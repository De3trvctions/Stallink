package catalog

import (
	"errors"
	"math"
)

var ErrInvalidPrice = errors.New("catalog: invalid price or price tier")

const (
	maxPriceTiers      = 50
	maxPricingQuantity = 100_000
)

// PriceTier describes the total price for an exact quantity bundle.
type PriceTier struct {
	Quantity        int   `json:"quantity"`
	TotalPriceMinor int64 `json:"totalPriceMinor"`
}

// ValidatePriceTiers checks that each configured bundle is a real discount
// against buying the same quantity at the base unit price.
func ValidatePriceTiers(unitPrice int64, tiers []PriceTier) error {
	if unitPrice <= 0 || len(tiers) > maxPriceTiers {
		return ErrInvalidPrice
	}

	seen := make(map[int]struct{}, len(tiers))
	for _, tier := range tiers {
		if tier.Quantity <= 1 || tier.Quantity > maxPricingQuantity || tier.TotalPriceMinor <= 0 {
			return ErrInvalidPrice
		}
		if _, exists := seen[tier.Quantity]; exists {
			return ErrInvalidPrice
		}
		seen[tier.Quantity] = struct{}{}
		if int64(tier.Quantity) > math.MaxInt64/unitPrice || tier.TotalPriceMinor >= unitPrice*int64(tier.Quantity) {
			return ErrInvalidPrice
		}
	}
	return nil
}

// BundleTotal returns the cheapest total using single units and the configured
// exact-quantity bundles. Bundles may be combined with each other and singles.
func BundleTotal(unitPrice int64, tiers []PriceTier, quantity int) (int64, error) {
	if quantity < 1 || quantity > maxPricingQuantity || ValidatePriceTiers(unitPrice, tiers) != nil {
		return 0, ErrInvalidPrice
	}

	best := make([]int64, quantity+1)
	for count := 1; count <= quantity; count++ {
		best[count] = math.MaxInt64
	}
	for count := 1; count <= quantity; count++ {
		if best[count-1] != math.MaxInt64 && best[count-1] <= math.MaxInt64-unitPrice {
			best[count] = best[count-1] + unitPrice
		}
		for _, tier := range tiers {
			if tier.Quantity > count {
				continue
			}
			previous := best[count-tier.Quantity]
			if previous == math.MaxInt64 || previous > math.MaxInt64-tier.TotalPriceMinor {
				continue
			}
			candidate := previous + tier.TotalPriceMinor
			if candidate < best[count] {
				best[count] = candidate
			}
		}
		if best[count] == math.MaxInt64 {
			return 0, ErrInvalidPrice
		}
	}
	return best[quantity], nil
}
