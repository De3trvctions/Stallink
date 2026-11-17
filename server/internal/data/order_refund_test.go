package data

import (
	"errors"
	"testing"
	"time"
)

func TestRefundConfirmDecision(t *testing.T) {
	confirmed := time.Now()
	tests := []struct {
		name         string
		status       string
		refundedAt   *time.Time
		wantDecision string
		wantErr      error
	}{
		{name: "already confirmed is idempotent", status: "cancelled", refundedAt: &confirmed, wantDecision: "idempotent"},
		{name: "waiting refund proceeds", status: "refund_required", wantDecision: "proceed"},
		{name: "plain cancelled without refund is not refundable", status: "cancelled", wantErr: ErrStatus},
		{name: "paid cannot be refund-confirmed", status: "paid", wantErr: ErrStatus},
		{name: "pending cannot be refund-confirmed", status: "pending", wantErr: ErrStatus},
		{name: "fulfilled cannot be refund-confirmed", status: "fulfilled", wantErr: ErrStatus},
		{name: "expired cannot be refund-confirmed", status: "expired", wantErr: ErrStatus},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision, err := refundConfirmDecision(tt.status, tt.refundedAt)
			if tt.wantErr != nil {
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err = %v", err)
			}
			if decision != tt.wantDecision {
				t.Fatalf("decision = %q, want %q", decision, tt.wantDecision)
			}
		})
	}
}

func TestReviewResetStatus(t *testing.T) {
	if got := reviewResetStatus(true); got != "approved" {
		t.Fatalf("auto approve on = %q, want approved", got)
	}
	if got := reviewResetStatus(false); got != "pending" {
		t.Fatalf("auto approve off = %q, want pending", got)
	}
}
