package data

// payCodeCurrencyMatches reports whether the pay code's currency is compatible
// with the order currency. A nil pay code currency follows the event default,
// so it always matches.
func payCodeCurrencyMatches(codeCurrency *string, orderCurrency string) bool {
	return codeCurrency == nil || *codeCurrency == orderCurrency
}

// cashSettlement validates tender against the pay-code kind. Cash requires a
// received amount that covers the total. QR and every other kind reject a
// cash amount so the drawer figures cannot be attached to the wrong method.
func cashSettlement(kind string, totalMinor int64, cashReceivedMinor *int64) (received, change *int64, err error) {
	if kind == "cash" {
		if cashReceivedMinor == nil || *cashReceivedMinor < totalMinor {
			return nil, nil, ErrInvalidCashReceived
		}
		receivedVal := *cashReceivedMinor
		changeVal := receivedVal - totalMinor
		return &receivedVal, &changeVal, nil
	}
	if cashReceivedMinor != nil {
		return nil, nil, ErrInvalidCashReceived
	}
	return nil, nil, nil
}

// payStillAllowed reports whether an existing pending order may still be paid.
// lineCount is every order line. eligibleCount is the subset whose good is
// approved and whose listing is still listed on that order's event.
func payStillAllowed(eventStatus string, allowScheduled bool, lineCount, eligibleCount int) error {
	if err := validateEventOpen(eventStatus, allowScheduled); err != nil {
		return err
	}
	if lineCount < 1 || eligibleCount != lineCount {
		return ErrNotListed
	}
	return nil
}

// poolRemaining computes the shared-pool sellable remainder and sold-out
// flag. Total is the unsold physical stock; Reserved is stock held by active
// pending orders. The remainder is clamped at zero so the public view never
// exposes a negative stock figure.
func poolRemaining(totalStock, reserved int) (remaining int, soldOut bool) {
	remaining = totalStock - reserved
	if remaining < 0 {
		remaining = 0
	}
	return remaining, remaining <= 0
}
