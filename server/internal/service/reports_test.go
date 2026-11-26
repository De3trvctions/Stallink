package service

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReportDate(t *testing.T) {
	today := time.Now().In(reportLocation).Format("2006-01-02")
	if date, ok := reportDate(""); !ok || date != today {
		t.Fatalf("empty date = %q, %v; want %q, true", date, ok, today)
	}
	if date, ok := reportDate(today); !ok || date != today {
		t.Fatalf("today = %q, %v; want %q, true", date, ok, today)
	}
	if _, ok := reportDate("2024-02-30"); ok {
		t.Fatal("invalid calendar date accepted")
	}
	if _, ok := reportDate("2999-01-01"); ok {
		t.Fatal("future date accepted")
	}
}

func TestReportTodayUsesReportLocation(t *testing.T) {
	t.Cleanup(func() { ConfigureReportLocation("") })

	ConfigureReportLocation("Asia/Tokyo")
	tokyo, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	today, timezone := reportToday()
	if want := time.Now().In(tokyo).Format("2006-01-02"); today != want {
		t.Fatalf("today = %q, want Tokyo date %q", today, want)
	}
	if timezone != "Asia/Tokyo" {
		t.Fatalf("timezone = %q, want Asia/Tokyo", timezone)
	}

	for _, configured := range []string{"", "Not/AZone"} {
		ConfigureReportLocation(configured)
		if _, timezone := reportToday(); timezone != "Asia/Kuala_Lumpur" {
			t.Fatalf("configured %q: timezone = %q, want Asia/Kuala_Lumpur", configured, timezone)
		}
	}
}

func TestDailyReportTodayRequiresSession(t *testing.T) {
	recorder := httptest.NewRecorder()
	(&API{}).DailyReportToday(recorder, httptest.NewRequest(http.MethodGet, "/api/admin/daily-reports/today", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("anonymous today: status %d, want 401", recorder.Code)
	}
}
