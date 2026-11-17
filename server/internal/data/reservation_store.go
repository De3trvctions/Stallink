package data

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/De3trvctions/Stallink/internal/catalog"
	"github.com/jackc/pgx/v5"
)

var ErrReservationExpired = errors.New("data: order inventory reservation expired")

type reservationLine struct {
	listingID ID
	goodID    ID
	qty       int
}

// inventoryLockOrder returns distinct goods and listings in the global lock order.
func inventoryLockOrder(lines []reservationLine) ([]ID, []ID) {
	goods := make(map[ID]struct{}, len(lines))
	listings := make(map[ID]struct{}, len(lines))
	for _, line := range lines {
		goods[line.goodID] = struct{}{}
		listings[line.listingID] = struct{}{}
	}
	goodIDs := make([]ID, 0, len(goods))
	for id := range goods {
		goodIDs = append(goodIDs, id)
	}
	listingIDs := make([]ID, 0, len(listings))
	for id := range listings {
		listingIDs = append(listingIDs, id)
	}
	sort.Slice(goodIDs, func(i, j int) bool { return goodIDs[i] < goodIDs[j] })
	sort.Slice(listingIDs, func(i, j int) bool { return listingIDs[i] < listingIDs[j] })
	return goodIDs, listingIDs
}

func reservationLinesMatchOrder(expected, active []reservationLine) bool {
	if len(expected) != len(active) {
		return false
	}
	expectedByListing := make(map[ID]reservationLine, len(expected))
	for _, line := range expected {
		if _, exists := expectedByListing[line.listingID]; exists {
			return false
		}
		expectedByListing[line.listingID] = line
	}
	activeByListing := make(map[ID]reservationLine, len(active))
	for _, line := range active {
		if _, exists := activeByListing[line.listingID]; exists {
			return false
		}
		activeByListing[line.listingID] = line
	}
	for listingID, want := range expectedByListing {
		got, exists := activeByListing[listingID]
		if !exists || got.goodID != want.goodID || got.qty != want.qty {
			return false
		}
	}
	return true
}

func reserveOrderInventory(ctx context.Context, tx pgx.Tx, shopID, orderID ID, ttl time.Duration) error {
	if ttl <= 0 {
		return fmt.Errorf("reservation ttl must be positive")
	}
	lines, err := orderReservationLines(ctx, tx, orderID)
	if err != nil {
		return err
	}
	expiresAt := ttl.Microseconds()
	for _, line := range lines {
		tag, err := tx.Exec(ctx, `
			UPDATE event_listings SET reserved_qty = reserved_qty + $3
			WHERE id = $1 AND shop_id = $2 AND listed_qty - sold_qty - reserved_qty >= $3`,
			line.listingID, shopID, line.qty)
		if err != nil {
			return fmt.Errorf("reserve order listing: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return catalog.ErrNotEnough
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO inventory_reservations(shop_id, order_id, listing_id, qty, state, expires_at)
			VALUES ($1,$2,$3,$4,'active',now() + ($5 * interval '1 microsecond'))`,
			shopID, orderID, line.listingID, line.qty, expiresAt); err != nil {
			return fmt.Errorf("write inventory reservation: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO inventory_movements(shop_id, order_id, listing_id, movement_type, qty) VALUES ($1,$2,$3,'reserve',$4)`, shopID, orderID, line.listingID, line.qty); err != nil {
			return fmt.Errorf("write reservation movement: %w", err)
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET reservation_expires_at = now() + ($2 * interval '1 microsecond') WHERE id = $1`, orderID, expiresAt); err != nil {
		return fmt.Errorf("set reservation expiration: %w", err)
	}
	return nil
}

func orderReservationLines(ctx context.Context, tx pgx.Tx, orderID ID) ([]reservationLine, error) {
	rows, err := tx.Query(ctx, `
		SELECT listing_id::text, good_id::text, sum(qty)::int
		FROM order_lines WHERE order_id = $1 GROUP BY listing_id, good_id ORDER BY listing_id`, orderID)
	if err != nil {
		return nil, fmt.Errorf("read order reservation lines: %w", err)
	}
	defer rows.Close()
	var lines []reservationLine
	for rows.Next() {
		var line reservationLine
		if err := rows.Scan(&line.listingID, &line.goodID, &line.qty); err != nil {
			return nil, fmt.Errorf("scan order reservation line: %w", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read order reservation lines: %w", err)
	}
	return lines, nil
}

func consumeOrderReservations(ctx context.Context, tx pgx.Tx, shopID, orderID ID) error {
	rows, err := tx.Query(ctx, `
		SELECT r.listing_id::text, COALESCE(l.good_id::text, ''), r.qty
		FROM inventory_reservations r
		LEFT JOIN event_listings l ON l.id = r.listing_id AND l.shop_id = r.shop_id
		WHERE r.shop_id = $1 AND r.order_id = $2 AND r.state = 'active' AND r.expires_at > now()
		ORDER BY r.listing_id FOR UPDATE OF r`, shopID, orderID)
	if err != nil {
		return fmt.Errorf("read active reservations: %w", err)
	}
	var active []reservationLine
	for rows.Next() {
		var item reservationLine
		var rawGoodID string
		if err := rows.Scan(&item.listingID, &rawGoodID, &item.qty); err != nil {
			rows.Close()
			return fmt.Errorf("scan active reservation: %w", err)
		}
		item.goodID, err = ParseID(rawGoodID)
		if err != nil {
			rows.Close()
			return fmt.Errorf("invalid good for reservation listing %s: %w", item.listingID, err)
		}
		active = append(active, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read active reservations: %w", err)
	}
	rows.Close()
	if len(active) == 0 {
		return ErrReservationExpired
	}
	expected, err := orderReservationLines(ctx, tx, orderID)
	if err != nil {
		return err
	}
	if !reservationLinesMatchOrder(expected, active) {
		return ErrReservationExpired
	}
	goodIDs, listingIDs := inventoryLockOrder(active)
	for _, goodID := range goodIDs {
		var total int
		if err := tx.QueryRow(ctx, `SELECT total_stock FROM goods WHERE id = $1 AND shop_id = $2 FOR UPDATE`, goodID, shopID).Scan(&total); err != nil {
			return fmt.Errorf("lock reserved good: %w", err)
		}
	}
	goodByListing := make(map[ID]ID, len(active))
	for _, item := range active {
		goodByListing[item.listingID] = item.goodID
	}
	for _, listingID := range listingIDs {
		var lockedGoodID ID
		if err := tx.QueryRow(ctx, `SELECT good_id::text FROM event_listings WHERE id = $1 AND shop_id = $2 FOR UPDATE`, listingID, shopID).Scan(&lockedGoodID); err != nil {
			return fmt.Errorf("lock reserved listing: %w", err)
		}
		if expected := goodByListing[listingID]; lockedGoodID != expected {
			return fmt.Errorf("reserved listing %s changed goods while locking", listingID)
		}
	}
	for _, item := range active {
		tag, err := tx.Exec(ctx, `UPDATE goods SET total_stock = total_stock - $2 WHERE id = $1 AND shop_id = $3 AND total_stock >= $2`, item.goodID, item.qty, shopID)
		if err != nil {
			return fmt.Errorf("consume reserved stock: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return catalog.ErrNotEnough
		}
		tag, err = tx.Exec(ctx, `UPDATE event_listings SET reserved_qty = reserved_qty - $2, sold_qty = sold_qty + $2 WHERE id = $1 AND reserved_qty >= $2 AND sold_qty + $2 <= listed_qty`, item.listingID, item.qty)
		if err != nil {
			return fmt.Errorf("consume listing reservation: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return catalog.ErrNotEnough
		}
		if _, err := tx.Exec(ctx, `UPDATE inventory_reservations SET state = 'consumed', updated_at = now() WHERE shop_id = $1 AND order_id = $2 AND listing_id = $3 AND state = 'active'`, shopID, orderID, item.listingID); err != nil {
			return fmt.Errorf("consume reservation record: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO inventory_movements(shop_id, order_id, listing_id, movement_type, qty) VALUES ($1,$2,$3,'sell',$4)`, shopID, orderID, item.listingID, item.qty); err != nil {
			return fmt.Errorf("write sale movement: %w", err)
		}
	}
	for _, goodID := range goodIDs {
		if err := assertAllocation(ctx, tx, goodID); err != nil {
			return err
		}
	}
	return nil
}

func releaseOrderReservations(ctx context.Context, tx pgx.Tx, shopID, orderID ID) error {
	rows, err := tx.Query(ctx, `
		SELECT listing_id::text, qty FROM inventory_reservations
		WHERE shop_id = $1 AND order_id = $2 AND state = 'active'
		ORDER BY listing_id FOR UPDATE`, shopID, orderID)
	if err != nil {
		return fmt.Errorf("read reservations to release: %w", err)
	}
	type activeReservation struct {
		listingID ID
		qty       int
	}
	var active []activeReservation
	for rows.Next() {
		var item activeReservation
		if err := rows.Scan(&item.listingID, &item.qty); err != nil {
			rows.Close()
			return fmt.Errorf("scan reservation to release: %w", err)
		}
		active = append(active, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return fmt.Errorf("read reservations to release: %w", err)
	}
	rows.Close()
	for _, item := range active {
		tag, err := tx.Exec(ctx, `UPDATE event_listings SET reserved_qty = reserved_qty - $2 WHERE id = $1 AND shop_id = $3 AND reserved_qty >= $2`, item.listingID, item.qty, shopID)
		if err != nil {
			return fmt.Errorf("release listing reservation: %w", err)
		}
		if tag.RowsAffected() != 1 {
			return catalog.ErrNotEnough
		}
		if _, err := tx.Exec(ctx, `UPDATE inventory_reservations SET state = 'released', updated_at = now() WHERE shop_id = $1 AND order_id = $2 AND listing_id = $3 AND state = 'active'`, shopID, orderID, item.listingID); err != nil {
			return fmt.Errorf("release reservation record: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO inventory_movements(shop_id, order_id, listing_id, movement_type, qty) VALUES ($1,$2,$3,'release',$4)`, shopID, orderID, item.listingID, item.qty); err != nil {
			return fmt.Errorf("write release movement: %w", err)
		}
	}
	return nil
}

// ExpireReservations releases due stock in bounded, concurrency-safe batches.
func (s *ShopStore) ExpireReservations(ctx context.Context, batchSize int) (int, error) {
	if batchSize <= 0 || batchSize > 1000 {
		batchSize = 100
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin reservation sweep: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	rows, err := tx.Query(ctx, `
		SELECT id::text, shop_id::text, opened_by::text FROM orders
		WHERE status = 'pending' AND reservation_mode = 'reserved' AND reservation_expires_at <= now()
		ORDER BY reservation_expires_at, id LIMIT $1 FOR UPDATE SKIP LOCKED`, batchSize)
	if err != nil {
		return 0, fmt.Errorf("select expired reservations: %w", err)
	}
	type expiredOrder struct{ id, shopID, actorID ID }
	var expired []expiredOrder
	for rows.Next() {
		var item expiredOrder
		if err := rows.Scan(&item.id, &item.shopID, &item.actorID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan expired reservation: %w", err)
		}
		expired = append(expired, item)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, fmt.Errorf("read expired reservations: %w", err)
	}
	rows.Close()
	for _, item := range expired {
		if err := releaseOrderReservations(ctx, tx, item.shopID, item.id); err != nil {
			return 0, err
		}
		if _, err := tx.Exec(ctx, `UPDATE orders SET status = 'expired', reservation_mode = 'released', state_version = state_version + 1 WHERE id = $1`, item.id); err != nil {
			return 0, fmt.Errorf("expire order: %w", err)
		}
		if err := insertAudit(ctx, tx, item.shopID, item.actorID, "order.expire", "order", item.id); err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit reservation sweep: %w", err)
	}
	return len(expired), nil
}
