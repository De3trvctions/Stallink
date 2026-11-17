//go:build integration

package data

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestConcurrentReservationConsumptionUsesStableLockOrder(t *testing.T) {
	dsn := os.Getenv("STALLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STALLINK_TEST_DATABASE_URL must point to an isolated test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()

	schema := fmt.Sprintf("reservation_lock_%d", time.Now().UnixNano())
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cleanupCancel()
		_, _ = admin.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE")
	}()

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}

	var shop, actor, eventOne, eventTwo, goodOne, goodTwo string
	if err := pool.QueryRow(ctx, `INSERT INTO shops(name,slug) VALUES ('Lock Shop','lock-shop') RETURNING id::text`).Scan(&shop); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO users(login_name,password_hash,role,shop_id) VALUES ('owner','hash','shop_admin',$1) RETURNING id::text`, shop).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	for _, target := range []*string{&eventOne, &eventTwo} {
		if err := pool.QueryRow(ctx, `INSERT INTO events(shop_id,name,starts_on,ends_on,currency,status) VALUES ($1,'event',CURRENT_DATE,CURRENT_DATE,'MYR','scheduled') RETURNING id::text`, shop).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	for _, target := range []*string{&goodOne, &goodTwo} {
		if err := pool.QueryRow(ctx, `INSERT INTO goods(shop_id,name,total_stock) VALUES ($1,'good',10) RETURNING id::text`, shop).Scan(target); err != nil {
			t.Fatal(err)
		}
	}

	var listingOne, listingTwo, listingThree, listingFour string
	for _, fixture := range []struct {
		event, good string
		listing     *string
	}{
		{eventOne, goodOne, &listingOne},
		{eventOne, goodTwo, &listingTwo},
		{eventTwo, goodTwo, &listingThree},
		{eventTwo, goodOne, &listingFour},
	} {
		if err := pool.QueryRow(ctx, `INSERT INTO event_listings(shop_id,event_id,good_id,listed_qty,unit_price_minor) VALUES ($1,$2,$3,5,100) RETURNING id::text`, shop, fixture.event, fixture.good).Scan(fixture.listing); err != nil {
			t.Fatal(err)
		}
	}

	var orderOne, orderTwo string
	for i, fixture := range []struct {
		event string
		order *string
	}{{eventOne, &orderOne}, {eventTwo, &orderTwo}} {
		number := fmt.Sprintf("AAAAA%c", 'A'+i)
		if err := pool.QueryRow(ctx, `INSERT INTO orders(shop_id,event_id,number,status,total_minor,currency,opened_by,opened_by_role,reservation_mode) VALUES ($1,$2,$3,'pending',100,'MYR',$4,'shop_admin','reserved') RETURNING id::text`, shop, fixture.event, number, actor).Scan(fixture.order); err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range []struct {
		order, listing string
	}{{orderOne, listingOne}, {orderOne, listingTwo}, {orderTwo, listingThree}, {orderTwo, listingFour}} {
		if _, err := pool.Exec(ctx, `INSERT INTO order_lines(order_id, good_id, listing_id, shop_id, event_id, good_name, unit_price_minor, qty, line_total_minor)
			SELECT $1, l.good_id, l.id, l.shop_id, l.event_id, 'good', 100, 1, 100 FROM event_listings l WHERE l.id=$2`, fixture.order, fixture.listing); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `UPDATE event_listings SET reserved_qty=1 WHERE id=$1`, fixture.listing); err != nil {
			t.Fatal(err)
		}
		if _, err := pool.Exec(ctx, `INSERT INTO inventory_reservations(shop_id,order_id,listing_id,qty,state,expires_at) VALUES ($1,$2,$3,1,'active',now()+interval '1 hour')`, shop, fixture.order, fixture.listing); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := pool.Exec(ctx, `CREATE FUNCTION pause_sell_movement() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.movement_type='sell' THEN PERFORM pg_sleep(0.35); END IF; RETURN NEW; END $$`); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `CREATE TRIGGER pause_sell_movement BEFORE INSERT ON inventory_movements FOR EACH ROW EXECUTE FUNCTION pause_sell_movement()`); err != nil {
		t.Fatal(err)
	}

	shopID := integrationID(t, shop)
	orderIDs := []ID{integrationID(t, orderOne), integrationID(t, orderTwo)}
	start := make(chan struct{})
	results := make(chan error, len(orderIDs))
	for _, orderID := range orderIDs {
		go func(orderID ID) {
			<-start
			tx, err := pool.Begin(ctx)
			if err != nil {
				results <- err
				return
			}
			defer func() { _ = tx.Rollback(ctx) }()
			if err := consumeOrderReservations(ctx, tx, shopID, orderID); err != nil {
				results <- err
				return
			}
			results <- tx.Commit(ctx)
		}(orderID)
	}
	close(start)
	for range orderIDs {
		if err := <-results; err != nil {
			t.Fatalf("concurrent reservation consumption failed: %v", err)
		}
	}
}
