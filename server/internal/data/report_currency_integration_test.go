//go:build integration

package data

import (
	"testing"
	"time"
)

func TestBuildDailyReportSeparatesCurrencies(t *testing.T) {
	pool, ctx := newIsolatedMigratedIntegrationPool(t, "p1_report_currency")
	var shop, actor, event, myrGood, sgdGood, myrListing, sgdListing string
	if err := pool.QueryRow(ctx, `INSERT INTO shops(name,slug) VALUES ('P1 Report Shop','p1-report-shop') RETURNING id::text`).Scan(&shop); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO users(login_name,password_hash,role,shop_id) VALUES ('owner','hash','shop_admin',$1) RETURNING id::text`, shop).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO events(shop_id,name,starts_on,ends_on,currency,status) VALUES ($1,'Shared Event',CURRENT_DATE,CURRENT_DATE,'MYR','scheduled') RETURNING id::text`, shop).Scan(&event); err != nil {
		t.Fatal(err)
	}
	for _, target := range []*string{&myrGood, &sgdGood} {
		if err := pool.QueryRow(ctx, `INSERT INTO goods(shop_id,name,total_stock) VALUES ($1,'item',5) RETURNING id::text`, shop).Scan(target); err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range []struct {
		good    string
		listing *string
	}{{myrGood, &myrListing}, {sgdGood, &sgdListing}} {
		if err := pool.QueryRow(ctx, `INSERT INTO event_listings(shop_id,event_id,good_id,listed_qty,unit_price_minor) VALUES ($1,$2,$3,5,10000) RETURNING id::text`, shop, event, fixture.good).Scan(fixture.listing); err != nil {
			t.Fatal(err)
		}
	}
	for _, fixture := range []struct {
		number, currency, status, good, listing string
		total                                   int64
		paidLine                                bool
	}{
		{"AAAAAA", "MYR", "paid", myrGood, myrListing, 10000, true},
		{"AAAAAB", "SGD", "paid", sgdGood, sgdListing, 4000, true},
		{"AAAAAC", "SGD", "pending", "", "", 0, false},
	} {
		var order string
		if err := pool.QueryRow(ctx, `INSERT INTO orders(shop_id,event_id,number,status,total_minor,currency,opened_by,opened_by_role) VALUES ($1,$2,$3,$4,$5,$6,$7,'shop_admin') RETURNING id::text`, shop, event, fixture.number, fixture.status, fixture.total, fixture.currency, actor).Scan(&order); err != nil {
			t.Fatal(err)
		}
		if fixture.paidLine {
			if _, err := pool.Exec(ctx, `INSERT INTO order_lines(order_id,good_id,listing_id,shop_id,event_id,good_name,unit_price_minor,qty,line_total_minor,snapshot_version) VALUES ($1,$2,$3,$4,$5,'item',$6,1,$6,1)`, order, fixture.good, fixture.listing, shop, event, fixture.total); err != nil {
				t.Fatal(err)
			}
		}
	}

	date := time.Now().In(time.FixedZone("MYT", 8*60*60)).Format("2006-01-02")
	report, err := NewShopStore(pool).BuildDailyReport(ctx, integrationID(t, shop), date, "manual", "")
	if err != nil {
		t.Fatal(err)
	}
	if report.Currency != "MYR" || report.Orders != 1 || report.PaidOrders != 1 || report.PendingOrders != 0 || report.ItemsSold != 1 {
		t.Fatalf("headline = %+v, want MYR-only headline counts", report)
	}
	if len(report.Currencies) != 2 || len(report.Events) != 2 {
		t.Fatalf("currency/event rows = %d/%d, want 2/2", len(report.Currencies), len(report.Events))
	}
	events := make(map[string]DailyReportEvent, len(report.Events))
	for _, row := range report.Events {
		events[row.Currency] = row
	}
	if myr, ok := events["MYR"]; !ok || myr.Orders != 1 || myr.RevenueMinor != 10000 {
		t.Fatalf("MYR event = %+v, want one order and 10000 minor units", myr)
	}
	if sgd, ok := events["SGD"]; !ok || sgd.Orders != 2 || sgd.RevenueMinor != 4000 {
		t.Fatalf("SGD event = %+v, want two orders and 4000 minor units", sgd)
	}
}

// K-Q7 = B / K-Q10 = A: new payments leave cash_received_minor NULL, so the
// snapshot's cashOverShortMinor (still used by the PDF) only reflects
// historical cash-tendered orders.
func TestBuildDailyReportCashOverShortOnlyCountsHistoricalCash(t *testing.T) {
	pool, ctx := newIsolatedMigratedIntegrationPool(t, "report_cash_history")
	var shop, actor, event string
	if err := pool.QueryRow(ctx, `INSERT INTO shops(name,slug) VALUES ('Cash History','cash-history') RETURNING id::text`).Scan(&shop); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO users(login_name,password_hash,role,shop_id) VALUES ('cashowner','hash','shop_admin',$1) RETURNING id::text`, shop).Scan(&actor); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `INSERT INTO events(shop_id,name,starts_on,ends_on,currency,status) VALUES ($1,'Cash Event',CURRENT_DATE,CURRENT_DATE,'MYR','live') RETURNING id::text`, shop).Scan(&event); err != nil {
		t.Fatal(err)
	}
	// Historical cash order: 100 minor over. New-style paid order: no tender recorded.
	if _, err := pool.Exec(ctx, `INSERT INTO orders(shop_id,event_id,number,status,total_minor,currency,opened_by,opened_by_role,cash_received_minor,cash_change_minor)
		VALUES ($1,$2,'CASHAA','paid',1000,'MYR',$3,'shop_admin',1100,100),
		       ($1,$2,'CASHAB','paid',2000,'MYR',$3,'shop_admin',NULL,NULL)`, shop, event, actor); err != nil {
		t.Fatal(err)
	}
	date := time.Now().In(time.FixedZone("MYT", 8*60*60)).Format("2006-01-02")
	report, err := NewShopStore(pool).BuildDailyReport(ctx, integrationID(t, shop), date, "manual", "")
	if err != nil {
		t.Fatal(err)
	}
	if report.CashOverShortMinor != 100 || len(report.Currencies) != 1 || report.Currencies[0].CashOverShortMinor != 100 {
		t.Fatalf("cash over/short = %d / %+v, want 100 from the historical order only", report.CashOverShortMinor, report.Currencies)
	}
}
