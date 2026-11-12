package data

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type CommandIdentity struct {
	ShopID      ID
	ActorID     ID
	ActorRole   string
	Operation   string
	Key         string
	RequestHash [32]byte
}

type OrderCommandResult struct {
	Order      Order
	StatusCode int
	Replayed   bool
}

var (
	ErrIdempotencyConflict = errors.New("data: idempotency key reused with another request")
	ErrCheckoutConflict    = errors.New("data: checkout identity reused with another request")
	ErrCommandInProgress   = errors.New("data: command is still in progress")
	ErrCommandUnauthorized = errors.New("data: command replay authorization failed")
)

func (s *ShopStore) runOrderCommand(
	ctx context.Context,
	identity CommandIdentity,
	statusCode int,
	mutate func(pgx.Tx) (Order, error),
) (OrderCommandResult, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return OrderCommandResult{}, fmt.Errorf("begin order command: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var commandID ID
	err = tx.QueryRow(ctx, `
		INSERT INTO order_commands(shop_id, actor_id, operation, key, request_hash)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (shop_id, actor_id, operation, key) DO NOTHING
		RETURNING id::text`, identity.ShopID, identity.ActorID, identity.Operation, identity.Key, identity.RequestHash[:]).Scan(&commandID)
	if errors.Is(err, pgx.ErrNoRows) {
		var storedHash, responseBody []byte
		var storedStatus int
		var completed bool
		err = tx.QueryRow(ctx, `
		SELECT request_hash, COALESCE(response_status, 0), COALESCE(response_body, 'null'::jsonb), completed_at IS NOT NULL
			FROM order_commands
			WHERE shop_id = $1 AND actor_id = $2 AND operation = $3 AND key = $4`,
			identity.ShopID, identity.ActorID, identity.Operation, identity.Key).
			Scan(&storedHash, &storedStatus, &responseBody, &completed)
		if err != nil {
			return OrderCommandResult{}, fmt.Errorf("read order command: %w", err)
		}
		if !bytes.Equal(storedHash, identity.RequestHash[:]) {
			return OrderCommandResult{}, ErrIdempotencyConflict
		}
		var authorized bool
		if err := tx.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM users u JOIN shops s ON s.id = u.shop_id
				WHERE u.id = $1 AND u.shop_id = $2 AND u.is_enabled AND s.is_enabled
			)`, identity.ActorID, identity.ShopID).Scan(&authorized); err != nil {
			return OrderCommandResult{}, fmt.Errorf("authorize command replay: %w", err)
		}
		if !authorized {
			return OrderCommandResult{}, ErrCommandUnauthorized
		}
		if !completed {
			return OrderCommandResult{}, ErrCommandInProgress
		}
		var order Order
		if err := json.Unmarshal(responseBody, &order); err != nil {
			return OrderCommandResult{}, fmt.Errorf("decode order command response: %w", err)
		}
		order, err = getOrderTx(ctx, tx, identity.ShopID, order.ID)
		if err != nil {
			return OrderCommandResult{}, fmt.Errorf("read current order after command replay: %w", err)
		}
		return OrderCommandResult{Order: order, StatusCode: storedStatus, Replayed: true}, nil
	}
	if err != nil {
		return OrderCommandResult{}, fmt.Errorf("claim order command: %w", err)
	}

	order, err := mutate(tx)
	if err != nil {
		return OrderCommandResult{}, err
	}
	responseBody, err := json.Marshal(order)
	if err != nil {
		return OrderCommandResult{}, fmt.Errorf("encode order command response: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		UPDATE order_commands
		SET order_id = $2, response_status = $3, response_body = $4, completed_at = now()
		WHERE id = $1`, commandID, order.ID, statusCode, responseBody); err != nil {
		return OrderCommandResult{}, fmt.Errorf("complete order command: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return OrderCommandResult{}, fmt.Errorf("commit order command: %w", err)
	}
	return OrderCommandResult{Order: order, StatusCode: statusCode}, nil
}
