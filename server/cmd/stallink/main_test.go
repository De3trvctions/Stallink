package main

import "testing"

func TestLocalSeedPasswordRequiresEnv(t *testing.T) {
	t.Setenv("STALLINK_LOCAL_SEED_PASSWORD", "")
	if _, err := localSeedPassword(); err == nil {
		t.Fatal("expected an empty seed password to fail")
	}
	t.Setenv("STALLINK_LOCAL_SEED_PASSWORD", "short")
	if _, err := localSeedPassword(); err == nil {
		t.Fatal("expected a short seed password to fail")
	}
	t.Setenv("STALLINK_LOCAL_SEED_PASSWORD", "local-seed-password")
	got, err := localSeedPassword()
	if err != nil || got != "local-seed-password" {
		t.Fatalf("localSeedPassword() = %q, %v", got, err)
	}
}
