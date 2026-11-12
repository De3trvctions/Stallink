package catalog

import "fmt"

func ValidCurrency(currency string) bool {
	return currency == "MYR" || currency == "SGD" || currency == "CNY"
}

func ValidEventStatus(status string) bool {
	return status == "scheduled" || status == "live" || status == "ended"
}

func FormatMoney(currency string, minor int64) (string, error) {
	var prefix string
	switch currency {
	case "MYR":
		prefix = "RM "
	case "SGD":
		prefix = "S$ "
	case "CNY":
		prefix = "¥ "
	default:
		return "", fmt.Errorf("catalog: unknown currency")
	}
	if minor < 0 {
		return "", fmt.Errorf("catalog: negative amount")
	}
	return fmt.Sprintf("%s%d.%02d", prefix, minor/100, minor%100), nil
}
