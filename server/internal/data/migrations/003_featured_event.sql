-- 店铺主推活动：管理员可指定哪个活动的商品作为店铺首页主推。
ALTER TABLE shops
	ADD COLUMN IF NOT EXISTS featured_event_id bigint REFERENCES events (id) ON DELETE SET NULL;
