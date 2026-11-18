// Package reviewswitch stores the platform-wide "auto approve" flag in Redis.
// The flag is fail-closed: a missing key, a Redis outage, or a nil switch
// means new goods stay pending until a platform admin turns approval on.
package reviewswitch

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/De3trvctions/Stallink/internal/logging"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const DefaultKey = "stallink:platform:auto_approve"

type Switch struct {
	client *redis.Client
	key    string
}

// New connects to Redis. When Redis is unreachable the switch stays off and
// Set reports an error, so a platform admin cannot believe a change was saved.
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
		logging.WarnContext(ctx, "review switch Redis unavailable; auto approve stays off", zap.Error(err))
		_ = client.Close()
		return &Switch{key: key}
	}
	return &Switch{client: client, key: key}
}

func (s *Switch) Enabled(ctx context.Context) bool {
	if s == nil || s.client == nil {
		return false
	}
	value, err := s.client.Get(ctx, s.key).Result()
	if errors.Is(err, redis.Nil) {
		if err := s.client.Set(ctx, s.key, "0", 0).Err(); err != nil {
			logging.WarnContext(ctx, "review switch default write failed; auto approve stays off", zap.Error(err))
			return false
		}
		return false
	}
	if err != nil {
		logging.WarnContext(ctx, "review switch read failed; auto approve stays off", zap.Error(err))
		return false
	}
	return value != "0"
}

func (s *Switch) Set(ctx context.Context, enabled bool) error {
	if s == nil || s.client == nil {
		return errors.New("review switch redis is unavailable")
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
