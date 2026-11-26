//go:build integration

package service_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/De3trvctions/Stallink/internal/server"
	"github.com/De3trvctions/Stallink/internal/service"
)

// K-Q1 = A, K-Q2 = A: the shop booth number and default currency fill blank
// fields when an event is created; customers see the event's own booth number
// first and fall back to the shop's.
func TestEventDefaultsAndDisplayBoothNumber(t *testing.T) {
	f := newWorkspaceFixture(t, "event-defaults-test")
	newEvent := func(name, booth, currency string) map[string]any {
		return map[string]any{
			"name": name, "startsOn": "2026-12-01", "endsOn": "2026-12-02", "city": "KL", "venue": "Hall",
			"boothNumber": booth, "currency": currency, "status": "scheduled",
		}
	}
	createEvent := func(body map[string]any) (int, string) {
		t.Helper()
		res := requestJSON(t, f.handler, http.MethodPost, "/api/staff/events", body, f.owner)
		return res.Code, res.Body.String()
	}
	type eventRow struct {
		ID                 string  `json:"id"`
		Name               string  `json:"name"`
		BoothNumber        string  `json:"boothNumber"`
		Currency           string  `json:"currency"`
		DisplayBoothNumber *string `json:"displayBoothNumber"`
	}
	staffEvents := func() map[string]eventRow {
		t.Helper()
		res := requestJSON(t, f.handler, http.MethodGet, "/api/staff/events", nil, f.owner)
		if res.Code != http.StatusOK {
			t.Fatalf("staff events: %d %s", res.Code, res.Body.String())
		}
		out := map[string]eventRow{}
		for _, e := range decodeBody[[]eventRow](t, res.Body.Bytes()) {
			out[e.Name] = e
		}
		return out
	}

	// No shop default currency yet: a blank currency is 400 event_currency_required
	// on all three create paths (event page, goods page, bundle page).
	bundles := server.NewHTTP(":0", "event-defaults-bundles", &service.API{DB: f.store, BundlesEnabled: true})
	draft := func(name string) map[string]any {
		return map[string]any{"name": name, "totalStock": 5, "listedQty": 5, "unitPriceMinor": 100, "isListed": true}
	}
	bundleBody := func(event map[string]any) map[string]any {
		return map[string]any{
			"name": "套餐", "unitPriceMinor": 150, "isListed": true, "event": event,
			"components": []map[string]any{{"draftKey": "a", "quantity": 1, "draft": draft("草稿A")}, {"draftKey": "b", "quantity": 1, "draft": draft("草稿B")}},
		}
	}
	for name, rec := range map[string]*httptest.ResponseRecorder{
		"event page": requestJSON(t, f.handler, http.MethodPost, "/api/staff/events", newEvent("无默认", "", ""), f.owner),
		"goods page": requestJSON(t, f.handler, http.MethodPost, "/api/staff/goods", map[string]any{
			"name": "海报", "totalStock": 3, "unitPriceMinor": 1000, "isListed": true, "event": newEvent("无默认2", "", ""),
		}, f.owner),
		"bundle page": requestJSON(t, bundles, http.MethodPost, "/api/staff/bundles", bundleBody(newEvent("无默认3", "", "")), f.owner),
	} {
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"event_currency_required"`) {
			t.Fatalf("%s blank currency without shop default: want 400 event_currency_required, got %d %s", name, rec.Code, rec.Body.String())
		}
	}
	// Bundle page with a bad name or dates is still event_required (unchanged).
	badBundleEvent := newEvent("", "", "")
	if res := requestJSON(t, bundles, http.MethodPost, "/api/staff/bundles", bundleBody(badBundleEvent), f.owner); res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "event_required") {
		t.Fatalf("bundle page invalid event: want 400 event_required, got %d %s", res.Code, res.Body.String())
	}
	if code, body := createEvent(newEvent("坏货币", "", "USD")); code != http.StatusBadRequest || !strings.Contains(body, "invalid_event") {
		t.Fatalf("bad currency: want 400 invalid_event, got %d %s", code, body)
	}

	if res := requestJSON(t, f.handler, http.MethodPatch, "/api/admin/shop", map[string]any{
		"name": "晚风", "isEnabled": true, "boothNumber": "A-27", "defaultCurrency": "SGD",
	}, f.owner); res.Code != http.StatusOK {
		t.Fatalf("set shop defaults: %d %s", res.Code, res.Body.String())
	}

	if code, body := createEvent(newEvent("用默认", "", "")); code != http.StatusCreated {
		t.Fatalf("create with defaults: %d %s", code, body)
	}
	if code, body := createEvent(newEvent("自填", "B-01", "MYR")); code != http.StatusCreated {
		t.Fatalf("create with own values: %d %s", code, body)
	}
	// Goods page "new event" path takes the same defaults.
	if res := requestJSON(t, f.handler, http.MethodPost, "/api/staff/goods", map[string]any{
		"name": "海报", "totalStock": 3, "unitPriceMinor": 1000, "isListed": true, "event": newEvent("商品页新建", "", ""),
	}, f.owner); res.Code != http.StatusCreated {
		t.Fatalf("goods with new event: %d %s", res.Code, res.Body.String())
	}
	events := staffEvents()
	if e := events["用默认"]; e.BoothNumber != "A-27" || e.Currency != "SGD" {
		t.Fatalf("blank fields must take shop defaults: %+v", e)
	}
	if e := events["自填"]; e.BoothNumber != "B-01" || e.Currency != "MYR" {
		t.Fatalf("explicit values must win: %+v", e)
	}
	if e := events["商品页新建"]; e.BoothNumber != "A-27" || e.Currency != "SGD" {
		t.Fatalf("goods-page new event must take shop defaults: %+v", e)
	}
	for name, e := range events {
		if e.DisplayBoothNumber != nil {
			t.Fatalf("workspace event %q must not carry displayBoothNumber", name)
		}
	}

	// Editing never re-applies defaults: clearing the booth number saves "".
	edit := newEvent("用默认", "", "SGD")
	if res := requestJSON(t, f.handler, http.MethodPatch, "/api/staff/events/"+events["用默认"].ID, edit, f.owner); res.Code != http.StatusOK {
		t.Fatalf("edit event: %d %s", res.Code, res.Body.String())
	}
	if res := requestJSON(t, f.handler, http.MethodPatch, "/api/staff/events/"+events["用默认"].ID, newEvent("用默认", "", ""), f.owner); res.Code != http.StatusBadRequest {
		t.Fatalf("editing with a blank currency must stay 400: %d %s", res.Code, res.Body.String())
	}
	if e := staffEvents()["用默认"]; e.BoothNumber != "" {
		t.Fatalf("edit must save the blank booth number: %+v", e)
	}

	publicShop := func() map[string]eventRow {
		t.Helper()
		res := requestJSON(t, f.handler, http.MethodGet, "/api/public/shops/"+f.shopSlug, nil, nil)
		if res.Code != http.StatusOK {
			t.Fatalf("public shop: %d %s", res.Code, res.Body.String())
		}
		body := decodeBody[struct {
			Events map[string][]eventRow `json:"events"`
		}](t, res.Body.Bytes())
		out := map[string]eventRow{}
		for _, group := range body.Events {
			for _, e := range group {
				out[e.Name] = e
			}
		}
		return out
	}
	display := func(e eventRow) string {
		t.Helper()
		if e.DisplayBoothNumber == nil {
			t.Fatalf("public event %q must carry displayBoothNumber", e.Name)
		}
		return *e.DisplayBoothNumber
	}
	pub := publicShop()
	if e := pub["自填"]; display(e) != "B-01" || e.BoothNumber != "B-01" {
		t.Fatalf("own booth number wins: %+v", e)
	}
	if e := pub["用默认"]; display(e) != "A-27" || e.BoothNumber != "" {
		t.Fatalf("blank event booth falls back to the shop's: %+v", e)
	}
	detail := requestJSON(t, f.handler, http.MethodGet, "/api/public/shops/"+f.shopSlug+"/events/"+pub["用默认"].ID, nil, nil)
	if detail.Code != http.StatusOK {
		t.Fatalf("event detail: %d %s", detail.Code, detail.Body.String())
	}
	if e := decodeBody[struct {
		Event eventRow `json:"event"`
	}](t, detail.Body.Bytes()).Event; display(e) != "A-27" {
		t.Fatalf("event detail displayBoothNumber: %+v", e)
	}

	// Both blank: "".
	if res := requestJSON(t, f.handler, http.MethodPatch, "/api/admin/shop", map[string]any{
		"name": "晚风", "isEnabled": true, "boothNumber": "", "defaultCurrency": "SGD",
	}, f.owner); res.Code != http.StatusOK {
		t.Fatalf("clear shop booth: %d %s", res.Code, res.Body.String())
	}
	if e := publicShop()["用默认"]; display(e) != "" {
		t.Fatalf("both blank must be empty: %+v", e)
	}
	landing := requestJSON(t, f.handler, http.MethodGet, "/api/public/landing", nil, nil)
	if landing.Code == http.StatusOK && !strings.Contains(landing.Body.String(), "displayBoothNumber") {
		t.Fatalf("landing shop events must carry displayBoothNumber: %s", landing.Body.String())
	}
}
