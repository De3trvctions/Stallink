package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/De3trvctions/Stallink/internal/catalog"
	"github.com/jackc/pgx/v5"
)

type Event struct {
	ID          ID
	Name        string
	StartsOn    time.Time
	EndsOn      time.Time
	City        string
	Venue       string
	BoothNumber string
	Currency    string
	Status      string
}

type GoodCard struct {
	ListingID      string
	GoodID         string
	EventID        string
	EventName      string
	Name           string
	Description    string
	Currency       string
	UnitPriceMinor int64
	PriceTiers     []catalog.PriceTier
	TotalStock     int
	ListedQty      int
	SoldQty        int
	ReservedQty    int
	Remaining      int
	IsListed       bool
	ReviewStatus   string
	ReviewNote     string
	ImageIDs       []string
}

type EventInput struct {
	Name, StartsOn, EndsOn, City, Venue, BoothNumber, Currency, Status string
}

type ListedGood struct {
	Name, Description string
	TotalStock        int
	ImagePaths        []string
	EventID           ID
	NewEvent          *EventInput
	ListedQty         int
	UnitPriceMinor    int64
	PriceTiers        []catalog.PriceTier
	IsListed          bool
	Approved          bool
}

type PlatformShopGoods struct {
	ShopID   string
	ShopName string
	ShopSlug string
	Goods    []PlatformGood
}

type PlatformGood struct {
	ID           string
	Name         string
	Description  string
	ImageIDs     []string
	ReviewStatus string
	ReviewNote   string
	TotalStock   int
	Listings     []PlatformListing
}

type PlatformListing struct {
	ListingID      string
	EventID        string
	EventName      string
	Currency       string
	UnitPriceMinor int64
	PriceTiers     []catalog.PriceTier
	ListedQty      int
	SoldQty        int
	ReservedQty    int
	IsListed       bool
}

func (s *ShopStore) SoleEnabledShop(ctx context.Context) (Shop, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, notice, slug FROM shops WHERE is_enabled ORDER BY created_at`)
	if err != nil {
		return Shop{}, fmt.Errorf("list enabled shops: %w", err)
	}
	defer rows.Close()
	var shops []Shop
	for rows.Next() {
		var shop Shop
		if err := rows.Scan(&shop.ID, &shop.Name, &shop.Notice, &shop.Slug); err != nil {
			return Shop{}, fmt.Errorf("scan shop: %w", err)
		}
		shops = append(shops, shop)
	}
	if err := rows.Err(); err != nil {
		return Shop{}, fmt.Errorf("list enabled shops: %w", err)
	}
	if len(shops) != 1 {
		return Shop{}, ErrNotFound
	}
	return shops[0], nil
}

func (s *ShopStore) ListShops(ctx context.Context) ([]Shop, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, notice, slug, is_enabled FROM shops ORDER BY created_at`)
	if err != nil {
		return nil, fmt.Errorf("list shops: %w", err)
	}
	defer rows.Close()
	var shops []Shop
	for rows.Next() {
		var shop Shop
		if err := rows.Scan(&shop.ID, &shop.Name, &shop.Notice, &shop.Slug, &shop.Enabled); err != nil {
			return nil, fmt.Errorf("scan shop: %w", err)
		}
		shops = append(shops, shop)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list shops: %w", err)
	}
	return shops, nil
}

func (s *ShopStore) UpdateShop(ctx context.Context, shopID ID, name, notice, slug string, enabled bool) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE shops SET name = $2, notice = $3, slug = $4, is_enabled = $5, updated_at = now()
		WHERE id = $1`, shopID, name, notice, slug, enabled)
	if isUnique(err) {
		return ErrConflict
	}
	if err != nil {
		return fmt.Errorf("update shop: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ShopStore) SetShopEnabled(ctx context.Context, shopID ID, enabled bool) error {
	tag, err := s.pool.Exec(ctx, `UPDATE shops SET is_enabled = $2, updated_at = now() WHERE id = $1`, shopID, enabled)
	if err != nil {
		return fmt.Errorf("set shop enabled: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *ShopStore) OpenShop(ctx context.Context, name, slug, notice, adminLogin, adminHash string) (ID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin open shop: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var shopID ID
	err = tx.QueryRow(ctx, `
		INSERT INTO shops (name, slug, notice) VALUES ($1, $2, $3) RETURNING id::text`, name, slug, notice).Scan(&shopID)
	if isUnique(err) {
		return 0, ErrConflict
	}
	if err != nil {
		return 0, fmt.Errorf("insert shop: %w", err)
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO users (login_name, password_hash, role, shop_id) VALUES ($1, $2, 'shop_admin', $3)`,
		adminLogin, adminHash, shopID)
	if isUnique(err) {
		return 0, ErrConflict
	}
	if err != nil {
		return 0, fmt.Errorf("insert shop admin: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit open shop: %w", err)
	}
	return shopID, nil
}

func (s *ShopStore) CreateEvent(ctx context.Context, shopID ID, in EventInput) (ID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin create event: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return 0, err
	}
	var id ID
	err = tx.QueryRow(ctx, `
		INSERT INTO events (shop_id, name, starts_on, ends_on, city, venue, booth_number, currency, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id::text`,
		shopID, in.Name, in.StartsOn, in.EndsOn, in.City, in.Venue, in.BoothNumber, in.Currency, in.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("create event: %w", err)
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit event: %w", err)
	}
	return id, nil
}

func (s *ShopStore) UpdateEvent(ctx context.Context, shopID, eventID ID, in EventInput) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update event: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE events SET name=$3, starts_on=$4, ends_on=$5, city=$6, venue=$7, booth_number=$8, currency=$9, status=$10
		WHERE id=$1 AND shop_id=$2`,
		eventID, shopID, in.Name, in.StartsOn, in.EndsOn, in.City, in.Venue, in.BoothNumber, in.Currency, in.Status)
	if err != nil {
		return fmt.Errorf("update event: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *ShopStore) ListEvents(ctx context.Context, shopID ID) ([]Event, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, starts_on, ends_on, city, venue, booth_number, currency, status
		FROM events WHERE shop_id = $1 ORDER BY starts_on, name`, shopID)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()
	return collectEvents(rows)
}

func (s *ShopStore) PublicCatalog(ctx context.Context, shopID ID) ([]Event, []GoodCard, error) {
	events, err := s.ListEvents(ctx, shopID)
	if err != nil {
		return nil, nil, err
	}
	goods, err := s.publicGoods(ctx, shopID, 0)
	if err != nil {
		return nil, nil, err
	}
	return events, goods, nil
}

func (s *ShopStore) PublicEventGoods(ctx context.Context, shopID, eventID ID) (Event, []GoodCard, error) {
	var event Event
	err := s.pool.QueryRow(ctx, `
		SELECT id::text, name, starts_on, ends_on, city, venue, booth_number, currency, status
		FROM events WHERE id = $1 AND shop_id = $2`, eventID, shopID).
		Scan(&event.ID, &event.Name, &event.StartsOn, &event.EndsOn, &event.City, &event.Venue, &event.BoothNumber, &event.Currency, &event.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return Event{}, nil, ErrNotFound
	}
	if err != nil {
		return Event{}, nil, fmt.Errorf("get event: %w", err)
	}
	goods, err := s.publicGoods(ctx, shopID, eventID)
	if err != nil {
		return Event{}, nil, err
	}
	return event, goods, nil
}

func (s *ShopStore) PublicListing(ctx context.Context, shopID, listingID ID) (GoodCard, error) {
	goods, err := s.publicGoods(ctx, shopID, 0)
	if err != nil {
		return GoodCard{}, err
	}
	for _, good := range goods {
		if good.ListingID == listingID.String() {
			return good, nil
		}
	}
	return GoodCard{}, ErrNotFound
}

func (s *ShopStore) publicGoods(ctx context.Context, shopID, eventID ID) ([]GoodCard, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT l.id::text, l.good_id::text, e.id::text, e.name, g.name, g.description, e.currency,
			l.unit_price_minor, l.price_tiers, l.listed_qty, l.sold_qty, l.reserved_qty, l.is_listed,
			g.review_status, g.review_note, g.total_stock,
			COALESCE((SELECT array_agg(i.id::text ORDER BY i.sort_order) FROM good_images i WHERE i.good_id = g.id), ARRAY[]::text[])
		FROM event_listings l
		JOIN goods g ON g.id = l.good_id
		JOIN events e ON e.id = l.event_id
		WHERE e.shop_id = $1 AND ($2 = 0 OR e.id = $2) AND l.is_listed AND l.listed_qty > l.sold_qty + l.reserved_qty
			AND g.review_status = 'approved'
			AND e.status IN ('live', 'scheduled') AND e.ends_on >= CURRENT_DATE
		ORDER BY e.starts_on, g.name`, shopID, eventID)
	if err != nil {
		return nil, fmt.Errorf("list public goods: %w", err)
	}
	defer rows.Close()
	return collectGoods(rows)
}

func (s *ShopStore) StaffListings(ctx context.Context, shopID, eventID ID) ([]GoodCard, error) {
	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT true FROM events WHERE id = $1 AND shop_id = $2`, eventID, shopID).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("staff event: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT l.id::text, l.good_id::text, e.id::text, e.name, g.name, g.description, e.currency,
			l.unit_price_minor, l.price_tiers, l.listed_qty, l.sold_qty, l.reserved_qty, l.is_listed,
			g.review_status, g.review_note, g.total_stock,
			COALESCE((SELECT array_agg(i.id::text ORDER BY i.sort_order) FROM good_images i WHERE i.good_id = g.id), ARRAY[]::text[])
		FROM event_listings l
		JOIN goods g ON g.id = l.good_id
		JOIN events e ON e.id = l.event_id
		WHERE e.id = $1 AND e.shop_id = $2
		ORDER BY g.name`, eventID, shopID)
	if err != nil {
		return nil, fmt.Errorf("staff listings: %w", err)
	}
	defer rows.Close()
	return collectGoods(rows)
}

func (s *ShopStore) CreateListedGood(ctx context.Context, shopID, actorID ID, in ListedGood) (goodID, listingID, eventID ID, err error) {
	if len(in.ImagePaths) > 8 {
		return 0, 0, 0, ErrImageLimit
	}
	if err := catalog.ValidatePriceTiers(in.UnitPriceMinor, in.PriceTiers); err != nil {
		return 0, 0, 0, err
	}
	tiers, err := marshalPriceTiers(in.PriceTiers)
	if err != nil {
		return 0, 0, 0, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("begin good: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return 0, 0, 0, err
	}
	eventID = in.EventID
	if in.NewEvent != nil {
		err = tx.QueryRow(ctx, `
			INSERT INTO events (shop_id, name, starts_on, ends_on, city, venue, booth_number, currency, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id::text`,
			shopID, in.NewEvent.Name, in.NewEvent.StartsOn, in.NewEvent.EndsOn, in.NewEvent.City, in.NewEvent.Venue, in.NewEvent.BoothNumber, in.NewEvent.Currency, in.NewEvent.Status).Scan(&eventID)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("create event for good: %w", err)
		}
	} else {
		var id string
		err = tx.QueryRow(ctx, `SELECT id::text FROM events WHERE id = $1 AND shop_id = $2`, eventID, shopID).Scan(&id)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, 0, ErrNotFound
		}
		if err != nil {
			return 0, 0, 0, fmt.Errorf("check event: %w", err)
		}
	}
	reviewStatus := "pending"
	if in.Approved {
		reviewStatus = "approved"
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO goods (shop_id, name, description, total_stock, review_status, reviewed_at)
		VALUES ($1,$2,$3,$4,$5,CASE WHEN $5 = 'approved' THEN now() ELSE NULL END) RETURNING id::text`,
		shopID, in.Name, in.Description, in.TotalStock, reviewStatus).Scan(&goodID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("insert good: %w", err)
	}
	for sortOrder, imagePath := range in.ImagePaths {
		if _, err := tx.Exec(ctx, `INSERT INTO good_images (good_id, path, sort_order) VALUES ($1, $2, $3)`, goodID, imagePath, sortOrder); err != nil {
			return 0, 0, 0, fmt.Errorf("insert good image: %w", err)
		}
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO event_listings (shop_id, event_id, good_id, listed_qty, unit_price_minor, price_tiers, is_listed)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id::text`,
		shopID, eventID, goodID, in.ListedQty, in.UnitPriceMinor, tiers, in.IsListed).Scan(&listingID)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("insert listing: %w", err)
	}
	if err := assertAllocation(ctx, tx, goodID); err != nil {
		return 0, 0, 0, err
	}
	if err := insertAudit(ctx, tx, shopID, actorID, "listing.create", "listing", listingID); err != nil {
		return 0, 0, 0, err
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return 0, 0, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, 0, fmt.Errorf("commit good: %w", err)
	}
	return goodID, listingID, eventID, nil
}

func (s *ShopStore) AddGoodImages(ctx context.Context, shopID, actorID, goodID ID, paths []string) ([]string, error) {
	if len(paths) == 0 {
		return []string{}, nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin good images: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return nil, err
	}
	var owned string
	err = tx.QueryRow(ctx, `SELECT id::text FROM goods WHERE id = $1 AND shop_id = $2 FOR UPDATE`, goodID, shopID).Scan(&owned)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("check good for images: %w", err)
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM good_images WHERE good_id = $1`, goodID).Scan(&count); err != nil {
		return nil, fmt.Errorf("count good images: %w", err)
	}
	if count+len(paths) > 8 {
		return nil, ErrImageLimit
	}
	ids := make([]string, 0, len(paths))
	for index, path := range paths {
		var id string
		if err := tx.QueryRow(ctx, `
			INSERT INTO good_images (good_id, path, sort_order) VALUES ($1, $2, $3) RETURNING id::text`,
			goodID, path, count+index).Scan(&id); err != nil {
			return nil, fmt.Errorf("insert good image: %w", err)
		}
		ids = append(ids, id)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE goods SET review_status = 'pending', review_note = '', reviewed_at = NULL WHERE id = $1`, goodID); err != nil {
		return nil, fmt.Errorf("reset good image review: %w", err)
	}
	if err := insertAudit(ctx, tx, shopID, actorID, "good.image.add", "good", goodID); err != nil {
		return nil, err
	}
	if err := invalidateBundlesForGood(ctx, tx, shopID, goodID); err != nil {
		return nil, err
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit good images: %w", err)
	}
	return ids, nil
}

func (s *ShopStore) DeleteGoodImage(ctx context.Context, shopID, actorID, goodID, imageID ID) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("begin good image delete: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return "", err
	}
	var owned string
	err = tx.QueryRow(ctx, `SELECT id::text FROM goods WHERE id = $1 AND shop_id = $2 FOR UPDATE`, goodID, shopID).Scan(&owned)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("check good for image delete: %w", err)
	}
	var path string
	var sortOrder int
	err = tx.QueryRow(ctx, `
		SELECT path, sort_order FROM good_images WHERE id = $1 AND good_id = $2 FOR UPDATE`, imageID, goodID).Scan(&path, &sortOrder)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("find good image: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM good_images WHERE id = $1`, imageID); err != nil {
		return "", fmt.Errorf("delete good image: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE good_images SET sort_order = sort_order + 100 WHERE good_id = $1 AND sort_order > $2`, goodID, sortOrder); err != nil {
		return "", fmt.Errorf("stage good image order: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE good_images SET sort_order = sort_order - 101 WHERE good_id = $1 AND sort_order > $2`, goodID, sortOrder+100); err != nil {
		return "", fmt.Errorf("compact good image order: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE goods SET review_status = 'pending', review_note = '', reviewed_at = NULL WHERE id = $1`, goodID); err != nil {
		return "", fmt.Errorf("reset good image review: %w", err)
	}
	if err := insertAudit(ctx, tx, shopID, actorID, "good.image.delete", "good", goodID); err != nil {
		return "", err
	}
	if err := invalidateBundlesForGood(ctx, tx, shopID, goodID); err != nil {
		return "", err
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("commit good image delete: %w", err)
	}
	return path, nil
}

func (s *ShopStore) ReorderGoodImages(ctx context.Context, shopID, actorID, goodID ID, imageIDs []ID) error {
	if len(imageIDs) > 8 {
		return ErrImageLimit
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin good image reorder: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return err
	}
	var owned string
	err = tx.QueryRow(ctx, `SELECT id::text FROM goods WHERE id = $1 AND shop_id = $2 FOR UPDATE`, goodID, shopID).Scan(&owned)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("check good for image reorder: %w", err)
	}
	rows, err := tx.Query(ctx, `SELECT id::text FROM good_images WHERE good_id = $1 ORDER BY sort_order`, goodID)
	if err != nil {
		return fmt.Errorf("list good images: %w", err)
	}
	existing := make([]ID, 0, len(imageIDs))
	for rows.Next() {
		var id ID
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return fmt.Errorf("scan good image: %w", err)
		}
		existing = append(existing, id)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read good images: %w", err)
	}
	rows.Close()
	if len(existing) != len(imageIDs) || len(imageIDs) > 8 {
		return ErrNotFound
	}
	existingIDs := make(map[ID]struct{}, len(existing))
	for _, id := range existing {
		existingIDs[id] = struct{}{}
	}
	seen := make(map[ID]struct{}, len(imageIDs))
	changed := false
	for index, id := range imageIDs {
		if _, ok := existingIDs[id]; !ok {
			return ErrNotFound
		}
		if _, ok := seen[id]; ok {
			return ErrNotFound
		}
		seen[id] = struct{}{}
		if existing[index] != id {
			changed = true
		}
	}
	if !changed {
		return tx.Commit(ctx)
	}
	if _, err := tx.Exec(ctx, `UPDATE good_images SET sort_order = sort_order + 100 WHERE good_id = $1`, goodID); err != nil {
		return fmt.Errorf("stage good image order: %w", err)
	}
	for index, id := range imageIDs {
		if _, err := tx.Exec(ctx, `UPDATE good_images SET sort_order = $3 WHERE good_id = $1 AND id = $2`, goodID, id, index); err != nil {
			return fmt.Errorf("update good image order: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE goods SET review_status = 'pending', review_note = '', reviewed_at = NULL WHERE id = $1`, goodID); err != nil {
		return fmt.Errorf("reset good image review: %w", err)
	}
	if err := invalidateBundlesForGood(ctx, tx, shopID, goodID); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, shopID, actorID, "good.image.reorder", "good", goodID); err != nil {
		return err
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *ShopStore) AddListing(ctx context.Context, shopID, actorID, goodID ID, in ListedGood) (ID, error) {
	if err := catalog.ValidatePriceTiers(in.UnitPriceMinor, in.PriceTiers); err != nil {
		return 0, err
	}
	tiers, err := marshalPriceTiers(in.PriceTiers)
	if err != nil {
		return 0, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin listing: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return 0, err
	}
	var owned string
	err = tx.QueryRow(ctx, `SELECT id::text FROM goods WHERE id = $1 AND shop_id = $2`, goodID, shopID).Scan(&owned)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("check good: %w", err)
	}
	var eventID ID
	err = tx.QueryRow(ctx, `SELECT id::text FROM events WHERE id = $1 AND shop_id = $2`, in.EventID, shopID).Scan(&eventID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("check event: %w", err)
	}
	var listingID ID
	err = tx.QueryRow(ctx, `
		INSERT INTO event_listings (shop_id, event_id, good_id, listed_qty, unit_price_minor, price_tiers, is_listed)
		VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id::text`,
		shopID, eventID, goodID, in.ListedQty, in.UnitPriceMinor, tiers, in.IsListed).Scan(&listingID)
	if isUnique(err) {
		return 0, ErrConflict
	}
	if err != nil {
		return 0, fmt.Errorf("insert listing: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE goods SET review_status = 'pending', review_note = '', reviewed_at = NULL WHERE id = $1`, goodID); err != nil {
		return 0, fmt.Errorf("reset good review: %w", err)
	}
	if err := invalidateBundlesForGood(ctx, tx, shopID, goodID); err != nil {
		return 0, err
	}
	if err := assertAllocation(ctx, tx, goodID); err != nil {
		return 0, err
	}
	if err := insertAudit(ctx, tx, shopID, actorID, "listing.create", "listing", listingID); err != nil {
		return 0, err
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit listing: %w", err)
	}
	return listingID, nil
}

func (s *ShopStore) UpdateListing(ctx context.Context, shopID, actorID, listingID ID, listedQty int, price int64, tiers []catalog.PriceTier, isListed bool) error {
	if err := catalog.ValidatePriceTiers(price, tiers); err != nil {
		return err
	}
	encodedTiers, err := marshalPriceTiers(tiers)
	if err != nil {
		return err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update listing: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return err
	}
	var goodID ID
	var sold int
	var reserved int
	var oldListed bool
	var oldPrice int64
	var oldTiers []byte
	err = tx.QueryRow(ctx, `
		SELECT l.good_id::text, l.sold_qty, l.reserved_qty, l.is_listed, l.unit_price_minor, l.price_tiers
		FROM event_listings l
		JOIN events e ON e.id = l.event_id
		WHERE l.id = $1 AND e.shop_id = $2
		FOR UPDATE OF l`, listingID, shopID).Scan(&goodID, &sold, &reserved, &oldListed, &oldPrice, &oldTiers)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock listing: %w", err)
	}
	if listedQty < sold+reserved {
		return catalog.ErrStockExceeded
	}
	priceChanged := oldPrice != price || string(oldTiers) != string(encodedTiers)
	eligibilityChanged := oldListed != isListed
	_, err = tx.Exec(ctx, `
		UPDATE event_listings SET listed_qty = $2, unit_price_minor = $3, price_tiers = $4, is_listed = $5, revision = revision + 1 WHERE id = $1`,
		listingID, listedQty, price, encodedTiers, isListed)
	if err != nil {
		return fmt.Errorf("update listing: %w", err)
	}
	if priceChanged {
		if _, err := tx.Exec(ctx, `UPDATE goods SET review_status = 'pending', review_note = '', reviewed_at = NULL WHERE id = $1`, goodID); err != nil {
			return fmt.Errorf("reset good review: %w", err)
		}
	}
	if priceChanged || eligibilityChanged {
		if err := invalidateBundlesForGood(ctx, tx, shopID, goodID); err != nil {
			return err
		}
	}
	if err := assertAllocation(ctx, tx, goodID); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, shopID, actorID, "listing.update", "listing", listingID); err != nil {
		return err
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *ShopStore) UpdateGoodStock(ctx context.Context, shopID, actorID, goodID ID, name, description string, total int) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin stock: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return err
	}
	var oldName, oldDescription string
	err = tx.QueryRow(ctx, `SELECT name, description FROM goods WHERE id = $1 AND shop_id = $2 FOR UPDATE`, goodID, shopID).Scan(&oldName, &oldDescription)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock good: %w", err)
	}
	metadataChanged := oldName != name || oldDescription != description
	tag, err := tx.Exec(ctx, `
		UPDATE goods SET
			name = $3,
			description = $4,
			total_stock = $5,
			review_status = CASE WHEN name IS DISTINCT FROM $3 OR description IS DISTINCT FROM $4 THEN 'pending' ELSE review_status END,
			review_note = CASE WHEN name IS DISTINCT FROM $3 OR description IS DISTINCT FROM $4 THEN '' ELSE review_note END,
			reviewed_at = CASE WHEN name IS DISTINCT FROM $3 OR description IS DISTINCT FROM $4 THEN NULL ELSE reviewed_at END
		WHERE id = $1 AND shop_id = $2`,
		goodID, shopID, name, description, total)
	if err != nil {
		return fmt.Errorf("update good: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	if metadataChanged {
		if err := invalidateBundlesForGood(ctx, tx, shopID, goodID); err != nil {
			return err
		}
	}
	if err := assertAllocation(ctx, tx, goodID); err != nil {
		return err
	}
	if err := insertAudit(ctx, tx, shopID, actorID, "stock.update", "good", goodID); err != nil {
		return err
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *ShopStore) ListPlatformGoods(ctx context.Context) ([]PlatformShopGoods, error) {
	shops, err := s.ListShops(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PlatformShopGoods, 0, len(shops))
	shopIndex := make(map[string]int, len(shops))
	for _, shop := range shops {
		shopIndex[shop.ID.String()] = len(out)
		out = append(out, PlatformShopGoods{
			ShopID: shop.ID.String(), ShopName: shop.Name, ShopSlug: shop.Slug, Goods: []PlatformGood{},
		})
	}

	rows, err := s.pool.Query(ctx, `
		SELECT s.id::text, COALESCE(g.id::text, ''), COALESCE(g.name, ''), COALESCE(g.description, ''),
			COALESCE(g.review_status, ''), COALESCE(g.review_note, ''), COALESCE(g.total_stock, 0),
			COALESCE((SELECT array_agg(i.id::text ORDER BY i.sort_order) FROM good_images i WHERE i.good_id = g.id), ARRAY[]::text[]),
			COALESCE(l.id::text, ''), COALESCE(e.id::text, ''), COALESCE(e.name, ''),
			COALESCE(e.currency, ''), COALESCE(l.unit_price_minor, 0),
			COALESCE(l.price_tiers, '[]'::jsonb), COALESCE(l.listed_qty, 0), COALESCE(l.sold_qty, 0), COALESCE(l.reserved_qty, 0),
			COALESCE(l.is_listed, false)
		FROM shops s
		LEFT JOIN goods g ON g.shop_id = s.id
		LEFT JOIN event_listings l ON l.good_id = g.id
		LEFT JOIN events e ON e.id = l.event_id
		ORDER BY s.created_at, s.id, g.created_at, g.id, e.starts_on, e.id`)
	if err != nil {
		return nil, fmt.Errorf("list platform goods: %w", err)
	}
	defer rows.Close()
	type goodLocation struct {
		shopIndex int
		goodIndex int
	}
	goodsByID := make(map[string]goodLocation)
	for rows.Next() {
		var shopID, goodID string
		var goodName, description, reviewStatus, reviewNote string
		var imageIDs []string
		var totalStock int
		var listing PlatformListing
		var tiers []byte
		if err := rows.Scan(
			&shopID, &goodID, &goodName, &description, &reviewStatus, &reviewNote, &totalStock, &imageIDs,
			&listing.ListingID, &listing.EventID, &listing.EventName, &listing.Currency,
			&listing.UnitPriceMinor, &tiers, &listing.ListedQty, &listing.SoldQty, &listing.ReservedQty, &listing.IsListed,
		); err != nil {
			return nil, fmt.Errorf("scan platform good: %w", err)
		}
		index, exists := shopIndex[shopID]
		if !exists {
			continue
		}
		location, exists := goodsByID[goodID]
		if !exists {
			if goodID == "" {
				continue
			}
			shop := &out[index]
			shop.Goods = append(shop.Goods, PlatformGood{
				ID: goodID, Name: goodName, Description: description, ReviewStatus: reviewStatus,
				ReviewNote: reviewNote, TotalStock: totalStock, ImageIDs: imageIDs, Listings: []PlatformListing{},
			})
			location = goodLocation{shopIndex: index, goodIndex: len(shop.Goods) - 1}
			goodsByID[goodID] = location
		}
		if listing.ListingID == "" {
			continue
		}
		if err := json.Unmarshal(tiers, &listing.PriceTiers); err != nil {
			return nil, fmt.Errorf("decode platform price tiers: %w", err)
		}
		good := &out[location.shopIndex].Goods[location.goodIndex]
		good.Listings = append(good.Listings, listing)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list platform goods: %w", err)
	}
	return out, nil
}

func (s *ShopStore) ReviewGood(ctx context.Context, actorID, goodID ID, status, note string) error {
	if status != "approved" && status != "rejected" {
		return ErrNotFound
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin good review: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var shopID ID
	err = tx.QueryRow(ctx, `SELECT shop_id::text FROM goods WHERE id = $1`, goodID).Scan(&shopID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("find good for review: %w", err)
	}
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return err
	}
	var owned string
	err = tx.QueryRow(ctx, `SELECT id::text FROM goods WHERE id = $1 AND shop_id = $2 FOR UPDATE`, goodID, shopID).Scan(&owned)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock good for review: %w", err)
	}
	if status == "approved" {
		note = ""
	}
	if _, err := tx.Exec(ctx, `
		UPDATE goods SET review_status = $2, review_note = $3, reviewed_at = now()
		WHERE id = $1`, goodID, status, note); err != nil {
		return fmt.Errorf("update good review: %w", err)
	}
	action := "good.approve"
	if status == "rejected" {
		action = "good.reject"
	}
	if err := insertAudit(ctx, tx, shopID, actorID, action, "good", goodID); err != nil {
		return err
	}
	if err := invalidateBundlesForGood(ctx, tx, shopID, goodID); err != nil {
		return err
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit good review: %w", err)
	}
	return nil
}

func assertAllocation(ctx context.Context, tx pgx.Tx, goodID ID) error {
	var total int
	if err := tx.QueryRow(ctx, `SELECT total_stock FROM goods WHERE id = $1`, goodID).Scan(&total); err != nil {
		return fmt.Errorf("read stock: %w", err)
	}
	rows, err := tx.Query(ctx, `SELECT listed_qty, sold_qty FROM event_listings WHERE good_id = $1`, goodID)
	if err != nil {
		return fmt.Errorf("read listings: %w", err)
	}
	defer rows.Close()
	var listings []catalog.ListingQty
	for rows.Next() {
		var item catalog.ListingQty
		if err := rows.Scan(&item.Listed, &item.Sold); err != nil {
			return fmt.Errorf("scan listing qty: %w", err)
		}
		listings = append(listings, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read listings: %w", err)
	}
	if err := catalog.ValidateAllocation(total, listings); err != nil {
		return err
	}
	return nil
}

func collectEvents(rows pgx.Rows) ([]Event, error) {
	var out []Event
	for rows.Next() {
		var event Event
		if err := rows.Scan(&event.ID, &event.Name, &event.StartsOn, &event.EndsOn, &event.City, &event.Venue, &event.BoothNumber, &event.Currency, &event.Status); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		out = append(out, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	return out, nil
}

func collectGoods(rows pgx.Rows) ([]GoodCard, error) {
	var out []GoodCard
	for rows.Next() {
		var good GoodCard
		var tiers []byte
		if err := rows.Scan(&good.ListingID, &good.GoodID, &good.EventID, &good.EventName, &good.Name, &good.Description, &good.Currency, &good.UnitPriceMinor, &tiers, &good.ListedQty, &good.SoldQty, &good.ReservedQty, &good.IsListed, &good.ReviewStatus, &good.ReviewNote, &good.TotalStock, &good.ImageIDs); err != nil {
			return nil, fmt.Errorf("scan good: %w", err)
		}
		good.Remaining = good.ListedQty - good.SoldQty - good.ReservedQty
		if err := json.Unmarshal(tiers, &good.PriceTiers); err != nil {
			return nil, fmt.Errorf("decode price tiers: %w", err)
		}
		out = append(out, good)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list goods: %w", err)
	}
	return out, nil
}

func marshalPriceTiers(tiers []catalog.PriceTier) ([]byte, error) {
	if tiers == nil {
		tiers = []catalog.PriceTier{}
	}
	encoded, err := json.Marshal(tiers)
	if err != nil {
		return nil, fmt.Errorf("encode price tiers: %w", err)
	}
	return encoded, nil
}

func insertAudit(ctx context.Context, tx pgx.Tx, shopID, actorID ID, action, subjectType string, subjectID ID) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO audit_logs (shop_id, actor_id, action, subject_type, subject_id)
		VALUES ($1,$2,$3,$4,$5)`, shopID, actorID, action, subjectType, subjectID)
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	return nil
}
