package catalog

import "errors"

var (
	ErrStockExceeded = errors.New("catalog: listed quantity exceeds stock")
	ErrNotEnough     = errors.New("catalog: not enough remaining")
)

type ListingQty struct {
	Listed int
	Sold   int
}

func ValidateAllocation(totalStock int, listings []ListingQty) error {
	if totalStock < 0 {
		return ErrStockExceeded
	}
	sum := 0
	for _, listing := range listings {
		if listing.Listed < 0 || listing.Sold < 0 || listing.Sold > listing.Listed {
			return ErrStockExceeded
		}
		sum += listing.Listed - listing.Sold
	}
	if sum > totalStock {
		return ErrStockExceeded
	}
	return nil
}

func ConfirmSale(totalStock int, listing ListingQty, qty int) (int, ListingQty, error) {
	if qty <= 0 || listing.Listed-listing.Sold < qty || totalStock < qty {
		return totalStock, listing, ErrNotEnough
	}
	listing.Sold += qty
	return totalStock - qty, listing, nil
}

func RestoreSale(totalStock int, listing ListingQty, qty int) (int, ListingQty, error) {
	if qty <= 0 || listing.Sold < qty {
		return totalStock, listing, ErrNotEnough
	}
	listing.Sold -= qty
	return totalStock + qty, listing, nil
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
