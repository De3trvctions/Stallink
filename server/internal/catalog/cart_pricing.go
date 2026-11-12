package catalog

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"
)

const (
	MaxCartItems       = 1000
	MaxSKUsPerPriceSet = 20
	MaxBundlesPerSet   = 10
	DefaultMaxStates   = 100000
)

var (
	ErrInvalidCart     = errors.New("catalog: invalid cart")
	ErrPricingTooLarge = errors.New("catalog: cart exceeds exact pricing limits")
)

type CartLineInput struct {
	Kind            string `json:"kind"`
	ListingID       string `json:"listingId,omitempty"`
	BundleListingID string `json:"bundleListingId,omitempty"`
	Qty             int    `json:"qty"`
}

type CartListing struct {
	ID             string
	GoodID         string
	Name           string
	UnitPriceMinor int64
	PriceTiers     []PriceTier
	Revision       int64
	Remaining      int
}

type CartBundleComponent struct {
	ListingID string
	GoodID    string
	Name      string
	Quantity  int
}

type CartBundle struct {
	ID             string
	BundleID       string
	Name           string
	UnitPriceMinor int64
	PriceTiers     []PriceTier
	Revision       int64
	Remaining      int
	Components     []CartBundleComponent
}

type PriceAllocation struct {
	ListingID      string `json:"listingId"`
	GoodID         string `json:"goodId"`
	Name           string `json:"name"`
	Quantity       int    `json:"quantity"`
	UnitPriceMinor int64  `json:"unitPriceMinor"`
	LineTotalMinor int64  `json:"lineTotalMinor"`
}

type PricingGroup struct {
	Kind         string            `json:"kind"`
	RuleID       string            `json:"ruleId"`
	RuleVersion  int64             `json:"ruleVersion"`
	Name         string            `json:"name"`
	Applications int               `json:"applications"`
	TotalMinor   int64             `json:"totalMinor"`
	Lines        []PriceAllocation `json:"lines"`
}

type CartQuote struct {
	Lines        []PriceAllocation `json:"lines"`
	Groups       []PricingGroup    `json:"groups"`
	TotalMinor   int64             `json:"totalMinor"`
	BaseMinor    int64             `json:"baseMinor"`
	SavingsMinor int64             `json:"savingsMinor"`
	ItemCount    int               `json:"itemCount"`
}

type cartOffer struct {
	key      string
	kind     string
	ruleID   string
	version  int64
	name     string
	price    int64
	vector   []int
	SKUCount int
}

type dpResult struct {
	cost   int64
	choice int
	ok     bool
}

type usage struct {
	offer int
	count int
}

// PriceCart expands bundle quantities into real SKU listings, then finds the
// exact lowest-cost combination of regular units, quantity tiers, and fixed
// bundles. It never approximates when a configured computation limit is hit.
func PriceCart(input []CartLineInput, listings map[string]CartListing, bundles map[string]CartBundle, maxStates int) (CartQuote, error) {
	if len(input) == 0 || len(input) > 100 || maxStates <= 0 {
		return CartQuote{}, ErrInvalidCart
	}
	if maxStates > DefaultMaxStates {
		maxStates = DefaultMaxStates
	}
	requested := make(map[string]int)
	itemCount := 0
	for _, line := range input {
		if line.Qty < 1 || line.Qty > MaxCartItems {
			return CartQuote{}, ErrInvalidCart
		}
		switch line.Kind {
		case "single":
			if line.ListingID == "" || line.BundleListingID != "" {
				return CartQuote{}, ErrInvalidCart
			}
			listing, ok := listings[line.ListingID]
			if !ok || !validCartListing(listing) {
				return CartQuote{}, ErrInvalidCart
			}
			if line.Qty > listing.Remaining {
				return CartQuote{}, ErrNotEnough
			}
			if requested[listing.ID] > MaxCartItems-line.Qty {
				return CartQuote{}, ErrPricingTooLarge
			}
			requested[listing.ID] += line.Qty
			itemCount += line.Qty
		case "bundle":
			if line.BundleListingID == "" || line.ListingID != "" {
				return CartQuote{}, ErrInvalidCart
			}
			bundle, ok := bundles[line.BundleListingID]
			if !ok || !validCartBundle(bundle) {
				return CartQuote{}, ErrInvalidCart
			}
			if line.Qty > bundle.Remaining {
				return CartQuote{}, ErrNotEnough
			}
			for _, component := range bundle.Components {
				if _, ok := listings[component.ListingID]; !ok || component.Quantity > MaxCartItems/line.Qty {
					return CartQuote{}, ErrInvalidCart
				}
				qty := component.Quantity * line.Qty
				if requested[component.ListingID] > MaxCartItems-qty {
					return CartQuote{}, ErrPricingTooLarge
				}
				requested[component.ListingID] += qty
				itemCount += qty
			}
		default:
			return CartQuote{}, ErrInvalidCart
		}
		if itemCount > MaxCartItems {
			return CartQuote{}, ErrPricingTooLarge
		}
	}
	if len(requested) == 0 {
		return CartQuote{}, ErrInvalidCart
	}

	listingIDs := make([]string, 0, len(requested))
	for listingID, qty := range requested {
		if qty <= 0 {
			return CartQuote{}, ErrInvalidCart
		}
		if qty > listings[listingID].Remaining {
			return CartQuote{}, ErrNotEnough
		}
		listingIDs = append(listingIDs, listingID)
	}
	sort.Strings(listingIDs)
	skuIndex := make(map[string]int, len(listingIDs))
	for index, id := range listingIDs {
		skuIndex[id] = index
	}

	offers := make([]cartOffer, 0, len(listings)+len(bundles))
	for _, id := range listingIDs {
		listing := listings[id]
		offers = append(offers, cartOffer{
			key: "single:" + id + ":1", kind: "regular", ruleID: id, version: positiveRevision(listing.Revision),
			name: listing.Name, price: listing.UnitPriceMinor, vector: unitVector(len(listingIDs), skuIndex[id]), SKUCount: 1,
		})
		if err := ValidatePriceTiers(listing.UnitPriceMinor, listing.PriceTiers); err != nil {
			return CartQuote{}, err
		}
		for _, tier := range listing.PriceTiers {
			vector := unitVector(len(listingIDs), skuIndex[id])
			vector[skuIndex[id]] = tier.Quantity
			offers = append(offers, cartOffer{
				key: fmt.Sprintf("single_tier:%s:%06d", id, tier.Quantity), kind: "single_tier", ruleID: id,
				version: positiveRevision(listing.Revision), name: listing.Name, price: tier.TotalPriceMinor,
				vector: vector, SKUCount: 1,
			})
		}
	}
	usedBundles := make(map[string]CartBundle)
	for _, bundle := range bundles {
		if !validCartBundle(bundle) || bundle.Remaining == 0 {
			continue
		}
		eligibleForCart := true
		for _, component := range bundle.Components {
			if _, selected := skuIndex[component.ListingID]; !selected {
				eligibleForCart = false
				break
			}
		}
		if eligibleForCart {
			usedBundles[bundle.ID] = bundle
		}
	}
	for _, bundle := range usedBundles {
		if err := ValidatePriceTiers(bundle.UnitPriceMinor, bundle.PriceTiers); err != nil {
			return CartQuote{}, err
		}
		vector := bundleVector(bundle, skuIndex, len(listingIDs), 1)
		if vector == nil {
			return CartQuote{}, ErrInvalidCart
		}
		offers = append(offers, cartOffer{
			key: "bundle:" + bundle.ID + ":1", kind: "bundle", ruleID: bundle.ID, version: positiveRevision(bundle.Revision),
			name: bundle.Name, price: bundle.UnitPriceMinor, vector: vector, SKUCount: len(bundle.Components),
		})
		for _, tier := range bundle.PriceTiers {
			tierVector := bundleVector(bundle, skuIndex, len(listingIDs), tier.Quantity)
			if tierVector == nil {
				return CartQuote{}, ErrInvalidCart
			}
			offers = append(offers, cartOffer{
				key: fmt.Sprintf("bundle_tier:%s:%06d", bundle.ID, tier.Quantity), kind: "bundle_tier", ruleID: bundle.ID,
				version: positiveRevision(bundle.Revision), name: bundle.Name, price: tier.TotalPriceMinor,
				vector: tierVector, SKUCount: len(bundle.Components),
			})
		}
	}
	sort.Slice(offers, func(i, j int) bool { return offers[i].key < offers[j].key })

	groups := connectedSKUGroups(offers, len(listingIDs))
	uses := make([]usage, 0, len(offers))
	for _, group := range groups {
		if len(group) > MaxSKUsPerPriceSet {
			return CartQuote{}, ErrPricingTooLarge
		}
		groupSet := make(map[int]struct{}, len(group))
		for _, idx := range group {
			groupSet[idx] = struct{}{}
		}
		localOffers := make([]cartOffer, 0)
		localMap := make([]int, 0)
		bundleIDs := make(map[string]struct{})
		for offerIndex, offer := range offers {
			touches := false
			for idx, qty := range offer.vector {
				if qty > 0 {
					if _, ok := groupSet[idx]; ok {
						touches = true
						break
					}
				}
			}
			if !touches {
				continue
			}
			local := cartOffer{key: offer.key, kind: offer.kind, ruleID: offer.ruleID, version: offer.version, name: offer.name, price: offer.price, vector: make([]int, len(group))}
			local.SKUCount = offer.SKUCount
			for localIndex, globalIndex := range group {
				local.vector[localIndex] = offer.vector[globalIndex]
			}
			localOffers = append(localOffers, local)
			localMap = append(localMap, offerIndex)
			if offer.kind == "bundle" || offer.kind == "bundle_tier" {
				bundleIDs[offer.ruleID] = struct{}{}
			}
		}
		if len(bundleIDs) > MaxBundlesPerSet {
			return CartQuote{}, ErrPricingTooLarge
		}
		counts := make([]int, len(group))
		for localIndex, globalIndex := range group {
			counts[localIndex] = requested[listingIDs[globalIndex]]
		}
		localUses, err := solveOffers(counts, localOffers, maxStates)
		if err != nil {
			return CartQuote{}, err
		}
		for _, item := range localUses {
			uses = append(uses, usage{offer: localMap[item.offer], count: item.count})
		}
	}

	quote := CartQuote{Groups: make([]PricingGroup, 0, len(uses)), ItemCount: itemCount}
	lineTotals := make(map[string]int64, len(requested))
	for _, use := range uses {
		offer := offers[use.offer]
		if use.count < 1 || offer.price > math.MaxInt64/int64(use.count) {
			return CartQuote{}, ErrInvalidPrice
		}
		groupTotal := offer.price * int64(use.count)
		allocations := make([]PriceAllocation, 0, offer.SKUCount)
		weights := make([]int64, 0, offer.SKUCount)
		for index, qtyPerUse := range offer.vector {
			qty := qtyPerUse * use.count
			if qty == 0 {
				continue
			}
			listing := listings[listingIDs[index]]
			if int64(qty) > math.MaxInt64/listing.UnitPriceMinor {
				return CartQuote{}, ErrInvalidPrice
			}
			weight := int64(qty) * listing.UnitPriceMinor
			allocations = append(allocations, PriceAllocation{ListingID: listing.ID, GoodID: listing.GoodID, Name: listing.Name, Quantity: qty, UnitPriceMinor: listing.UnitPriceMinor})
			weights = append(weights, weight)
		}
		if err := allocateGroupAmount(groupTotal, allocations, weights); err != nil {
			return CartQuote{}, err
		}
		for _, allocation := range allocations {
			if lineTotals[allocation.ListingID] > math.MaxInt64-allocation.LineTotalMinor {
				return CartQuote{}, ErrInvalidPrice
			}
			lineTotals[allocation.ListingID] += allocation.LineTotalMinor
		}
		if quote.TotalMinor > math.MaxInt64-groupTotal {
			return CartQuote{}, ErrInvalidPrice
		}
		quote.TotalMinor += groupTotal
		quote.Groups = append(quote.Groups, PricingGroup{
			Kind: offer.kind, RuleID: offer.ruleID, RuleVersion: offer.version, Name: offer.name,
			Applications: use.count, TotalMinor: groupTotal, Lines: allocations,
		})
	}
	sort.Slice(quote.Groups, func(i, j int) bool {
		if quote.Groups[i].RuleID == quote.Groups[j].RuleID {
			return quote.Groups[i].Kind < quote.Groups[j].Kind
		}
		return quote.Groups[i].RuleID < quote.Groups[j].RuleID
	})
	quote.Lines = make([]PriceAllocation, 0, len(requested))
	for _, listingID := range listingIDs {
		listing := listings[listingID]
		qty := requested[listingID]
		if int64(qty) > math.MaxInt64/listing.UnitPriceMinor {
			return CartQuote{}, ErrInvalidPrice
		}
		base := int64(qty) * listing.UnitPriceMinor
		if quote.BaseMinor > math.MaxInt64-base {
			return CartQuote{}, ErrInvalidPrice
		}
		quote.BaseMinor += base
		quote.Lines = append(quote.Lines, PriceAllocation{
			ListingID: listing.ID, GoodID: listing.GoodID, Name: listing.Name,
			Quantity: qty, UnitPriceMinor: listing.UnitPriceMinor, LineTotalMinor: lineTotals[listingID],
		})
	}
	quote.SavingsMinor = quote.BaseMinor - quote.TotalMinor
	return quote, nil
}

func solveOffers(counts []int, offers []cartOffer, maxStates int) ([]usage, error) {
	memo := make(map[string]dpResult)
	states := 0
	var solve func() (dpResult, error)
	solve = func() (dpResult, error) {
		key := pricingStateKey(counts)
		if result, ok := memo[key]; ok {
			return result, nil
		}
		states++
		if states > maxStates {
			return dpResult{}, ErrPricingTooLarge
		}
		pivot := -1
		for index, count := range counts {
			if count > 0 {
				pivot = index
				break
			}
		}
		if pivot == -1 {
			result := dpResult{cost: 0, choice: -1, ok: true}
			memo[key] = result
			return result, nil
		}
		best := dpResult{}
		bestKey := ""
		for offerIndex, offer := range offers {
			if offer.vector[pivot] == 0 || offer.price < 0 {
				continue
			}
			fits := true
			for index, quantity := range offer.vector {
				if quantity > counts[index] {
					fits = false
					break
				}
			}
			if !fits {
				continue
			}
			for index, quantity := range offer.vector {
				counts[index] -= quantity
			}
			rest, err := solve()
			for index, quantity := range offer.vector {
				counts[index] += quantity
			}
			if err != nil {
				return dpResult{}, err
			}
			if !rest.ok || rest.cost > math.MaxInt64-offer.price {
				continue
			}
			candidate := rest.cost + offer.price
			if !best.ok || candidate < best.cost || (candidate == best.cost && (bestKey == "" || offer.key < bestKey)) {
				best = dpResult{cost: candidate, choice: offerIndex, ok: true}
				bestKey = offer.key
			}
		}
		if best.ok {
			memo[key] = best
		}
		return best, nil
	}
	result, err := solve()
	if err != nil {
		return nil, err
	}
	if !result.ok {
		return nil, ErrInvalidPrice
	}
	uses := make(map[int]int)
	for {
		key := pricingStateKey(counts)
		state := memo[key]
		if state.choice < 0 {
			break
		}
		choice := state.choice
		uses[choice]++
		for index, quantity := range offers[choice].vector {
			counts[index] -= quantity
		}
	}
	out := make([]usage, 0, len(uses))
	for index, count := range uses {
		out = append(out, usage{offer: index, count: count})
	}
	sort.Slice(out, func(i, j int) bool { return offers[out[i].offer].key < offers[out[j].offer].key })
	return out, nil
}

func connectedSKUGroups(offers []cartOffer, count int) [][]int {
	parent := make([]int, count)
	for index := range parent {
		parent[index] = index
	}
	var find func(int) int
	find = func(index int) int {
		if parent[index] != index {
			parent[index] = find(parent[index])
		}
		return parent[index]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	for _, offer := range offers {
		first := -1
		for index, qty := range offer.vector {
			if qty == 0 {
				continue
			}
			if first == -1 {
				first = index
			} else {
				union(first, index)
			}
		}
	}
	byRoot := make(map[int][]int)
	for index := 0; index < count; index++ {
		root := find(index)
		byRoot[root] = append(byRoot[root], index)
	}
	groups := make([][]int, 0, len(byRoot))
	for _, group := range byRoot {
		groups = append(groups, group)
	}
	sort.Slice(groups, func(i, j int) bool { return groups[i][0] < groups[j][0] })
	return groups
}

func bundleVector(bundle CartBundle, indexes map[string]int, size, multiplier int) []int {
	vector := make([]int, size)
	seen := make(map[string]struct{}, len(bundle.Components))
	if len(bundle.Components) < 2 || len(bundle.Components) > MaxSKUsPerPriceSet {
		return nil
	}
	for _, component := range bundle.Components {
		index, ok := indexes[component.ListingID]
		if !ok || component.Quantity < 1 || component.Quantity > MaxCartItems/multiplier {
			return nil
		}
		if _, duplicate := seen[component.ListingID]; duplicate {
			return nil
		}
		seen[component.ListingID] = struct{}{}
		vector[index] = component.Quantity * multiplier
	}
	return vector
}

func validCartListing(listing CartListing) bool {
	return listing.ID != "" && listing.GoodID != "" && listing.Name != "" && listing.UnitPriceMinor > 0 && listing.Remaining >= 0
}

func validCartBundle(bundle CartBundle) bool {
	return bundle.ID != "" && bundle.BundleID != "" && bundle.Name != "" && bundle.UnitPriceMinor > 0 && bundle.Remaining >= 0 && len(bundle.Components) >= 2
}

func unitVector(size, index int) []int {
	vector := make([]int, size)
	vector[index] = 1
	return vector
}

func positiveRevision(revision int64) int64 {
	if revision < 1 {
		return 1
	}
	return revision
}

func pricingStateKey(counts []int) string {
	encoded := make([]byte, len(counts)*4)
	for index, count := range counts {
		binary.LittleEndian.PutUint32(encoded[index*4:], uint32(count))
	}
	return string(encoded)
}

func allocateGroupAmount(total int64, allocations []PriceAllocation, weights []int64) error {
	var weightTotal int64
	for _, weight := range weights {
		if weight <= 0 || weightTotal > math.MaxInt64-weight {
			return ErrInvalidPrice
		}
		weightTotal += weight
	}
	type remainder struct {
		index int
		value *big.Int
		key   string
	}
	remainders := make([]remainder, 0, len(weights))
	var allocated int64
	denominator := big.NewInt(weightTotal)
	for index, weight := range weights {
		product := new(big.Int).Mul(big.NewInt(total), big.NewInt(weight))
		quotient, rest := new(big.Int), new(big.Int)
		quotient.QuoRem(product, denominator, rest)
		share := quotient.Int64()
		allocations[index].LineTotalMinor = share
		allocated += share
		remainders = append(remainders, remainder{index: index, value: rest, key: allocations[index].ListingID})
	}
	remaining := total - allocated
	sort.Slice(remainders, func(i, j int) bool {
		cmp := remainders[i].value.Cmp(remainders[j].value)
		if cmp == 0 {
			return remainders[i].key < remainders[j].key
		}
		return cmp > 0
	})
	for index := int64(0); index < remaining; index++ {
		allocations[remainders[index%int64(len(remainders))].index].LineTotalMinor++
	}
	return nil
}
