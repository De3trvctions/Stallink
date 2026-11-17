package service

import "strings"

// validLoginName mirrors the users.login_name CHECK: 3–32 runes and no
// whitespace. Kept as a pure helper so the rule can be unit-tested without a
// request.
func validLoginName(name string) bool {
	if len([]rune(name)) < 3 || len([]rune(name)) > 32 {
		return false
	}
	return !strings.ContainsAny(name, " \t\n")
}

// validProfileName mirrors display_name <= 80 and contact <= 120 runes.
func validProfileName(display, contact string) bool {
	return len([]rune(display)) <= 80 && len([]rune(contact)) <= 120
}

// validPayCodeName mirrors pay_codes.name BETWEEN 1 AND 40 runes; the empty
// check stays at the call site.
func validPayCodeName(name string) bool {
	return len([]rune(name)) <= 40
}
