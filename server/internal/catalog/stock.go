package catalog

import "errors"

var (
	ErrStockExceeded = errors.New("catalog: stock exceeds available pool")
	ErrNotEnough     = errors.New("catalog: not enough remaining")
)

// Pool is the good-level shared stock across every event listing of one good.
// Total is the unsold physical stock; Reserved is stock held by active pending
// orders. Sold units already left Total and are tracked per listing as a sales
// ledger, so they must not be subtracted again here.
type Pool struct {
	Total    int
	Reserved int
}

// Available returns how many units may be reserved or sold from the shared
// pool right now, clamped at zero for display.
func (p Pool) Available() int {
	if v := p.Total - p.Reserved; v > 0 {
		return v
	}
	return 0
}

// CanAlloc reports whether qty can be taken from the shared pool.
func (p Pool) CanAlloc(qty int) bool {
	return qty > 0 && p.Total-p.Reserved >= qty
}

func CanTransit(from, to string) bool {
	switch to {
	case "paid":
		return from == "pending"
	case "fulfilled":
		return from == "paid"
	case "cancelled":
		return from == "pending" || from == "paid" || from == "fulfilled"
	default:
		return false
	}
}
