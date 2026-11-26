package data

import (
	"context"
	"fmt"
)

// ListOrderPage returns only summary fields for one bounded page. timezone is
// the IANA report zone used to slice order dates; callers pass a resolved zone.
func (s *ShopStore) ListOrderPage(ctx context.Context, shopID ID, eventID, date, search, status, timezone string, limit, offset int) ([]Order, bool, error) {
	timezone = NormalizeReportTimezone(timezone)
	rows, err := s.pool.Query(ctx, `SELECT o.id::text, o.number, o.status, o.booth_name, o.currency, o.pay_code_name,
 opener.login_name, o.opened_by_role, COALESCE(payer.login_name, ''), COALESCE(payer.role, ''),
 o.event_id::text, e.name, o.total_minor, o.state_version, o.created_at, o.reservation_expires_at, o.refunded_at
 FROM orders o JOIN users opener ON opener.id=o.opened_by LEFT JOIN users payer ON payer.id=o.paid_by JOIN events e ON e.id=o.event_id
 WHERE o.shop_id=$1 AND ($2='' OR o.event_id::text=$2)
 AND ($4='' OR (o.created_at AT TIME ZONE $7)::date = $4::date)
 AND ($8='' OR o.status=$8)
 AND ($3='' OR strpos(lower(o.number),lower($3))>0 OR strpos(lower(e.name),lower($3))>0 OR strpos(lower(opener.login_name),lower($3))>0 OR strpos(lower(o.booth_name),lower($3))>0)
 ORDER BY o.created_at DESC, o.id DESC LIMIT $5 OFFSET $6`, shopID, eventID, search, date, limit+1, offset, timezone, status)
	if err != nil {
		return nil, false, fmt.Errorf("list order page: %w", err)
	}
	defer rows.Close()
	out := make([]Order, 0, limit+1)
	for rows.Next() {
		var o Order
		if err := rows.Scan(&o.ID, &o.Number, &o.Status, &o.BoothName, &o.Currency, &o.PayCodeName, &o.OpenedByLogin, &o.OpenedByRole, &o.PaidByLogin, &o.PaidByRole, &o.EventID, &o.EventName, &o.TotalMinor, &o.StateVersion, &o.CreatedAt, &o.ReservationExpiresAt, &o.RefundedAt); err != nil {
			return nil, false, fmt.Errorf("scan order page: %w", err)
		}
		out = append(out, o)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("read order page: %w", err)
	}
	rows.Close()
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	for i := range out {
		out[i].Lines, err = s.orderLines(ctx, out[i].ID)
		if err != nil {
			return nil, false, err
		}
	}
	return out, more, nil
}

// auditSubjectSnapshotLabel is the slot for the write-time snapshot
// detail->'subject'->>'label' (K-Q5 = C, read path prefers the snapshot).
// audit_logs.detail only arrives with the review-fix plan's migration 014;
// until then the slot is NULL and every row falls back to the live name.
// S3 phase 2 replaces it with `a.detail->'subject'->>'label'`.
const auditSubjectSnapshotLabel = `NULL::text`

// auditSubjectLabelSQL resolves a display name for each audit subject from
// rows of the same shop: order number, good name, bundle name, or
// "good · event" for a listing. Unknown types and missing rows yield an empty string.
const auditSubjectLabelSQL = `COALESCE(` + auditSubjectSnapshotLabel + `, CASE a.subject_type
      WHEN 'order'   THEN (SELECT o.number FROM orders o WHERE o.id = a.subject_id AND o.shop_id = a.shop_id)
      WHEN 'good'    THEN (SELECT g.name FROM goods g WHERE g.id = a.subject_id AND g.shop_id = a.shop_id)
      WHEN 'bundle'  THEN (SELECT b.name FROM bundles b WHERE b.id = a.subject_id AND b.shop_id = a.shop_id)
      WHEN 'listing' THEN (SELECT g.name || ' · ' || e.name FROM event_listings el
                             JOIN goods g ON g.id = el.good_id AND g.shop_id = el.shop_id
                             JOIN events e ON e.id = el.event_id AND e.shop_id = el.shop_id
                            WHERE el.id = a.subject_id AND el.shop_id = a.shop_id)
    END, '')`

func (s *ShopStore) ListAuditPage(ctx context.Context, shopID ID, search string, limit, offset int) ([]AuditEntry, bool, error) {
	rows, err := s.pool.Query(ctx, `
WITH labelled AS (
  SELECT a.id, a.action, u.login_name, a.subject_type, a.subject_id, a.created_at,
    `+auditSubjectLabelSQL+` AS label
  FROM audit_logs a JOIN users u ON u.id = a.actor_id
  WHERE a.shop_id = $1
)
SELECT action, login_name, subject_type, subject_id::text, label, created_at
FROM labelled
WHERE ($2 = '' OR strpos(lower(action), lower($2)) > 0 OR strpos(lower(login_name), lower($2)) > 0
   OR strpos(lower(subject_type), lower($2)) > 0 OR strpos(subject_id::text, lower($2)) > 0
   OR strpos(lower(label), lower($2)) > 0
   OR strpos(CASE action
        WHEN 'bundle.approve' THEN '审核通过套餐' WHEN 'bundle.create' THEN '创建套餐' WHEN 'bundle.update' THEN '编辑套餐'
        WHEN 'bundle.reject' THEN '拒绝套餐' WHEN 'good.approve' THEN '审核通过商品' WHEN 'good.reject' THEN '拒绝商品'
        WHEN 'good.image.add' THEN '添加商品图片' WHEN 'good.image.delete' THEN '删除商品图片'
        WHEN 'good.image.reorder' THEN '调整商品图片顺序' WHEN 'listing.create' THEN '上架商品'
        WHEN 'listing.update' THEN '修改商品上架信息' WHEN 'stock.update' THEN '调整商品库存'
        WHEN 'order.create' THEN '创建订单' WHEN 'order.pay' THEN '确认收款' WHEN 'order.cancel' THEN '取消订单'
        WHEN 'order.refund_required' THEN '收款后取消待退款' WHEN 'order.refund_confirm' THEN '确认退款'
        WHEN 'order.fulfill' THEN '完成交付' WHEN 'order.expire' THEN '订单预留过期' ELSE '' END, $2) > 0)
ORDER BY created_at DESC, id DESC LIMIT $3 OFFSET $4`, shopID, search, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list audit page: %w", err)
	}
	defer rows.Close()
	out := make([]AuditEntry, 0, limit+1)
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.Action, &e.ActorLogin, &e.SubjectType, &e.SubjectID, &e.SubjectLabel, &e.CreatedAt); err != nil {
			return nil, false, fmt.Errorf("scan audit page: %w", err)
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, false, fmt.Errorf("read audit page: %w", err)
	}
	more := len(out) > limit
	if more {
		out = out[:limit]
	}
	return out, more, nil
}
