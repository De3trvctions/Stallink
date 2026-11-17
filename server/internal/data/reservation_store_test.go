package data

import (
	"testing"
)

func TestReservationLinesMatchOrder(t *testing.T) {
	expected := []reservationLine{
		{listingID: 11, goodID: 101, qty: 2},
		{listingID: 12, goodID: 102, qty: 1},
	}

	tests := []struct {
		name   string
		active []reservationLine
		want   bool
	}{
		{name: "complete in different order", active: []reservationLine{
			{listingID: 12, goodID: 102, qty: 1},
			{listingID: 11, goodID: 101, qty: 2},
		}, want: true},
		{name: "missing listing", active: []reservationLine{{listingID: 11, goodID: 101, qty: 2}}},
		{name: "extra listing", active: []reservationLine{
			{listingID: 11, goodID: 101, qty: 2},
			{listingID: 12, goodID: 102, qty: 1},
			{listingID: 13, goodID: 103, qty: 1},
		}},
		{name: "quantity mismatch", active: []reservationLine{
			{listingID: 11, goodID: 101, qty: 1},
			{listingID: 12, goodID: 102, qty: 1},
		}},
		{name: "good mismatch", active: []reservationLine{
			{listingID: 11, goodID: 999, qty: 2},
			{listingID: 12, goodID: 102, qty: 1},
		}},
		{name: "duplicate listing", active: []reservationLine{
			{listingID: 11, goodID: 101, qty: 1},
			{listingID: 11, goodID: 101, qty: 1},
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reservationLinesMatchOrder(expected, tt.active); got != tt.want {
				t.Fatalf("reservationLinesMatchOrder() = %v, want %v", got, tt.want)
			}
		})
	}
}
