//go:build integration

package data

import (
	"errors"
	"testing"
)

func TestPayOrderRejectsPartiallyExpiredReservations(t *testing.T) {
	pool, ctx := newIsolatedMigratedIntegrationPool(t, "p1_partial_reservation")
	var shop, actor, event, goodOne, goodTwo, listingOne, listingTwo, order, payCode string
	if err := pool.QueryRow(ctx, `INSERT INTO shops(name,slug) VALUES ('P1 Reservation Shop','p1-reservation-shop') RETURNING id::text`).Scan(&shop); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO users(login_name,password_hash,role,shop_id) VALUES ('owner','hash','shop_admin',$1) RETURNING id::text`, shop).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO events(shop_id,name,starts_on,ends_on,currency,status) VALUES ($1,'P1 Event',CURRENT_DATE,CURRENT_DATE,'MYR','scheduled') RETURNING id::text`, shop).Scan(&event); err != nil {
		t.Fatal(err)
	}
	for _, target := range []*string{&goodOne, &goodTwo} {
		if err := pool.QueryRow(ctx, `INSERT INTO goods(shop_id,name,total_stock) VALUES ($1,'item',5) RETURNING id::text`, shop).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range []struct {
		good, name string
		listing    *string
	}{{goodOne, "first", &listingOne}, {goodTwo, "second", &listingTwo}} {
		if err := pool.QueryRow(ctx, `INSERT INTO event_listings(shop_id,event_id,good_id,listed_qty,unit_price_minor) VALUES ($1,$2,$3,5,100) RETURNING id::text`, shop, event, fixture.good).Scan(fixture.listing); err != nil {
			t.Fatal(err)
		}
	}
	if err := pool.QueryRow(ctx, `INSERT INTO orders(shop_id,event_id,number,status,total_minor,currency,opened_by,opened_by_role,reservation_mode) VALUES ($1,$2,'AAAAAA','pending',300,'MYR',$3,'shop_admin','reserved') RETURNING id::text`, shop, event, actor).Scan(&order); err != nil {
		t.Fatal(err)
	}
	for _, fixture := range []struct {
		good, listing string
		qty           int
	}{{goodOne, listingOne, 2}, {goodTwo, listingTwo, 1}} {
		if _, err := pool.Exec(ctx, `INSERT INTO order_lines(order_id,good_id,listing_id,shop_id,event_id,good_name,unit_price_minor,qty,line_total_minor,snapshot_version) VALUES ($1,$2,$3,$4,$5,'item',100,$6,$7,1)`, order, fixture.good, fixture.listing, shop, event, fixture.qty, fixture.qty*100); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE event_listings SET reserved_qty=$2 WHERE id=$1`, fixture.listing, fixture.qty); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `INSERT INTO inventory_reservations(shop_id,order_id,listing_id,qty,state,expires_at) VALUES ($1,$2,$3,2,'active',now()+interval '1 hour'),($1,$2,$4,1,'active',now()-interval '1 second')`, shop, order, listingOne, listingTwo); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO pay_codes(shop_id,name,image_path,kind,is_default) VALUES ($1,'Cash','unused.png','cash',true) RETURNING id::text`, shop).Scan(&payCode); err != nil {
		t.Fatal(err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	shopID, actorID := integrationID(t, shop), integrationID(t, actor)
	orderID, payCodeID := integrationID(t, order), integrationID(t, payCode)
	_, err = payOrderTx(ctx, tx, shopID, actorID, orderID, payCodeID, nil)
	if !errors.Is(err, ErrReservationExpired) {
		_ = tx.Rollback(ctx)
		t.Fatalf("payOrderTx error = %v, want ErrReservationExpired", err)
	}
	var status string
	if err := tx.QueryRow(ctx, `SELECT status FROM orders WHERE id=$1`, orderID).Scan(&status); err != nil || status != "pending" {
		_ = tx.Rollback(ctx)
		t.Fatalf("order status = %q; err = %v, want pending", status, err)
	}
	for _, fixture := range []struct {
		good, listing string
		reserved      int
	}{{goodOne, listingOne, 2}, {goodTwo, listingTwo, 1}} {
		var stock, sold, reserved int
		if err := tx.QueryRow(ctx, `SELECT total_stock FROM goods WHERE id=$1`, fixture.good).Scan(&stock); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatal(err)
		}
		if err := tx.QueryRow(ctx, `SELECT sold_qty,reserved_qty FROM event_listings WHERE id=$1`, fixture.listing).Scan(&sold, &reserved); err != nil {
			_ = tx.Rollback(ctx)
			t.Fatal(err)
		}
		if stock != 5 || sold != 0 || reserved != fixture.reserved {
			_ = tx.Rollback(ctx)
			t.Fatalf("listing %s after rejected payment: stock=%d sold=%d reserved=%d", fixture.listing, stock, sold, reserved)
		}
	}
	if err := tx.Rollback(ctx); err != nil {
		t.Fatal(err)
	}
}
