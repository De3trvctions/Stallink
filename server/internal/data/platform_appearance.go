package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// PlatformAppearanceActor is who last published the platform theme.
type PlatformAppearanceActor struct {
	ID   ID
	Name string
}

// PlatformAppearanceMeta is the theme plus when and by whom it was last
// published. UpdatedAt and UpdatedBy are nil when it was never published (or
// the publisher's account is gone, for UpdatedBy).
type PlatformAppearanceMeta struct {
	Theme     string
	UpdatedAt *time.Time
	UpdatedBy *PlatformAppearanceActor
}

func (s *ShopStore) PlatformTheme(ctx context.Context) (string, error) {
	var theme string
	if err := s.pool.QueryRow(ctx, `SELECT COALESCE(
		(SELECT theme FROM platform_appearance WHERE singleton), 'classic'
	)`).Scan(&theme); err != nil {
		return "", fmt.Errorf("read platform theme: %w", err)
	}
	return theme, nil
}

// PlatformAppearanceMeta reads the theme with its publish metadata. The
// publisher name is the display name, falling back to the login name.
func (s *ShopStore) PlatformAppearanceMeta(ctx context.Context) (PlatformAppearanceMeta, error) {
	var meta PlatformAppearanceMeta
	var actorID *ID
	var actorName *string
	err := s.pool.QueryRow(ctx, `
		SELECT pa.theme, pa.updated_at, u.id, COALESCE(NULLIF(u.display_name, ''), u.login_name)
		FROM platform_appearance pa
		LEFT JOIN users u ON u.id = pa.updated_by
		WHERE pa.singleton`).Scan(&meta.Theme, &meta.UpdatedAt, &actorID, &actorName)
	if errors.Is(err, pgx.ErrNoRows) {
		return PlatformAppearanceMeta{Theme: "classic"}, nil
	}
	if err != nil {
		return PlatformAppearanceMeta{}, fmt.Errorf("read platform appearance meta: %w", err)
	}
	if actorID != nil && actorName != nil {
		meta.UpdatedBy = &PlatformAppearanceActor{ID: *actorID, Name: *actorName}
	}
	return meta, nil
}

// SetPlatformTheme publishes a theme. Every call is a publish, even with an
// unchanged theme, so updated_at and updated_by are always overwritten.
func (s *ShopStore) SetPlatformTheme(ctx context.Context, theme string, actorID ID) error {
	if _, err := s.pool.Exec(ctx, `INSERT INTO platform_appearance (singleton, theme, updated_at, updated_by)
		VALUES (true, $1, now(), $2)
		ON CONFLICT (singleton) DO UPDATE SET theme = EXCLUDED.theme, updated_at = now(), updated_by = EXCLUDED.updated_by`, theme, actorID); err != nil {
		return fmt.Errorf("set platform theme: %w", err)
	}
	return nil
}
