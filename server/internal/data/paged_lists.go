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
 o.event_id::text, e.name, o.total_minor, o.state_version, o.created_at, o.reservation_expires_at, o.refunded_at,
 o.cash_received_minor, o.cash_change_minor
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
		if err := rows.Scan(&o.ID, &o.Number, &o.Status, &o.BoothName, &o.Currency, &o.PayCodeName, &o.OpenedByLogin, &o.OpenedByRole, &o.PaidByLogin, &o.PaidByRole, &o.EventID, &o.EventName, &o.TotalMinor, &o.StateVersion, &o.CreatedAt, &o.ReservationExpiresAt, &o.RefundedAt, &o.CashReceivedMinor, &o.CashChangeMinor); err != nil {
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

func (s *ShopStore) ListAuditPage(ctx context.Context, shopID ID, search string, limit, offset int) ([]AuditEntry, bool, error) {
	rows, err := s.pool.Query(ctx, `SELECT a.action,u.login_name,a.subject_type,a.subject_id::text,to_char(a.created_at,'YYYY-MM-DD HH24:MI')
 FROM audit_logs a JOIN users u ON u.id=a.actor_id
 WHERE a.shop_id=$1 AND ($2='' OR strpos(lower(a.action),lower($2))>0 OR strpos(lower(u.login_name),lower($2))>0 OR strpos(lower(a.subject_type),lower($2))>0 OR strpos(a.subject_id::text,lower($2))>0 OR strpos(
 CASE a.action WHEN 'bundle.approve' THEN '审核通过套餐' WHEN 'bundle.create' THEN '创建套餐' WHEN 'bundle.update' THEN '编辑套餐' WHEN 'bundle.reject' THEN '拒绝套餐' WHEN 'good.approve' THEN '审核通过商品' WHEN 'good.reject' THEN '拒绝商品' WHEN 'good.image.add' THEN '添加商品图片' WHEN 'good.image.delete' THEN '删除商品图片' WHEN 'good.image.reorder' THEN '调整商品图片顺序' WHEN 'listing.create' THEN '上架商品' WHEN 'listing.update' THEN '修改商品上架信息' WHEN 'stock.update' THEN '调整商品库存' WHEN 'order.create' THEN '创建订单' WHEN 'order.pay' THEN '确认收款' WHEN 'order.cancel' THEN '取消订单' WHEN 'order.refund_required' THEN '收款后取消待退款' WHEN 'order.refund_confirm' THEN '确认退款' WHEN 'order.fulfill' THEN '完成交付' WHEN 'order.expire' THEN '订单预留过期' ELSE '' END, $2)>0)
 ORDER BY a.created_at DESC,a.id DESC LIMIT $3 OFFSET $4`, shopID, search, limit+1, offset)
	if err != nil {
		return nil, false, fmt.Errorf("list audit page: %w", err)
	}
	defer rows.Close()
	out := make([]AuditEntry, 0, limit+1)
	for rows.Next() {
		var e AuditEntry
		if err := rows.Scan(&e.Action, &e.ActorLogin, &e.SubjectType, &e.SubjectID, &e.CreatedAt); err != nil {
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
