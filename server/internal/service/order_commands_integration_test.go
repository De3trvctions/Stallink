//go:build integration

package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestOrderCommandsAreDurableAndIdempotent(t *testing.T) {
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
	login := "itowner" + suffix
	slug := "it-" + suffix
	shopID, err := store.OpenShop(ctx, "Idempotency Test", slug, "", login, passwordHash)
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
	eventID, err := store.CreateEvent(ctx, shopID, data.EventInput{Name: "Test Event", StartsOn: starts, EndsOn: ends, Currency: "MYR", Status: "scheduled"})
	if err != nil {
		t.Fatal(err)
	}
	_, listingID, _, err := store.CreateListedGood(ctx, shopID, account.ID, data.ListedGood{
		Name: "Test Good", TotalStock: 100, EventID: eventID, ListedQty: 100,
		UnitPriceMinor: 1000, IsListed: true, Approved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	payCodeID, err := store.CreatePayCode(ctx, shopID, "Cash", "unused", true)
	if err != nil {
		t.Fatal(err)
	}
	handler := server.NewHTTP(":0", "order-command-test", &service.API{
		DB: store, RequireIdempotency: true, ReservationsEnabled: true, ReservationTTL: 15 * time.Minute,
	})
	cookie := &http.Cookie{Name: "stallink_session", Value: plainToken}
	clientOrderID := uuid.NewString()
	key := uuid.NewString()
	quote := requestQuote(t, handler, cookie, eventID, listingID, 1, 1)
	body := map[string]any{
		"clientOrderId": clientOrderID, "quoteId": quote.ID,
		"expectedTotalMinor": quote.TotalMinor, "boothName": "Booth A",
	}

	const parallel = 50
	responses := make([]*httptest.ResponseRecorder, parallel)
	var wg sync.WaitGroup
	for i := range responses {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			responses[i] = commandRequest(t, handler, http.MethodPost, "/api/staff/orders", body, cookie, key)
		}(i)
	}
	wg.Wait()
	var orderID string
	for i, response := range responses {
		if response.Code != http.StatusCreated {
			t.Fatalf("parallel create %d: status %d, body %s", i, response.Code, response.Body.String())
		}
		var got struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if orderID == "" {
			orderID = got.ID
		}
		if got.ID != orderID {
			t.Fatalf("parallel duplicate checkout returned order %q, want %q", got.ID, orderID)
		}
	}
	assertCount(t, ctx, pool, `SELECT count(*) FROM orders WHERE shop_id = $1`, shopID, 1)
	assertCount(t, ctx, pool, `SELECT count(*) FROM audit_logs WHERE subject_id = $1 AND action = 'order.create'`, orderID, 1)
	assertCount(t, ctx, pool, `SELECT count(*) FROM inventory_reservations WHERE order_id = $1 AND state = 'active'`, orderID, 1)
	assertCount(t, ctx, pool, `SELECT count(*) FROM event_listings WHERE id = $1 AND reserved_qty = 1`, listingID, 1)

	changedQuote := requestQuote(t, handler, cookie, eventID, listingID, 2, 2)
	changed := map[string]any{
		"clientOrderId": clientOrderID, "quoteId": changedQuote.ID,
		"expectedTotalMinor": changedQuote.TotalMinor, "boothName": "Booth A",
	}
	if response := commandRequest(t, handler, http.MethodPost, "/api/staff/orders", changed, cookie, key); response.Code != http.StatusConflict {
		t.Fatalf("same key with changed request: status %d, body %s", response.Code, response.Body.String())
	}
	if response := commandRequest(t, handler, http.MethodPost, "/api/staff/orders", body, cookie, uuid.NewString()); response.Code != http.StatusCreated {
		t.Fatalf("new command key for existing checkout: status %d, body %s", response.Code, response.Body.String())
	} else {
		var got struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil || got.ID != orderID {
			t.Fatalf("new key did not recover checkout: body %s, err %v", response.Body.String(), err)
		}
	}
	if response := commandRequest(t, handler, http.MethodPost, "/api/staff/orders", changed, cookie, uuid.NewString()); response.Code != http.StatusConflict {
		t.Fatalf("changed request for existing checkout: status %d, body %s", response.Code, response.Body.String())
	}

	payPath := "/api/staff/orders/" + orderID + "/pay"
	payBody := map[string]string{"payCodeId": payCodeID}
	payKey := uuid.NewString()
	for attempt := 0; attempt < 2; attempt++ {
		response := commandRequest(t, handler, http.MethodPost, payPath, payBody, cookie, payKey)
		if response.Code != http.StatusOK {
			t.Fatalf("pay attempt %d: status %d, body %s", attempt, response.Code, response.Body.String())
		}
	}
	assertCount(t, ctx, pool, `SELECT count(*) FROM order_payments WHERE order_id = $1`, orderID, 1)
	assertCount(t, ctx, pool, `SELECT count(*) FROM audit_logs WHERE subject_id = $1 AND action = 'order.pay'`, orderID, 1)
	assertCount(t, ctx, pool, `SELECT count(*) FROM inventory_reservations WHERE order_id = $1 AND state = 'consumed'`, orderID, 1)
	assertCount(t, ctx, pool, `SELECT count(*) FROM inventory_movements WHERE order_id = $1 AND movement_type = 'sell'`, orderID, 1)
	assertCount(t, ctx, pool, `SELECT count(*) FROM event_listings WHERE id = $1 AND reserved_qty = 0 AND sold_qty = 1`, listingID, 1)

	fulfillPath := "/api/staff/orders/" + orderID + "/fulfill"
	fulfillKey := uuid.NewString()
	for attempt := 0; attempt < 2; attempt++ {
		if response := commandRequest(t, handler, http.MethodPost, fulfillPath, map[string]any{}, cookie, fulfillKey); response.Code != http.StatusOK {
			t.Fatalf("fulfill attempt %d: status %d, body %s", attempt, response.Code, response.Body.String())
		}
	}
	assertCount(t, ctx, pool, `SELECT count(*) FROM audit_logs WHERE subject_id = $1 AND action = 'order.fulfill'`, orderID, 1)
	if response := commandRequest(t, handler, http.MethodPost, "/api/staff/orders/"+orderID+"/cancel", map[string]any{}, cookie, uuid.NewString()); response.Code != http.StatusOK {
		t.Fatalf("cancel fulfilled order: status %d, body %s", response.Code, response.Body.String())
	} else {
		var got struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil || got.Status != "refund_required" {
			t.Fatalf("cancel did not require a refund: body %s, err %v", response.Body.String(), err)
		}
	}
	assertCount(t, ctx, pool, `SELECT count(*) FROM audit_logs WHERE subject_id = $1 AND action = 'order.refund_required'`, orderID, 1)
	if response := commandRequest(t, handler, http.MethodPost, payPath, payBody, cookie, payKey); response.Code != http.StatusOK {
		t.Fatalf("replay paid command after later cancel: status %d, body %s", response.Code, response.Body.String())
	} else {
		var got struct {
			Status       string `json:"status"`
			StateVersion int64  `json:"stateVersion"`
		}
		if err := json.Unmarshal(response.Body.Bytes(), &got); err != nil || got.Status != "refund_required" || got.StateVersion < 3 {
			t.Fatalf("replay returned stale order state: body %s, err %v", response.Body.String(), err)
		}
	}

	legacyHandler := server.NewHTTP(":0", "legacy-reservation-test", &service.API{
		DB: store, RequireIdempotency: true, ReservationsEnabled: false,
	})
	legacyClientOrderID := uuid.NewString()
	legacyQuote := requestQuote(t, legacyHandler, cookie, eventID, listingID, 1, 3)
	legacyBody := map[string]any{
		"clientOrderId": legacyClientOrderID, "quoteId": legacyQuote.ID,
		"expectedTotalMinor": legacyQuote.TotalMinor, "boothName": "Legacy Booth",
	}
	legacyCreate := commandRequest(t, legacyHandler, http.MethodPost, "/api/staff/orders", legacyBody, cookie, uuid.NewString())
	if legacyCreate.Code != http.StatusCreated {
		t.Fatalf("create legacy unreserved order: status %d, body %s", legacyCreate.Code, legacyCreate.Body.String())
	}
	var legacyOrder struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(legacyCreate.Body.Bytes(), &legacyOrder); err != nil {
		t.Fatal(err)
	}
	reservePath := "/api/staff/orders/" + legacyOrder.ID + "/reserve"
	reserveKey := uuid.NewString()
	for attempt := 0; attempt < 2; attempt++ {
		response := commandRequest(t, handler, http.MethodPost, reservePath, map[string]any{}, cookie, reserveKey)
		if response.Code != http.StatusOK {
			t.Fatalf("reserve legacy order attempt %d: status %d, body %s", attempt, response.Code, response.Body.String())
		}
	}
	assertCount(t, ctx, pool, `SELECT count(*) FROM inventory_reservations WHERE order_id = $1 AND state = 'active'`, legacyOrder.ID, 1)
	assertCount(t, ctx, pool, `SELECT count(*) FROM audit_logs WHERE subject_id = $1 AND action = 'order.reserve'`, legacyOrder.ID, 1)
	if response := commandRequest(t, handler, http.MethodPost, "/api/staff/orders/"+legacyOrder.ID+"/cancel", map[string]any{}, cookie, uuid.NewString()); response.Code != http.StatusOK {
		t.Fatalf("cancel reserved legacy order: status %d, body %s", response.Code, response.Body.String())
	}
	assertCount(t, ctx, pool, `SELECT count(*) FROM inventory_reservations WHERE order_id = $1 AND state = 'released'`, legacyOrder.ID, 1)

	missingPayload, _ := json.Marshal(body)
	missingKey := httptest.NewRequest(http.MethodPost, "/api/staff/orders", bytes.NewReader(missingPayload))
	missingKey.Header.Set("Content-Type", "application/json")
	missingKey.AddCookie(cookie)
	missingResponse := httptest.NewRecorder()
	handler.ServeHTTP(missingResponse, missingKey)
	if missingResponse.Code != http.StatusBadRequest || !bytes.Contains(missingResponse.Body.Bytes(), []byte(`"error":"idempotency_key_required"`)) {
		t.Fatalf("missing idempotency key: status %d, body %s", missingResponse.Code, missingResponse.Body.String())
	}
}

type integrationQuote struct {
	ID         string `json:"id"`
	TotalMinor int64  `json:"totalMinor"`
}

func requestQuote(t *testing.T, handler http.Handler, cookie *http.Cookie, eventID, listingID string, qty int, cartVersion int) integrationQuote {
	t.Helper()
	body := map[string]any{
		"eventId": eventID, "cartVersion": cartVersion,
		"lines": []map[string]any{{"kind": "single", "listingId": listingID, "qty": qty}},
	}
	response := commandRequest(t, handler, http.MethodPost, "/api/staff/order-quotes", body, cookie, uuid.NewString())
	if response.Code != http.StatusOK {
		t.Fatalf("create order quote: status %d, body %s", response.Code, response.Body.String())
	}
	var quote integrationQuote
	if err := json.Unmarshal(response.Body.Bytes(), &quote); err != nil {
		t.Fatal(err)
	}
	return quote
}

func isolatedSchemaPool(t *testing.T, ctx context.Context, dsn string) *pgxpool.Pool {
	t.Helper()
	adminConfig, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminConfig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(adminPool.Close)
	schema := fmt.Sprintf("stallink_it_%d", time.Now().UnixNano())
	if _, err := adminPool.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_, _ = adminPool.Exec(cleanupCtx, "DROP SCHEMA "+schema+" CASCADE")
	})
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)
	return pool
}

func commandRequest(t *testing.T, handler http.Handler, method, path string, body any, cookie *http.Cookie, key string) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(method, path, bytes.NewReader(encoded))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Idempotency-Key", key)
	request.AddCookie(cookie)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertCount(t *testing.T, ctx context.Context, pool *pgxpool.Pool, query string, arg string, want int) {
	t.Helper()
	var got int
	if err := pool.QueryRow(ctx, query, arg).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("count query returned %d, want %d: %s", got, want, query)
	}
}
