package service

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
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
	return strconv.FormatFloat(float64(minor)/100, 'f', 2, 64)
}

// renderDailyReportCSV turns a built report into the export CSV. currency
// filters detail rows to one currency; an empty value keeps the headline
// currency. Amounts are expressed in major units, never raw minor integers.
func renderDailyReportCSV(report data.DailyReport, currency string) []byte {
	if currency == "" {
		currency = report.Currency
	}
	var headline data.DailyReportCurrency
	headline.Orders = report.Orders
	headline.PaidOrders = report.PaidOrders
	headline.RevenueMinor = report.RevenueMinor
	headline.RefundMinor = report.RefundMinor
	headline.Currency = report.Currency
	for _, c := range report.Currencies {
		if c.Currency == currency {
			headline = c
			break
		}
	}

	var rows [][]string
	add := func(group, curr, item string, orders int, amountMinor int64) {
		rows = append(rows, []string{group, curr, item, strconv.Itoa(orders), minorToMajor(amountMinor)})
	}

	add("总览", headline.Currency, "已付款订单", headline.PaidOrders, headline.RevenueMinor)
	// The currency breakdown always lists every currency; the selected-currency
	// filter applies to the event/good/pay-code detail rows below.
	for _, c := range report.Currencies {
		add("按币种", c.Currency, "全部订单", c.Orders, c.RevenueMinor)
	}
	for _, e := range report.Events {
		if currency != "" && e.Currency != currency {
			continue
		}
		add("按活动", e.Currency, e.Name, e.Orders, e.RevenueMinor)
	}
	for _, g := range report.TopGoods {
		if currency != "" && g.Currency != currency {
			continue
		}
		add("按商品", g.Currency, g.Name, g.Quantity, g.RevenueMinor)
	}
	for _, p := range report.PayCodes {
		if currency != "" && p.Currency != currency {
			continue
		}
		add("按收款方式", p.Currency, p.Name, p.Orders, p.RevenueMinor)
	}
	for _, c := range report.Currencies {
		if currency != "" && c.Currency != currency {
			continue
		}
		if c.RefundMinor != 0 {
			add("退款待处理", c.Currency, "待退金额", 0, c.RefundMinor)
		}
	}

	var buf strings.Builder
	writer := csv.NewWriter(&buf)
	_ = writer.Write([]string{"分组", "币种", "项目", "订单数", "金额"})
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

// ConfigureReportLocation applies the configured report zone.
func ConfigureReportLocation(name string) {
	reportLocation = data.ReportLocation(name)
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
