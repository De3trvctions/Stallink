-- Goods need the same review revision bundles already have, so a platform
-- decision cannot approve content that changed after the page was loaded.
ALTER TABLE goods ADD COLUMN IF NOT EXISTS revision bigint NOT NULL DEFAULT 1 CHECK (revision > 0);
