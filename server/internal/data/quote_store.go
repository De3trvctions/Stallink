package data

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/De3trvctions/Stallink/internal/catalog"
	"github.com/jackc/pgx/v5"
)

var (
	ErrQuoteExpired = errors.New("data: quote expired")
	ErrQuoteStale   = errors.New("data: quote catalog changed")
	ErrQuoteInvalid = errors.New("data: quote is invalid for this actor or checkout")
)

type OrderQuote struct {
	ID              ID
	ShopID          ID
	ActorID         ID
	EventID         ID
	CartVersion     int64
	CatalogRevision int64
	Snapshot        catalog.CartQuote
	TotalMinor      int64
	ExpiresAt       time.Time
}

func (s *ShopStore) CreateOrderQuote(
	ctx context.Context,
	shopID, actorID, eventID ID,
	cartVersion int64,
	lines []catalog.CartLineInput,
	ttl time.Duration,
	maxStates int,
	bundlesEnabled bool,
) (OrderQuote, error) {
	if cartVersion < 0 || len(lines) == 0 || ttl <= 0 {
		return OrderQuote{}, catalog.ErrInvalidCart
	}
	canonicalLines, err := canonicalCartLines(lines)
	if err != nil {
		return OrderQuote{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead})
	if err != nil {
		return OrderQuote{}, fmt.Errorf("begin order quote: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var revision int64
	if err := tx.QueryRow(ctx, `SELECT catalog_revision FROM shops WHERE id = $1 AND is_enabled`, shopID).Scan(&revision); errors.Is(err, pgx.ErrNoRows) {
		return OrderQuote{}, ErrNotFound
	} else if err != nil {
		return OrderQuote{}, fmt.Errorf("read quote catalog revision: %w", err)
	}
	var exists bool
	if err := tx.QueryRow(ctx, `SELECT true FROM events WHERE id = $1 AND shop_id = $2`, eventID, shopID).Scan(&exists); errors.Is(err, pgx.ErrNoRows) {
		return OrderQuote{}, ErrNotFound
	} else if err != nil {
		return OrderQuote{}, fmt.Errorf("check quote event: %w", err)
	}
	listingRows, err := tx.Query(ctx, `
		SELECT l.id::text, g.id::text, g.name, l.unit_price_minor, l.price_tiers,
			l.revision, l.listed_qty - l.sold_qty - l.reserved_qty
		FROM event_listings l
		JOIN goods g ON g.id = l.good_id AND g.shop_id = l.shop_id
		WHERE l.shop_id = $1 AND l.event_id = $2 AND l.is_listed AND g.review_status = 'approved'
			AND l.listed_qty > l.sold_qty + l.reserved_qty`, shopID, eventID)
	if err != nil {
		return OrderQuote{}, fmt.Errorf("read quote listings: %w", err)
	}
	listings := make(map[string]catalog.CartListing)
	for listingRows.Next() {
		var listing catalog.CartListing
		var tiers []byte
		if err := listingRows.Scan(&listing.ID, &listing.GoodID, &listing.Name, &listing.UnitPriceMinor, &tiers, &listing.Revision, &listing.Remaining); err != nil {
			listingRows.Close()
			return OrderQuote{}, fmt.Errorf("scan quote listing: %w", err)
		}
		if err := json.Unmarshal(tiers, &listing.PriceTiers); err != nil {
			listingRows.Close()
			return OrderQuote{}, fmt.Errorf("decode quote listing tiers: %w", err)
		}
		listings[listing.ID] = listing
	}
	if err := listingRows.Err(); err != nil {
		listingRows.Close()
		return OrderQuote{}, fmt.Errorf("read quote listings: %w", err)
	}
	listingRows.Close()
	bundles := make(map[string]catalog.CartBundle)
	if bundlesEnabled {
		cards, err := listBundlesFrom(ctx, tx, shopID, eventID, true)
		if err != nil {
			return OrderQuote{}, err
		}
		for _, card := range cards {
			bundle := catalog.CartBundle{
				ID: card.ListingID, BundleID: card.BundleID, Name: card.Name,
				UnitPriceMinor: card.UnitPriceMinor,
				Revision:       card.Revision, Remaining: card.Remaining,
			}
			for _, component := range card.Components {
				bundle.Components = append(bundle.Components, catalog.CartBundleComponent{
					ListingID: component.ListingID, GoodID: component.GoodID, Name: component.GoodName, Quantity: component.Quantity,
				})
			}
			bundles[bundle.ID] = bundle
		}
	}
	quote, err := catalog.PriceCart(canonicalLines, listings, bundles, maxStates)
	if err != nil {
		return OrderQuote{}, err
	}
	snapshot, err := json.Marshal(quote)
	if err != nil {
		return OrderQuote{}, fmt.Errorf("encode quote snapshot: %w", err)
	}
	inputJSON, err := json.Marshal(struct {
		EventID     ID
		CartVersion int64
		Lines       []catalog.CartLineInput
	}{eventID, cartVersion, canonicalLines})
	if err != nil {
		return OrderQuote{}, fmt.Errorf("encode quote input: %w", err)
	}
	inputHash := sha256.Sum256(inputJSON)
	result := OrderQuote{
		ShopID: shopID, ActorID: actorID, EventID: eventID, CartVersion: cartVersion,
		CatalogRevision: revision, Snapshot: quote, TotalMinor: quote.TotalMinor,
	}
	err = tx.QueryRow(ctx, `
		INSERT INTO order_quotes(shop_id, actor_id, event_id, cart_version, input_hash, catalog_revision, snapshot, total_minor, expires_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,now() + ($9 * interval '1 microsecond'))
		RETURNING id::text, expires_at`,
		shopID, actorID, eventID, cartVersion, inputHash[:], revision, snapshot, quote.TotalMinor, ttl.Microseconds()).Scan(&result.ID, &result.ExpiresAt)
	if err != nil {
		return OrderQuote{}, fmt.Errorf("save order quote: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return OrderQuote{}, fmt.Errorf("commit order quote: %w", err)
	}
	return result, nil
}

func canonicalCartLines(lines []catalog.CartLineInput) ([]catalog.CartLineInput, error) {
	quantities := make(map[string]int, len(lines))
	for _, line := range lines {
		if line.Qty < 1 || line.Qty > catalog.MaxCartItems {
			return nil, catalog.ErrInvalidCart
		}
		id := line.ListingID
		if line.Kind == "bundle" {
			id = line.BundleListingID
		}
		if (line.Kind != "single" && line.Kind != "bundle") || id == "" ||
			(line.Kind == "single" && line.BundleListingID != "") || (line.Kind == "bundle" && line.ListingID != "") {
			return nil, catalog.ErrInvalidCart
		}
		key := line.Kind + ":" + id
		if quantities[key] > catalog.MaxCartItems-line.Qty {
			return nil, catalog.ErrPricingTooLarge
		}
		quantities[key] += line.Qty
	}
	keys := make([]string, 0, len(quantities))
	for key := range quantities {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	canonical := make([]catalog.CartLineInput, 0, len(keys))
	for _, key := range keys {
		kind, id, _ := strings.Cut(key, ":")
		line := catalog.CartLineInput{Kind: kind, Qty: quantities[key]}
		if kind == "single" {
			line.ListingID = id
		} else {
			line.BundleListingID = id
		}
		canonical = append(canonical, line)
	}
	return canonical, nil
}
