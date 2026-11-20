# 共享池库存模型改造方案（方案 A · 已落地）

> 状态：方案 A 纯共享池已实施。后端共享池扣减、前端去配额字段、迁移 `010_shared_pool_sold_ledger.sql` 均已落地；`go build/vet/test -tags integration`、前端 `tsc/eslint/next build` 全部通过。

## 1. 现状：配额制（当前线上逻辑）

**表结构**
- `goods.total_stock`：未售出的共享余量。
- `event_listings`：`listed_qty`（本场配额）、`sold_qty`（本场已售）、`reserved_qty`（本场待支付预占）。

**校验与可售**
- 配额校验 `catalog.ValidateAllocation`：`Σ(listed_qty − sold_qty) ≤ total_stock`，即各场未售配额之和不能超过总库存。
- 单场可售 `remaining = listed_qty − sold_qty − reserved_qty`（listing 维度）。

**扣减链路**
- 预占 `reserveOrderInventory`：按 listing 余量 `reserved_qty += qty`，条件 `listed − sold − reserved ≥ qty`。
- 成交 `consumeOrderReservations`：`total_stock` 递减，listing `reserved → sold`，并复核配额约束。
- 释放 `releaseOrderReservations`：listing `reserved_qty −= qty`。
- 退款恢复（`order_store.go`）：按 listing 维度把 sold 回补库存。

**读取侧**
- `quote_store.go`、`bundle_store.go`：按 `listed − sold − reserved` 取每场余量。
- `cart_pricing.go`：按 `listing.Remaining` 校验加购；套餐组件同理。
- 前端：弹窗「本场可售总数」上限 = 可分配额度；卡片展示 已售 / 已分配 / 可分配；加入活动需要分配数量。

**截图现象的成因**：「测试123」占满 18 件配额后，另一场可分额度被挤到 0；那场已售 12 件被配额锁死，数量不能降到 12 以下，于是报错「可售数量不能低于已售加预占 12 件」。

## 2. 目标：共享池（方案 A · 纯共享，按业务举例）

- `total_stock` = 未售出物理库存（成交扣减、退款回补）；`sold_qty` 是分场销售台账，**已不在 total 之内**，不参与池减法；`reserved_qty` 是待支付预占。
- **可售 `available = total_stock − Σreserved_qty`**（钳到 0），全局一个池；sold 不再从 total 重复扣减。
- 每场 listing **不再持有配额**：只存 基础单价 / 优惠包 / 上下架；同一件商品两场都可以上架，对外展示同一个 available。
- 下单预占在 **good 维度**原子校验 `available ≥ 需求量`，锁 good 行；同一订单内多行同一商品先合并计数再校验。
- 成交扣共享池；退款回补共享池；预占超时释放机制保留。
- 售罄口径：`available ≤ 0` 即全场售罄；单场不再有独立「配额售罄」。

## 3. 改动清单

### 后端

1. **`catalog/stock.go`**
   - 移除 `ValidateAllocation` 的跨场配额求和校验。
   - 新增共享池计算/校验：`Available(total, sold, reserved)` 与池不足判断。
   - `ConfirmSale / RestoreSale` 改为 good 维度累计，不再依赖 listing 的 listed 配额。

2. **`reservation_store.go`**
   - 预占：按 good 聚合订单行数量 → 锁 good 行 → 读 `Σsold + Σreserved` → 校验池余量 ≥ 聚合需求；再回各 listing `reserved_qty += qty`（保留分场预占记录，用于报表与释放）。
   - 成交：`total_stock` 递减、listing `reserved → sold`；去掉 `sold + qty ≤ listed_qty` 守卫；去掉成交后的配额复核 `assertAllocation`。
   - 释放：逻辑不变（`reserved_qty −= qty`）。

3. **`order_store.go`（退款/恢复路径）**：可用性与回补改 good 维度池；sold 仍回补到对应 listing 便于分场报表。

4. **`quote_store.go`**：listing 余量改 good 级 available（JOIN 聚合子查询）；可售过滤条件改为 `pool > 0`。

5. **`bundle_store.go`**：套餐组件余量改 good 级 available；套餐可售 = `min(组件池余量 / 单套组件数)`。

6. **`good_image_store.go`**：在售/可售判断条件同步为池口径。

7. **`staff.go` / `catalog_store.go`**：保存校验去掉 listedQty 配额约束；入参 listedQty 可选并忽略；新增 listing 只带价格与上下架，不再分配数量。

8. **迁移 `010_*.sql`**：放宽或删除 CHECK `sold_qty ≤ listed_qty`；`listed_qty` 列先保留为兼容字段（后续再评估清理）；历史 sold/reserved 语义不变，无需重算存量数据。

### 前端

1. 类型与草稿：移除 listedQty 配额字段，保留 soldQty / reservedQty 用于展示。
2. 编辑弹窗：去掉「本场可售总数」输入；每场只留 基础单价 / 优惠包 / 上下架；「加入活动」只选活动和价格，不填数量。
3. 卡片库存摘要：已售 / 待支付预占 / 共享剩余；售罄文案改为「共享库存售罄」。
4. 配额条（StockMeter）改为共享池展示，去掉「已分配」口径。

## 4. 保持不变

- 订单幂等链路：Idempotency-Key 重放、`order_commands` 重放表、`client_order_id` 唯一索引、`inventory_movements` 唯一约束、`state_version` 乐观锁。
- 预占 TTL 与超时释放。
- 价格、优惠包、平台审核流程；套餐组成与收银台交互。

## 5. 风险与边界

- 两场同时热卖：付款瞬间可能被另一单抢光，靠 good 行锁 + 池校验兜底；待支付订单在 TTL 内占用池。
- 套餐占用多个组件池：套餐可售按最紧组件计算。
- 历史数据：旧 `listed_qty` 不再参与校验；迁移不阻断存量订单。
- 分场销量报表仍按 `listing.sold_qty` 统计，不受影响。

## 6. 测试更新要点

- 并发预占：同一池多 listing 下单，池不足返回 `ErrNotEnough`；同幂等键重放仍只生成 1 单。
- 跨场共享：两场挂同一商品，合计售出不超过 `total_stock`。
- 退款回补池；套餐组件池占用与最紧组件约束。
- 现有配额制用例（`Σ listed−sold ≤ stock`）改写为共享池语义。

## 待确认

- **方案 A（推荐，纯共享池）**：去掉本场数量，两场共享同一池。
- 若实际需要「单场保底/封顶」，再议方案 B：保留 listed_qty 作为可选单场上限（≤ 总库存），不做跨场配额切分。
