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

// TestStaffGoodsCatalog covers the good-centric management surface: the
// shop-wide good list with shared stock, and the atomic profile+listings save
// that must never half-apply across events.
func TestStaffGoodsCatalog(t *testing.T) {
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
	login := "gcowner" + suffix
	shopID, err := store.OpenShop(ctx, "Goods Catalog Test", "gc-"+suffix, "", login, passwordHash)
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
	eventA, err := store.CreateEvent(ctx, shopID, data.EventInput{Name: "Event A", StartsOn: starts, EndsOn: ends, Currency: "MYR", Status: "live"})
	if err != nil {
		t.Fatal(err)
	}
	eventB, err := store.CreateEvent(ctx, shopID, data.EventInput{Name: "Event B", StartsOn: starts, EndsOn: ends, Currency: "MYR", Status: "scheduled"})
	if err != nil {
		t.Fatal(err)
	}
	goodID, listingA, _, err := store.CreateListedGood(ctx, shopID, account.ID, data.ListedGood{
		Name: "Shared Good", TotalStock: 10, EventID: eventA,
		UnitPriceMinor: 1000, IsListed: true, Approved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	listingB, err := store.AddListing(ctx, shopID, account.ID, goodID, data.ListedGood{
		EventID: eventB, UnitPriceMinor: 1200, IsListed: true,
	}, true)
	if err != nil {
		t.Fatal(err)
	}
	otherGoodID, otherListing, _, err := store.CreateListedGood(ctx, shopID, account.ID, data.ListedGood{
		Name: "Other Good", TotalStock: 5, EventID: eventA,
		UnitPriceMinor: 900, IsListed: true, Approved: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	_ = otherGoodID
	handler := server.NewHTTP(":0", "goods-catalog-test", &service.API{DB: store})
	cookie := &http.Cookie{Name: "stallink_session", Value: plainToken}

	type catalogListing struct {
		ListingID      string `json:"listingId"`
		EventID        string `json:"eventId"`
		SoldQty        int    `json:"soldQty"`
		Remaining      int    `json:"remaining"`
		UnitPriceMinor int64  `json:"unitPriceMinor"`
		IsListed       bool   `json:"isListed"`
	}
	type catalogGood struct {
		ID           string           `json:"id"`
		Name         string           `json:"name"`
		TotalStock   int              `json:"totalStock"`
		AvailableQty int              `json:"availableQty"`
		SoldQty      int              `json:"soldQty"`
		Listings     []catalogListing `json:"listings"`
	}
	readCatalog := func() []catalogGood {
		t.Helper()
		response := commandRequest(t, handler, http.MethodGet, "/api/staff/goods", nil, cookie, "")
		if response.Code != http.StatusOK {
			t.Fatalf("staff goods: status %d, body %s", response.Code, response.Body.String())
		}
		var goods []catalogGood
		if err := json.Unmarshal(response.Body.Bytes(), &goods); err != nil {
			t.Fatal(err)
		}
		return goods
	}
	findGood := func(goods []catalogGood, id data.ID) catalogGood {
		t.Helper()
		for _, good := range goods {
			if good.ID == id.String() {
				return good
			}
		}
		t.Fatalf("good %s missing from catalog", id)
		return catalogGood{}
	}

	goods := readCatalog()
	if len(goods) != 2 {
		t.Fatalf("catalog has %d goods, want 2", len(goods))
	}
	shared := findGood(goods, goodID)
	if shared.TotalStock != 10 || shared.AvailableQty != 10 || shared.SoldQty != 0 || len(shared.Listings) != 2 {
		t.Fatalf("shared good aggregates wrong: %+v", shared)
	}
	byID := map[string]catalogListing{}
	for _, listing := range shared.Listings {
		byID[listing.ListingID] = listing
	}
	if byID[listingA.String()].Remaining != 10 || byID[listingB.String()].UnitPriceMinor != 1200 {
		t.Fatalf("listing rows wrong: %+v", byID)
	}

	// Atomic save: profile + both event listings in one request.
	save := map[string]any{
		"name": "Renamed Good", "description": "", "totalStock": 12,
		"listings": []map[string]any{
			{"listingId": listingA.String(), "unitPriceMinor": 1100, "priceTiers": []any{}, "isListed": true},
			{"listingId": listingB.String(), "unitPriceMinor": 1300, "priceTiers": []any{}, "isListed": false},
		},
	}
	if response := commandRequest(t, handler, http.MethodPatch, "/api/staff/goods/"+goodID.String(), save, cookie, uuid.NewString()); response.Code != http.StatusOK {
		t.Fatalf("atomic save: status %d, body %s", response.Code, response.Body.String())
	}
	shared = findGood(readCatalog(), goodID)
	if shared.Name != "Renamed Good" || shared.TotalStock != 12 || shared.AvailableQty != 12 {
		t.Fatalf("atomic save not reflected: %+v", shared)
	}
	byID = map[string]catalogListing{}
	for _, listing := range shared.Listings {
		byID[listing.ListingID] = listing
	}
	if byID[listingA.String()].UnitPriceMinor != 1100 {
		t.Fatalf("listing A not saved: %+v", byID[listingA.String()])
	}
	if byID[listingB.String()].UnitPriceMinor != 1300 || byID[listingB.String()].IsListed {
		t.Fatalf("listing B not saved: %+v", byID[listingB.String()])
	}
	assertCount(t, ctx, pool, `SELECT count(*) FROM audit_logs WHERE subject_id = $1 AND action = 'stock.update'`, goodID, 1)
	assertCount(t, ctx, pool, `SELECT count(*) FROM audit_logs WHERE subject_id = $1 AND action = 'listing.update'`, listingA, 1)
	assertCount(t, ctx, pool, `SELECT count(*) FROM audit_logs WHERE subject_id = $1 AND action = 'listing.update'`, listingB, 1)

	// A listing owned by another good must be rejected without partial writes.
	crossSave := map[string]any{
		"name": "Cross Good", "description": "", "totalStock": 12,
		"listings": []map[string]any{
			{"listingId": otherListing.String(), "unitPriceMinor": 900, "priceTiers": []any{}, "isListed": true},
		},
	}
	if response := commandRequest(t, handler, http.MethodPatch, "/api/staff/goods/"+goodID.String(), crossSave, cookie, uuid.NewString()); response.Code != http.StatusConflict {
		t.Fatalf("cross-good listing save: status %d, body %s", response.Code, response.Body.String())
	}
	if findGood(readCatalog(), goodID).Name != "Renamed Good" {
		t.Fatal("cross-good save leaked profile changes")
	}

	// Duplicate listing ids are rejected.
	dupSave := map[string]any{
		"name": "Renamed Good", "description": "", "totalStock": 12,
		"listings": []map[string]any{
			{"listingId": listingA.String(), "unitPriceMinor": 1100, "priceTiers": []any{}, "isListed": true},
			{"listingId": listingA.String(), "unitPriceMinor": 1100, "priceTiers": []any{}, "isListed": true},
		},
	}
	if response := commandRequest(t, handler, http.MethodPatch, "/api/staff/goods/"+goodID.String(), dupSave, cookie, uuid.NewString()); response.Code != http.StatusConflict {
		t.Fatalf("duplicate listing save: status %d, body %s", response.Code, response.Body.String())
	}

	// The legacy profile-only path still works unchanged.
	legacy := map[string]any{"name": "Renamed Good", "description": "", "totalStock": 20}
	if response := commandRequest(t, handler, http.MethodPatch, "/api/staff/goods/"+goodID.String(), legacy, cookie, uuid.NewString()); response.Code != http.StatusOK {
		t.Fatalf("legacy profile save: status %d, body %s", response.Code, response.Body.String())
	}
	if findGood(readCatalog(), goodID).TotalStock != 20 {
		t.Fatal("legacy profile save not reflected")
	}
}