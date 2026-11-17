package data

// payCodeCurrencyMatches reports whether the pay code's currency is compatible
// with the order currency. A nil pay code currency follows the event default,
// so it always matches.
func payCodeCurrencyMatches(codeCurrency *string, orderCurrency string) bool {
	return codeCurrency == nil || *codeCurrency == orderCurrency
}

// cashSettlement validates cash tender against the order total and returns the
// recorded received and change amounts. Non-cash callers pass nil and receive
// nil amounts back. A received amount below the total is rejected.
func cashSettlement(totalMinor int64, cashReceivedMinor *int64) (received, change *int64, err error) {
	if cashReceivedMinor == nil {
		return nil, nil, nil
	}
	if *cashReceivedMinor < totalMinor {
		return nil, nil, ErrInvalidCashReceived
	}
	receivedVal := *cashReceivedMinor
	changeVal := receivedVal - totalMinor
	return &receivedVal, &changeVal, nil
}

// remainingSoldOut computes the sellable remainder and sold-out flag. The
// remainder is clamped at zero so the public view never exposes a negative
// stock figure.
func remainingSoldOut(listedQty, soldQty, reservedQty int) (remaining int, soldOut bool) {
	remaining = listedQty - soldQty - reservedQty
	if remaining < 0 {
		remaining = 0
	}
	return remaining, remaining <= 0
}
