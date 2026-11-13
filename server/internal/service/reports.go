package service

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/logging"
	"go.uber.org/zap"
)

var reportLocation = time.FixedZone("MYT", 8*60*60)

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
	report, err := a.DB.BuildDailyReport(r.Context(), *account.ShopID, date, "manual")
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

func GenerateScheduledReports(ctx context.Context, store *data.ShopStore, date string) {
	shops, err := store.ListShops(ctx)
	if err != nil {
		logging.ErrorContext(ctx, "list shops for daily reports", zap.Error(err))
		return
	}
	saved, kept, failed := 0, 0, 0
	for _, shop := range shops {
		report, err := store.BuildDailyReport(ctx, shop.ID, date, "scheduled")
		if err != nil {
			failed++
			logging.ErrorContext(ctx, "build scheduled daily report", zap.String("shop_id", shop.ID.String()), zap.String("date", date), zap.Error(err))
			continue
		}
		wrote, err := store.SaveScheduledDailyReport(ctx, shop.ID, report)
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
