package service

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/De3trvctions/Stallink/internal/data"
)

func TestValidLoginName(t *testing.T) {
	valid := []string{"abc", "sales_01", "一二三四", "abcdefghijklmnopqrstuvwxyz123456"}
	for _, name := range valid {
		if !validLoginName(name) {
			t.Errorf("validLoginName(%q) = false, want true", name)
		}
	}
	invalid := []string{"", "ab", strings.Repeat("x", 33), "sales name", "sales\tname", "sales\nname"}
	for _, name := range invalid {
		if validLoginName(name) {
			t.Errorf("validLoginName(%q) = true, want false", name)
		}
	}
}

func TestValidProfileName(t *testing.T) {
	if !validProfileName("", "") {
		t.Fatal("empty display/contact should be allowed")
	}
	if !validProfileName(strings.Repeat("一", 80), strings.Repeat("x", 120)) {
		t.Fatal("boundary lengths should be allowed")
	}
	if validProfileName(strings.Repeat("一", 81), "x") {
		t.Fatal("display name over 80 runes should be rejected")
	}
	if validProfileName("x", strings.Repeat("x", 121)) {
		t.Fatal("contact over 120 runes should be rejected")
	}
}

func TestValidPayCodeName(t *testing.T) {
	if !validPayCodeName(strings.Repeat("x", 40)) {
		t.Fatal("40-rune pay code name should be allowed")
	}
	if validPayCodeName(strings.Repeat("x", 41)) {
		t.Fatal("41-rune pay code name should be rejected")
	}
}

func TestValidEventLengths(t *testing.T) {
	base := data.EventInput{
		Name: "展会", StartsOn: "2026-11-16", EndsOn: "2026-11-17",
		Currency: "MYR", Status: "live",
	}
	if !validEvent(base) {
		t.Fatal("baseline event should be valid")
	}

	tooLong := map[string]func(data.EventInput) data.EventInput{
		"name":  func(e data.EventInput) data.EventInput { e.Name = strings.Repeat("x", 121); return e },
		"venue": func(e data.EventInput) data.EventInput { e.Venue = strings.Repeat("x", 121); return e },
		"booth": func(e data.EventInput) data.EventInput { e.BoothNumber = strings.Repeat("x", 41); return e },
	}
	for field, mutate := range tooLong {
		if validEvent(mutate(base)) {
			t.Fatalf("event with overlong %s should be rejected", field)
		}
	}

	boundary := base
	boundary.Venue = strings.Repeat("x", 120)
	boundary.BoothNumber = strings.Repeat("x", 40)
	boundary.Name = strings.Repeat("x", 120)
	if !validEvent(boundary) {
		t.Fatal("event at CHECK length boundaries should be accepted")
	}
}

func TestValidNewEventAllowsOnlyBlankCurrency(t *testing.T) {
	base := data.EventInput{Name: "展会", StartsOn: "2026-11-16", EndsOn: "2026-11-17", Status: "live"}
	if !validNewEvent(base) {
		t.Fatal("a new event may leave currency blank for the shop default")
	}
	if validEvent(base) {
		t.Fatal("editing must still require a currency")
	}
	for _, bad := range []string{"USD", "myr", " MYR"} {
		e := base
		e.Currency = bad
		if validNewEvent(e) {
			t.Fatalf("currency %q must be rejected", bad)
		}
	}
	e := base
	e.Name = ""
	if validNewEvent(e) {
		t.Fatal("other checks still apply to new events")
	}
}

func TestWriteDataErrEventCurrencyCodes(t *testing.T) {
	for _, c := range []struct {
		err  error
		code string
	}{
		{data.ErrEventCurrencyRequired, "event_currency_required"},
		{fmt.Errorf("create event: %w", data.ErrEventCurrencyRequired), "event_currency_required"},
		{data.ErrInvalidEvent, "invalid_event"},
	} {
		rec := httptest.NewRecorder()
		writeDataErr(rec, httptest.NewRequest(http.MethodPost, "/api/staff/events", nil), c.err)
		if rec.Code != http.StatusBadRequest || !strings.Contains(rec.Body.String(), `"`+c.code+`"`) {
			t.Fatalf("%v: want 400 %s, got %d %s", c.err, c.code, rec.Code, rec.Body.String())
		}
	}
}
