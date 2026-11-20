package catalog

import "testing"

func TestSharedPoolAvailable(t *testing.T) {
	pool := Pool{Total: 10, Reserved: 3}
	if avail := pool.Available(); avail != 7 {
		t.Fatalf("expected 7 available, got %d", avail)
	}
	if !pool.CanAlloc(7) || pool.CanAlloc(8) {
		t.Fatal("pool should allocate up to 7")
	}
	negative := Pool{Total: 2, Reserved: 5}
	if avail := negative.Available(); avail != 0 {
		t.Fatalf("expected clamped 0, got %d", avail)
	}
	if negative.CanAlloc(1) {
		t.Fatal("empty pool must reject allocation")
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
