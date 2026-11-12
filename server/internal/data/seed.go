package data

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/De3trvctions/Stallink/internal/auth"
	"github.com/De3trvctions/Stallink/internal/catalog"
)

func EnsurePlatformAdmin(ctx context.Context, store *ShopStore, login, password string) error {
	exists, err := store.HasPlatformAdmin(ctx)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return err
	}
	_, err = store.CreateAccount(ctx, login, hash, "platform_admin", nil, "", "")
	return err
}

// SeedLocalDemo creates a deterministic, two-store local fixture in an empty
// database. It never runs as part of normal server startup.
func SeedLocalDemo(ctx context.Context, store *ShopStore, password string) error {
	if len(password) < 8 {
		return fmt.Errorf("local seed password must contain at least 8 characters")
	}
	type seedGood struct {
		name, description string
		stock, listed     int
		price             int64
		tiers             []catalog.PriceTier
	}
	type seedShop struct {
		slug, name, eventName, starts, ends, venue, booth string
		goods                                             []seedGood
		bundleName                                        string
		bundleGoods                                       []int
		bundlePrice                                       int64
	}
	shops := []seedShop{
		{
			slug: "alpha", name: "甲店", eventName: "Comic Fiesta 2026", starts: "2026-12-19", ends: "2026-12-20", venue: "KLCC", booth: "A-12",
			goods: []seedGood{
				{name: "闪膜明信片", description: "一套四张，覆闪膜。", stock: 30, listed: 12, price: 1990},
				{name: "亚克力立牌", description: "本场限定立牌。", stock: 20, listed: 8, price: 3900},
				{name: "马口铁徽章", description: "三枚一组。", stock: 40, listed: 20, price: 1500},
				{name: "拍立得小卡", description: "随机一张。", stock: 16, listed: 10, price: 2500},
			},
			bundleName: "会场双件套", bundleGoods: []int{1, 2}, bundlePrice: 4900,
		},
		{
			slug: "beta", name: "Stallink Beta", eventName: "Kuala Lumpur Art Market", starts: "2026-12-05", ends: "2026-12-06", venue: "MITEC", booth: "B-06",
			goods: []seedGood{
				{name: "纪念徽章六件套", description: "单枚 RM40；集齐 6 枚套装价 RM150。", stock: 36, listed: 36, price: 4000, tiers: []catalog.PriceTier{{Quantity: 6, TotalPriceMinor: 15000}}},
				{name: "角色贴纸单张", description: "单张 RM10；任意两张 RM15。", stock: 60, listed: 60, price: 1000, tiers: []catalog.PriceTier{{Quantity: 2, TotalPriceMinor: 1500}}},
				{name: "限定帆布袋", description: "厚磅棉布，活动限定印花。", stock: 24, listed: 24, price: 3500},
				{name: "透明收藏卡", description: "PET 透明材质，单张出售。", stock: 50, listed: 50, price: 1200},
			},
			bundleName: "精选双件套", bundleGoods: []int{2, 3}, bundlePrice: 4200,
		},
	}
	users := []struct {
		login string
		role  string
		shop  string
	}{
		{login: "platform-admin", role: "platform_admin"},
		{login: "alpha-boss", role: "shop_admin", shop: "alpha"},
		{login: "alpha-sales", role: "sales", shop: "alpha"},
		{login: "beta-boss", role: "shop_admin", shop: "beta"},
		{login: "beta-sales", role: "sales", shop: "beta"},
		{login: "demo-customer", role: "customer"},
	}
	hashes := make(map[string]string, len(users))
	for _, user := range users {
		hash, err := auth.HashPassword(password)
		if err != nil {
			return fmt.Errorf("hash local seed password: %w", err)
		}
		hashes[user.login] = hash
	}

	tx, err := store.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin local seed: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	lockQuery := "LOCK TABLE "
	for i, table := range localResetTables {
		if i > 0 {
			lockQuery += ", "
		}
		lockQuery += pgxIdentifier(table)
	}
	lockQuery += " IN EXCLUSIVE MODE"
	if _, err := tx.Exec(ctx, lockQuery); err != nil {
		return fmt.Errorf("lock local seed: %w", err)
	}
	for _, table := range localResetTables {
		var hasRows bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM `+pgxIdentifier(table)+` LIMIT 1)`).Scan(&hasRows); err != nil {
			return fmt.Errorf("check local seed table %s: %w", table, err)
		}
		if hasRows {
			return fmt.Errorf("local seed requires empty application tables; run reset-local first")
		}
	}
	shopIDs := make(map[string]ID, len(shops))
	for _, shop := range shops {
		var shopID ID
		if err := tx.QueryRow(ctx, `INSERT INTO shops(name, slug, notice) VALUES ($1,$2,$3) RETURNING id`, shop.name, shop.slug, "Local development store").Scan(&shopID); err != nil {
			return fmt.Errorf("create local shop %s: %w", shop.slug, err)
		}
		shopIDs[shop.slug] = shopID
	}
	for _, user := range users {
		var shopID any
		if user.shop != "" {
			shopID = shopIDs[user.shop]
		}
		if _, err := tx.Exec(ctx, `INSERT INTO users(login_name,password_hash,role,shop_id) VALUES ($1,$2,$3,$4)`, user.login, hashes[user.login], user.role, shopID); err != nil {
			return fmt.Errorf("create local user %s: %w", user.login, err)
		}
	}
	for _, shop := range shops {
		shopID := shopIDs[shop.slug]
		var eventID ID
		if err := tx.QueryRow(ctx, `INSERT INTO events(shop_id,name,starts_on,ends_on,city,venue,booth_number,currency,status)
			VALUES ($1,$2,$3,$4,'Kuala Lumpur',$5,$6,'MYR','scheduled') RETURNING id`, shopID, shop.eventName, shop.starts, shop.ends, shop.venue, shop.booth).Scan(&eventID); err != nil {
			return fmt.Errorf("create local event %s: %w", shop.slug, err)
		}
		goodIDs := make([]ID, len(shop.goods))
		for i, good := range shop.goods {
			if good.stock < 0 || good.listed < 0 || good.listed > good.stock || catalog.ValidatePriceTiers(good.price, good.tiers) != nil {
				return fmt.Errorf("invalid local product fixture %s/%s", shop.slug, good.name)
			}
			if good.tiers == nil {
				good.tiers = []catalog.PriceTier{}
			}
			tiers, err := json.Marshal(good.tiers)
			if err != nil {
				return fmt.Errorf("encode local price tiers: %w", err)
			}
			if err := tx.QueryRow(ctx, `INSERT INTO goods(shop_id,name,description,total_stock,review_status,reviewed_at)
				VALUES ($1,$2,$3,$4,'approved',now()) RETURNING id`, shopID, good.name, good.description, good.stock).Scan(&goodIDs[i]); err != nil {
				return fmt.Errorf("create local product %s/%s: %w", shop.slug, good.name, err)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO event_listings(shop_id,event_id,good_id,listed_qty,unit_price_minor,price_tiers,is_listed)
				VALUES ($1,$2,$3,$4,$5,$6,true)`, shopID, eventID, goodIDs[i], good.listed, good.price, tiers); err != nil {
				return fmt.Errorf("list local product %s/%s: %w", shop.slug, good.name, err)
			}
		}
		var bundleID ID
		if err := tx.QueryRow(ctx, `INSERT INTO bundles(shop_id,name,description,review_status,reviewed_at)
			VALUES ($1,$2,'固定商品组合优惠','approved',now()) RETURNING id`, shopID, shop.bundleName).Scan(&bundleID); err != nil {
			return fmt.Errorf("create local bundle %s: %w", shop.slug, err)
		}
		for sortOrder, goodIndex := range shop.bundleGoods {
			if goodIndex < 0 || goodIndex >= len(goodIDs) {
				return fmt.Errorf("invalid product index in local bundle %s", shop.slug)
			}
			if _, err := tx.Exec(ctx, `INSERT INTO bundle_components(bundle_id,shop_id,good_id,quantity,sort_order) VALUES ($1,$2,$3,1,$4)`, bundleID, shopID, goodIDs[goodIndex], sortOrder); err != nil {
				return fmt.Errorf("add local bundle component %s: %w", shop.slug, err)
			}
		}
		if _, err := tx.Exec(ctx, `INSERT INTO event_bundle_listings(shop_id,event_id,bundle_id,unit_price_minor,price_tiers,is_listed)
			VALUES ($1,$2,$3,$4,'[]'::jsonb,true)`, shopID, eventID, bundleID, shop.bundlePrice); err != nil {
			return fmt.Errorf("list local bundle %s: %w", shop.slug, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit local seed: %w", err)
	}
	return nil
}
