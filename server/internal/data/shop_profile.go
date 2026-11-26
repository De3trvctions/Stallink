package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ShopProfileEvent is the event the shop-profile preview shows.
type ShopProfileEvent struct {
	ID          ID
	Name        string
	BoothNumber string
	Currency    string
	Status      string
}

// ShopProfile is the owner-facing view of a shop. Slug is platform-assigned
// and read-only for the owner.
type ShopProfile struct {
	ID              ID
	Name            string
	Notice          string
	Slug            string
	Enabled         bool
	Contact         string
	BoothNumber     string
	DefaultCurrency *string
	FeaturedEvent   *ShopProfileEvent
	OnSaleCount     int
	UpdatedAt       time.Time
}

// ShopProfileInput is every owner-editable shop field. It has no slug on
// purpose: the platform assigns it (K-Q3 = B, K-Q9 = A).
type ShopProfileInput struct {
	Name            string
	Notice          string
	Enabled         bool
	Contact         string
	BoothNumber     string
	DefaultCurrency *string
}

// ShopProfile reads the owner-facing profile in one statement so the preview
// event and its on-sale count come from the same snapshot. It does not filter
// on is_enabled: a disabled shop's sessions are already rejected by auth.
//
// The preview event is the featured event, else the first live one, else the
// first scheduled one; ended events never show. onSaleCount counts listings
// of that event that are listed, approved and still sellable from the shared
// pool (total_stock minus every listing's reserved_qty for the good). Bundles
// are not counted.
func (s *ShopStore) ShopProfile(ctx context.Context, shopID ID) (ShopProfile, error) {
	var p ShopProfile
	var evID *ID
	var evName, evBooth, evCurrency, evStatus *string
	err := s.pool.QueryRow(ctx, `
SELECT s.id, s.name, s.notice, s.slug, s.is_enabled, s.contact, s.booth_number, s.default_currency, s.updated_at,
  ev.id, ev.name, ev.booth_number, ev.currency, ev.status,
  COALESCE((
    SELECT count(*) FROM event_listings el
    JOIN goods g ON g.id = el.good_id AND g.shop_id = el.shop_id
    WHERE el.shop_id = s.id AND el.event_id = ev.id AND el.is_listed AND g.review_status = 'approved'
      AND g.total_stock > COALESCE((SELECT SUM(l2.reserved_qty) FROM event_listings l2 WHERE l2.good_id = g.id AND l2.shop_id = g.shop_id), 0)
  ), 0)
FROM shops s
LEFT JOIN LATERAL (
  SELECT e.id, e.name, e.booth_number, e.currency, e.status
  FROM events e
  WHERE e.shop_id = s.id AND e.status <> 'ended'
  ORDER BY (e.id IS NOT DISTINCT FROM s.featured_event_id) DESC, (e.status = 'live') DESC, e.starts_on, e.id
  LIMIT 1
) ev ON true
WHERE s.id = $1`, shopID).Scan(&p.ID, &p.Name, &p.Notice, &p.Slug, &p.Enabled, &p.Contact, &p.BoothNumber, &p.DefaultCurrency, &p.UpdatedAt,
		&evID, &evName, &evBooth, &evCurrency, &evStatus, &p.OnSaleCount)
	if errors.Is(err, pgx.ErrNoRows) {
		return ShopProfile{}, ErrNotFound
	}
	if err != nil {
		return ShopProfile{}, fmt.Errorf("query shop profile: %w", err)
	}
	if evID != nil {
		p.FeaturedEvent = &ShopProfileEvent{ID: *evID, Name: *evName, BoothNumber: *evBooth, Currency: *evCurrency, Status: *evStatus}
	}
	return p, nil
}

// UpdateShopProfile replaces every owner-editable field and never writes the
// slug. Disabling the shop revokes all its users' sessions in the same
// transaction, so a failed save revokes nothing (B-5).
func (s *ShopStore) UpdateShopProfile(ctx context.Context, shopID ID, in ShopProfileInput) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update shop profile: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `
		UPDATE shops SET name = $2, notice = $3, is_enabled = $4, contact = $5, booth_number = $6, default_currency = $7, updated_at = now()
		WHERE id = $1`, shopID, in.Name, in.Notice, in.Enabled, in.Contact, in.BoothNumber, in.DefaultCurrency)
	if err != nil {
		return fmt.Errorf("update shop profile: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if !in.Enabled {
		if _, err := tx.Exec(ctx, `DELETE FROM sessions WHERE user_id IN (SELECT id FROM users WHERE shop_id = $1)`, shopID); err != nil {
			return fmt.Errorf("revoke shop sessions: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit update shop profile: %w", err)
	}
	return nil
}
