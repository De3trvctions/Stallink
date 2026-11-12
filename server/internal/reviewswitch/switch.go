// Package reviewswitch stores the platform-wide "auto approve" flag in Redis.
// The flag is fail-open: a missing key means approval is on, and the key is
// created with the default value on first read.
package reviewswitch

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/De3trvctions/Stallink/internal/logging"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const DefaultKey = "stallink:platform:auto_approve"

type Switch struct {
	client *redis.Client
	key    string
	mu     sync.RWMutex
	local  bool
}

// New connects to Redis. When Redis is unreachable it falls back to an
// in-process flag that still defaults to approval on, so the platform keeps
// working instead of blocking every submission.
func New(addr, password string, db int, key string) *Switch {
	if strings.TrimSpace(key) == "" {
		key = DefaultKey
	}
	if strings.TrimSpace(addr) == "" {
		addr = "127.0.0.1:6379"
	}
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password, DB: db})
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		logging.WarnContext(ctx, "review switch Redis unavailable; using in-process default", zap.Error(err))
		_ = client.Close()
		return &Switch{key: key, local: true}
	}
	return &Switch{client: client, key: key, local: true}
}

func (s *Switch) Enabled(ctx context.Context) bool {
	if s == nil {
		return true
	}
	if s.client == nil {
		return s.value()
	}
	value, err := s.client.Get(ctx, s.key).Result()
	if errors.Is(err, redis.Nil) {
		if err := s.client.Set(ctx, s.key, "1", 0).Err(); err != nil {
			return true
		}
		s.store(true)
		return true
	}
	if err != nil {
		logging.WarnContext(ctx, "review switch read failed; using in-process value", zap.Error(err))
		return s.value()
	}
	enabled := value != "0"
	s.store(enabled)
	return enabled
}

func (s *Switch) Set(ctx context.Context, enabled bool) error {
	if s == nil {
		return nil
	}
	s.store(enabled)
	if s.client == nil {
		return nil
	}
	value := "0"
	if enabled {
		value = "1"
	}
	return s.client.Set(ctx, s.key, value, 0).Err()
}

func (s *Switch) Close() error {
	if s == nil || s.client == nil {
		return nil
	}
	return s.client.Close()
}

func (s *Switch) value() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.local
}

func (s *Switch) store(enabled bool) {
	s.mu.Lock()
	s.local = enabled
	s.mu.Unlock()
}
