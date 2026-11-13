CREATE TABLE IF NOT EXISTS daily_reports (
	shop_id bigint NOT NULL REFERENCES shops(id) ON DELETE CASCADE,
	report_date date NOT NULL,
	payload jsonb NOT NULL,
	source text NOT NULL CHECK (source IN ('manual', 'scheduled')),
	generated_at timestamptz NOT NULL DEFAULT now(),
	PRIMARY KEY (shop_id, report_date)
);
CREATE INDEX IF NOT EXISTS daily_reports_shop_date_idx ON daily_reports(shop_id, report_date DESC);
