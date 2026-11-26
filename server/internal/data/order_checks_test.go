package data

import (
	"errors"
	"testing"
)

func TestPayCodeCurrencyMatches(t *testing.T) {
	mYR := "MYR"
	cases := []struct {
		name      string
		code      *string
		order     string
		wantMatch bool
	}{
		{"nil pay code follows order currency", nil, "MYR", true},
		{"same currency matches", &mYR, "MYR", true},
		{"different currency rejects", &mYR, "SGD", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := payCodeCurrencyMatches(tc.code, tc.order); got != tc.wantMatch {
				t.Fatalf("payCodeCurrencyMatches(%v, %q) = %v, want %v", tc.code, tc.order, got, tc.wantMatch)
			}
		})
	}
}

func TestPayStillAllowed(t *testing.T) {
	cases := []struct {
		name           string
		status         string
		allowScheduled bool
		lines          int
		eligible       int
		want           error
	}{
		{name: "live approved listing can be paid", status: "live", allowScheduled: false, lines: 2, eligible: 2},
		{name: "scheduled order can be paid when pre-sale is on", status: "scheduled", allowScheduled: true, lines: 1, eligible: 1},
		{name: "scheduled order is refused when pre-sale is off", status: "scheduled", allowScheduled: false, lines: 1, eligible: 1, want: ErrEventNotOpen},
		{name: "ended event is refused", status: "ended", allowScheduled: true, lines: 1, eligible: 1, want: ErrEventNotOpen},
		{name: "rejected or unlisted line is refused", status: "live", allowScheduled: true, lines: 2, eligible: 1, want: ErrNotListed},
		{name: "empty order is refused", status: "live", allowScheduled: true, lines: 0, eligible: 0, want: ErrNotListed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := payStillAllowed(tc.status, tc.allowScheduled, tc.lines, tc.eligible)
			if !errors.Is(err, tc.want) {
				t.Fatalf("payStillAllowed() = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestPoolRemainingSoldOut(t *testing.T) {
	cases := []struct {
		name          string
		total, reserved int
		wantRemaining int
		wantSoldOut   bool
	}{
		{"in stock", 10, 2, 8, false},
		{"exactly zero", 5, 5, 0, true},
		{"over reserved clamps to zero", 3, 12, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			remaining, soldOut := poolRemaining(tc.total, tc.reserved)
			if remaining != tc.wantRemaining || soldOut != tc.wantSoldOut {
				t.Fatalf("poolRemaining(%d,%d) = %d,%v; want %d,%v",
					tc.total, tc.reserved, remaining, soldOut, tc.wantRemaining, tc.wantSoldOut)
			}
		})
	}
}

func TestValidateEventOpen(t *testing.T) {
	cases := []struct {
		status       string
		allowSchedul bool
		wantOpen     bool
	}{
		{"live", true, true},
		{"live", false, true},
		{"scheduled", true, true},
		{"scheduled", false, false},
		{"ended", true, false},
		{"ended", false, false},
		{"unknown", true, false},
	}
	for _, tc := range cases {
		got := validateEventOpen(tc.status, tc.allowSchedul)
		if (got == nil) != tc.wantOpen {
			t.Fatalf("validateEventOpen(%q, %v) = %v; want open=%v", tc.status, tc.allowSchedul, got, tc.wantOpen)
		}
	}
}
