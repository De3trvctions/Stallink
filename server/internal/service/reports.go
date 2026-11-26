package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/De3trvctions/Stallink/internal/catalog"
	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/logging"
	"go.uber.org/zap"
)

// csvBOM lets Excel open the exported report without garbling UTF-8 text.
var csvBOM = []byte{0xEF, 0xBB, 0xBF}

// minorToMajor renders minor currency units as a two-decimal major-unit amount.
func minorToMajor(minor int64) string {
	sign := ""
	if minor < 0 {
		sign = "-"
		minor = -minor
	}
	return fmt.Sprintf("%s%d.%02d", sign, minor/100, minor%100)
}

// spreadsheetText keeps a catalog name from being executed as a formula when
// the daily report is opened in a spreadsheet.
func spreadsheetText(value string) string {
	trimmed := strings.TrimLeft(value, " \t\r\n")
	if trimmed == "" {
		return value
	}
	switch trimmed[0] {
	case '=', '+', '-', '@':
		return "'" + value
	default:
		return value
	}
}

// dailyReportCSVHeader is the fixed five-column layout (design note §8.3, R2).
// 数量 is an order count or an item count depending on 项目.
var dailyReportCSVHeader = []string{"分组", "币种", "项目", "数量", "金额"}

// renderDailyReportCSV turns a built report into the export CSV. currency
// filters detail rows to one currency; an empty value keeps the headline
// currency. Amounts are expressed in major units, never raw minor integers.
// Cells that do not apply to a row stay empty instead of 0. Cash over/short is
// deliberately not exported (C-2 = A); the snapshot still keeps it for the PDF.
func renderDailyReportCSV(report data.DailyReport, currency string) []byte {
	if currency == "" {
		currency = report.Currency
	}
	var rows [][]string
	add := func(group, curr, item string, quantity, amountMinor *int64) {
		qty, amount := "", ""
		if quantity != nil {
			qty = strconv.FormatInt(*quantity, 10)
		}
		if amountMinor != nil {
			amount = minorToMajor(*amountMinor)
		}
		rows = append(rows, []string{group, curr, spreadsheetText(item), qty, amount})
	}
	count := func(n int) *int64 {
		v := int64(n)
		return &v
	}
	money := func(minor int64) *int64 { return &minor }

	if currency == report.Currency {
		add("总览", currency, "收款总额", count(report.PaidOrders), money(report.RevenueMinor))
		add("总览", currency, "订单数", count(report.Orders), nil)
		add("总览", currency, "待付款订单", count(report.PendingOrders), nil)
		add("总览", currency, "已关闭订单", count(report.ClosedOrders), nil)
		add("总览", currency, "退款订单", count(report.RefundOrders), nil)
		add("总览", currency, "售出件数", count(report.ItemsSold), nil)
		add("总览", currency, "平均每单", nil, money(report.AverageMinor))
		add("总览", currency, "优惠金额", nil, money(report.DiscountMinor))
		add("总览", currency, "待退金额", nil, money(report.RefundMinor))
	} else {
		// Non-primary currencies only carry the per-currency fields; rows
		// without a field are omitted rather than guessed (design note issue 11).
		selected := data.DailyReportCurrency{Currency: currency}
		for _, c := range report.Currencies {
			if c.Currency == currency {
				selected = c
				break
			}
		}
		add("总览", currency, "收款总额", count(selected.PaidOrders), money(selected.RevenueMinor))
		add("总览", currency, "订单数", count(selected.Orders), nil)
		add("总览", currency, "优惠金额", nil, money(selected.DiscountMinor))
		add("总览", currency, "待退金额", nil, money(selected.RefundMinor))
	}
	// The currency breakdown always lists every currency; the selected-currency
	// filter applies to the event/good/pay-code detail rows below.
	for _, c := range report.Currencies {
		add("按币种", c.Currency, "全部订单", count(c.Orders), money(c.RevenueMinor))
		add("按币种", c.Currency, "已付款订单", count(c.PaidOrders), nil)
		add("按币种", c.Currency, "优惠金额", nil, money(c.DiscountMinor))
		add("按币种", c.Currency, "待退金额", nil, money(c.RefundMinor))
	}
	for _, e := range report.Events {
		if e.Currency != currency {
			continue
		}
		add("按活动", e.Currency, e.Name, count(e.Orders), money(e.RevenueMinor))
		add("按活动", e.Currency, e.Name+" 优惠", nil, money(e.DiscountMinor))
	}
	for _, g := range report.TopGoods {
		if g.Currency != currency {
			continue
		}
		add("按商品", g.Currency, g.Name, count(g.Quantity), money(g.RevenueMinor))
	}
	for _, p := range report.PayCodes {
		if p.Currency != currency {
			continue
		}
		add("按收款方式", p.Currency, p.Name, count(p.Orders), money(p.RevenueMinor))
	}

	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	_ = writer.Write(dailyReportCSVHeader)
	for _, row := range rows {
		_ = writer.Write(row)
	}
	writer.Flush()
	out := append(csvBOM, []byte(buf.String())...)
	return out
}

// reportLocation is resolved from reports.timezone at startup; an empty or
// invalid configured zone falls back to the KL fixed offset used historically.
var reportLocation = data.ReportLocation("")

// reportTimezoneName is the IANA name reported to clients next to the report
// "today"; an empty or invalid configured zone reports the default zone.
var reportTimezoneName = data.NormalizeReportTimezone("")

// ConfigureReportLocation applies the configured report zone.
func ConfigureReportLocation(name string) {
	reportLocation = data.ReportLocation(name)
	reportTimezoneName = data.NormalizeReportTimezone(name)
}

// reportToday is the newest date the daily report endpoints accept, computed
// in the report zone, together with that zone's IANA name.
func reportToday() (today, timezone string) {
	return time.Now().In(reportLocation).Format("2006-01-02"), reportTimezoneName
}

// DailyReportToday exposes the server-side report "today" so the workspace
// does not default to the browser's local date (design note R4 / §8.4).
func (a *API) DailyReportToday(w http.ResponseWriter, r *http.Request) {
	if _, ok := a.shopAdmin(w, r); !ok {
		return
	}
	today, timezone := reportToday()
	writeJSON(w, http.StatusOK, map[string]string{"today": today, "timezone": timezone})
}

func reportDate(value string) (string, bool) {
	if value == "" {
		return time.Now().In(reportLocation).Format("2006-01-02"), true
	}
	parsed, err := time.ParseInLocation("2006-01-02", value, reportLocation)
	if err != nil || parsed.After(time.Now().In(reportLocation)) {
		return "", false
	}
	return parsed.Format("2006-01-02"), true
}

func (a *API) GetDailyReport(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	date, valid := reportDate(r.URL.Query().Get("date"))
	if !valid {
		writeErr(w, http.StatusBadRequest, "invalid_date")
		return
	}
	payload, err := a.DB.GetDailyReport(r.Context(), *account.ShopID, date)
	if errors.Is(err, data.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "report_not_generated")
		return
	}
	if err != nil {
		logging.ErrorContext(r.Context(), "read daily report", zap.String("shop_id", account.ShopID.String()), zap.String("date", date), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (a *API) GenerateDailyReport(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	var body struct {
		Date string `json:"date"`
	}
	if err := readJSON(r, &body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_json")
		return
	}
	date, valid := reportDate(body.Date)
	if !valid {
		writeErr(w, http.StatusBadRequest, "invalid_date")
		return
	}
	report, err := a.DB.BuildDailyReport(r.Context(), *account.ShopID, date, "manual", a.ReportTimezone)
	if err != nil {
		logging.ErrorContext(r.Context(), "build daily report", zap.String("shop_id", account.ShopID.String()), zap.String("date", date), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if err := a.DB.UpsertDailyReport(r.Context(), *account.ShopID, report); err != nil {
		logging.ErrorContext(r.Context(), "save daily report", zap.String("shop_id", account.ShopID.String()), zap.String("date", date), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	logging.InfoContext(r.Context(), "daily report generated", zap.String("shop_id", account.ShopID.String()), zap.String("date", date), zap.String("source", "manual"))
	writeJSON(w, http.StatusOK, report)
}

func (a *API) ExportDailyReport(w http.ResponseWriter, r *http.Request) {
	account, ok := a.shopAdmin(w, r)
	if !ok {
		return
	}
	date, valid := reportDate(r.URL.Query().Get("date"))
	if !valid {
		writeErr(w, http.StatusBadRequest, "invalid_date")
		return
	}
	currency := strings.TrimSpace(r.URL.Query().Get("currency"))
	if currency != "" && !catalog.ValidCurrency(currency) {
		writeErr(w, http.StatusBadRequest, "invalid_currency")
		return
	}
	snapshot, err := a.DB.GetDailyReport(r.Context(), *account.ShopID, date)
	if errors.Is(err, data.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "report_not_generated")
		return
	}
	if err != nil {
		logging.ErrorContext(r.Context(), "export daily report", zap.String("shop_id", account.ShopID.String()), zap.String("date", date), zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	var report data.DailyReport
	if err := json.Unmarshal(snapshot, &report); err != nil {
		logging.ErrorContext(r.Context(), "decode daily report for export", zap.Error(err))
		writeErr(w, http.StatusInternalServerError, "internal")
		return
	}
	if currency == "" {
		currency = report.Currency
	}
	payload := renderDailyReportCSV(report, currency)
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="daily-report-`+date+`-`+currency+`.csv"`)
	w.Header().Set("Content-Length", strconv.Itoa(len(payload)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(payload)
}

func GenerateScheduledReports(ctx context.Context, store *data.ShopStore, date, timezone string) {
	shops, err := store.ListShops(ctx)
	if err != nil {
		logging.ErrorContext(ctx, "list shops for daily reports", zap.Error(err))
		return
	}
	saved, kept, failed := 0, 0, 0
	for _, shop := range shops {
		report, err := store.BuildDailyReport(ctx, shop.ID, date, "scheduled", timezone)
		if err != nil {
			failed++
			logging.ErrorContext(ctx, "build scheduled daily report", zap.String("shop_id", shop.ID.String()), zap.String("date", date), zap.Error(err))
			continue
		}
		wrote, err := store.SaveScheduledDailyReport(ctx, shop.ID, report, timezone)
		if err != nil {
			failed++
			logging.ErrorContext(ctx, "save scheduled daily report", zap.String("shop_id", shop.ID.String()), zap.String("date", date), zap.Error(err))
			continue
		}
		if wrote {
			saved++
		} else {
			kept++
		}
	}
	logging.InfoContext(ctx, "scheduled daily reports finished",
		zap.String("date", date), zap.Int("shops", len(shops)),
		zap.Int("saved", saved), zap.Int("kept", kept), zap.Int("failed", failed))
}
