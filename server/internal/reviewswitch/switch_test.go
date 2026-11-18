package reviewswitch

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestNilClientStaysOff(t *testing.T) {
	sw := &Switch{}
	if sw.Enabled(context.Background()) {
		t.Fatal("switch without Redis must stay off")
	}
	if err := sw.Set(context.Background(), true); err == nil {
		t.Fatal("set without Redis must fail")
	}
}

func TestMissingKeyDefaultsToOffAndIsCreated(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	probe := redis.NewClient(&redis.Options{Addr: "127.0.0.1:6379"})
	if err := probe.Ping(ctx).Err(); err != nil {
		t.Skip("redis is not available")
	}
	key := "stallink:test:auto_approve"
	if err := probe.Del(ctx, key).Err(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = probe.Del(context.Background(), key).Err() }()

	sw := New("127.0.0.1:6379", "", 0, key)
	defer func() { _ = sw.Close() }()
	if sw.Enabled(ctx) {
		t.Fatal("missing key should default to approval off")
	}
	value, err := probe.Get(ctx, key).Result()
	if err != nil || value != "0" {
		t.Fatalf("default key should be created as 0, got %q err %v", value, err)
	}
	if err := sw.Set(ctx, true); err != nil {
		t.Fatal(err)
	}
	if !sw.Enabled(ctx) {
		t.Fatal("switch should be on after Set(true)")
	}
}
