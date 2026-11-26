//go:build integration

package service_test

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/De3trvctions/Stallink/internal/auth"
	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/server"
	"github.com/De3trvctions/Stallink/internal/service"
	"github.com/google/uuid"
)

// K-Q10 = A / K-Q17 = A: payment no longer takes a cash-received amount. A
// client that still sends cashReceivedMinor is ignored, not rejected, and the
// order keeps cash_received_minor / cash_change_minor NULL.
func TestPayIgnoresCashReceivedMinor(t *testing.T) {
	dsn := os.Getenv("STALLINK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("STALLINK_TEST_DATABASE_URL must point to an isolated test database")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
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
	loginName := "payowner" + suffix
	shopID, err := store.OpenShop(ctx, "Cash Test", "cash-"+suffix, "", loginName, passwordHash)
	if err != nil {
		t.Fatal(err)
	}
	account, err := store.AccountByLogin(ctx, loginName)
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
	today := time.Now().Format(time.DateOnly)
	eventID, err := store.CreateEvent(ctx, shopID, data.EventInput{Name: "Cash Event", StartsOn: today, EndsOn: today, Currency: "MYR", Status: "live"})
	if err != nil {
		t.Fatal(err)
	}
	_, listingID, _, err := store.CreateListedGood(ctx, shopID, account.ID, data.ListedGood{
		Name: "Cash Good", TotalStock: 100, EventID: eventID, ListedQty: 100,
		UnitPriceMinor: 1000, IsListed: true, Approved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	cashCode, err := store.CreateCashPayCode(ctx, shopID, "现金", true)
	if err != nil {
		t.Fatal(err)
	}
	handler := server.NewHTTP(":0", "pay-cash-test", &service.API{DB: store, RequireIdempotency: true})
	cookie := &http.Cookie{Name: "stallink_session", Value: plainToken}

	for name, body := range map[string]map[string]any{
		"without cash amount":    {"payCodeId": cashCode.String()},
		"legacy cash amount":     {"payCodeId": cashCode.String(), "cashReceivedMinor": 1},
		"legacy negative amount": {"payCodeId": cashCode.String(), "cashReceivedMinor": -5},
	} {
		quote := requestQuote(t, handler, cookie, eventID, listingID, 1, 1)
		created := commandRequest(t, handler, http.MethodPost, "/api/staff/orders", map[string]any{
			"clientOrderId": uuid.NewString(), "quoteId": quote.ID, "expectedTotalMinor": quote.TotalMinor, "boothName": "",
		}, cookie, uuid.NewString())
		if created.Code != http.StatusCreated {
			t.Fatalf("%s: create order %d %s", name, created.Code, created.Body.String())
		}
		var order struct {
			ID string `json:"id"`
		}
		if err := json.Unmarshal(created.Body.Bytes(), &order); err != nil {
			t.Fatal(err)
		}
		paid := commandRequest(t, handler, http.MethodPost, "/api/staff/orders/"+order.ID+"/pay", body, cookie, uuid.NewString())
		if paid.Code != http.StatusOK {
			t.Fatalf("%s: pay %d %s", name, paid.Code, paid.Body.String())
		}
		var got map[string]any
		if err := json.Unmarshal(paid.Body.Bytes(), &got); err != nil {
			t.Fatal(err)
		}
		if got["status"] != "paid" {
			t.Fatalf("%s: status %v", name, got["status"])
		}
		for _, key := range []string{"cashReceivedMinor", "cashChangeMinor"} {
			if _, ok := got[key]; ok {
				t.Fatalf("%s: order JSON must not include %s: %s", name, key, paid.Body.String())
			}
		}
		assertCount(t, ctx, pool, `SELECT count(*) FROM orders WHERE id = $1 AND cash_received_minor IS NULL AND cash_change_minor IS NULL`, order.ID, 1)
	}
}
