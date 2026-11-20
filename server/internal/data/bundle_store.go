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

type BundleComponentInput struct {
	GoodID   ID
	DraftKey string
	Quantity int
	Draft    *BundleGoodDraft
}

type BundleGoodDraft struct {
	Name           string
	Description    string
	TotalStock     int
	ListedQty      int
	UnitPriceMinor int64
	PriceTiers     []catalog.PriceTier
	IsListed       bool
	ImagePaths     []string
}

type BundleInput struct {
	Name           string
	Description    string
	EventID        ID
	NewEvent       *EventInput
	Components     []BundleComponentInput
	UnitPriceMinor int64
	PriceTiers     []catalog.PriceTier
	IsListed       bool
	ImagePaths     []string
	Approved       bool
}

type BundleComponent struct {
	GoodID         string
	GoodName       string
	Quantity       int
	ImageIDs       []string
	ListingID      string
	UnitPriceMinor int64
	ReviewStatus   string
	IsListed       bool
	Remaining      int
}

type BundleCard struct {
	ListingID      string
	BundleID       string
	ShopID         string
	ShopName       string
	ShopSlug       string
	EventID        string
	EventName      string
	Currency       string
	Name           string
	Description    string
	UnitPriceMinor int64
	PriceTiers     []catalog.PriceTier
	Remaining      int
	IsListed       bool
	ReviewStatus   string
	ReviewNote     string
	Revision       int64
	ImageIDs       []string
	Components     []BundleComponent
}

type PlatformShopBundles struct {
	ShopID   string
	ShopName string
	ShopSlug string
	Bundles  []BundleCard
}

func (s *ShopStore) CreateBundle(ctx context.Context, shopID, actorID ID, in BundleInput) (ID, ID, error) {
	if in.Name == "" || len([]rune(in.Name)) > 120 || len([]rune(in.Description)) > 4000 ||
		len(in.Components) < 2 || len(in.Components) > 20 || len(in.ImagePaths) > 8 || in.UnitPriceMinor <= 0 ||
		catalog.ValidatePriceTiers(in.UnitPriceMinor, in.PriceTiers) != nil || (in.EventID == 0) == (in.NewEvent == nil) {
		return 0, 0, ErrInvalidBundle
	}
	for _, path := range in.ImagePaths {
		if path == "" {
			return 0, 0, ErrInvalidBundle
		}
	}
	tiers, err := marshalPriceTiers(in.PriceTiers)
	if err != nil {
		return 0, 0, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, 0, fmt.Errorf("begin bundle: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return 0, 0, err
	}
	var eventID ID
	if in.NewEvent != nil {
		if !validBundleEvent(*in.NewEvent) {
			return 0, 0, ErrInvalidBundle
		}
		err = tx.QueryRow(ctx, `
			INSERT INTO events(shop_id, name, starts_on, ends_on, city, venue, booth_number, currency, status)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id::text`,
			shopID, in.NewEvent.Name, in.NewEvent.StartsOn, in.NewEvent.EndsOn, in.NewEvent.City,
			in.NewEvent.Venue, in.NewEvent.BoothNumber, in.NewEvent.Currency, in.NewEvent.Status).Scan(&eventID)
		if err != nil {
			return 0, 0, fmt.Errorf("create bundle event: %w", err)
		}
	} else {
		err = tx.QueryRow(ctx, `SELECT id::text FROM events WHERE id = $1 AND shop_id = $2`, in.EventID, shopID).Scan(&eventID)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, 0, ErrNotFound
		}
		if err != nil {
			return 0, 0, fmt.Errorf("find bundle event: %w", err)
		}
	}

	var bundleID ID
	if err := tx.QueryRow(ctx, `
		INSERT INTO bundles(shop_id, name, description, review_status, reviewed_at, revision)
		VALUES ($1,$2,$3,CASE WHEN $4 THEN 'approved' ELSE 'pending' END,CASE WHEN $4 THEN now() ELSE NULL END,1)
		RETURNING id::text`, shopID, in.Name, in.Description, in.Approved).Scan(&bundleID); err != nil {
		return 0, 0, fmt.Errorf("insert bundle: %w", err)
	}
	seenGoods := make(map[ID]struct{}, len(in.Components))
	seenDrafts := make(map[string]struct{}, len(in.Components))
	for order, component := range in.Components {
		if component.Quantity < 1 || component.Quantity > 100 {
			return 0, 0, ErrInvalidBundle
		}
		var goodID ID
		switch {
		case component.GoodID != 0 && component.DraftKey == "" && component.Draft == nil:
			goodID = component.GoodID
			if _, exists := seenGoods[goodID]; exists {
				return 0, 0, ErrInvalidBundle
			}
			var listed bool
			err := tx.QueryRow(ctx, `
				SELECT l.is_listed FROM goods g
				JOIN event_listings l ON l.good_id = g.id AND l.event_id = $3
				WHERE g.id = $1 AND g.shop_id = $2 FOR SHARE OF g, l`, goodID, shopID, eventID).Scan(&listed)
			if errors.Is(err, pgx.ErrNoRows) || (err == nil && !listed) {
				return 0, 0, ErrNotListed
			}
			if err != nil {
				return 0, 0, fmt.Errorf("validate bundle component: %w", err)
			}
		case component.GoodID == 0 && component.DraftKey != "" && component.Draft != nil:
			if _, exists := seenDrafts[component.DraftKey]; exists {
				return 0, 0, ErrInvalidBundle
			}
			seenDrafts[component.DraftKey] = struct{}{}
			goodID, err = createBundleGoodDraft(ctx, tx, shopID, eventID, *component.Draft, in.Approved)
			if err != nil {
				return 0, 0, err
			}
		default:
			return 0, 0, ErrInvalidBundle
		}
		seenGoods[goodID] = struct{}{}
		if _, err := tx.Exec(ctx, `
			INSERT INTO bundle_components(bundle_id, shop_id, good_id, quantity, sort_order)
			VALUES ($1,$2,$3,$4,$5)`, bundleID, shopID, goodID, component.Quantity, order); err != nil {
			return 0, 0, fmt.Errorf("insert bundle component: %w", err)
		}
	}
	var listingID ID
	err = tx.QueryRow(ctx, `
		INSERT INTO event_bundle_listings(shop_id, event_id, bundle_id, unit_price_minor, price_tiers, is_listed)
		VALUES ($1,$2,$3,$4,$5,$6) RETURNING id::text`, shopID, eventID, bundleID, in.UnitPriceMinor, tiers, in.IsListed).Scan(&listingID)
	if isUnique(err) {
		return 0, 0, ErrConflict
	}
	if err != nil {
		return 0, 0, fmt.Errorf("insert bundle listing: %w", err)
	}
	for order, imagePath := range in.ImagePaths {
		if _, err := tx.Exec(ctx, `INSERT INTO bundle_images(bundle_id, shop_id, path, sort_order) VALUES ($1,$2,$3,$4)`, bundleID, shopID, imagePath, order); err != nil {
			return 0, 0, fmt.Errorf("insert bundle image: %w", err)
		}
	}
	if err := insertAudit(ctx, tx, shopID, actorID, "bundle.create", "bundle", bundleID); err != nil {
		return 0, 0, err
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return 0, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, 0, fmt.Errorf("commit bundle: %w", err)
	}
	return bundleID, listingID, nil
}

func (s *ShopStore) UpdateBundle(ctx context.Context, shopID, actorID, bundleID ID, expectedRevision int64, in BundleInput, replaceImages bool) ([]string, error) {
	if expectedRevision <= 0 || in.EventID == 0 || in.NewEvent != nil || in.Name == "" || len([]rune(in.Name)) > 120 ||
		len([]rune(in.Description)) > 4000 || len(in.Components) < 2 || len(in.Components) > 20 ||
		len(in.ImagePaths) > 8 || in.UnitPriceMinor <= 0 || catalog.ValidatePriceTiers(in.UnitPriceMinor, in.PriceTiers) != nil {
		return nil, ErrInvalidBundle
	}
	for _, component := range in.Components {
		if component.Quantity < 1 || component.Quantity > 100 || !((component.GoodID != 0 && component.Draft == nil && component.DraftKey == "") || (component.GoodID == 0 && component.Draft != nil && component.DraftKey != "")) {
			return nil, ErrInvalidBundle
		}
	}
	for _, path := range in.ImagePaths {
		if path == "" {
			return nil, ErrInvalidBundle
		}
	}
	tiers, err := marshalPriceTiers(in.PriceTiers)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("begin update bundle: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return nil, err
	}
	var currentRevision int64
	err = tx.QueryRow(ctx, `SELECT revision FROM bundles WHERE id = $1 AND shop_id = $2 FOR UPDATE`, bundleID, shopID).Scan(&currentRevision)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("lock bundle to update: %w", err)
	}
	if currentRevision != expectedRevision {
		return nil, ErrConflict
	}
	var listingID string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM event_bundle_listings WHERE bundle_id = $1 AND shop_id = $2 AND event_id = $3 FOR UPDATE`, bundleID, shopID, in.EventID).Scan(&listingID); errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, fmt.Errorf("lock bundle listing to update: %w", err)
	}
	seen := make(map[ID]struct{}, len(in.Components))
	seenDrafts := make(map[string]struct{}, len(in.Components))
	for i := range in.Components {
		component := &in.Components[i]
		if component.Draft != nil {
			if _, duplicate := seenDrafts[component.DraftKey]; duplicate {
				return nil, ErrInvalidBundle
			}
			seenDrafts[component.DraftKey] = struct{}{}
			id, err := createBundleGoodDraft(ctx, tx, shopID, in.EventID, *component.Draft, false)
			if err != nil {
				return nil, err
			}
			component.GoodID = id
		}
		if _, duplicate := seen[component.GoodID]; duplicate {
			return nil, ErrInvalidBundle
		}
		seen[component.GoodID] = struct{}{}
		var isListed bool
		err := tx.QueryRow(ctx, `SELECT l.is_listed FROM goods g JOIN event_listings l ON l.good_id = g.id AND l.shop_id = g.shop_id
			WHERE g.id = $1 AND g.shop_id = $2 AND l.event_id = $3 FOR SHARE OF g, l`, component.GoodID, shopID, in.EventID).Scan(&isListed)
		if errors.Is(err, pgx.ErrNoRows) || (err == nil && !isListed) {
			return nil, ErrNotListed
		}
		if err != nil {
			return nil, fmt.Errorf("validate updated bundle component: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE bundles SET name = $3, description = $4, review_status = 'pending', review_note = '', reviewed_at = NULL, revision = revision + 1
		WHERE id = $1 AND shop_id = $2`, bundleID, shopID, in.Name, in.Description); err != nil {
		return nil, fmt.Errorf("update bundle metadata: %w", err)
	}
	if _, err := tx.Exec(ctx, `DELETE FROM bundle_components WHERE bundle_id = $1 AND shop_id = $2`, bundleID, shopID); err != nil {
		return nil, fmt.Errorf("replace bundle components: %w", err)
	}
	for index, component := range in.Components {
		if _, err := tx.Exec(ctx, `INSERT INTO bundle_components(bundle_id, shop_id, good_id, quantity, sort_order) VALUES ($1,$2,$3,$4,$5)`, bundleID, shopID, component.GoodID, component.Quantity, index); err != nil {
			return nil, fmt.Errorf("insert updated bundle component: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		UPDATE event_bundle_listings SET unit_price_minor = $2, price_tiers = $3, is_listed = $4, revision = revision + 1
		WHERE id = $1`, listingID, in.UnitPriceMinor, tiers, in.IsListed); err != nil {
		return nil, fmt.Errorf("update bundle listing: %w", err)
	}
	var oldPaths []string
	if replaceImages {
		rows, err := tx.Query(ctx, `SELECT path FROM bundle_images WHERE bundle_id = $1 AND shop_id = $2 ORDER BY sort_order FOR UPDATE`, bundleID, shopID)
		if err != nil {
			return nil, fmt.Errorf("read old bundle images: %w", err)
		}
		for rows.Next() {
			var path string
			if err := rows.Scan(&path); err != nil {
				rows.Close()
				return nil, fmt.Errorf("scan old bundle image: %w", err)
			}
			oldPaths = append(oldPaths, path)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return nil, fmt.Errorf("read old bundle images: %w", err)
		}
		rows.Close()
		if _, err := tx.Exec(ctx, `DELETE FROM bundle_images WHERE bundle_id = $1 AND shop_id = $2`, bundleID, shopID); err != nil {
			return nil, fmt.Errorf("delete replaced bundle images: %w", err)
		}
		for index, path := range in.ImagePaths {
			if _, err := tx.Exec(ctx, `INSERT INTO bundle_images(bundle_id, shop_id, path, sort_order) VALUES ($1,$2,$3,$4)`, bundleID, shopID, path, index); err != nil {
				return nil, fmt.Errorf("insert replacement bundle image: %w", err)
			}
		}
	}
	if err := insertAudit(ctx, tx, shopID, actorID, "bundle.update", "bundle", bundleID); err != nil {
		return nil, err
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("commit bundle update: %w", err)
	}
	return oldPaths, nil
}

func createBundleGoodDraft(ctx context.Context, tx pgx.Tx, shopID, eventID ID, draft BundleGoodDraft, approved bool) (ID, error) {
	if draft.Name == "" || len([]rune(draft.Name)) > 120 || len([]rune(draft.Description)) > 4000 || draft.TotalStock < 0 ||
		!draft.IsListed || len(draft.ImagePaths) > 8 ||
		catalog.ValidatePriceTiers(draft.UnitPriceMinor, draft.PriceTiers) != nil {
		return 0, ErrInvalidBundle
	}
	tiers, err := marshalPriceTiers(draft.PriceTiers)
	if err != nil {
		return 0, err
	}
	var goodID ID
	if err := tx.QueryRow(ctx, `
		INSERT INTO goods(shop_id, name, description, total_stock, review_status, reviewed_at)
		VALUES ($1,$2,$3,$4,CASE WHEN $5 THEN 'approved' ELSE 'pending' END,CASE WHEN $5 THEN now() ELSE NULL END)
		RETURNING id::text`, shopID, draft.Name, draft.Description, draft.TotalStock, approved).Scan(&goodID); err != nil {
		return 0, fmt.Errorf("insert bundle component good: %w", err)
	}
	for order, imagePath := range draft.ImagePaths {
		if _, err := tx.Exec(ctx, `INSERT INTO good_images(good_id, path, sort_order) VALUES ($1,$2,$3)`, goodID, imagePath, order); err != nil {
			return 0, fmt.Errorf("insert bundle component image: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO event_listings(shop_id, event_id, good_id, listed_qty, unit_price_minor, price_tiers, is_listed)
		VALUES ($1,$2,$3,0,$4,$5,true)`, shopID, eventID, goodID, draft.UnitPriceMinor, tiers); err != nil {
		return 0, fmt.Errorf("insert bundle component listing: %w", err)
	}
	if err := assertPool(ctx, tx, shopID, goodID); err != nil {
		return 0, err
	}
	return goodID, nil
}

func validBundleEvent(in EventInput) bool {
	start, startErr := time.Parse(time.DateOnly, in.StartsOn)
	end, endErr := time.Parse(time.DateOnly, in.EndsOn)
	return in.Name != "" && catalog.ValidCurrency(in.Currency) && catalog.ValidEventStatus(in.Status) &&
		startErr == nil && endErr == nil && !end.Before(start)
}

func (s *ShopStore) StaffBundles(ctx context.Context, shopID, eventID ID) ([]BundleCard, error) {
	return s.listBundles(ctx, shopID, eventID, false)
}

func (s *ShopStore) PublicBundles(ctx context.Context, shopID, eventID ID) ([]BundleCard, error) {
	return s.listBundles(ctx, shopID, eventID, true)
}

func (s *ShopStore) listBundles(ctx context.Context, shopID, eventID ID, publicOnly bool) ([]BundleCard, error) {
	return listBundlesFrom(ctx, s.pool, shopID, eventID, publicOnly)
}

type rowQuerier interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}

func listBundlesFrom(ctx context.Context, queryer rowQuerier, shopID, eventID ID, publicOnly bool) ([]BundleCard, error) {
	rows, err := queryer.Query(ctx, `
		SELECT eb.id::text, b.id::text, s.id::text, s.name, s.slug,
			e.id::text, e.name, e.currency, b.name, b.description, eb.unit_price_minor, eb.price_tiers,
			COALESCE(availability.remaining, 0), eb.is_listed, b.review_status, b.review_note, b.revision,
			COALESCE((SELECT array_agg(bi.id::text ORDER BY bi.sort_order) FROM bundle_images bi WHERE bi.bundle_id = b.id), ARRAY[]::text[]),
			COALESCE(bc.good_id::text, ''), COALESCE(g.name, ''), COALESCE(bc.quantity, 0),
			COALESCE(l.id::text, ''), COALESCE(l.unit_price_minor, 0), COALESCE(g.review_status, ''),
			COALESCE(l.is_listed, false), COALESCE(GREATEST(g.total_stock - COALESCE((SELECT SUM(l2.reserved_qty) FROM event_listings l2 WHERE l2.good_id = g.id AND l2.shop_id = g.shop_id), 0), 0), 0),
			COALESCE((SELECT array_agg(gi.id::text ORDER BY gi.sort_order) FROM good_images gi WHERE gi.good_id = g.id), ARRAY[]::text[])
		FROM event_bundle_listings eb
		JOIN bundles b ON b.id = eb.bundle_id AND b.shop_id = eb.shop_id
		JOIN shops s ON s.id = b.shop_id
		JOIN events e ON e.id = eb.event_id AND e.shop_id = eb.shop_id
		LEFT JOIN bundle_components bc ON bc.bundle_id = b.id AND bc.shop_id = b.shop_id
		LEFT JOIN goods g ON g.id = bc.good_id AND g.shop_id = bc.shop_id
		LEFT JOIN event_listings l ON l.good_id = g.id AND l.event_id = e.id AND l.shop_id = e.shop_id
		LEFT JOIN LATERAL (
			SELECT min(GREATEST(cg.total_stock - COALESCE((SELECT SUM(l2.reserved_qty) FROM event_listings l2 WHERE l2.good_id = cg.id AND l2.shop_id = cg.shop_id), 0), 0) / c.quantity) AS remaining,
				count(*) AS components,
				count(*) FILTER (WHERE cg.review_status = 'approved' AND cl.is_listed) AS eligible
			FROM bundle_components c
			JOIN goods cg ON cg.id = c.good_id AND cg.shop_id = c.shop_id
			LEFT JOIN event_listings cl ON cl.good_id = c.good_id AND cl.event_id = e.id AND cl.shop_id = c.shop_id
			WHERE c.bundle_id = b.id AND c.shop_id = b.shop_id
		) availability ON true
		WHERE ($1 = 0 OR s.id = $1) AND ($2 = 0 OR e.id = $2)
			AND (NOT $3 OR (s.is_enabled AND eb.is_listed AND b.review_status = 'approved'
				AND e.status IN ('live', 'scheduled') AND e.ends_on >= CURRENT_DATE
				AND availability.components >= 2 AND availability.components = availability.eligible AND availability.remaining > 0))
		ORDER BY s.created_at, e.starts_on, b.created_at, bc.sort_order`, shopID, eventID, publicOnly)
	if err != nil {
		return nil, fmt.Errorf("list bundles: %w", err)
	}
	defer rows.Close()
	bundles := make([]BundleCard, 0)
	index := make(map[string]int)
	for rows.Next() {
		var card BundleCard
		var tiers []byte
		var component BundleComponent
		if err := rows.Scan(&card.ListingID, &card.BundleID, &card.ShopID, &card.ShopName, &card.ShopSlug,
			&card.EventID, &card.EventName, &card.Currency, &card.Name, &card.Description, &card.UnitPriceMinor, &tiers,
			&card.Remaining, &card.IsListed, &card.ReviewStatus, &card.ReviewNote, &card.Revision, &card.ImageIDs,
			&component.GoodID, &component.GoodName, &component.Quantity, &component.ListingID, &component.UnitPriceMinor,
			&component.ReviewStatus, &component.IsListed, &component.Remaining, &component.ImageIDs); err != nil {
			return nil, fmt.Errorf("scan bundle: %w", err)
		}
		if err := json.Unmarshal(tiers, &card.PriceTiers); err != nil {
			return nil, fmt.Errorf("decode bundle price tiers: %w", err)
		}
		idx, exists := index[card.ListingID]
		if !exists {
			card.Components = make([]BundleComponent, 0, 4)
			if card.ImageIDs == nil {
				card.ImageIDs = []string{}
			}
			bundles = append(bundles, card)
			idx = len(bundles) - 1
			index[card.ListingID] = idx
		}
		if component.GoodID != "" {
			bundles[idx].Components = append(bundles[idx].Components, component)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list bundles: %w", err)
	}
	return bundles, nil
}

func (s *ShopStore) ListPlatformBundles(ctx context.Context) ([]PlatformShopBundles, error) {
	bundles, err := s.listBundles(ctx, 0, 0, false)
	if err != nil {
		return nil, err
	}
	shops, err := s.ListShops(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]PlatformShopBundles, 0, len(shops))
	index := make(map[string]int, len(shops))
	for _, shop := range shops {
		index[shop.ID.String()] = len(result)
		result = append(result, PlatformShopBundles{ShopID: shop.ID.String(), ShopName: shop.Name, ShopSlug: shop.Slug, Bundles: []BundleCard{}})
	}
	for _, bundle := range bundles {
		if shopIndex, ok := index[bundle.ShopID]; ok {
			result[shopIndex].Bundles = append(result[shopIndex].Bundles, bundle)
		}
	}
	return result, nil
}

func (s *ShopStore) ReviewBundle(ctx context.Context, actorID, bundleID ID, status, note string, expectedRevision int64) error {
	if status != "approved" && status != "rejected" {
		return ErrInvalidBundle
	}
	if expectedRevision <= 0 {
		return ErrInvalidBundle
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin bundle review: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var shopID ID
	err = tx.QueryRow(ctx, `SELECT shop_id::text FROM bundles WHERE id = $1`, bundleID).Scan(&shopID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("find bundle for review: %w", err)
	}
	if err := lockShopForCatalogWrite(ctx, tx, shopID); err != nil {
		return err
	}
	var owned string
	if err := tx.QueryRow(ctx, `SELECT id::text FROM bundles WHERE id = $1 AND shop_id = $2 FOR UPDATE`, bundleID, shopID).Scan(&owned); errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	} else if err != nil {
		return fmt.Errorf("lock bundle for review: %w", err)
	}
	if status == "approved" {
		var invalid bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM event_bundle_listings eb
				WHERE eb.bundle_id = $1 AND (
					SELECT count(*) FROM bundle_components bc
					JOIN goods g ON g.id = bc.good_id AND g.shop_id = bc.shop_id
					LEFT JOIN event_listings l ON l.good_id = g.id AND l.event_id = eb.event_id AND l.shop_id = eb.shop_id
					WHERE bc.bundle_id = eb.bundle_id AND (g.review_status <> 'approved' OR l.id IS NULL OR NOT l.is_listed)
				) > 0
			) OR NOT EXISTS (SELECT 1 FROM bundle_components WHERE bundle_id = $1)`, bundleID).Scan(&invalid); err != nil {
			return fmt.Errorf("validate bundle review: %w", err)
		}
		if invalid {
			return ErrBundleComponents
		}
		note = ""
	}
	tag, err := tx.Exec(ctx, `UPDATE bundles SET review_status = $2, review_note = $3, reviewed_at = now(), revision = revision + 1 WHERE id = $1 AND revision = $4`, bundleID, status, note, expectedRevision)
	if err != nil {
		return fmt.Errorf("update bundle review: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrConflict
	}
	if err := bumpCatalogRevision(ctx, tx, shopID); err != nil {
		return err
	}
	action := "bundle.approve"
	if status == "rejected" {
		action = "bundle.reject"
	}
	if err := insertAudit(ctx, tx, shopID, actorID, action, "bundle", bundleID); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit bundle review: %w", err)
	}
	return nil
}

func lockShopForCatalogWrite(ctx context.Context, tx pgx.Tx, shopID ID) error {
	var id string
	err := tx.QueryRow(ctx, `SELECT id::text FROM shops WHERE id = $1 AND is_enabled FOR UPDATE`, shopID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("lock catalog shop: %w", err)
	}
	return nil
}

func invalidateBundlesForGood(ctx context.Context, tx pgx.Tx, shopID, goodID ID) error {
	if _, err := tx.Exec(ctx, `
		UPDATE bundles SET review_status = 'pending', review_note = 'A component changed and needs review.',
			reviewed_at = NULL, revision = revision + 1
		WHERE shop_id = $1 AND id IN (SELECT bundle_id FROM bundle_components WHERE shop_id = $1 AND good_id = $2)
			AND review_status <> 'pending'`, shopID, goodID); err != nil {
		return fmt.Errorf("invalidate dependent bundles: %w", err)
	}
	return nil
}

func bumpCatalogRevision(ctx context.Context, tx pgx.Tx, shopID ID) error {
	tag, err := tx.Exec(ctx, `UPDATE shops SET catalog_revision = catalog_revision + 1 WHERE id = $1`, shopID)
	if err != nil {
		return fmt.Errorf("bump catalog revision: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}
