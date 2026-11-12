package data

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5/pgtype"
)

// ErrInvalidID reports an empty, non-canonical, non-positive, or overflowing entity ID.
var ErrInvalidID = errors.New("data: invalid id")

// ID is a PostgreSQL BIGINT entity identifier. JSON uses decimal strings so
// JavaScript clients retain exact values beyond Number.MAX_SAFE_INTEGER.
type ID int64

// ParseID parses a canonical positive base-10 PostgreSQL BIGINT ID.
func ParseID(raw string) (ID, error) {
	if raw == "" || (len(raw) > 1 && raw[0] == '0') {
		return 0, ErrInvalidID
	}
	for _, char := range raw {
		if char < '0' || char > '9' {
			return 0, ErrInvalidID
		}
	}
	value, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || value <= 0 {
		return 0, ErrInvalidID
	}
	return ID(value), nil
}

// String returns the decimal representation used by APIs and logs.
func (id ID) String() string {
	return strconv.FormatInt(int64(id), 10)
}

// Int64Value implements pgx's BIGINT value interface. Zero is allowed for
// optional query-filter sentinels; persisted entity IDs must be positive.
func (id ID) Int64Value() (pgtype.Int8, error) {
	if id < 0 {
		return pgtype.Int8{}, ErrInvalidID
	}
	return pgtype.Int8{Int64: int64(id), Valid: true}, nil
}

// ScanInt64 implements pgx's BIGINT scanner and rejects NULL or non-positive IDs.
func (id *ID) ScanInt64(value pgtype.Int8) error {
	if id == nil || !value.Valid || value.Int64 <= 0 {
		return ErrInvalidID
	}
	*id = ID(value.Int64)
	return nil
}

// ScanText implements pgx's text scanner for queries that cast IDs to text.
func (id *ID) ScanText(value pgtype.Text) error {
	if id == nil || !value.Valid {
		return ErrInvalidID
	}
	parsed, err := ParseID(value.String)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}

// MarshalJSON serializes IDs as decimal strings so JavaScript preserves precision.
func (id ID) MarshalJSON() ([]byte, error) {
	if id <= 0 {
		return nil, ErrInvalidID
	}
	return json.Marshal(id.String())
}

// UnmarshalJSON accepts only decimal-string IDs; JSON numbers can lose precision.
func (id *ID) UnmarshalJSON(encoded []byte) error {
	if id == nil {
		return ErrInvalidID
	}
	var raw string
	if err := json.Unmarshal(encoded, &raw); err != nil {
		return ErrInvalidID
	}
	parsed, err := ParseID(raw)
	if err != nil {
		return err
	}
	*id = parsed
	return nil
}
