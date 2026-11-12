package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Account struct {
	ID           ID
	LoginName    string
	Role         string
	ShopID       *ID
	ShopSlug     *string
	DisplayName  string
	Contact      string
	PasswordHash string
	Enabled      bool
}

func (s *ShopStore) CreateAccount(ctx context.Context, login, hash, role string, shopID *ID, display, contact string) (ID, error) {
	var id ID
	err := s.pool.QueryRow(ctx, `
		INSERT INTO users (login_name, password_hash, role, shop_id, display_name, contact)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id::text`, login, hash, role, shopID, display, contact).Scan(&id)
	if isUnique(err) {
		return 0, ErrConflict
	}
	if err != nil {
		return 0, fmt.Errorf("create account: %w", err)
	}
	return id, nil
}

func (s *ShopStore) AccountByLogin(ctx context.Context, login string) (Account, error) {
	return s.scanAccount(s.pool.QueryRow(ctx, accountSelect+` WHERE u.login_name = $1`, login))
}

func (s *ShopStore) InsertSession(ctx context.Context, userID ID, tokenHash []byte, expires time.Time) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`, userID, tokenHash, expires)
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *ShopStore) DeleteSession(ctx context.Context, tokenHash []byte) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE token_hash = $1`, tokenHash)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func (s *ShopStore) AccountByToken(ctx context.Context, tokenHash []byte) (Account, error) {
	return s.scanAccount(s.pool.QueryRow(ctx, accountSelect+`
		JOIN sessions sess ON sess.user_id = u.id
		WHERE sess.token_hash = $1 AND sess.expires_at > now() AND u.is_enabled`, tokenHash))
}

func (s *ShopStore) UpdateProfile(ctx context.Context, userID ID, display, contact string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE users SET display_name = $2, contact = $3
		WHERE id = $1 AND role = 'customer'`, userID, display, contact)
	if err != nil {
		return fmt.Errorf("update profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ShopStore) SetSalesEnabled(ctx context.Context, shopID, userID ID, enabled bool) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE users SET is_enabled = $3
		WHERE id = $1 AND shop_id = $2 AND role = 'sales'`, userID, shopID, enabled)
	if err != nil {
		return fmt.Errorf("set sales enabled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetSalesPassword updates the credential and revokes existing sessions atomically.
func (s *ShopStore) SetSalesPassword(ctx context.Context, shopID, userID ID, hash string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin sales password reset: %w", err)
	}
	defer tx.Rollback(ctx)
	tag, err := tx.Exec(ctx, `
		UPDATE users SET password_hash = $3
		WHERE id = $1 AND shop_id = $2 AND role = 'sales'`, userID, shopID, hash)
	if err != nil {
		return fmt.Errorf("set sales password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("revoke sales sessions after password reset: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit sales password reset: %w", err)
	}
	return nil
}

func (s *ShopStore) ListSales(ctx context.Context, shopID ID) ([]Account, error) {
	rows, err := s.pool.Query(ctx, accountSelect+` WHERE u.shop_id = $1 AND u.role = 'sales' ORDER BY u.login_name`, shopID)
	if err != nil {
		return nil, fmt.Errorf("list sales: %w", err)
	}
	defer rows.Close()
	var out []Account
	for rows.Next() {
		account, err := scanAccountRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, account)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list sales: %w", err)
	}
	return out, nil
}

func (s *ShopStore) HasPlatformAdmin(ctx context.Context) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users WHERE role = 'platform_admin'`).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("count platform admin: %w", err)
	}
	return n > 0, nil
}

const accountSelect = `
SELECT u.id::text, u.login_name, u.role, u.shop_id::text, s.slug, u.display_name, u.contact, u.password_hash, u.is_enabled
FROM users u
LEFT JOIN shops s ON s.id = u.shop_id`

func (s *ShopStore) scanAccount(row pgx.Row) (Account, error) {
	account, err := scanAccountRow(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Account{}, ErrNotFound
	}
	if err != nil {
		return Account{}, fmt.Errorf("query account: %w", err)
	}
	return account, nil
}

func scanAccountRow(row pgx.Row) (Account, error) {
	var account Account
	var shopID *ID
	var shopSlug *string
	err := row.Scan(&account.ID, &account.LoginName, &account.Role, &shopID, &shopSlug, &account.DisplayName, &account.Contact, &account.PasswordHash, &account.Enabled)
	if err != nil {
		return Account{}, err
	}
	account.ShopID = shopID
	account.ShopSlug = shopSlug
	return account, nil
}

func isUnique(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
