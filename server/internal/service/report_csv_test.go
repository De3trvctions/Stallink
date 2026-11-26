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
		Currency:           "MYR",
		Orders:             5,
		PaidOrders:         2,
		PendingOrders:      1,
		ClosedOrders:       1,
		RefundOrders:       1,
		ItemsSold:          3,
		RevenueMinor:       1500,
		DiscountMinor:      120,
		RefundMinor:        200,
		CashOverShortMinor: 50,
		CashTenderedOrders: 1,
		AverageMinor:       750,
		Currencies: []data.DailyReportCurrency{
			{Currency: "MYR", Orders: 5, PaidOrders: 2, RevenueMinor: 1500, DiscountMinor: 120, RefundMinor: 200, CashOverShortMinor: 50, CashTenderedOrders: 1},
			{Currency: "SGD", Orders: 1, PaidOrders: 1, RevenueMinor: 800, DiscountMinor: 30, RefundMinor: 0},
		},
		Events: []data.DailyReportEvent{
			{EventID: "e1", Name: "Bazaar", Currency: "MYR", Orders: 4, RevenueMinor: 1500, DiscountMinor: 120},
			{EventID: "e2", Name: "Lion City", Currency: "SGD", Orders: 1, RevenueMinor: 800, DiscountMinor: 0},
		},
		TopGoods: []data.DailyReportGood{
			{Currency: "MYR", Name: "Tea, latte", Quantity: 3, RevenueMinor: 1500},
			{Currency: "SGD", Name: "Jam", Quantity: 1, RevenueMinor: 800},
		},
		PayCodes: []data.DailyReportPayCode{
			{Currency: "MYR", Name: "QR", Orders: 2, RevenueMinor: 1500},
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
	wantHeader := []string{"分组", "币种", "项目", "数量", "金额"}
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
	for _, want := range []string{"总览", "按币种", "按活动", "按商品", "按收款方式"} {
		if !groups[want] {
			t.Fatalf("missing report group %q in %v", want, groups)
		}
	}
	if groups["退款待处理"] {
		t.Fatal("pending refunds are now an overview row, not a separate group")
	}
	for _, row := range rows {
		if len(row) != 5 {
			t.Fatalf("every row must have 5 columns, got %v", row)
		}
		if strings.Contains(row[2], "长短款") || strings.Contains(row[2], "实收") {
			t.Fatalf("cash over/short and cash tendered orders must not be exported (C-2 = A, K-Q18 = A): %v", row)
		}
	}
}

func rowsInGroup(rows [][]string, group string) [][]string {
	var out [][]string
	for _, row := range rows[1:] {
		if row[0] == group {
			out = append(out, row)
		}
	}
	return out
}

func TestRenderDailyReportCSVOverviewRows(t *testing.T) {
	rows := rowsInGroup(parseCSV(t, renderDailyReportCSV(sampleReport(), "MYR")), "总览")
	want := [][]string{
		{"总览", "MYR", "收款总额", "2", "15.00"},
		{"总览", "MYR", "订单数", "5", ""},
		{"总览", "MYR", "待付款订单", "1", ""},
		{"总览", "MYR", "已关闭订单", "1", ""},
		{"总览", "MYR", "退款订单", "1", ""},
		{"总览", "MYR", "售出件数", "3", ""},
		{"总览", "MYR", "平均每单", "", "7.50"},
		{"总览", "MYR", "优惠金额", "", "1.20"},
		{"总览", "MYR", "待退金额", "", "2.00"},
	}
	if len(rows) != len(want) {
		t.Fatalf("overview rows = %v, want %v", rows, want)
	}
	for i := range want {
		if strings.Join(rows[i], "|") != strings.Join(want[i], "|") {
			t.Fatalf("overview row %d = %v, want %v", i, rows[i], want[i])
		}
	}
	report := sampleReport()
	report.RefundMinor = 0
	for _, row := range rowsInGroup(parseCSV(t, renderDailyReportCSV(report, "MYR")), "总览") {
		if row[2] == "待退金额" {
			if row[4] != "0.00" {
				t.Fatalf("zero pending refund must still be written as 0.00, got %v", row)
			}
			return
		}
	}
	t.Fatal("pending refund row must be written even when zero")
}

func TestRenderDailyReportCSVCurrencyRows(t *testing.T) {
	rows := rowsInGroup(parseCSV(t, renderDailyReportCSV(sampleReport(), "MYR")), "按币种")
	got := make([]string, 0, len(rows))
	for _, row := range rows {
		got = append(got, strings.Join(row[1:], "|"))
	}
	want := []string{
		"MYR|全部订单|5|15.00", "MYR|已付款订单|2|", "MYR|优惠金额||1.20", "MYR|待退金额||2.00",
		"SGD|全部订单|1|8.00", "SGD|已付款订单|1|", "SGD|优惠金额||0.30", "SGD|待退金额||0.00",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("currency rows = %v, want %v", got, want)
	}
}

func TestRenderDailyReportCSVEventDiscountRow(t *testing.T) {
	rows := rowsInGroup(parseCSV(t, renderDailyReportCSV(sampleReport(), "MYR")), "按活动")
	want := [][]string{
		{"按活动", "MYR", "Bazaar", "4", "15.00"},
		{"按活动", "MYR", "Bazaar 优惠", "", "1.20"},
	}
	if len(rows) != len(want) {
		t.Fatalf("event rows = %v, want %v", rows, want)
	}
	for i := range want {
		if strings.Join(rows[i], "|") != strings.Join(want[i], "|") {
			t.Fatalf("event row %d = %v, want %v", i, rows[i], want[i])
		}
	}
}

func TestRenderDailyReportCSVGoodsQuantity(t *testing.T) {
	rows := rowsInGroup(parseCSV(t, renderDailyReportCSV(sampleReport(), "MYR")), "按商品")
	if len(rows) != 1 || rows[0][2] != "Tea, latte" || rows[0][3] != "3" || rows[0][4] != "15.00" {
		t.Fatalf("goods rows must carry quantity in 数量: %v", rows)
	}
	pay := rowsInGroup(parseCSV(t, renderDailyReportCSV(sampleReport(), "MYR")), "按收款方式")
	if len(pay) != 1 || pay[0][3] != "2" || pay[0][4] != "15.00" {
		t.Fatalf("pay code rows: %v", pay)
	}
}

func TestRenderDailyReportCSVNonPrimaryCurrency(t *testing.T) {
	rows := rowsInGroup(parseCSV(t, renderDailyReportCSV(sampleReport(), "SGD")), "总览")
	items := map[string][]string{}
	for _, row := range rows {
		if row[1] != "SGD" {
			t.Fatalf("non-primary overview must use the selected currency: %v", row)
		}
		items[row[2]] = row
	}
	for _, missing := range []string{"待付款订单", "已关闭订单", "退款订单", "售出件数"} {
		if _, ok := items[missing]; ok {
			t.Fatalf("non-primary overview must omit %q (no per-currency field)", missing)
		}
	}
	if got := items["收款总额"]; got == nil || got[3] != "1" || got[4] != "8.00" {
		t.Fatalf("non-primary 收款总额 = %v", got)
	}
	if got := items["订单数"]; got == nil || got[3] != "1" {
		t.Fatalf("non-primary 订单数 = %v", got)
	}
	if got := items["优惠金额"]; got == nil || got[4] != "0.30" {
		t.Fatalf("non-primary 优惠金额 = %v", got)
	}
	if got := items["待退金额"]; got == nil || got[4] != "0.00" {
		t.Fatalf("non-primary 待退金额 = %v", got)
	}
}

func TestRenderDailyReportCSVNeutralizesFormulas(t *testing.T) {
	report := sampleReport()
	report.TopGoods[0].Name = `=HYPERLINK("https://evil.example","明细")`
	report.Events[0].Name = "+1 小卡"
	rows := parseCSV(t, renderDailyReportCSV(report, "MYR"))
	var sawFormula, sawPlus bool
	for _, row := range rows[1:] {
		if row[0] == "按商品" && strings.HasPrefix(row[2], "'=") {
			sawFormula = true
		}
		if row[0] == "按活动" && row[2] == "'+1 小卡" {
			sawPlus = true
		}
		if row[4] != "" && (row[4][0] == '=' || row[4][0] == '\'') {
			t.Fatalf("amount column should stay numeric, got %q", row[4])
		}
	}
	if !sawFormula || !sawPlus {
		t.Fatalf("formula names were not neutralized: %v", rows)
	}
}

func TestRenderDailyReportCSVCurrencyFilter(t *testing.T) {
	detailGroups := map[string]bool{"按活动": true, "按商品": true, "按收款方式": true}
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
	// An empty currency (全部币种) still exports only the headline currency.
	all := parseCSV(t, renderDailyReportCSV(sampleReport(), ""))
	for _, row := range all[1:] {
		if (detailGroups[row[0]] || row[0] == "总览") && row[1] != "MYR" {
			t.Fatalf("empty currency must export only the primary currency, got %v", row)
		}
	}
}

func TestRenderDailyReportCSVAmountMajorUnit(t *testing.T) {
	amountRe := regexp.MustCompile(`^-?\d+\.\d{2}$`)
	rows := parseCSV(t, renderDailyReportCSV(sampleReport(), "MYR"))
	for _, row := range rows[1:] {
		if row[4] != "" && !amountRe.MatchString(row[4]) {
			t.Fatalf("amount %q is not a two-decimal major-unit value", row[4])
		}
		if row[3] != "" && !regexp.MustCompile(`^\d+$`).MatchString(row[3]) {
			t.Fatalf("quantity %q is not a plain integer", row[3])
		}
		if row[3] == "" && row[4] == "" {
			t.Fatalf("row has neither quantity nor amount: %v", row)
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
