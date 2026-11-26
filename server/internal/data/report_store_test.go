package data

import (
	"encoding/json"
	"testing"
	"time"
)

func TestApplyHeadlineCurrencyUsesTopCurrencyPaidOrders(t *testing.T) {
	report := DailyReport{
		Orders: 3, PaidOrders: 2,
		Currencies: []DailyReportCurrency{
			{Currency: "MYR", Orders: 1, PaidOrders: 1, RevenueMinor: 10000, DiscountMinor: 500},
			{Currency: "SGD", Orders: 2, PaidOrders: 1, RevenueMinor: 4000, DiscountMinor: 100},
		},
	}
	applyHeadlineCurrency(&report)
	if report.Currency != "MYR" || report.RevenueMinor != 10000 || report.DiscountMinor != 500 {
		t.Fatalf("headline = %+v", report)
	}
	if report.Orders != 1 || report.PaidOrders != 1 {
		t.Fatalf("headline counts = orders %d, paid %d; want 1, 1", report.Orders, report.PaidOrders)
	}
	if report.AverageMinor != 10000 {
		t.Fatalf("average = %d, want 10000 from the top currency alone", report.AverageMinor)
	}
}

func TestApplyHeadlineCurrencyCopiesRefundMinor(t *testing.T) {
	report := DailyReport{
		Orders: 3, PaidOrders: 2,
		Currencies: []DailyReportCurrency{
			{Currency: "MYR", Orders: 1, PaidOrders: 1, RevenueMinor: 10000, RefundMinor: 2500},
			{Currency: "SGD", Orders: 2, PaidOrders: 1, RevenueMinor: 4000, RefundMinor: 300},
		},
	}
	applyHeadlineCurrency(&report)
	if report.RefundMinor != 2500 {
		t.Fatalf("headline refundMinor = %d, want 2500 from the top currency", report.RefundMinor)
	}
}

func TestApplyHeadlineCurrencyWithoutCurrencyRows(t *testing.T) {
	report := DailyReport{PaidOrders: 2, RevenueMinor: 80}
	applyHeadlineCurrency(&report)
	if report.AverageMinor != 40 {
		t.Fatalf("average = %d, want 40", report.AverageMinor)
	}
}

func TestKeepManualDailyReport(t *testing.T) {
	location := time.FixedZone("MYT", 8*60*60)
	dayEnd := time.Date(2026, 11, 13, 0, 0, 0, 0, location)
	preview := time.Date(2026, 11, 12, 23, 0, 0, 0, location)
	closed := time.Date(2026, 11, 13, 0, 5, 0, 0, location)
	if keepManualDailyReport(preview, dayEnd) {
		t.Fatal("preview generated before the business day ended was kept")
	}
	if !keepManualDailyReport(closed, dayEnd) {
		t.Fatal("manual report generated after the business day ended was replaced")
	}
	if !keepManualDailyReport(dayEnd, dayEnd) {
		t.Fatal("manual report generated at day end was replaced")
	}
}

func TestApplyHeadlineCurrencyCopiesCashTenderedOrders(t *testing.T) {
	report := DailyReport{
		Currencies: []DailyReportCurrency{
			{Currency: "MYR", PaidOrders: 1, RevenueMinor: 10000, CashTenderedOrders: 3},
			{Currency: "SGD", PaidOrders: 1, RevenueMinor: 4000, CashTenderedOrders: 1},
		},
	}
	applyHeadlineCurrency(&report)
	if report.CashTenderedOrders != 3 {
		t.Fatalf("headline cashTenderedOrders = %d, want 3 from the top currency", report.CashTenderedOrders)
	}
}

// Snapshots stored before K-Q18 lack cashTenderedOrders; a missing key must be
// filled from the recomputed counts, never read as 0.
func TestFillMissingCashTenderedOrders(t *testing.T) {
	counts := map[string]int{"MYR": 2, "SGD": 1}
	old := []byte(`{"currency":"MYR","cashOverShortMinor":100,"extra":{"keep":true},"currencies":[{"currency":"MYR","cashOverShortMinor":100},{"currency":"SGD"},{"currency":"CNY"}]}`)
	out, changed, err := fillMissingCashTenderedOrders(old, counts)
	if err != nil || !changed {
		t.Fatalf("old snapshot must be filled: changed=%v err=%v", changed, err)
	}
	var got struct {
		Currency           string          `json:"currency"`
		CashTenderedOrders *int            `json:"cashTenderedOrders"`
		CashOverShortMinor int64           `json:"cashOverShortMinor"`
		Extra              map[string]bool `json:"extra"`
		Currencies         []struct {
			Currency           string `json:"currency"`
			CashTenderedOrders *int   `json:"cashTenderedOrders"`
		} `json:"currencies"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatal(err)
	}
	if got.CashTenderedOrders == nil || *got.CashTenderedOrders != 2 || got.CashOverShortMinor != 100 || !got.Extra["keep"] {
		t.Fatalf("root fill or untouched fields wrong: %s", out)
	}
	want := map[string]int{"MYR": 2, "SGD": 1, "CNY": 0}
	for _, c := range got.Currencies {
		if c.CashTenderedOrders == nil || *c.CashTenderedOrders != want[c.Currency] {
			t.Fatalf("currency %s: %s", c.Currency, out)
		}
	}

	stored := []byte(`{"currency":"MYR","cashTenderedOrders":0,"currencies":[{"currency":"MYR","cashTenderedOrders":0}]}`)
	out, changed, err = fillMissingCashTenderedOrders(stored, counts)
	if err != nil || changed || string(out) != string(stored) {
		t.Fatalf("a snapshot that already has the field must be returned as is: changed=%v err=%v %s", changed, err, out)
	}
}

func TestSnapshotNeedsCashTenderedOrders(t *testing.T) {
	for payload, want := range map[string]bool{
		`{"currency":"MYR","currencies":[]}`:                                                                 true,
		`{"currency":"MYR","cashTenderedOrders":0,"currencies":[{"currency":"MYR"}]}`:                        true,
		`{"currency":"MYR","cashTenderedOrders":1,"currencies":[{"currency":"MYR","cashTenderedOrders":1}]}`: false,
	} {
		got, err := snapshotNeedsCashTenderedOrders([]byte(payload))
		if err != nil || got != want {
			t.Fatalf("%s: got %v %v, want %v", payload, got, err, want)
		}
	}
}
