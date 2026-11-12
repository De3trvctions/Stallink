package auth

import "testing"

func TestPasswordRoundTripAndLength(t *testing.T) {
	hash, err := HashPassword("correct-horse")
	if err != nil {
		t.Fatal(err)
	}
	if !CheckPassword(hash, "correct-horse") || CheckPassword(hash, "wrong-horse") {
		t.Fatal("bcrypt check")
	}
	if _, err := HashPassword("short"); err == nil {
		t.Fatal("expected short password to fail")
	}
}
