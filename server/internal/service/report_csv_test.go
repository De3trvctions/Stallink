package service

import (
	"encoding/csv"
	"regexp"
	"strings"
	"testing"

	"github.com/De3trvctions/Stallink/internal/data"
)

func sampleReport() data.DailyReport {
	return data.DailyReport{
		Currency:     "MYR",
		Orders:       2,
		PaidOrders:   1,
		RevenueMinor: 1500,
		RefundMinor:  200,
		Currencies: []data.DailyReportCurrency{
			{Currency: "MYR", Orders: 2, PaidOrders: 1, RevenueMinor: 1500, RefundMinor: 200},
			{Currency: "SGD", Orders: 1, PaidOrders: 1, RevenueMinor: 800, RefundMinor: 0},
		},
		Events: []data.DailyReportEvent{
			{EventID: "e1", Name: "Bazaar", Currency: "MYR", Orders: 1, RevenueMinor: 1500},
		},
		TopGoods: []data.DailyReportGood{
			{Currency: "MYR", Name: "Tea, latte", Quantity: 2, RevenueMinor: 1500},
			{Currency: "SGD", Name: "Jam", Quantity: 1, RevenueMinor: 800},
		},
		PayCodes: []data.DailyReportPayCode{
			{Currency: "MYR", Name: "QR", Orders: 1, RevenueMinor: 1500},
		},
	}
}

func parseCSV(t *testing.T, payload []byte) [][]string {
	t.Helper()
	reader := csv.NewReader(strings.NewReader(string(payload[3:]))) // strip BOM
	rows, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("parse exported CSV: %v", err)
	}
	return rows
}

func TestRenderDailyReportCSVHeaderAndGroups(t *testing.T) {
	payload := renderDailyReportCSV(sampleReport(), "MYR")
	if len(payload) < 3 || payload[0] != 0xEF || payload[1] != 0xBB || payload[2] != 0xBF {
		t.Fatal("export CSV is missing the UTF-8 BOM")
	}
	rows := parseCSV(t, payload)
	wantHeader := []string{"分组", "币种", "项目", "订单数", "金额"}
	if len(rows[0]) != 5 {
		t.Fatalf("header columns = %d, want 5", len(rows[0]))
	}
	for i, want := range wantHeader {
		if rows[0][i] != want {
			t.Fatalf("header[%d] = %q, want %q", i, rows[0][i], want)
		}
	}
	groups := map[string]bool{}
	for _, row := range rows[1:] {
		groups[row[0]] = true
	}
	for _, want := range []string{"总览", "按币种", "按活动", "按商品", "按收款方式", "退款待处理"} {
		if !groups[want] {
			t.Fatalf("missing report group %q in %v", want, groups)
		}
	}
}

func TestRenderDailyReportCSVCurrencyFilter(t *testing.T) {
	detailGroups := map[string]bool{"按活动": true, "按商品": true, "按收款方式": true, "退款待处理": true}
	filtered := parseCSV(t, renderDailyReportCSV(sampleReport(), "MYR"))
	for _, row := range filtered[1:] {
		if detailGroups[row[0]] && row[1] == "SGD" {
			t.Fatalf("filtered detail still contains SGD row: %v", row)
		}
	}
	sgd := parseCSV(t, renderDailyReportCSV(sampleReport(), "SGD"))
	foundJam := false
	for _, row := range sgd[1:] {
		if row[0] == "按商品" && row[2] == "Jam" {
			foundJam = true
		}
	}
	if !foundJam {
		t.Fatal("SGD-filtered CSV should include the SGD good")
	}
}

func TestRenderDailyReportCSVAmountMajorUnit(t *testing.T) {
	amountRe := regexp.MustCompile(`^-?\d+\.\d{2}$`)
	rows := parseCSV(t, renderDailyReportCSV(sampleReport(), "MYR"))
	for _, row := range rows[1:] {
		if !amountRe.MatchString(row[4]) {
			t.Fatalf("amount %q is not a two-decimal major-unit value", row[4])
		}
	}
	// Comma inside the good name must be quoted by the CSV writer.
	var escaped string
	for _, row := range rows[1:] {
		if strings.Contains(row[2], "Tea, latte") {
			escaped = row[2]
		}
	}
	if escaped == "" {
		t.Fatal("good name with a comma was not exported")
	}
}
