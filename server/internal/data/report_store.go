package data

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DefaultReportTimezone is the zone used when the operator configured none.
const DefaultReportTimezone = "Asia/Kuala_Lumpur"

// NormalizeReportTimezone returns a zone name PostgreSQL accepts in
// AT TIME ZONE. An empty or unknown name would make every dated query fail with
// SQLSTATE 22023, so callers must resolve it before building SQL parameters.
func NormalizeReportTimezone(timezone string) string {
	if timezone != "" {
		if _, err := time.LoadLocation(timezone); err == nil {
			return timezone
		}
	}
	return DefaultReportTimezone
}

// ReportLocation resolves the configured report timezone; an empty or unknown
// zone falls back to the Kuala Lumpur fixed offset used historically.
func ReportLocation(timezone string) *time.Location {
	if timezone != "" {
		if loc, err := time.LoadLocation(timezone); err == nil {
			return loc
		}
	}
	return time.FixedZone("MYT", 8*60*60)
}

type DailyReportCurrency struct {
	Currency      string `json:"currency"`
	Orders        int    `json:"orders"`
	PaidOrders    int    `json:"paidOrders"`
	RevenueMinor  int64  `json:"revenueMinor"`
	DiscountMinor int64  `json:"discountMinor"`
	// RefundMinor is collected money still awaiting refund confirmation
	// (status = 'refund_required'). These orders are excluded from RevenueMinor;
	// once the refund is confirmed the order moves to cancelled and leaves this
	// figure. It lets the shop reconcile the cash drawer against pending refunds.
	RefundMinor int64 `json:"refundMinor"`
	// CashOverShortMinor is cash received minus order total for cash-tendered
	// orders only (orders with cash_received_minor set). Positive is over,
	// negative is short. Non-cash orders do not participate.
	CashOverShortMinor int64 `json:"cashOverShortMinor"`
}

type DailyReportEvent struct {
	EventID       string `json:"eventId"`
	Name          string `json:"name"`
	Currency      string `json:"currency"`
	Orders        int    `json:"orders"`
	RevenueMinor  int64  `json:"revenueMinor"`
	DiscountMinor int64  `json:"discountMinor"`
}

type DailyReportGood struct {
	Currency     string `json:"currency"`
	Name         string `json:"name"`
	Quantity     int    `json:"quantity"`
	RevenueMinor int64  `json:"revenueMinor"`
}

type DailyReportPayCode struct {
	Currency     string `json:"currency"`
	Name         string `json:"name"`
	Orders       int    `json:"orders"`
	RevenueMinor int64  `json:"revenueMinor"`
}

type DailyReport struct {
	Date               string                `json:"date"`
	GeneratedAt        string                `json:"generatedAt"`
	Source             string                `json:"source"`
	Currency           string                `json:"currency"`
	Orders             int                   `json:"orders"`
	PaidOrders         int                   `json:"paidOrders"`
	PendingOrders      int                   `json:"pendingOrders"`
	ClosedOrders       int                   `json:"closedOrders"`
	RefundOrders       int                   `json:"refundOrders"`
	ItemsSold          int                   `json:"itemsSold"`
	RevenueMinor       int64                 `json:"revenueMinor"`
	DiscountMinor      int64                 `json:"discountMinor"`
	RefundMinor        int64                 `json:"refundMinor"`
	CashOverShortMinor int64                 `json:"cashOverShortMinor"`
	AverageMinor       int64                 `json:"averageMinor"`
	Currencies         []DailyReportCurrency `json:"currencies"`
	Events             []DailyReportEvent    `json:"events"`
	TopGoods           []DailyReportGood     `json:"topGoods"`
	PayCodes           []DailyReportPayCode  `json:"payCodes"`
}

func beginDailyReportSnapshot(ctx context.Context, pool *pgxpool.Pool) (pgx.Tx, error) {
	return pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
}

func (s *ShopStore) BuildDailyReport(ctx context.Context, shopID ID, date, source, timezone string) (DailyReport, error) {
	// Every dated query below passes timezone straight into AT TIME ZONE, which
	// rejects an empty name, so resolve it once here.
	timezone = NormalizeReportTimezone(timezone)
	location := ReportLocation(timezone)
	if _, parseErr := time.ParseInLocation("2006-01-02", date, location); parseErr != nil {
		return DailyReport{}, ErrNotFound
	}
	report := DailyReport{
		Date: date, Source: source,
		GeneratedAt: time.Now().In(location).Format(time.RFC3339),
		Currencies:  []DailyReportCurrency{}, Events: []DailyReportEvent{},
		TopGoods: []DailyReportGood{}, PayCodes: []DailyReportPayCode{},
	}
	tx, err := beginDailyReportSnapshot(ctx, s.pool)
	if err != nil {
		return DailyReport{}, fmt.Errorf("begin daily report snapshot: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err := tx.QueryRow(ctx, `
		SELECT currency FROM events WHERE shop_id = $1 ORDER BY starts_on DESC, id DESC LIMIT 1`,
		shopID).Scan(&report.Currency); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return DailyReport{}, fmt.Errorf("shop currency: %w", err)
	}
	if report.Currency == "" {
		report.Currency = "MYR"
	}

	// line_discount is list price (unit_price_minor * qty) minus the charged line
	// total. applyHeadlineCurrency chooses the highest-revenue currency for the
	// headline; Currencies still lists every currency.
	rows, err := tx.Query(ctx, `
		SELECT o.currency, count(*),
			count(*) FILTER (WHERE o.status IN ('paid', 'fulfilled')),
			COALESCE(sum(o.total_minor) FILTER (WHERE o.status IN ('paid', 'fulfilled')), 0),
			COALESCE(sum(d.line_discount) FILTER (WHERE o.status IN ('paid', 'fulfilled')), 0),
			COALESCE(sum(o.total_minor) FILTER (WHERE o.status = 'refund_required'), 0),
			COALESCE(sum(o.cash_received_minor) FILTER (WHERE o.cash_received_minor IS NOT NULL), 0)
				- COALESCE(sum(o.total_minor) FILTER (WHERE o.cash_received_minor IS NOT NULL), 0)
		FROM orders o
		LEFT JOIN LATERAL (
			SELECT COALESCE(sum(l.unit_price_minor * l.qty - l.line_total_minor), 0) AS line_discount
			FROM order_lines l WHERE l.order_id = o.id
		) d ON true
		WHERE o.shop_id = $1 AND (o.created_at AT TIME ZONE $2)::date = $3::date
		GROUP BY o.currency
		ORDER BY 4 DESC, 3 DESC, 1`, shopID, timezone, date)
	if err != nil {
		return DailyReport{}, fmt.Errorf("daily currencies: %w", err)
	}
	for rows.Next() {
		var currency DailyReportCurrency
		if err := rows.Scan(&currency.Currency, &currency.Orders, &currency.PaidOrders, &currency.RevenueMinor, &currency.DiscountMinor, &currency.RefundMinor, &currency.CashOverShortMinor); err != nil {
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

	err = tx.QueryRow(ctx, `
		SELECT count(*),
			count(*) FILTER (WHERE status IN ('paid', 'fulfilled')),
			count(*) FILTER (WHERE status = 'pending'),
			count(*) FILTER (WHERE status IN ('cancelled', 'expired')),
			count(*) FILTER (WHERE status = 'refund_required'),
			COALESCE(sum(total_minor) FILTER (WHERE status = 'refund_required'), 0),
			COALESCE(sum(cash_received_minor) FILTER (WHERE cash_received_minor IS NOT NULL), 0)
				- COALESCE(sum(total_minor) FILTER (WHERE cash_received_minor IS NOT NULL), 0)
		FROM orders
		WHERE shop_id = $1 AND currency = $4 AND (created_at AT TIME ZONE $2)::date = $3::date`,
		shopID, timezone, date, report.Currency).
		Scan(&report.Orders, &report.PaidOrders, &report.PendingOrders, &report.ClosedOrders, &report.RefundOrders, &report.RefundMinor, &report.CashOverShortMinor)
	if err != nil {
		return DailyReport{}, fmt.Errorf("daily totals: %w", err)
	}

	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(sum(l.qty), 0)
		FROM order_lines l
		JOIN orders o ON o.id = l.order_id AND o.shop_id = $1
		WHERE (o.created_at AT TIME ZONE $2)::date = $3::date
			AND o.status IN ('paid', 'fulfilled') AND o.currency = $4`,
		shopID, timezone, date, report.Currency).Scan(&report.ItemsSold); err != nil {
		return DailyReport{}, fmt.Errorf("daily items: %w", err)
	}

	rows, err = tx.Query(ctx, `
		SELECT o.event_id::text, e.name, o.currency, count(*),
			COALESCE(sum(o.total_minor) FILTER (WHERE o.status IN ('paid', 'fulfilled')), 0),
			COALESCE(sum(d.line_discount) FILTER (WHERE o.status IN ('paid', 'fulfilled')), 0)
		FROM orders o
		JOIN events e ON e.id = o.event_id AND e.shop_id = o.shop_id
		LEFT JOIN LATERAL (
			SELECT COALESCE(sum(l.unit_price_minor * l.qty - l.line_total_minor), 0) AS line_discount
			FROM order_lines l WHERE l.order_id = o.id
		) d ON true
		WHERE o.shop_id = $1 AND (o.created_at AT TIME ZONE $2)::date = $3::date
		GROUP BY o.event_id, e.name, o.currency
		ORDER BY 5 DESC, 4 DESC, e.name, o.currency`, shopID, timezone, date)
	if err != nil {
		return DailyReport{}, fmt.Errorf("daily events: %w", err)
	}
	for rows.Next() {
		var event DailyReportEvent
		if err := rows.Scan(&event.EventID, &event.Name, &event.Currency, &event.Orders, &event.RevenueMinor, &event.DiscountMinor); err != nil {
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

	rows, err = tx.Query(ctx, `
		SELECT currency, good_id, good_name, qty, revenue FROM (
			SELECT o.currency, l.good_id::text AS good_id, l.good_name,
				sum(l.qty) AS qty, COALESCE(sum(l.line_total_minor), 0) AS revenue,
				ROW_NUMBER() OVER (PARTITION BY o.currency
					ORDER BY COALESCE(sum(l.line_total_minor), 0) DESC, sum(l.qty) DESC, l.good_name) AS rn
			FROM order_lines l
			JOIN orders o ON o.id = l.order_id AND o.shop_id = $1
			WHERE (o.created_at AT TIME ZONE $2)::date = $3::date
				AND o.status IN ('paid', 'fulfilled')
			GROUP BY o.currency, l.good_id, l.good_name
		) ranked
		WHERE rn <= 5
		ORDER BY currency, revenue DESC, qty DESC, good_name`, shopID, timezone, date)
	if err != nil {
		return DailyReport{}, fmt.Errorf("daily top goods: %w", err)
	}
	for rows.Next() {
		var good DailyReportGood
		var goodID string
		if err := rows.Scan(&good.Currency, &goodID, &good.Name, &good.Quantity, &good.RevenueMinor); err != nil {
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

	rows, err = tx.Query(ctx, `
		SELECT currency, COALESCE(NULLIF(pay_code_name, ''), '未记录'), count(*),
			COALESCE(sum(total_minor), 0)
		FROM orders
		WHERE shop_id = $1 AND (created_at AT TIME ZONE $2)::date = $3::date
			AND status IN ('paid', 'fulfilled')
		GROUP BY currency, 2
		ORDER BY currency, 4 DESC, 3 DESC, 2`, shopID, timezone, date)
	if err != nil {
		return DailyReport{}, fmt.Errorf("daily pay codes: %w", err)
	}
	for rows.Next() {
		var code DailyReportPayCode
		if err := rows.Scan(&code.Currency, &code.Name, &code.Orders, &code.RevenueMinor); err != nil {
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

	if err := tx.Commit(ctx); err != nil {
		return DailyReport{}, fmt.Errorf("commit daily report snapshot: %w", err)
	}
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
		report.Orders = top.Orders
		report.PaidOrders = top.PaidOrders
		report.RevenueMinor = top.RevenueMinor
		report.DiscountMinor = top.DiscountMinor
		report.RefundMinor = top.RefundMinor
		report.CashOverShortMinor = top.CashOverShortMinor
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

func reportDayEnd(date, timezone string) (time.Time, error) {
	day, err := time.ParseInLocation("2006-01-02", date, ReportLocation(timezone))
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
func (s *ShopStore) SaveScheduledDailyReport(ctx context.Context, shopID ID, report DailyReport, timezone string) (bool, error) {
	dayEnd, err := reportDayEnd(report.Date, timezone)
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
