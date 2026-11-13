package service

import (
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
