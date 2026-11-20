//go:build integration

package service_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/De3trvctions/Stallink/internal/auth"
	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/server"
	"github.com/De3trvctions/Stallink/internal/service"
	"github.com/google/uuid"
)

// TestConcurrentCheckoutNeverOversells hammers a small shared pool with many
// parallel buyers. The pool may be quoted optimistically up front, but the
// reserve step must serialize on the good row: at most STOCK orders may be
// paid, sold_qty must equal the number of paid orders, and no unit may go
// negative or be sold twice.
func TestConcurrentCheckoutNeverOversells(t *testing.T) {
	dsn := os.Getenv("STALLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STALLINK_TEST_DATABASE_URL must point to an isolated test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	pool := isolatedSchemaPool(t, ctx, dsn)
	if err := data.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	store := data.NewShopStore(pool)
	passwordHash, err := auth.HashPassword("integration-password")
	if err != nil {
		t.Fatal(err)
	}
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	login := "itrace" + suffix
	slug := "race-" + suffix
	shopID, err := store.OpenShop(ctx, "Race Shop", slug, "", login, passwordHash)
	if err != nil {
		t.Fatal(err)
	}
	account, err := store.AccountByLogin(ctx, login)
	if err != nil {
		t.Fatal(err)
	}
	plainToken, tokenHash, err := auth.NewToken()
	if err != nil {
		t.Fatal(err)
	}
	if err := store.InsertSession(ctx, account.ID, tokenHash, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	starts := time.Now().Format(time.DateOnly)
	ends := time.Now().AddDate(0, 0, 1).Format(time.DateOnly)
	eventID, err := store.CreateEvent(ctx, shopID, data.EventInput{Name: "Race Event", StartsOn: starts, EndsOn: ends, Currency: "MYR", Status: "live"})
	if err != nil {
		t.Fatal(err)
	}

	const stock = 5
	_, listingID, _, err := store.CreateListedGood(ctx, shopID, account.ID, data.ListedGood{
		Name: "Race Good", TotalStock: stock, EventID: eventID, ListedQty: stock,
		UnitPriceMinor: 1000, IsListed: true, Approved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	payCodeID, err := store.CreatePayCode(ctx, shopID, "Cash", "unused", nil, true)
	if err != nil {
		t.Fatal(err)
	}

	handler := server.NewHTTP(":0", "race-test", &service.API{
		DB: store, RequireIdempotency: true, ReservationsEnabled: true, ReservationTTL: 15 * time.Minute,
	})
	cookie := &http.Cookie{Name: "stallink_session", Value: plainToken}

	const buyers = 20
	type result struct {
		created   bool
		paid      bool
		createCode int
		payCode   int
		payBody   string
	}
	results := make([]result, buyers)
	var wg sync.WaitGroup
	start := make(chan struct{})
	for i := 0; i < buyers; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			quote, ok := quoteAttempt(t, handler, cookie, eventID, listingID, 1)
			if !ok {
				return
			}
			clientOrderID := uuid.NewString()
			body := map[string]any{
				"clientOrderId": clientOrderID, "quoteId": quote.ID,
				"expectedTotalMinor": quote.TotalMinor, "boothName": "Booth",
			}
			createResp := commandRequest(t, handler, http.MethodPost, "/api/staff/orders", body, cookie, uuid.NewString())
			results[i].createCode = createResp.Code
			if createResp.Code != http.StatusCreated {
				return // not enough stock at reserve step
			}
			var created struct {
				ID string `json:"id"`
			}
			if err := json.Unmarshal(createResp.Body.Bytes(), &created); err != nil || created.ID == "" {
				t.Errorf("decode created order: %s err %v", createResp.Body.String(), err)
				return
			}
			results[i].created = true
			payResp := commandRequest(t, handler, http.MethodPost, "/api/staff/orders/"+created.ID+"/pay",
				map[string]any{"payCodeId": payCodeID.String()}, cookie, uuid.NewString())
			results[i].payCode = payResp.Code
			results[i].payBody = payResp.Body.String()
			if payResp.Code == http.StatusOK {
				results[i].paid = true
			}
		}(i)
	}
	close(start)
	wg.Wait()

	created, paid := 0, 0
	var payFailures []string
	createCodes := map[int]int{}
	for _, r := range results {
		createCodes[r.createCode]++
		if r.created {
			created++
		}
		if r.paid {
			paid++
		} else if r.created {
			payFailures = append(payFailures, fmt.Sprintf("code=%d body=%s", r.payCode, r.payBody))
		}
	}
	t.Logf("create status distribution: %v", createCodes)
	if len(payFailures) > 0 {
		t.Logf("non-paid created orders: %v", payFailures)
	}
	if paid > stock {
		t.Fatalf("oversold: %d paid orders against stock %d", paid, stock)
	}
	if created < paid {
		t.Fatalf("paid %d orders but only %d created", paid, created)
	}

	var paidOrders, pendingOrders int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE status='paid'`).Scan(&paidOrders); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM orders WHERE status='pending'`).Scan(&pendingOrders); err != nil {
		t.Fatal(err)
	}
	var totalStock, sold, reserved int
	if err := pool.QueryRow(ctx, `SELECT total_stock FROM goods WHERE id = (SELECT good_id FROM event_listings WHERE id=$1)`, listingID).Scan(&totalStock); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT sold_qty, reserved_qty FROM event_listings WHERE id=$1`, listingID).Scan(&sold, &reserved); err != nil {
		t.Fatal(err)
	}
	if sold != paid {
		t.Fatalf("sold_qty=%d, want %d (paid orders)", sold, paid)
	}
	if totalStock != stock-paid {
		t.Fatalf("remaining total_stock=%d, want %d", totalStock, stock-paid)
	}
	if reserved != pendingOrders {
		t.Fatalf("reserved_qty=%d, want %d (one active reservation per pending order)", reserved, pendingOrders)
	}
	if paidOrders != paid {
		t.Fatalf("paid orders in DB=%d, want %d", paidOrders, paid)
	}
	// Every pending order must still hold an active reservation that reconciles
	// with the pool, and no order may have consumed more than the pool allowed.
	if pendingOrders > stock-paid {
		t.Fatalf("%d pending reservations exceed free pool %d", pendingOrders, stock-paid)
	}
	var poolViolations int
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM goods g WHERE g.total_stock < COALESCE((SELECT SUM(l.reserved_qty) FROM event_listings l WHERE l.good_id=g.id),0)`).Scan(&poolViolations); err != nil {
		t.Fatal(err)
	}
	if poolViolations != 0 {
		t.Fatalf("%d goods violate the shared-pool invariant total>=reserved", poolViolations)
	}
	t.Logf("race finished: %d buyers, %d created, %d paid, stock remaining %d", buyers, created, paid, totalStock)
}

// quoteAttempt returns the quote and true on 200, otherwise an empty quote and
// false. Unlike requestQuote it never calls t.Fatalf from a goroutine.
func quoteAttempt(t *testing.T, handler http.Handler, cookie *http.Cookie, eventID, listingID data.ID, qty int) (integrationQuote, bool) {
	body := map[string]any{
		"eventId": eventID.String(), "cartVersion": 1,
		"lines": []map[string]any{{"kind": "single", "listingId": listingID.String(), "qty": qty}},
	}
	resp := commandRequest(t, handler, http.MethodPost, "/api/staff/order-quotes", body, cookie, uuid.NewString())
	if resp.Code != http.StatusOK {
		return integrationQuote{}, false
	}
	var quote integrationQuote
	if err := json.Unmarshal(resp.Body.Bytes(), &quote); err != nil {
		t.Errorf("decode quote: %v body %s", err, resp.Body.String())
		return integrationQuote{}, false
	}
	return quote, true
}
