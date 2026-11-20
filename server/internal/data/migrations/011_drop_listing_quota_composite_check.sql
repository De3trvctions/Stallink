-- Shared-pool stock: migration 010 relaxed the per-column quota guards but
-- missed the legacy table-level composite constraint that still bounded
-- sold_qty by listed_qty. Under the good-level shared pool, sold_qty is just
-- a per-listing sales ledger and must not be capped by the vestigial listed_qty
-- column. Drop that composite check; keep the non-negative column guards.
ALTER TABLE event_listings DROP CONSTRAINT IF EXISTS event_listings_check;
