package data

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/De3trvctions/Stallink/internal/catalog"
	"github.com/jackc/pgx/v5"
)

type OrderLineIn struct {
	ListingID ID
	Qty       int
}

type OrderLine struct {
	GoodName       string
	UnitPriceMinor int64
	Qty            int
	LineTotalMinor int64
}

type Order struct {
	ID                   ID
	Number               string
	Status               string
	BoothName            string
	Currency             string
	PayCodeName          string
	OpenedByLogin        string
	OpenedByRole         string
	PaidByLogin          string
	PaidByRole           string
	EventID              ID
	EventName            string
	TotalMinor           int64
	StateVersion         int64
	CreatedAt            time.Time
	ReservationExpiresAt *time.Time
	RefundedAt           *time.Time
	CashReceivedMinor    *int64
	CashChangeMinor      *int64
	Lines                []OrderLine
	PricingGroups        []catalog.PricingGroup
}

type PayCode struct {
	ID        ID
	Name      string
	Kind      string
	Currency  *string
	IsDefault bool
	IsEnabled bool
}

type AuditEntry struct {
	Action      string
	ActorLogin  string
	SubjectType string
	SubjectID   ID
	CreatedAt   string
}

func (s *ShopStore) CreateOrderCommand(ctx context.Context, identity CommandIdentity, clientOrderID string, quoteID ID, expectedTotal int64, boothName string, reservationsEnabled bool, reservationTTL time.Duration, allowScheduled bool) (OrderCommandResult, error) {
	checkoutHash := hashCheckout(identity, clientOrderID)
	return s.runOrderCommand(ctx, identity, 201, func(tx pgx.Tx) (Order, error) {
		var existingID ID
		var existingHash []byte
		err := tx.QueryRow(ctx, `SELECT id::text, checkout_hash FROM orders WHERE shop_id = $1 AND client_order_id = $2`, identity.ShopID, clientOrderID).Scan(&existingID, &existingHash)
		if err == nil {
			if !bytes.Equal(existingHash, checkoutHash[:]) {
				return Order{}, ErrCheckoutConflict
			}
			return getOrderTx(ctx, tx, identity.ShopID, existingID)
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return Order{}, fmt.Errorf("lookup checkout: %w", err)
		}
		quote, err := loadQuoteForCheckout(ctx, tx, identity, quoteID, clientOrderID, expectedTotal)
		if err != nil {
			return Order{}, err
		}
		return createOrderFromQuoteTx(ctx, tx, identity, clientOrderID, checkoutHash, boothName, quote, reservationsEnabled, reservationTTL, allowScheduled)
	})
}

func hashCheckout(identity CommandIdentity, clientOrderID string) [32]byte {
	canonical, _ := json.Marshal(struct {
		ShopID        ID
		ActorID       ID
		ClientOrderID string
		RequestHash   [32]byte
	}{identity.ShopID, identity.ActorID, clientOrderID, identity.RequestHash})
	return sha256.Sum256(canonical)
}

func loadQuoteForCheckout(ctx context.Context, tx pgx.Tx, identity CommandIdentity, quoteID ID, clientOrderID string, expectedTotal int64) (OrderQuote, error) {
	var quote OrderQuote
	var snapshot []byte
	var consumedBy string
	var unexpired bool
	var currentRevision int64
	if err := tx.QueryRow(ctx, `SELECT catalog_revision FROM shops WHERE id = $1 AND is_enabled FOR SHARE`, identity.ShopID).Scan(&currentRevision); errors.Is(err, pgx.ErrNoRows) {
		return OrderQuote{}, ErrNotFound
	} else if err != nil {
		return OrderQuote{}, fmt.Errorf("check quote revision: %w", err)
	}
	err := tx.QueryRow(ctx, `
		SELECT id::text, shop_id::text, actor_id::text, event_id::text, cart_version, catalog_revision,
			snapshot, total_minor, expires_at, expires_at > now(), COALESCE(consumed_by_client_order_id::text, '')
		FROM order_quotes WHERE id = $1 AND shop_id = $2 AND actor_id = $3 FOR UPDATE`,
		quoteID, identity.ShopID, identity.ActorID).Scan(&quote.ID, &quote.ShopID, &quote.ActorID, &quote.EventID,
		&quote.CartVersion, &quote.CatalogRevision, &snapshot, &quote.TotalMinor, &quote.ExpiresAt, &unexpired, &consumedBy)
	if errors.Is(err, pgx.ErrNoRows) {
		return OrderQuote{}, ErrQuoteInvalid
	}
	if err != nil {
		return OrderQuote{}, fmt.Errorf("load quote for checkout: %w", err)
	}
	if !unexpired {
		return OrderQuote{}, ErrQuoteExpired
	}
	if quote.CatalogRevision != currentRevision {
		return OrderQuote{}, ErrQuoteStale
	}
	if quote.TotalMinor != expectedTotal || (consumedBy != "" && consumedBy != clientOrderID) {
		return OrderQuote{}, ErrQuoteInvalid
	}
	if err := json.Unmarshal(snapshot, &quote.Snapshot); err != nil {
		return OrderQuote{}, fmt.Errorf("decode quote snapshot: %w", err)
	}
	return quote, nil
}

func createOrderFromQuoteTx(ctx context.Context, tx pgx.Tx, identity CommandIdentity, clientOrderID string, checkoutHash [32]byte, boothName string, quote OrderQuote, reservationsEnabled bool, reservationTTL time.Duration, allowScheduled bool) (Order, error) {
	var currency, eventStatus string
	if err := tx.QueryRow(ctx, `SELECT currency, status FROM events WHERE id = $1 AND shop_id = $2`, quote.EventID, identity.ShopID).Scan(&currency, &eventStatus); errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	} else if err != nil {
		return Order{}, fmt.Errorf("read quote event: %w", err)
	}
	if err := validateEventOpen(eventStatus, allowScheduled); err != nil {
		return Order{}, err
	}
	orderID, created, err := insertOrderHeader(ctx, tx, identity, clientOrderID, checkoutHash, quote.EventID, boothName, quote.TotalMinor, currency, reservationsEnabled, reservationTTL)
	if err != nil {
		return Order{}, err
	}
	if !created {
		return getOrderTx(ctx, tx, identity.ShopID, orderID)
	}
	var computedTotal int64
	for _, group := range quote.Snapshot.Groups {
		groupSnapshot, err := json.Marshal(group)
		if err != nil {
			return Order{}, fmt.Errorf("encode order pricing group: %w", err)
		}
		groupID, err := ParseID(group.RuleID)
		if err != nil {
			return Order{}, fmt.Errorf("invalid pricing rule id %q: %w", group.RuleID, err)
		}
		var pricingGroupID ID
		if err := tx.QueryRow(ctx, `
			INSERT INTO order_pricing_groups(shop_id, order_id, rule_kind, rule_id, rule_version, name_snapshot, applications, total_minor, snapshot)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id::text`,
			identity.ShopID, orderID, group.Kind, groupID, group.RuleVersion, group.Name, group.Applications, group.TotalMinor, groupSnapshot).Scan(&pricingGroupID); err != nil {
			return Order{}, fmt.Errorf("insert order pricing group: %w", err)
		}
		var lineTotal int64
		for _, line := range group.Lines {
			if line.Quantity <= 0 || line.UnitPriceMinor < 0 || line.LineTotalMinor < 0 || lineTotal > math.MaxInt64-line.LineTotalMinor {
				return Order{}, catalog.ErrInvalidPrice
			}
			lineTotal += line.LineTotalMinor
			goodID, err := ParseID(line.GoodID)
			if err != nil {
				return Order{}, fmt.Errorf("invalid quoted good id %q: %w", line.GoodID, err)
			}
			listingID, err := ParseID(line.ListingID)
			if err != nil {
				return Order{}, fmt.Errorf("invalid quoted listing id %q: %w", line.ListingID, err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO order_lines(order_id, good_id, listing_id, shop_id, event_id, pricing_group_id, good_name,
					unit_price_minor, qty, line_total_minor, component_snapshot, snapshot_version)
				VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,1)`,
				orderID, goodID, listingID, identity.ShopID, quote.EventID, pricingGroupID, line.Name,
				line.UnitPriceMinor, line.Quantity, line.LineTotalMinor, groupSnapshot); err != nil {
				return Order{}, fmt.Errorf("insert order pricing line: %w", err)
			}
		}
		if lineTotal != group.TotalMinor || computedTotal > math.MaxInt64-group.TotalMinor {
			return Order{}, catalog.ErrInvalidPrice
		}
		computedTotal += group.TotalMinor
	}
	if computedTotal != quote.TotalMinor {
		return Order{}, catalog.ErrInvalidPrice
	}
	if reservationsEnabled {
		if err := reserveOrderInventory(ctx, tx, identity.ShopID, orderID, reservationTTL); err != nil {
			return Order{}, err
		}
	}
	tag, err := tx.Exec(ctx, `UPDATE order_quotes SET consumed_by_client_order_id = $2 WHERE id = $1 AND (consumed_by_client_order_id IS NULL OR consumed_by_client_order_id = $2)`, quote.ID, clientOrderID)
	if err != nil {
		return Order{}, fmt.Errorf("consume order quote: %w", err)
	}
	if tag.RowsAffected() != 1 {
		return Order{}, ErrQuoteInvalid
	}
	if err := insertAudit(ctx, tx, identity.ShopID, identity.ActorID, "order.create", "order", orderID); err != nil {
		return Order{}, err
	}
	return getOrderTx(ctx, tx, identity.ShopID, orderID)
}

func insertOrderHeader(ctx context.Context, tx pgx.Tx, identity CommandIdentity, clientOrderID string, checkoutHash [32]byte, eventID ID, boothName string, total int64, currency string, reservationsEnabled bool, reservationTTL time.Duration) (ID, bool, error) {
	mode := "legacy_unreserved"
	if reservationsEnabled {
		if reservationTTL <= 0 {
			return 0, false, fmt.Errorf("reservation ttl must be positive")
		}
		mode = "reserved"
	}
	for attempt := 0; attempt < 5; attempt++ {
		number, err := newOrderNumber()
		if err != nil {
			return 0, false, err
		}
		if _, err := tx.Exec(ctx, "SAVEPOINT order_number"); err != nil {
			return 0, false, fmt.Errorf("order savepoint: %w", err)
		}
		var orderID ID
		err = tx.QueryRow(ctx, `
			INSERT INTO orders(shop_id,event_id,number,status,booth_name,total_minor,currency,opened_by,opened_by_role,client_order_id,checkout_hash,reservation_mode,reservation_expires_at)
			VALUES ($1,$2,$3,'pending',$4,$5,$6,$7,$8,$9,$10,$11,CASE WHEN $12 THEN now() + ($13 * interval '1 microsecond') ELSE NULL END)
			ON CONFLICT DO NOTHING RETURNING id::text`, identity.ShopID, eventID, number, boothName, total, currency,
			identity.ActorID, identity.ActorRole, clientOrderID, checkoutHash[:], mode, reservationsEnabled, reservationTTL.Microseconds()).Scan(&orderID)
		if err == nil {
			if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT order_number"); err != nil {
				return 0, false, fmt.Errorf("release order savepoint: %w", err)
			}
			return orderID, true, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return 0, false, fmt.Errorf("insert order header: %w", err)
		}
		var existingID ID
		var existingHash []byte
		lookupErr := tx.QueryRow(ctx, `SELECT id::text, checkout_hash FROM orders WHERE shop_id = $1 AND client_order_id = $2`, identity.ShopID, clientOrderID).Scan(&existingID, &existingHash)
		if lookupErr == nil {
			if !bytes.Equal(existingHash, checkoutHash[:]) {
				return 0, false, ErrCheckoutConflict
			}
			if _, err := tx.Exec(ctx, "RELEASE SAVEPOINT order_number"); err != nil {
				return 0, false, fmt.Errorf("release checkout savepoint: %w", err)
			}
			return existingID, false, nil
		}
		if !errors.Is(lookupErr, pgx.ErrNoRows) {
			return 0, false, fmt.Errorf("find checkout after conflict: %w", lookupErr)
		}
		if _, err := tx.Exec(ctx, "ROLLBACK TO SAVEPOINT order_number"); err != nil {
			return 0, false, fmt.Errorf("rollback order savepoint: %w", err)
		}
	}
	return 0, false, fmt.Errorf("could not allocate order number")
}

func (s *ShopStore) PayOrderCommand(ctx context.Context, identity CommandIdentity, orderID, payCodeID ID, cashReceivedMinor *int64) (OrderCommandResult, error) {
	return s.runOrderCommand(ctx, identity, 200, func(tx pgx.Tx) (Order, error) {
		return payOrderTx(ctx, tx, identity.ShopID, identity.ActorID, orderID, payCodeID, cashReceivedMinor)
	})
}

func payOrderTx(ctx context.Context, tx pgx.Tx, shopID, actorID, orderID, payCodeID ID, cashReceivedMinor *int64) (Order, error) {
	if payCodeID == 0 {
		return Order{}, ErrPayCode
	}
	var status, reservationMode, orderCurrency string
	var totalMinor int64
	err := tx.QueryRow(ctx, `SELECT status, reservation_mode, currency, total_minor FROM orders WHERE id = $1 AND shop_id = $2 FOR UPDATE`, orderID, shopID).Scan(&status, &reservationMode, &orderCurrency, &totalMinor)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("lock order: %w", err)
	}
	if status == "paid" || status == "fulfilled" {
		var existingPayCodeID int64
		if err := tx.QueryRow(ctx, `SELECT COALESCE(pay_code_id, 0) FROM orders WHERE id = $1`, orderID).Scan(&existingPayCodeID); err != nil {
			return Order{}, fmt.Errorf("read completed payment: %w", err)
		}
		if ID(existingPayCodeID) != payCodeID {
			return Order{}, ErrConflict
		}
		return getOrderTx(ctx, tx, shopID, orderID)
	}
	if !catalog.CanTransit(status, "paid") {
		return Order{}, ErrStatus
	}
	var payName, payKind string
	var payCurrency *string
	err = tx.QueryRow(ctx, `SELECT name, kind, currency FROM pay_codes WHERE id = $1 AND shop_id = $2 AND is_enabled`, payCodeID, shopID).Scan(&payName, &payKind, &payCurrency)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrPayCode
	}
	if err != nil {
		return Order{}, fmt.Errorf("pay code: %w", err)
	}
	if !payCodeCurrencyMatches(payCurrency, orderCurrency) {
		return Order{}, ErrPayCodeCurrencyMismatch
	}
	cashReceived, cashChange, err := cashSettlement(totalMinor, cashReceivedMinor)
	if err != nil {
		return Order{}, err
	}
	if reservationMode == "reserved" {
		if err := consumeOrderReservations(ctx, tx, shopID, orderID); err != nil {
			return Order{}, err
		}
	} else if err := applyStock(ctx, tx, orderID, true); err != nil {
		return Order{}, err
	}
	_, err = tx.Exec(ctx, `
		UPDATE orders SET status = 'paid', reservation_mode = CASE WHEN reservation_mode = 'reserved' THEN 'consumed' ELSE reservation_mode END,
			pay_code_id = $2, pay_code_name = $3, paid_by = $4, cash_received_minor = $5, cash_change_minor = $6,
			state_version = state_version + 1 WHERE id = $1`,
		orderID, payCodeID, payName, actorID, cashReceived, cashChange)
	if err != nil {
		return Order{}, fmt.Errorf("mark paid: %w", err)
	}
	if err := insertAudit(ctx, tx, shopID, actorID, "order.pay", "order", orderID); err != nil {
		return Order{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO order_payments(order_id, shop_id, actor_id, pay_code_id, pay_code_name_snapshot, amount_minor, currency, pay_code_currency) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		orderID, shopID, actorID, payCodeID, payName, totalMinor, orderCurrency, payCurrency); err != nil {
		return Order{}, fmt.Errorf("record order payment: %w", err)
	}
	return getOrderTx(ctx, tx, shopID, orderID)
}

func (s *ShopStore) FulfillOrderCommand(ctx context.Context, identity CommandIdentity, orderID ID) (OrderCommandResult, error) {
	return s.runOrderCommand(ctx, identity, 200, func(tx pgx.Tx) (Order, error) {
		return fulfillOrderTx(ctx, tx, identity.ShopID, identity.ActorID, orderID)
	})
}

func (s *ShopStore) ReserveOrderCommand(ctx context.Context, identity CommandIdentity, orderID ID, reservationTTL time.Duration) (OrderCommandResult, error) {
	return s.runOrderCommand(ctx, identity, 200, func(tx pgx.Tx) (Order, error) {
		var status, reservationMode string
		err := tx.QueryRow(ctx, `SELECT status, reservation_mode FROM orders WHERE id = $1 AND shop_id = $2 FOR UPDATE`, orderID, identity.ShopID).Scan(&status, &reservationMode)
		if errors.Is(err, pgx.ErrNoRows) {
			return Order{}, ErrNotFound
		}
		if err != nil {
			return Order{}, fmt.Errorf("lock order for reservation: %w", err)
		}
		if status != "pending" {
			return Order{}, ErrStatus
		}
		if reservationMode == "reserved" {
			return getOrderTx(ctx, tx, identity.ShopID, orderID)
		}
		if reservationMode != "legacy_unreserved" {
			return Order{}, ErrStatus
		}
		if err := reserveOrderInventory(ctx, tx, identity.ShopID, orderID, reservationTTL); err != nil {
			return Order{}, err
		}
		if _, err := tx.Exec(ctx, `UPDATE orders SET reservation_mode = 'reserved', state_version = state_version + 1 WHERE id = $1`, orderID); err != nil {
			return Order{}, fmt.Errorf("mark order reserved: %w", err)
		}
		if err := insertAudit(ctx, tx, identity.ShopID, identity.ActorID, "order.reserve", "order", orderID); err != nil {
			return Order{}, err
		}
		return getOrderTx(ctx, tx, identity.ShopID, orderID)
	})
}

func fulfillOrderTx(ctx context.Context, tx pgx.Tx, shopID, actorID, orderID ID) (Order, error) {
	var status string
	err := tx.QueryRow(ctx, `SELECT status FROM orders WHERE id = $1 AND shop_id = $2 FOR UPDATE`, orderID, shopID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("lock order: %w", err)
	}
	if status == "fulfilled" {
		return getOrderTx(ctx, tx, shopID, orderID)
	}
	if !catalog.CanTransit(status, "fulfilled") {
		return Order{}, ErrStatus
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET status = 'fulfilled', state_version = state_version + 1 WHERE id = $1`, orderID); err != nil {
		return Order{}, fmt.Errorf("fulfill order: %w", err)
	}
	if err := insertAudit(ctx, tx, shopID, actorID, "order.fulfill", "order", orderID); err != nil {
		return Order{}, err
	}
	return getOrderTx(ctx, tx, shopID, orderID)
}

func (s *ShopStore) CancelOrderCommand(ctx context.Context, identity CommandIdentity, orderID ID) (OrderCommandResult, error) {
	return s.runOrderCommand(ctx, identity, 200, func(tx pgx.Tx) (Order, error) {
		return cancelOrderTx(ctx, tx, identity.ShopID, identity.ActorID, orderID)
	})
}

func cancelOrderTx(ctx context.Context, tx pgx.Tx, shopID, actorID, orderID ID) (Order, error) {
	var status, reservationMode string
	err := tx.QueryRow(ctx, `SELECT status, reservation_mode FROM orders WHERE id = $1 AND shop_id = $2 FOR UPDATE`, orderID, shopID).Scan(&status, &reservationMode)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("lock order: %w", err)
	}
	switch status {
	case "cancelled", "expired", "refund_required":
		return getOrderTx(ctx, tx, shopID, orderID)
	case "pending":
		if reservationMode == "reserved" {
			if err := releaseOrderReservations(ctx, tx, shopID, orderID); err != nil {
				return Order{}, err
			}
			reservationMode = "released"
		}
		status = "cancelled"
	case "paid", "fulfilled":
		// This endpoint records that a refund is required. It does not claim
		// money was returned and does not put sold stock back on the shelf.
		status = "refund_required"
	default:
		return Order{}, ErrStatus
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET status = $2, reservation_mode = $3, state_version = state_version + 1 WHERE id = $1`, orderID, status, reservationMode); err != nil {
		return Order{}, fmt.Errorf("cancel order: %w", err)
	}
	action := "order.cancel"
	if status == "refund_required" {
		action = "order.refund_required"
	}
	if err := insertAudit(ctx, tx, shopID, actorID, action, "order", orderID); err != nil {
		return Order{}, err
	}
	return getOrderTx(ctx, tx, shopID, orderID)
}

// ConfirmRefundCommand closes the refund_required -> cancelled exit: stock is
// restored to the shelf, a reverse `refund` movement is written, and refunded_at
// records when the money was actually returned. A repeat call on an already
// confirmed order is idempotent and does not restore stock a second time.
func (s *ShopStore) ConfirmRefundCommand(ctx context.Context, identity CommandIdentity, orderID ID) (OrderCommandResult, error) {
	return s.runOrderCommand(ctx, identity, 200, func(tx pgx.Tx) (Order, error) {
		return confirmRefundTx(ctx, tx, identity.ShopID, identity.ActorID, orderID)
	})
}

// refundConfirmDecision classifies the locked order before any refund mutation.
// "idempotent" means the refund was already confirmed (cancelled with
// refunded_at set) and stock must not be restored twice; "proceed" means the
// order is waiting for refund confirmation. Any other status is ErrStatus.
func refundConfirmDecision(status string, refundedAt *time.Time) (string, error) {
	if status == "cancelled" && refundedAt != nil {
		return "idempotent", nil
	}
	if status != "refund_required" {
		return "", ErrStatus
	}
	return "proceed", nil
}

func confirmRefundTx(ctx context.Context, tx pgx.Tx, shopID, actorID, orderID ID) (Order, error) {
	var status string
	var refundedAt *time.Time
	err := tx.QueryRow(ctx, `SELECT status, refunded_at FROM orders WHERE id = $1 AND shop_id = $2 FOR UPDATE`, orderID, shopID).Scan(&status, &refundedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("lock order for refund: %w", err)
	}
	decision, err := refundConfirmDecision(status, refundedAt)
	if err != nil {
		return Order{}, err
	}
	if decision == "idempotent" {
		return getOrderTx(ctx, tx, shopID, orderID)
	}
	if err := applyStock(ctx, tx, orderID, false); err != nil {
		return Order{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO inventory_movements(shop_id, order_id, listing_id, movement_type, qty)
		SELECT $1, $2, listing_id, 'refund', SUM(qty)
		FROM order_lines WHERE order_id = $2
		GROUP BY listing_id`, shopID, orderID); err != nil {
		return Order{}, fmt.Errorf("write refund movements: %w", err)
	}
	if _, err := tx.Exec(ctx, `UPDATE orders SET status = 'cancelled', refunded_at = now(), state_version = state_version + 1 WHERE id = $1`, orderID); err != nil {
		return Order{}, fmt.Errorf("confirm refund: %w", err)
	}
	if err := insertAudit(ctx, tx, shopID, actorID, "order.refund_confirm", "order", orderID); err != nil {
		return Order{}, err
	}
	return getOrderTx(ctx, tx, shopID, orderID)
}

func applyStock(ctx context.Context, tx pgx.Tx, orderID ID, sell bool) error {
	rows, err := tx.Query(ctx, `
		SELECT ol.listing_id::text, ol.good_id::text, ol.qty
		FROM order_lines ol WHERE ol.order_id = $1 ORDER BY ol.good_id, ol.listing_id`, orderID)
	if err != nil {
		return fmt.Errorf("order lines: %w", err)
	}
	defer rows.Close()
	type line struct {
		listingID, goodID ID
		qty               int
	}
	var lines []line
	for rows.Next() {
		var item line
		if err := rows.Scan(&item.listingID, &item.goodID, &item.qty); err != nil {
			return fmt.Errorf("scan order line: %w", err)
		}
		lines = append(lines, item)
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("order lines: %w", err)
	}
	seenGoods := map[ID]struct{}{}
	for _, item := range lines {
		var total, listed, sold, reserved int
		err := tx.QueryRow(ctx, `SELECT total_stock FROM goods WHERE id = $1 FOR UPDATE`, item.goodID).Scan(&total)
		if err != nil {
			return fmt.Errorf("lock good: %w", err)
		}
		err = tx.QueryRow(ctx, `SELECT listed_qty, sold_qty, reserved_qty FROM event_listings WHERE id = $1 FOR UPDATE`, item.listingID).Scan(&listed, &sold, &reserved)
		if err != nil {
			return fmt.Errorf("lock listing: %w", err)
		}
		var nextTotal int
		var next catalog.ListingQty
		if sell {
			if listed-sold-reserved < item.qty {
				return catalog.ErrNotEnough
			}
			nextTotal, next, err = catalog.ConfirmSale(total, catalog.ListingQty{Listed: listed, Sold: sold}, item.qty)
		} else {
			nextTotal, next, err = catalog.RestoreSale(total, catalog.ListingQty{Listed: listed, Sold: sold}, item.qty)
		}
		if err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `UPDATE goods SET total_stock = $2 WHERE id = $1`, item.goodID, nextTotal); err != nil {
			return fmt.Errorf("write stock: %w", err)
		}
		if _, err := tx.Exec(ctx, `UPDATE event_listings SET sold_qty = $2 WHERE id = $1`, item.listingID, next.Sold); err != nil {
			return fmt.Errorf("write sold: %w", err)
		}
		seenGoods[item.goodID] = struct{}{}
	}
	for goodID := range seenGoods {
		if err := assertAllocation(ctx, tx, goodID); err != nil {
			return err
		}
	}
	return nil
}

func (s *ShopStore) GetOrder(ctx context.Context, shopID, orderID ID) (Order, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return Order{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	return getOrderTx(ctx, tx, shopID, orderID)
}

func (s *ShopStore) GetOrderByClientOrderID(ctx context.Context, shopID ID, clientOrderID string) (Order, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{AccessMode: pgx.ReadOnly})
	if err != nil {
		return Order{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var orderID ID
	err = tx.QueryRow(ctx, `SELECT id::text FROM orders WHERE shop_id = $1 AND client_order_id = $2`, shopID, clientOrderID).Scan(&orderID)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("lookup checkout: %w", err)
	}
	return getOrderTx(ctx, tx, shopID, orderID)
}

func getOrderTx(ctx context.Context, tx pgx.Tx, shopID, orderID ID) (Order, error) {
	var order Order
	err := tx.QueryRow(ctx, `
		SELECT o.id::text, o.number, o.status, o.booth_name, o.currency, o.pay_code_name,
			opener.login_name, o.opened_by_role, COALESCE(payer.login_name, ''), COALESCE(payer.role, ''),
			o.event_id::text, e.name, o.total_minor, o.state_version, o.created_at, o.reservation_expires_at, o.refunded_at,
			o.cash_received_minor, o.cash_change_minor
		FROM orders o
		JOIN users opener ON opener.id = o.opened_by
		LEFT JOIN users payer ON payer.id = o.paid_by
		JOIN events e ON e.id = o.event_id
		WHERE o.id = $1 AND o.shop_id = $2`, orderID, shopID).
		Scan(&order.ID, &order.Number, &order.Status, &order.BoothName, &order.Currency, &order.PayCodeName,
			&order.OpenedByLogin, &order.OpenedByRole, &order.PaidByLogin, &order.PaidByRole,
			&order.EventID, &order.EventName, &order.TotalMinor, &order.StateVersion, &order.CreatedAt, &order.ReservationExpiresAt, &order.RefundedAt,
			&order.CashReceivedMinor, &order.CashChangeMinor)
	if errors.Is(err, pgx.ErrNoRows) {
		return Order{}, ErrNotFound
	}
	if err != nil {
		return Order{}, fmt.Errorf("read order snapshot: %w", err)
	}
	rows, err := tx.Query(ctx, `
		SELECT good_name, unit_price_minor, qty, line_total_minor
		FROM order_lines WHERE order_id = $1 ORDER BY id`, orderID)
	if err != nil {
		return Order{}, fmt.Errorf("read order lines: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var line OrderLine
		if err := rows.Scan(&line.GoodName, &line.UnitPriceMinor, &line.Qty, &line.LineTotalMinor); err != nil {
			return Order{}, fmt.Errorf("scan order line: %w", err)
		}
		order.Lines = append(order.Lines, line)
	}
	if err := rows.Err(); err != nil {
		return Order{}, fmt.Errorf("read order lines: %w", err)
	}
	groupRows, err := tx.Query(ctx, `SELECT snapshot FROM order_pricing_groups WHERE order_id = $1 ORDER BY id`, orderID)
	if err != nil {
		return Order{}, fmt.Errorf("read order pricing groups: %w", err)
	}
	defer groupRows.Close()
	for groupRows.Next() {
		var snapshot []byte
		var group catalog.PricingGroup
		if err := groupRows.Scan(&snapshot); err != nil {
			return Order{}, fmt.Errorf("scan order pricing group: %w", err)
		}
		if err := json.Unmarshal(snapshot, &group); err != nil {
			return Order{}, fmt.Errorf("decode order pricing group: %w", err)
		}
		order.PricingGroups = append(order.PricingGroups, group)
	}
	if err := groupRows.Err(); err != nil {
		return Order{}, fmt.Errorf("read order pricing groups: %w", err)
	}
	return order, nil
}

func (s *ShopStore) ListOrders(ctx context.Context, shopID ID, eventID string) ([]Order, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT o.id::text, o.number, o.status, o.booth_name, o.currency, o.pay_code_name,
			opener.login_name, o.opened_by_role, COALESCE(payer.login_name, ''), COALESCE(payer.role, ''),
			o.event_id::text, e.name, o.total_minor, o.state_version, o.created_at, o.reservation_expires_at, o.refunded_at,
			o.cash_received_minor, o.cash_change_minor
		FROM orders o
		JOIN users opener ON opener.id = o.opened_by
		LEFT JOIN users payer ON payer.id = o.paid_by
		JOIN events e ON e.id = o.event_id
		WHERE o.shop_id = $1 AND ($2 = '' OR o.event_id::text = $2)
		ORDER BY o.created_at DESC`, shopID, eventID)
	if err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	defer rows.Close()
	var orders []Order
	for rows.Next() {
		var order Order
		if err := rows.Scan(&order.ID, &order.Number, &order.Status, &order.BoothName, &order.Currency, &order.PayCodeName, &order.OpenedByLogin, &order.OpenedByRole, &order.PaidByLogin, &order.PaidByRole, &order.EventID, &order.EventName, &order.TotalMinor, &order.StateVersion, &order.CreatedAt, &order.ReservationExpiresAt, &order.RefundedAt, &order.CashReceivedMinor, &order.CashChangeMinor); err != nil {
			return nil, fmt.Errorf("scan order: %w", err)
		}
		orders = append(orders, order)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list orders: %w", err)
	}
	for i := range orders {
		lines, err := s.orderLines(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].Lines = lines
		groups, err := s.orderPricingGroups(ctx, orders[i].ID)
		if err != nil {
			return nil, err
		}
		orders[i].PricingGroups = groups
	}
	return orders, nil
}

func (s *ShopStore) orderPricingGroups(ctx context.Context, orderID ID) ([]catalog.PricingGroup, error) {
	rows, err := s.pool.Query(ctx, `SELECT snapshot FROM order_pricing_groups WHERE order_id = $1 ORDER BY id`, orderID)
	if err != nil {
		return nil, fmt.Errorf("order pricing groups: %w", err)
	}
	defer rows.Close()
	var groups []catalog.PricingGroup
	for rows.Next() {
		var snapshot []byte
		var group catalog.PricingGroup
		if err := rows.Scan(&snapshot); err != nil {
			return nil, fmt.Errorf("scan pricing group: %w", err)
		}
		if err := json.Unmarshal(snapshot, &group); err != nil {
			return nil, fmt.Errorf("decode pricing group: %w", err)
		}
		groups = append(groups, group)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("order pricing groups: %w", err)
	}
	return groups, nil
}

func (s *ShopStore) orderLines(ctx context.Context, orderID ID) ([]OrderLine, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT good_name, unit_price_minor, qty, line_total_minor FROM order_lines WHERE order_id = $1 ORDER BY id`, orderID)
	if err != nil {
		return nil, fmt.Errorf("order lines: %w", err)
	}
	defer rows.Close()
	var lines []OrderLine
	for rows.Next() {
		var line OrderLine
		if err := rows.Scan(&line.GoodName, &line.UnitPriceMinor, &line.Qty, &line.LineTotalMinor); err != nil {
			return nil, fmt.Errorf("scan line: %w", err)
		}
		lines = append(lines, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("order lines: %w", err)
	}
	return lines, nil
}

func (s *ShopStore) ListAudit(ctx context.Context, shopID ID) ([]AuditEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.action, u.login_name, a.subject_type, a.subject_id::text, to_char(a.created_at, 'YYYY-MM-DD HH24:MI')
		FROM audit_logs a JOIN users u ON u.id = a.actor_id
		WHERE a.shop_id = $1 ORDER BY a.created_at DESC LIMIT 100`, shopID)
	if err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	defer rows.Close()
	var out []AuditEntry
	for rows.Next() {
		var entry AuditEntry
		if err := rows.Scan(&entry.Action, &entry.ActorLogin, &entry.SubjectType, &entry.SubjectID, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit: %w", err)
		}
		out = append(out, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list audit: %w", err)
	}
	return out, nil
}

func (s *ShopStore) CreatePayCode(ctx context.Context, shopID ID, name, path string, currency *string, isDefault bool) (ID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin pay code: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if isDefault {
		if _, err := tx.Exec(ctx, `UPDATE pay_codes SET is_default = false WHERE shop_id = $1`, shopID); err != nil {
			return 0, fmt.Errorf("clear default pay code: %w", err)
		}
	}
	var id ID
	err = tx.QueryRow(ctx, `
		INSERT INTO pay_codes (shop_id, name, image_path, currency, is_default, is_enabled)
		VALUES ($1,$2,$3,$4,$5,true) RETURNING id::text`, shopID, name, path, currency, isDefault).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert pay code: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit pay code: %w", err)
	}
	return id, nil
}

// CreateCashPayCode inserts the shop's single cash method. A repeat call returns
// the existing row. The shop row lock stops two requests from inserting together.
func (s *ShopStore) CreateCashPayCode(ctx context.Context, shopID ID, name string, isDefault bool) (ID, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin cash pay code: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var locked ID
	err = tx.QueryRow(ctx, `SELECT id::text FROM shops WHERE id = $1 FOR UPDATE`, shopID).Scan(&locked)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("lock shop for cash pay code: %w", err)
	}
	var existing ID
	err = tx.QueryRow(ctx, `
		SELECT id::text FROM pay_codes WHERE shop_id = $1 AND kind = 'cash' ORDER BY id LIMIT 1`, shopID).Scan(&existing)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return 0, fmt.Errorf("commit cash pay code: %w", err)
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("read cash pay code: %w", err)
	}
	if isDefault {
		if _, err := tx.Exec(ctx, `UPDATE pay_codes SET is_default = false WHERE shop_id = $1`, shopID); err != nil {
			return 0, fmt.Errorf("clear default pay code: %w", err)
		}
	}
	var id ID
	err = tx.QueryRow(ctx, `
		INSERT INTO pay_codes (shop_id, name, image_path, kind, is_default, is_enabled)
		VALUES ($1,$2,'','cash',$3,true) RETURNING id::text`, shopID, name, isDefault).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert cash pay code: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit cash pay code: %w", err)
	}
	return id, nil
}

func (s *ShopStore) UpdatePayCode(ctx context.Context, shopID, id ID, name string, currency *string, enabled, isDefault bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin update pay code: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if isDefault {
		if _, err := tx.Exec(ctx, `UPDATE pay_codes SET is_default = false WHERE shop_id = $1`, shopID); err != nil {
			return fmt.Errorf("clear default pay code: %w", err)
		}
	}
	if !enabled {
		isDefault = false
	}
	tag, err := tx.Exec(ctx, `
		UPDATE pay_codes SET name = $3, currency = COALESCE($4, currency), is_enabled = $5, is_default = $6 WHERE id = $1 AND shop_id = $2`,
		id, shopID, name, currency, enabled, isDefault)
	if err != nil {
		return fmt.Errorf("update pay code: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return tx.Commit(ctx)
}

func (s *ShopStore) ListPayCodes(ctx context.Context, shopID ID, enabledOnly bool) ([]PayCode, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id::text, name, kind, currency, is_default, is_enabled FROM pay_codes
		WHERE shop_id = $1 AND (NOT $2 OR is_enabled)
		ORDER BY is_default DESC, name`, shopID, enabledOnly)
	if err != nil {
		return nil, fmt.Errorf("list pay codes: %w", err)
	}
	defer rows.Close()
	var out []PayCode
	for rows.Next() {
		var code PayCode
		if err := rows.Scan(&code.ID, &code.Name, &code.Kind, &code.Currency, &code.IsDefault, &code.IsEnabled); err != nil {
			return nil, fmt.Errorf("scan pay code: %w", err)
		}
		out = append(out, code)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list pay codes: %w", err)
	}
	return out, nil
}

func (s *ShopStore) PayCodePath(ctx context.Context, id ID) (shopID ID, path string, err error) {
	err = s.pool.QueryRow(ctx, `SELECT shop_id::text, image_path FROM pay_codes WHERE id = $1`, id).Scan(&shopID, &path)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", ErrNotFound
	}
	if err != nil {
		return 0, "", fmt.Errorf("pay code file: %w", err)
	}
	return shopID, path, nil
}

func (s *ShopStore) GoodImagePath(ctx context.Context, id ID) (string, error) {
	var path string
	err := s.pool.QueryRow(ctx, `
		SELECT i.path
		FROM good_images i
		JOIN goods g ON g.id = i.good_id
		JOIN shops s ON s.id = g.shop_id
		WHERE i.id = $1 AND s.is_enabled AND g.review_status = 'approved'`, id).Scan(&path)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("good image: %w", err)
	}
	return path, nil
}

func newOrderNumber() (string, error) {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, 6)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("order number: %w", err)
	}
	for i := range buf {
		buf[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(buf), nil
}
