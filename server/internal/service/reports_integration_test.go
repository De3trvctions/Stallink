//go:build integration

package service_test

import (
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDailyReportTodayRolesAndPayload(t *testing.T) {
	f := newWorkspaceFixture(t, "report-today-test")
	const endpoint = "/api/admin/daily-reports/today"

	if res := requestJSON(t, f.handler, http.MethodGet, endpoint, nil, nil); res.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous: want 401, got %d", res.Code)
	}
	if res := requestJSON(t, f.handler, http.MethodGet, endpoint, nil, f.salesCookie(t)); res.Code != http.StatusForbidden {
		t.Fatalf("sales: want 403, got %d", res.Code)
	}
	if res := requestJSON(t, f.handler, http.MethodGet, endpoint, nil, f.platform); res.Code != http.StatusForbidden {
		t.Fatalf("platform admin: want 403, got %d", res.Code)
	}
	res := requestJSON(t, f.handler, http.MethodGet, endpoint, nil, f.owner)
	if res.Code != http.StatusOK {
		t.Fatalf("owner: %d %s", res.Code, res.Body.String())
	}
	body := decodeBody[map[string]string](t, res.Body.Bytes())
	if len(body) != 2 || body["timezone"] != "Asia/Kuala_Lumpur" {
		t.Fatalf("want exactly {today, timezone} with default zone, got %v", body)
	}
	if _, err := time.Parse("2006-01-02", body["today"]); err != nil {
		t.Fatalf("today must be YYYY-MM-DD, got %q", body["today"])
	}
	if _, ok := body["date"]; ok {
		t.Fatal("field is named today, not date")
	}
	// Later dates are still rejected by the other report endpoints.
	tomorrow, _ := time.Parse("2006-01-02", body["today"])
	next := tomorrow.AddDate(0, 0, 1).Format("2006-01-02")
	if res := requestJSON(t, f.handler, http.MethodGet, "/api/admin/daily-reports?date="+next, nil, f.owner); res.Code != http.StatusBadRequest || !strings.Contains(res.Body.String(), "invalid_date") {
		t.Fatalf("date after today: want 400 invalid_date, got %d %s", res.Code, res.Body.String())
	}
}

func TestExportDailyReportFilenameUsesPrimaryCurrency(t *testing.T) {
	f := newWorkspaceFixture(t, "report-export-test")
	today := decodeBody[map[string]string](t, requestJSON(t, f.handler, http.MethodGet, "/api/admin/daily-reports/today", nil, f.owner).Body.Bytes())["today"]
	if res := requestJSON(t, f.handler, http.MethodPost, "/api/staff/events", map[string]string{
		"name": "新加坡场", "startsOn": today, "endsOn": today, "currency": "SGD", "status": "live",
	}, f.owner); res.Code != http.StatusCreated {
		t.Fatalf("create event: %d %s", res.Code, res.Body.String())
	}
	if res := requestJSON(t, f.handler, http.MethodPost, "/api/admin/daily-reports", map[string]string{"date": today}, f.owner); res.Code != http.StatusOK {
		t.Fatalf("generate report: %d %s", res.Code, res.Body.String())
	}
	res := requestJSON(t, f.handler, http.MethodGet, "/api/admin/daily-reports/export?date="+today, nil, f.owner)
	if res.Code != http.StatusOK {
		t.Fatalf("export: %d %s", res.Code, res.Body.String())
	}
	want := `attachment; filename="daily-report-` + today + `-SGD.csv"`
	if got := res.Header().Get("Content-Disposition"); got != want {
		t.Fatalf("Content-Disposition = %q, want %q", got, want)
	}
	lines := strings.Split(strings.TrimPrefix(res.Body.String(), "\ufeff"), "\n")
	if strings.TrimSpace(lines[0]) != "分组,币种,项目,数量,金额" {
		t.Fatalf("CSV header = %q", lines[0])
	}
}
