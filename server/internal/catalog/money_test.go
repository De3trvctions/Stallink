package catalog

import "testing"

func TestFormatMoney(t *testing.T) {
	cases := []struct {
		currency string
		minor    int64
		want     string
	}{
		{"MYR", 1990, "RM 19.90"},
		{"SGD", 1200, "S$ 12.00"},
		{"CNY", 3900, "¥ 39.00"},
	}
	for _, tc := range cases {
		got, err := FormatMoney(tc.currency, tc.minor)
		if err != nil || got != tc.want {
			t.Fatalf("%s %d => %q %v", tc.currency, tc.minor, got, err)
		}
	}
	if _, err := FormatMoney("USD", 100); err == nil {
		t.Fatal("expected USD to fail")
	}
}
