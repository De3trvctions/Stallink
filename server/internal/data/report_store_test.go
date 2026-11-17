package data

import (
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
