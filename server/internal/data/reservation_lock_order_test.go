package data

import (
	"reflect"
	"testing"
)

func TestInventoryLockOrderSortsAndDeduplicates(t *testing.T) {
	lines := []reservationLine{
		{listingID: 40, goodID: 90, qty: 1},
		{listingID: 10, goodID: 20, qty: 1},
		{listingID: 30, goodID: 20, qty: 2},
		{listingID: 5, goodID: 5, qty: 1},
	}

	goodIDs, listingIDs := inventoryLockOrder(lines)
	if want := []ID{5, 20, 90}; !reflect.DeepEqual(goodIDs, want) {
		t.Fatalf("good lock order = %v, want %v", goodIDs, want)
	}
	if want := []ID{5, 10, 30, 40}; !reflect.DeepEqual(listingIDs, want) {
		t.Fatalf("listing lock order = %v, want %v", listingIDs, want)
	}
}
