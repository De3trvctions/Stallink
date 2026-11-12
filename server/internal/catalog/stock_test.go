package catalog

import "testing"

func TestAllocationAndConfirm(t *testing.T) {
	listings := []ListingQty{{Listed: 6, Sold: 0}, {Listed: 4, Sold: 0}}
	if err := ValidateAllocation(10, listings); err != nil {
		t.Fatal(err)
	}
	if err := ValidateAllocation(9, listings); err == nil {
		t.Fatal("expected 6+4 to exceed 9")
	}
	total, listing, err := ConfirmSale(10, ListingQty{Listed: 6, Sold: 0}, 1)
	if err != nil || total != 9 || listing.Sold != 1 {
		t.Fatalf("%d %+v %v", total, listing, err)
	}
	after := []ListingQty{listing, {Listed: 4, Sold: 0}}
	if err := ValidateAllocation(total, after); err != nil {
		t.Fatal(err)
	}
	if _, _, err := ConfirmSale(total, listing, 6); err == nil {
		t.Fatal("expected remaining 5 to reject 6")
	}
	restoredTotal, restored, err := RestoreSale(total, listing, 1)
	if err != nil || restoredTotal != 10 || restored.Sold != 0 {
		t.Fatalf("restore %d %+v %v", restoredTotal, restored, err)
	}
}

func TestOrderStatus(t *testing.T) {
	if !CanTransit("pending", "paid") || !CanTransit("paid", "cancelled") || !CanTransit("fulfilled", "cancelled") {
		t.Fatal("expected allowed transitions")
	}
	if CanTransit("pending", "fulfilled") || CanTransit("cancelled", "paid") {
		t.Fatal("expected forbidden transitions")
	}
}
