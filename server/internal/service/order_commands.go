package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/De3trvctions/Stallink/internal/data"
	"github.com/De3trvctions/Stallink/internal/logging"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

func (a *API) commandIdentity(w http.ResponseWriter, r *http.Request, account data.Account, operation string, payload any) (data.CommandIdentity, bool) {
	key := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if key == "" {
		if a.RequireIdempotency {
			writeErr(w, http.StatusBadRequest, "idempotency_key_required")
			return data.CommandIdentity{}, false
		}
		key = uuid.NewString()
	}
	parsed, err := uuid.Parse(key)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_idempotency_key")
		return data.CommandIdentity{}, false
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "invalid_command")
		return data.CommandIdentity{}, false
	}
	shopID := data.ID(0)
	if account.ShopID != nil {
		shopID = *account.ShopID
	}
	return data.CommandIdentity{
		ShopID: shopID, ActorID: account.ID, ActorRole: account.Role,
		Operation: operation, Key: parsed.String(), RequestHash: sha256.Sum256(encoded),
	}, true
}

func writeOrderCommand(w http.ResponseWriter, result data.OrderCommandResult) {
	if result.Replayed {
		w.Header().Set("Idempotency-Replayed", "true")
	}
	writeJSON(w, result.StatusCode, orderJSON(result.Order))
}

func (a *API) orderActionCommand(w http.ResponseWriter, r *http.Request, account data.Account, operation string, payload any, apply func(data.CommandIdentity) (data.OrderCommandResult, error)) {
	identity, ok := a.commandIdentity(w, r, account, operation, payload)
	if !ok {
		return
	}
	started := time.Now()
	result, err := apply(identity)
	logOrderCommandResult(r.Context(), identity, result, err, started)
	if err != nil {
		writeDataErr(w, r, err)
		return
	}
	writeOrderCommand(w, result)
}

func logOrderCommandResult(ctx context.Context, identity data.CommandIdentity, result data.OrderCommandResult, err error, started time.Time) {
	keyHash := sha256.Sum256([]byte(identity.Key))
	fields := []zap.Field{
		zap.String("shop_id", identity.ShopID.String()),
		zap.String("actor_id", identity.ActorID.String()),
		zap.String("operation", identity.Operation),
		zap.String("command_key_hash", hex.EncodeToString(keyHash[:8])),
		zap.Duration("duration", time.Since(started)),
	}
	if result.Order.ID != 0 {
		fields = append(fields, zap.String("order_id", result.Order.ID.String()), zap.Int64("state_version", result.Order.StateVersion))
	}
	if err != nil {
		fields = append(fields, zap.String("error_code", orderCommandErrorCode(err)))
		if orderCommandErrorCode(err) == "internal" {
			logging.ErrorContext(ctx, "order command failed", append(fields, zap.Error(err))...)
		} else {
			logging.WarnContext(ctx, "order command rejected", fields...)
		}
		return
	}
	fields = append(fields, zap.Bool("replayed", result.Replayed))
	logging.InfoContext(ctx, "order command completed", fields...)
}

func orderCommandErrorCode(err error) string {
	switch {
	case errors.Is(err, data.ErrNotFound):
		return "not_found"
	case errors.Is(err, data.ErrIdempotencyConflict):
		return "idempotency_conflict"
	case errors.Is(err, data.ErrConflict):
		return "conflict"
	case errors.Is(err, data.ErrCheckoutConflict):
		return "checkout_conflict"
	case errors.Is(err, data.ErrCommandInProgress):
		return "command_in_progress"
	case errors.Is(err, data.ErrCommandUnauthorized):
		return "forbidden"
	case errors.Is(err, data.ErrReservationExpired):
		return "reservation_expired"
	case errors.Is(err, data.ErrQuoteExpired):
		return "quote_expired"
	case errors.Is(err, data.ErrQuoteStale):
		return "quote_stale"
	case errors.Is(err, data.ErrQuoteInvalid):
		return "quote_invalid"
	case errors.Is(err, data.ErrStatus):
		return "invalid_status"
	case errors.Is(err, data.ErrNotListed):
		return "not_listed"
	case errors.Is(err, data.ErrEventNotOpen):
		return "event_not_open"
	case errors.Is(err, data.ErrPayCode):
		return "pay_code_required"
	case errors.Is(err, data.ErrPayCodeCurrencyMismatch):
		return "pay_code_currency_mismatch"
	default:
		return "internal"
	}
}

func canonicalOrderLines(lines []data.OrderLineIn) []data.OrderLineIn {
	quantities := make(map[data.ID]int, len(lines))
	for _, line := range lines {
		quantities[line.ListingID] += line.Qty
	}
	listingIDs := make([]data.ID, 0, len(quantities))
	for listingID := range quantities {
		listingIDs = append(listingIDs, listingID)
	}
	sort.Slice(listingIDs, func(i, j int) bool { return listingIDs[i] < listingIDs[j] })
	out := make([]data.OrderLineIn, 0, len(listingIDs))
	for _, listingID := range listingIDs {
		out = append(out, data.OrderLineIn{ListingID: listingID, Qty: quantities[listingID]})
	}
	return out
}
