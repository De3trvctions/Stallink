package data

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

func (s *ShopStore) GoodImagePathForShop(ctx context.Context, shopID, goodID, imageID ID) (string, error) {
	var path string
	err := s.pool.QueryRow(ctx, `
		SELECT i.path
		FROM good_images i
		JOIN goods g ON g.id = i.good_id
		WHERE i.id = $1 AND g.id = $2 AND g.shop_id = $3`, imageID, goodID, shopID).Scan(&path)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("good image for shop: %w", err)
	}
	return path, nil
}

func (s *ShopStore) GoodImagePathForPlatform(ctx context.Context, goodID, imageID ID) (string, error) {
	var path string
	err := s.pool.QueryRow(ctx, `
		SELECT i.path
		FROM good_images i
		JOIN goods g ON g.id = i.good_id
		WHERE i.id = $1 AND g.id = $2`, imageID, goodID).Scan(&path)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("good image for platform: %w", err)
	}
	return path, nil
}

func (s *ShopStore) BundleImagePathForShop(ctx context.Context, shopID, bundleID, imageID ID) (string, error) {
	var imagePath string
	err := s.pool.QueryRow(ctx, `
		SELECT i.path FROM bundle_images i
		JOIN bundles b ON b.id = i.bundle_id AND b.shop_id = i.shop_id
		WHERE i.id = $1 AND b.id = $2 AND b.shop_id = $3`, imageID, bundleID, shopID).Scan(&imagePath)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("bundle image for shop: %w", err)
	}
	return imagePath, nil
}

func (s *ShopStore) BundleImagePathForPlatform(ctx context.Context, bundleID, imageID ID) (string, error) {
	var imagePath string
	err := s.pool.QueryRow(ctx, `SELECT path FROM bundle_images WHERE id = $1 AND bundle_id = $2`, imageID, bundleID).Scan(&imagePath)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("bundle image for platform: %w", err)
	}
	return imagePath, nil
}

func (s *ShopStore) PublicBundleImagePath(ctx context.Context, imageID ID) (string, error) {
	var imagePath string
	err := s.pool.QueryRow(ctx, `
		SELECT bi.path FROM bundle_images bi
		JOIN bundles b ON b.id = bi.bundle_id AND b.shop_id = bi.shop_id
		WHERE bi.id = $1 AND b.review_status = 'approved' AND EXISTS (
			SELECT 1 FROM event_bundle_listings eb
			JOIN shops s ON s.id = eb.shop_id AND s.is_enabled
			WHERE eb.bundle_id = b.id AND eb.shop_id = b.shop_id AND eb.is_listed
			AND NOT EXISTS (
				SELECT 1 FROM bundle_components bc
				JOIN goods g ON g.id = bc.good_id AND g.shop_id = bc.shop_id
				LEFT JOIN event_listings l ON l.good_id = g.id AND l.event_id = eb.event_id AND l.shop_id = eb.shop_id
				WHERE bc.bundle_id = b.id AND (g.review_status <> 'approved' OR l.id IS NULL OR NOT l.is_listed
					OR l.listed_qty - l.sold_qty - l.reserved_qty < bc.quantity)
			)
		)
		LIMIT 1`, imageID).Scan(&imagePath)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("public bundle image: %w", err)
	}
	return imagePath, nil
}
