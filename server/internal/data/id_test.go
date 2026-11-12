package data

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
)

func TestParseID(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want ID
		ok   bool
	}{
		{name: "one", raw: "1", want: 1, ok: true},
		{name: "ordinary value", raw: "42", want: 42, ok: true},
		{name: "maximum bigint", raw: "9223372036854775807", want: ID(math.MaxInt64), ok: true},
		{name: "empty", raw: "", ok: false},
		{name: "zero", raw: "0", ok: false},
		{name: "leading zero", raw: "01", ok: false},
		{name: "negative", raw: "-1", ok: false},
		{name: "plus sign", raw: "+1", ok: false},
		{name: "whitespace", raw: " 1", ok: false},
		{name: "fraction", raw: "1.0", ok: false},
		{name: "uuid", raw: "82d375b7-5ef9-4371-a657-4fa1775955b0", ok: false},
		{name: "overflow", raw: "9223372036854775808", ok: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := ParseID(test.raw)
			if test.ok {
				if err != nil {
					t.Fatalf("ParseID(%q) error = %v", test.raw, err)
				}
				if got != test.want {
					t.Fatalf("ParseID(%q) = %d, want %d", test.raw, got, test.want)
				}
				return
			}
			if err == nil {
				t.Fatalf("ParseID(%q) = %d, want an error", test.raw, got)
			}
		})
	}
}

func TestIDJSONUsesDecimalString(t *testing.T) {
	encoded, err := json.Marshal(ID(42))
	if err != nil {
		t.Fatalf("marshal ID: %v", err)
	}
	if string(encoded) != "\"42\"" {
		t.Fatalf("marshaled ID = %s, want quoted decimal string", encoded)
	}

	var decoded ID
	if err := json.Unmarshal([]byte("\"9007199254740993\""), &decoded); err != nil {
		t.Fatalf("unmarshal ID: %v", err)
	}
	if decoded != ID(9007199254740993) {
		t.Fatalf("unmarshaled ID = %d, want 9007199254740993", decoded)
	}

	if err := json.Unmarshal([]byte("42"), &decoded); err == nil {
		t.Fatal("numeric JSON ID accepted; want a decimal string")
	}
}

func TestIDImplementsPGXInt64Codec(t *testing.T) {
	var valuer pgtype.Int64Valuer = ID(42)
	value, err := valuer.Int64Value()
	if err != nil {
		t.Fatalf("encode ID: %v", err)
	}
	if !value.Valid || value.Int64 != 42 {
		t.Fatalf("encoded ID = %#v, want valid value 42", value)
	}

	var got ID
	var scanner pgtype.Int64Scanner = &got
	if err := scanner.ScanInt64(pgtype.Int8{Int64: 42, Valid: true}); err != nil {
		t.Fatalf("scan ID: %v", err)
	}
	if got != 42 {
		t.Fatalf("scanned ID = %d, want 42", got)
	}
}
