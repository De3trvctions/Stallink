package data

import "testing"

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

func TestCashSettlement(t *testing.T) {
	t.Run("non-cash leaves amounts nil", func(t *testing.T) {
		gotReceived, gotChange, err := cashSettlement(1000, nil)
		if err != nil || gotReceived != nil || gotChange != nil {
			t.Fatalf("cashSettlement(1000, nil) = %v, %v, %v; want nil nil nil", gotReceived, gotChange, err)
		}
	})
	received := int64(1500)
	gotReceived, gotChange, err := cashSettlement(1000, &received)
	if err != nil {
		t.Fatalf("cashSettlement returned error: %v", err)
	}
	if gotReceived == nil || *gotReceived != 1500 {
		t.Fatalf("received = %v, want 1500", gotReceived)
	}
	if gotChange == nil || *gotChange != 500 {
		t.Fatalf("change = %v, want 500", gotChange)
	}
	exact := int64(1000)
	gotReceived, gotChange, err = cashSettlement(1000, &exact)
	if err != nil || gotChange == nil || *gotChange != 0 {
		t.Fatalf("exact total: received=%v change=%v err=%v", gotReceived, gotChange, err)
	}
	under := int64(900)
	if _, _, err := cashSettlement(1000, &under); err != ErrInvalidCashReceived {
		t.Fatalf("underpaid error = %v, want ErrInvalidCashReceived", err)
	}
}

func TestRemainingSoldOut(t *testing.T) {
	cases := []struct {
		name                   string
		listed, sold, reserved int
		wantRemaining          int
		wantSoldOut            bool
	}{
		{"in stock", 10, 3, 2, 5, false},
		{"exactly zero", 10, 5, 5, 0, true},
		{"over sold clamps to zero", 10, 12, 3, 0, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			remaining, soldOut := remainingSoldOut(tc.listed, tc.sold, tc.reserved)
			if remaining != tc.wantRemaining || soldOut != tc.wantSoldOut {
				t.Fatalf("remainingSoldOut(%d,%d,%d) = %d,%v; want %d,%v",
					tc.listed, tc.sold, tc.reserved, remaining, soldOut, tc.wantRemaining, tc.wantSoldOut)
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
