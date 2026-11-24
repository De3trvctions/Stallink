package data

import (
	"context"
	"fmt"
)

func (s *ShopStore) PlatformTheme(ctx context.Context) (string, error) {
	var theme string
	if err := s.pool.QueryRow(ctx, `SELECT COALESCE(
		(SELECT theme FROM platform_appearance WHERE singleton), 'classic'
	)`).Scan(&theme); err != nil {
		return "", fmt.Errorf("read platform theme: %w", err)
	}
	return theme, nil
}

func (s *ShopStore) SetPlatformTheme(ctx context.Context, theme string) error {
	if _, err := s.pool.Exec(ctx, `INSERT INTO platform_appearance (singleton, theme)
		VALUES (true, $1)
		ON CONFLICT (singleton) DO UPDATE SET theme = EXCLUDED.theme`, theme); err != nil {
		return fmt.Errorf("set platform theme: %w", err)
	}
	return nil
}
