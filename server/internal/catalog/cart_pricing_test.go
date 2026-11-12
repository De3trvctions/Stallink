package catalog

import (
	"errors"
	"math"
	"testing"
)

func TestPriceCartChoosesLowestCombination(t *testing.T) {
	tests := []struct {
		name     string
		lines    []CartLineInput
		listings map[string]CartListing
		bundles  map[string]CartBundle
		want     int64
	}{
		{
			name:     "automatic fixed bundle",
			lines:    []CartLineInput{{Kind: "single", ListingID: "a", Qty: 1}, {Kind: "single", ListingID: "b", Qty: 1}, {Kind: "single", ListingID: "c", Qty: 1}},
			listings: testListings("a", "b", "c"),
			bundles:  map[string]CartBundle{"abc": testBundle("abc", "ABC", 5500, component("a", 1), component("b", 1), component("c", 1))},
			want:     5500,
		},
		{
			name:  "single quantity offer wins over overlapping bundle",
			lines: []CartLineInput{{Kind: "single", ListingID: "a", Qty: 2}, {Kind: "single", ListingID: "b", Qty: 1}},
			listings: map[string]CartListing{
				"a": testListing("a", 1000, 8, []PriceTier{{Quantity: 2, TotalPriceMinor: 1500}}),
				"b": testListing("b", 1000, 8, nil),
			},
			bundles: map[string]CartBundle{"ab": testBundle("ab", "AB", 1600, component("a", 1), component("b", 1))},
			want:    2500,
		},
		{
			name:     "overlapping bundles choose the global minimum",
			lines:    []CartLineInput{{Kind: "single", ListingID: "a", Qty: 1}, {Kind: "single", ListingID: "b", Qty: 1}, {Kind: "single", ListingID: "c", Qty: 1}, {Kind: "single", ListingID: "d", Qty: 1}},
			listings: testListings("a", "b", "c", "d"),
			bundles: map[string]CartBundle{
				"abc": testBundle("abc", "ABC", 5500, component("a", 1), component("b", 1), component("c", 1)),
				"bcd": testBundle("bcd", "BCD", 5000, component("b", 1), component("c", 1), component("d", 1)),
			},
			want: 7000,
		},
		{
			name:     "explicit bundle expands to real item quantities",
			lines:    []CartLineInput{{Kind: "bundle", BundleListingID: "abc", Qty: 1}},
			listings: testListings("a", "b", "c"),
			bundles:  map[string]CartBundle{"abc": testBundle("abc", "ABC", 5500, component("a", 1), component("b", 2), component("c", 1))},
			want:     5500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quote, err := PriceCart(tt.lines, tt.listings, tt.bundles, DefaultMaxStates)
			if err != nil {
				t.Fatal(err)
			}
			if quote.TotalMinor != tt.want {
				t.Fatalf("total = %d, want %d (groups: %+v)", quote.TotalMinor, tt.want, quote.Groups)
			}
			var lineTotal int64
			for _, line := range quote.Lines {
				lineTotal += line.LineTotalMinor
			}
			if lineTotal != quote.TotalMinor {
				t.Fatalf("line totals = %d, quote total = %d", lineTotal, quote.TotalMinor)
			}
		})
	}
}

func TestPriceCartRejectsInvalidAndOverBudgetCarts(t *testing.T) {
	listings := testListings("a", "b", "c")
	valid := []CartLineInput{{Kind: "single", ListingID: "a", Qty: 1}}
	if _, err := PriceCart([]CartLineInput{{Kind: "single", ListingID: "a", Qty: 0}}, listings, nil, DefaultMaxStates); !errors.Is(err, ErrInvalidCart) {
		t.Fatalf("zero quantity error = %v", err)
	}
	if _, err := PriceCart(valid, listings, nil, 1); !errors.Is(err, ErrPricingTooLarge) {
		t.Fatalf("state limit error = %v", err)
	}
	tooExpensive := map[string]CartListing{"a": testListing("a", math.MaxInt64, 5, nil)}
	if _, err := PriceCart([]CartLineInput{{Kind: "single", ListingID: "a", Qty: 2}}, tooExpensive, nil, DefaultMaxStates); !errors.Is(err, ErrInvalidPrice) {
		t.Fatalf("overflow error = %v", err)
	}
}

func TestPriceCartMatchesExhaustiveOracle(t *testing.T) {
	listings := map[string]CartListing{
		"a": testListing("a", 1100, 5, []PriceTier{{Quantity: 2, TotalPriceMinor: 1900}}),
		"b": testListing("b", 900, 5, []PriceTier{{Quantity: 3, TotalPriceMinor: 2300}}),
		"c": testListing("c", 1300, 5, nil),
	}
	bundles := map[string]CartBundle{
		"ab": testBundle("ab", "AB", 1700, component("a", 1), component("b", 1)),
		"bc": testBundle("bc", "BC", 1950, component("b", 1), component("c", 1)),
	}
	lines := []CartLineInput{{Kind: "single", ListingID: "a", Qty: 2}, {Kind: "single", ListingID: "b", Qty: 2}, {Kind: "single", ListingID: "c", Qty: 1}}
	quote, err := PriceCart(lines, listings, bundles, DefaultMaxStates)
	if err != nil {
		t.Fatal(err)
	}
	want := brutePrice([]int{2, 2, 1}, []testOffer{
		{vector: []int{1, 0, 0}, price: 1100}, {vector: []int{2, 0, 0}, price: 1900},
		{vector: []int{0, 1, 0}, price: 900}, {vector: []int{0, 3, 0}, price: 2300},
		{vector: []int{0, 0, 1}, price: 1300},
		{vector: []int{1, 1, 0}, price: 1700}, {vector: []int{0, 1, 1}, price: 1950},
	})
	if quote.TotalMinor != want {
		t.Fatalf("exact quote = %d, exhaustive oracle = %d", quote.TotalMinor, want)
	}
}

type testOffer struct {
	vector []int
	price  int64
}

func brutePrice(counts []int, offers []testOffer) int64 {
	pivot := -1
	for index, count := range counts {
		if count > 0 {
			pivot = index
			break
		}
	}
	if pivot < 0 {
		return 0
	}
	best := int64(math.MaxInt64)
	for _, offer := range offers {
		if offer.vector[pivot] == 0 {
			continue
		}
		fits := true
		for i, q := range offer.vector {
			if q > counts[i] {
				fits = false
				break
			}
		}
		if !fits {
			continue
		}
		for i, q := range offer.vector {
			counts[i] -= q
		}
		rest := brutePrice(counts, offers)
		for i, q := range offer.vector {
			counts[i] += q
		}
		if rest <= math.MaxInt64-offer.price && rest+offer.price < best {
			best = rest + offer.price
		}
	}
	return best
}

func testListing(id string, price int64, remaining int, tiers []PriceTier) CartListing {
	return CartListing{ID: id, GoodID: id + "-good", Name: id, UnitPriceMinor: price, Remaining: remaining, PriceTiers: tiers, Revision: 1}
}

func testListings(ids ...string) map[string]CartListing {
	out := make(map[string]CartListing, len(ids))
	for _, id := range ids {
		out[id] = testListing(id, 2000, 20, nil)
	}
	return out
}

func component(id string, qty int) CartBundleComponent {
	return CartBundleComponent{ListingID: id, GoodID: id + "-good", Name: id, Quantity: qty}
}

func testBundle(id, name string, price int64, components ...CartBundleComponent) CartBundle {
	return CartBundle{ID: id, BundleID: id + "-bundle", Name: name, UnitPriceMinor: price, Remaining: 20, Components: components, Revision: 1}
}
