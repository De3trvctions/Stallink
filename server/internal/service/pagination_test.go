package service

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPageParams(t *testing.T) {
	tests := []struct {
		query       string
		page, limit int
		search      string
		ok          bool
	}{
		{"", 1, 20, "", true}, {"?page=3&limit=50&q=%20ABC%20", 3, 50, "ABC", true},
		{"?page=0", 0, 0, "", false}, {"?limit=51", 0, 0, "", false}, {"?page=100001", 0, 0, "", false}, {"?q=" + strings.Repeat("a", 101), 0, 0, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			p, l, q, ok := pageParams(httptest.NewRequest("GET", "/"+tt.query, nil))
			if p != tt.page || l != tt.limit || q != tt.search || ok != tt.ok {
				t.Fatalf("got %d %d %q %v", p, l, q, ok)
			}
		})
	}
}
