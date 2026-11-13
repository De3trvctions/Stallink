package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const reportTimezone = "Asia/Kuala_Lumpur"

type DailyReportCurrency struct {
	Currency      string `json:"currency"`
	Orders        int    `json:"orders"`
	PaidOrders    int    `json:"paidOrders"`
	RevenueMinor  int64  `json:"revenueMinor"`
	DiscountMinor int64  `json:"discountMinor"`
}

type DailyReportEvent struct {
	EventID       string `json:"eventId"`
	Name          string `json:"name"`
	Orders        int    `json:"orders"`
	RevenueMinor  int64  `json:"revenueMinor"`
	DiscountMinor int64  `json:"discountMinor"`
}

type DailyReportGood struct {
	Name         string `json:"name"`
	Quantity     int    `json:"quantity"`
	RevenueMinor int64  `json:"revenueMinor"`
}

type DailyReportPayCode struct {
	Name         string `json:"name"`
	Orders       int    `json:"orders"`
	RevenueMinor int64  `json:"revenueMinor"`
}

type DailyReport struct {
	Date          string                `json:"date"`
	GeneratedAt   string                `json:"generatedAt"`
	Source        string                `json:"source"`
	Currency      string                `json:"currency"`
	Orders        int                   `json:"orders"`
	PaidOrders    int                   `json:"paidOrders"`
	PendingOrders int                   `json:"pendingOrders"`
	ClosedOrders  int                   `json:"closedOrders"`
	RefundOrders  int                   `json:"refundOrders"`
	ItemsSold     int                   `json:"itemsSold"`
	RevenueMinor  int64                 `json:"revenueMinor"`
	DiscountMinor int64                 `json:"discountMinor"`
	AverageMinor  int64                 `json:"averageMinor"`
	Currencies    []DailyReportCurrency `json:"currencies"`
	Events        []DailyReportEvent    `json:"events"`
	TopGoods      []DailyReportGood     `json:"topGoods"`
	PayCodes      []DailyReportPayCode  `json:"payCodes"`
}

func (s *ShopStore) BuildDailyReport(ctx context.Context, shopID ID, date string, source string) (DailyReport, error) {
	if _, parseErr := time.ParseInLocation("2006-01-02", date, time.FixedZone("MYT", 8*60*60)); parseErr != nil {
		return DailyReport{}, ErrNotFound
	}
	report := DailyReport{
		Date: date, Source: source,
		GeneratedAt: time.Now().In(time.FixedZone("MYT", 8*60*60)).Format(time.RFC3339),
		Currencies:  []DailyReportCurrency{}, Events: []DailyReportEvent{},
		TopGoods: []DailyReportGood{}, PayCodes: []DailyReportPayCode{},
	}
	if err := s.pool.QueryRow(ctx, `
		SELECT currency FROM events WHERE shop_id = $1 ORDER BY starts_on DESC, id DESC LIMIT 1`,
		shopID).Scan(&report.Currency); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return DailyReport{}, fmt.Errorf("shop currency: %w", err)
	}
	if report.Currency == "" {
		report.Currency = "MYR"
	}

	err := s.pool.QueryRow(ctx, `
		SELECT count(*),
			count(*) FILTER (WHERE status IN ('paid', 'fulfilled')),
			count(*) FILTER (WHERE status = 'pending'),
			count(*) FILTER (WHERE status IN ('cancelled', 'expired')),
			count(*) FILTER (WHERE status = 'refund_required')
		FROM orders
		WHERE shop_id = $1 AND (created_at AT TIME ZONE $2)::date = $3::date`,
		shopID, reportTimezone, date).
		Scan(&report.Orders, &report.PaidOrders, &report.PendingOrders, &report.ClosedOrders, &report.RefundOrders)
	if err != nil {
		return DailyReport{}, fmt.Errorf("daily totals: %w", err)
	}

	// line_discount is list price (unit_price_minor * qty) minus the charged line
	// total. applyHeadlineCurrency then keeps only the highest-revenue currency
	// on the headline totals; Currencies still lists every currency.
	rows, err := s.pool.Query(ctx, `
		SELECT o.currency, count(*),
			count(*) FILTER (WHERE o.status IN ('paid', 'fulfilled')),
			COALESCE(sum(o.total_minor) FILTER (WHERE o.status IN ('paid', 'fulfilled')), 0),
			COALESCE(sum(d.line_discount) FILTER (WHERE o.status IN ('paid', 'fulfilled')), 0)
		FROM orders o
		LEFT JOIN LATERAL (
			SELECT COALESCE(sum(l.unit_price_minor * l.qty - l.line_total_minor), 0) AS line_discount
			FROM order_lines l WHERE l.order_id = o.id
		) d ON true
		WHERE o.shop_id = $1 AND (o.created_at AT TIME ZONE $2)::date = $3::date
		GROUP BY o.currency
		ORDER BY 4 DESC, 3 DESC, 1`, shopID, reportTimezone, date)
	if err != nil {
		return DailyReport{}, fmt.Errorf("daily currencies: %w", err)
	}
	for rows.Next() {
		var currency DailyReportCurrency
		if err := rows.Scan(&currency.Currency, &currency.Orders, &currency.PaidOrders, &currency.RevenueMinor, &currency.DiscountMinor); err != nil {
			rows.Close()
			return DailyReport{}, fmt.Errorf("scan daily currency: %w", err)
		}
		report.Currencies = append(report.Currencies, currency)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return DailyReport{}, fmt.Errorf("daily currencies: %w", err)
	}
	rows.Close()
	applyHeadlineCurrency(&report)

	if err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(sum(l.qty), 0)
		FROM order_lines l
		JOIN orders o ON o.id = l.order_id AND o.shop_id = $1
		WHERE (o.created_at AT TIME ZONE $2)::date = $3::date AND o.status IN ('paid', 'fulfilled')`,
		shopID, reportTimezone, date).Scan(&report.ItemsSold); err != nil {
		return DailyReport{}, fmt.Errorf("daily items: %w", err)
	}

	rows, err = s.pool.Query(ctx, `
		SELECT o.event_id::text, e.name, count(*),
			COALESCE(sum(o.total_minor) FILTER (WHERE o.status IN ('paid', 'fulfilled')), 0),
			COALESCE(sum(d.line_discount) FILTER (WHERE o.status IN ('paid', 'fulfilled')), 0)
		FROM orders o
		JOIN events e ON e.id = o.event_id AND e.shop_id = o.shop_id
		LEFT JOIN LATERAL (
			SELECT COALESCE(sum(l.unit_price_minor * l.qty - l.line_total_minor), 0) AS line_discount
			FROM order_lines l WHERE l.order_id = o.id
		) d ON true
		WHERE o.shop_id = $1 AND (o.created_at AT TIME ZONE $2)::date = $3::date
		GROUP BY o.event_id, e.name
		ORDER BY 4 DESC, 3 DESC, e.name`, shopID, reportTimezone, date)
	if err != nil {
		return DailyReport{}, fmt.Errorf("daily events: %w", err)
	}
	for rows.Next() {
		var event DailyReportEvent
		if err := rows.Scan(&event.EventID, &event.Name, &event.Orders, &event.RevenueMinor, &event.DiscountMinor); err != nil {
			rows.Close()
			return DailyReport{}, fmt.Errorf("scan daily event: %w", err)
		}
		report.Events = append(report.Events, event)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return DailyReport{}, fmt.Errorf("daily events: %w", err)
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT l.good_name, sum(l.qty), COALESCE(sum(l.line_total_minor), 0)
		FROM order_lines l
		JOIN orders o ON o.id = l.order_id AND o.shop_id = $1
		WHERE (o.created_at AT TIME ZONE $2)::date = $3::date
			AND o.status IN ('paid', 'fulfilled') AND o.currency = $4
		GROUP BY l.good_name
		ORDER BY 3 DESC, 2 DESC, l.good_name
		LIMIT 5`, shopID, reportTimezone, date, report.Currency)
	if err != nil {
		return DailyReport{}, fmt.Errorf("daily top goods: %w", err)
	}
	for rows.Next() {
		var good DailyReportGood
		if err := rows.Scan(&good.Name, &good.Quantity, &good.RevenueMinor); err != nil {
			rows.Close()
			return DailyReport{}, fmt.Errorf("scan daily good: %w", err)
		}
		report.TopGoods = append(report.TopGoods, good)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return DailyReport{}, fmt.Errorf("daily top goods: %w", err)
	}
	rows.Close()

	rows, err = s.pool.Query(ctx, `
		SELECT COALESCE(NULLIF(pay_code_name, ''), '未记录'), count(*),
			COALESCE(sum(total_minor), 0)
		FROM orders
		WHERE shop_id = $1 AND (created_at AT TIME ZONE $2)::date = $3::date
			AND status IN ('paid', 'fulfilled') AND currency = $4
		GROUP BY 1
		ORDER BY 3 DESC, 2 DESC`, shopID, reportTimezone, date, report.Currency)
	if err != nil {
		return DailyReport{}, fmt.Errorf("daily pay codes: %w", err)
	}
	for rows.Next() {
		var code DailyReportPayCode
		if err := rows.Scan(&code.Name, &code.Orders, &code.RevenueMinor); err != nil {
			rows.Close()
			return DailyReport{}, fmt.Errorf("scan daily pay code: %w", err)
		}
		report.PayCodes = append(report.PayCodes, code)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return DailyReport{}, fmt.Errorf("daily pay codes: %w", err)
	}
	rows.Close()

	return report, nil
}

// applyHeadlineCurrency copies the highest-revenue currency onto the report
// headline. Average uses that currency's paid orders, not the all-currency count.
// With no currency rows, average uses the paid count and revenue already on the report.
func applyHeadlineCurrency(report *DailyReport) {
	paid := report.PaidOrders
	revenue := report.RevenueMinor
	if len(report.Currencies) > 0 && report.Currencies[0].Currency != "" {
		top := report.Currencies[0]
		report.Currency = top.Currency
		report.RevenueMinor = top.RevenueMinor
		report.DiscountMinor = top.DiscountMinor
		paid = top.PaidOrders
		revenue = top.RevenueMinor
	}
	if paid > 0 {
		report.AverageMinor = revenue / int64(paid)
	}
}

// keepManualDailyReport reports whether a manual snapshot survives a scheduled
// run. dayEnd is the exclusive end of the report date. A preview taken before
// that instant is replaced. A manual report taken at or after that instant stays.
func keepManualDailyReport(generatedAt, dayEnd time.Time) bool {
	return !generatedAt.Before(dayEnd)
}

func reportDayEnd(date string) (time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", date, time.FixedZone("MYT", 8*60*60))
	if err != nil {
		return time.Time{}, err
	}
	return day.AddDate(0, 0, 1), nil
}

// SaveScheduledDailyReport writes one close-of-day snapshot.
// A missing row is inserted and a scheduled row is refreshed. A manual preview
// generated before the business day ended is replaced. A manual report generated
// at or after that midnight is left unchanged. saved is false when that manual
// report is kept.
func (s *ShopStore) SaveScheduledDailyReport(ctx context.Context, shopID ID, report DailyReport) (bool, error) {
	dayEnd, err := reportDayEnd(report.Date)
	if err != nil {
		return false, fmt.Errorf("scheduled report date: %w", err)
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return false, fmt.Errorf("marshal daily report: %w", err)
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin scheduled daily report: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var source string
	var generatedAt time.Time
	err = tx.QueryRow(ctx, `
		SELECT source, generated_at FROM daily_reports
		WHERE shop_id = $1 AND report_date = $2::date
		FOR UPDATE`, shopID, report.Date).Scan(&source, &generatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO daily_reports(shop_id, report_date, payload, source, generated_at)
			VALUES ($1, $2::date, $3::jsonb, $4, now())`,
			shopID, report.Date, payload, report.Source); err != nil {
			return false, fmt.Errorf("insert scheduled daily report: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit scheduled daily report: %w", err)
		}
		return true, nil
	}
	if err != nil {
		return false, fmt.Errorf("read daily report: %w", err)
	}
	if source == "manual" && keepManualDailyReport(generatedAt, dayEnd) {
		if err := tx.Commit(ctx); err != nil {
			return false, fmt.Errorf("commit kept daily report: %w", err)
		}
		return false, nil
	}
	if _, err := tx.Exec(ctx, `
		UPDATE daily_reports
		SET payload = $3::jsonb, source = $4, generated_at = now()
		WHERE shop_id = $1 AND report_date = $2::date`,
		shopID, report.Date, payload, report.Source); err != nil {
		return false, fmt.Errorf("update scheduled daily report: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, fmt.Errorf("commit scheduled daily report: %w", err)
	}
	return true, nil
}

func (s *ShopStore) UpsertDailyReport(ctx context.Context, shopID ID, report DailyReport) error {
	payload, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshal daily report: %w", err)
	}
	_, err = s.pool.Exec(ctx, `
		INSERT INTO daily_reports(shop_id, report_date, payload, source, generated_at)
		VALUES ($1, $2::date, $3::jsonb, $4, now())
		ON CONFLICT (shop_id, report_date) DO UPDATE
		SET payload = EXCLUDED.payload, source = EXCLUDED.source, generated_at = now()`,
		shopID, report.Date, payload, report.Source)
	if err != nil {
		return fmt.Errorf("save daily report: %w", err)
	}
	return nil
}

func (s *ShopStore) GetDailyReport(ctx context.Context, shopID ID, date string) (json.RawMessage, error) {
	var payload []byte
	err := s.pool.QueryRow(ctx, `
		SELECT payload FROM daily_reports WHERE shop_id = $1 AND report_date = $2::date`,
		shopID, date).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("read daily report: %w", err)
	}
	return json.RawMessage(payload), nil
}
