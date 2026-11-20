-- Shared-pool stock: sold_qty is a per-listing sales ledger and reserved_qty
-- is held on the good-level shared pool. They are no longer bounded by a
-- per-event quota (listed_qty). Drop both old quota checks on existing
-- databases and keep only the non-negative guards.
ALTER TABLE event_listings DROP CONSTRAINT IF EXISTS event_listings_sold_qty_check;
ALTER TABLE event_listings DROP CONSTRAINT IF EXISTS event_listings_sold_reserved_check;
ALTER TABLE event_listings ADD CONSTRAINT event_listings_sold_qty_check CHECK (sold_qty >= 0);
