//go:build integration

package data

import "testing"

func TestConfirmRefundRestoresStockAndAggregatesMovements(t *testing.T) {
	pool, ctx := newIsolatedMigratedIntegrationPool(t, "refund_movement")
	var shop, actor, event, good, listing, order string
	if err := pool.QueryRow(ctx, `INSERT INTO shops(name,slug) VALUES ('Refund Shop','refund-shop') RETURNING id::text`).Scan(&shop); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO users(login_name,password_hash,role,shop_id) VALUES ('owner','hash','shop_admin',$1) RETURNING id::text`, shop).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO events(shop_id,name,starts_on,ends_on,currency,status) VALUES ($1,'Event',CURRENT_DATE,CURRENT_DATE,'MYR','live') RETURNING id::text`, shop).Scan(&event); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO goods(shop_id,name,total_stock) VALUES ($1,'item',2) RETURNING id::text`, shop).Scan(&good); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO event_listings(shop_id,event_id,good_id,listed_qty,sold_qty,unit_price_minor) VALUES ($1,$2,$3,5,3,100) RETURNING id::text`, shop, event, good).Scan(&listing); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO orders(shop_id,event_id,number,status,total_minor,currency,opened_by,opened_by_role,reservation_mode) VALUES ($1,$2,'AAAAAA','refund_required',300,'MYR',$3,'shop_admin','consumed') RETURNING id::text`, shop, event, actor).Scan(&order); err != nil {
		t.Fatal(err)
	}
	for _, qty := range []int{1, 2} {
		if _, err := pool.Exec(ctx, `INSERT INTO order_lines(order_id,good_id,listing_id,shop_id,event_id,good_name,unit_price_minor,qty,line_total_minor,snapshot_version) VALUES ($1,$2,$3,$4,$5,'item',100,$6,$7,1)`, order, good, listing, shop, event, qty, qty*100); err != nil {
			t.Fatal(err)
		}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	shopID, actorID, orderID := integrationID(t, shop), integrationID(t, actor), integrationID(t, order)
	result, err := confirmRefundTx(ctx, tx, shopID, actorID, orderID)
	if err != nil {
		t.Fatalf("confirm refund: %v", err)
	}
	if result.Status != "cancelled" || result.RefundedAt == nil {
		t.Fatalf("unexpected order: status %s refunded %v", result.Status, result.RefundedAt)
	}
	var total, sold, movementCount, movementQty int
	if err := tx.QueryRow(ctx, `SELECT total_stock FROM goods WHERE id=$1`, good).Scan(&total); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT sold_qty FROM event_listings WHERE id=$1`, listing).Scan(&sold); err != nil {
		t.Fatal(err)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*),coalesce(sum(qty),0) FROM inventory_movements WHERE order_id=$1 AND movement_type='refund'`, order).Scan(&movementCount, &movementQty); err != nil {
		t.Fatal(err)
	}
	if total != 5 || sold != 0 || movementCount != 1 || movementQty != 3 {
		t.Fatalf("stock=%d sold=%d movement count=%d qty=%d", total, sold, movementCount, movementQty)
	}
	if _, err := confirmRefundTx(ctx, tx, shopID, actorID, orderID); err != nil {
		t.Fatalf("repeat refund: %v", err)
	}
	if err := tx.QueryRow(ctx, `SELECT count(*),coalesce(sum(qty),0) FROM inventory_movements WHERE order_id=$1 AND movement_type='refund'`, order).Scan(&movementCount, &movementQty); err != nil {
		t.Fatal(err)
	}
	if movementCount != 1 || movementQty != 3 {
		t.Fatalf("repeat refund movement count=%d qty=%d", movementCount, movementQty)
	}
}
