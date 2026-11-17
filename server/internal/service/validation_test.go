package service

import (
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
