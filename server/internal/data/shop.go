package data

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotFound                = errors.New("data: not found")
	ErrConflict                = errors.New("data: conflict")
	ErrStatus                  = errors.New("data: invalid status")
	ErrPayCode                 = errors.New("data: pay code required")
	ErrPayCodeCurrencyMismatch = errors.New("data: pay code currency does not match order currency")
	ErrInvalidCashReceived     = errors.New("data: cash received must cover the order total")
	ErrNotListed               = errors.New("data: not listed")
	ErrImageLimit              = errors.New("data: image limit")
	ErrInvalidBundle           = errors.New("data: invalid bundle")
	ErrBundleComponents        = errors.New("data: bundle components are not eligible")
	ErrEventNotOpen            = errors.New("data: event is not open for orders")
)

type Shop struct {
	ID              ID
	Name            string
	Notice          string
	Slug            string
	Enabled         bool
	FeaturedEventID ID
}

type ShopStore struct {
	pool *pgxpool.Pool
}

func NewPool(ctx context.Context, source string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(source)
	if err != nil {
		// Connection strings can contain credentials, so avoid echoing parser errors.
		return nil, fmt.Errorf("parse database source: invalid connection string")
	}
	cfg.MaxConns = 10
	cfg.MinConns = 0
	cfg.MaxConnLifetime = 30 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return pool, nil
}

func NewShopStore(pool *pgxpool.Pool) *ShopStore {
	return &ShopStore{pool: pool}
}

func (s *ShopStore) EnabledShop(ctx context.Context, slug string) (Shop, error) {
	var shop Shop
	var featuredEventID int64
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, name, notice, slug, COALESCE(featured_event_id, 0)
		FROM shops
		WHERE slug = $1 AND is_enabled`, slug).Scan(&shop.ID, &shop.Name, &shop.Notice, &shop.Slug, &featuredEventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Shop{}, ErrNotFound
	}
	if err != nil {
		return Shop{}, fmt.Errorf("query shop: %w", err)
	}
	shop.FeaturedEventID = ID(featuredEventID)
	return shop, nil
}

func (s *ShopStore) SetFeaturedEvent(ctx context.Context, shopID, eventID ID) error {
	if eventID != 0 {
		var exists bool
		err := s.pool.QueryRow(ctx, `SELECT true FROM events WHERE id = $1 AND shop_id = $2`, eventID, shopID).Scan(&exists)
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("check featured event: %w", err)
		}
	}
	value := any(nil)
	if eventID != 0 {
		value = int64(eventID)
	}
	_, err := s.pool.Exec(ctx, `UPDATE shops SET featured_event_id = $2, updated_at = now() WHERE id = $1`, int64(shopID), value)
	if err != nil {
		return fmt.Errorf("set featured event: %w", err)
	}
	return nil
}
