# Stallink 财务、库存与 UI/UX 质量修复实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 superpowers:subagent-driven-development（推荐）或 superpowers:executing-plans 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法跟踪进度。执行 Go 工作前先加载项目要求的 Go 技能；前端改动遵循现有纸白、宋体、钴蓝、细线视觉语言。

**目标：** 修复库存部分扣减、多币种日报口径、日报非一致快照和质量门禁失效，并收口 PDF、弹窗和移动端可读性，使当前版本达到可重复验证的生产候选状态。

**架构：** 库存收款采用“订单应有数量与有效预留逐行一致”的全有或全无校验；日报在同一 `READ ONLY / REPEATABLE READ` 事务内生成，主币种总览与分明细币种的数据口径分离。前端用异步目录快照回调协调购物车，PDF 等待真实资源加载，弹窗统一使用原生 `<dialog>`。

**技术栈：** Go、pgx/PostgreSQL、Kratos HTTP、Next.js 16、React 19、TypeScript、ESLint、Vitest、Testing Library、浏览器打印能力。

**审查基线：** `9eab534 feat(shop): 完善收银库存、订单筛选与销售日报导出`

**执行边界：** 本计划编写阶段不修改业务代码、不提交 Git。集成测试只能使用独立测试数据库或独立 schema，禁止对开发/生产数据做过期时间篡改。每个任务完成后可单独暂存并提交；提交动作必须由用户确认。

---

## 文件职责

| 文件 | 职责 |
|---|---|
| `server/internal/data/integration_test_helpers_test.go` | integration 测试共用的字符串到 `ID` 转换辅助函数 |
| `server/internal/data/paged_lists_integration_test.go`、`bundle_update_integration_test.go` | 修复 integration 测试中的 `string` / `ID` 类型边界 |
| `server/internal/data/reservation_store.go` | 收款前全量核对有效预留，拒绝部分消耗 |
| `server/internal/data/reservation_store_integration_test.go` | 覆盖单条过期、多条过期、完整预留三种收款路径 |
| `server/internal/data/report_store.go` | 使用一致读事务，按币种生成活动明细并统一总览口径 |
| `server/internal/data/report_store_test.go`、`report_store_integration_test.go` | 覆盖主币种计数和多币种 SQL 聚合 |
| `web/src/app/counter/page.tsx` | 目录快照协调购物车，处理预留过期，不再在 Effect 体中同步级联 setState |
| `web/src/app/counter/page.test.tsx` | 验证数量裁剪和预留过期恢复逻辑 |
| `web/src/components/select-menu.tsx`、`select-menu.test.tsx` | 保持 Portal 下拉行为并补键盘回归测试 |
| `web/src/app/admin/reports/page.tsx` | 活动金额使用行币种，PDF 等待样式与字体，失败时可见报错 |
| `web/src/app/admin/pay-codes/page.tsx` | 收款码预览改为原生弹窗 |
| `web/src/app/globals.css` | 原生弹窗、关键文字和移动导航滚动条的定向样式收口 |
| `web/vitest.config.ts`、`web/src/test/setup.ts`、`web/package.json` | 建立前端单元/组件测试入口 |

## 任务 1：恢复 integration 测试编译和质量门禁

**文件：**
- 创建：`server/internal/data/integration_test_helpers_test.go`
- 修改：`server/internal/data/paged_lists_integration_test.go`
- 修改：`server/internal/data/bundle_update_integration_test.go`

- [ ] **步骤 1：添加 integration 专用 ID 转换辅助函数**

```go
//go:build integration

package data

import "testing"

func integrationID(t *testing.T, raw string) ID {
	t.Helper()
	id, err := ParseID(raw)
	if err != nil {
		t.Fatalf("ParseID(%q): %v", raw, err)
	}
	return id
}
```

- [ ] **步骤 2：在分页测试中区分 SQL 字符串 ID 与数据层类型 ID**

保留数据库 `RETURNING id::text` 的字符串变量，在调用 store 前显式转换：

```go
shopID := integrationID(t, shop)
otherID := integrationID(t, other)
eventID := integrationID(t, event)

p1, more, err := store.ListOrderPage(ctx, shopID, "", "", "", 20, 0)
result, _, err := store.ListOrderPage(ctx, shopID, eventID.String(), "", "顾客12", 20, 0)
result, _, err = store.ListOrderPage(ctx, otherID, "", "", "", 20, 0)
```

将该文件中所有 `ListOrderPage`、`ListAuditPage` 的店铺参数分别改为 `shopID`、`otherID`，SQL fixture 参数仍使用原字符串。

- [ ] **步骤 3：修复套餐更新测试的实体 ID 类型**

```go
shopID := integrationID(t, shop)
actorID := integrationID(t, actor)
eventID := integrationID(t, event)
oneID := integrationID(t, one)
twoID := integrationID(t, two)

input := BundleInput{
	Name: "bundle", EventID: eventID, UnitPriceMinor: 150, IsListed: true,
	Components: []BundleComponentInput{{GoodID: oneID, Quantity: 1}, {GoodID: twoID, Quantity: 1}},
}
id, _, err := store.CreateBundle(ctx, shopID, actorID, input)
_, err = store.UpdateBundle(ctx, shopID, actorID, id, 1, input, false)
```

- [ ] **步骤 4：只编译全部 integration 测试**

运行：

```bash
cd /Users/kelden/go/src/Stallink/server
go test -tags=integration ./... -run '^$'
```

预期：所有包编译成功，不再出现 `cannot use ... (variable of type string) as ID value`。若编译器继续指出其他 integration 文件中的同类调用，在这些文件内采用相同的“SQL 用字符串、store 用 `integrationID`”边界修复，不改变测试业务语义。

- [ ] **步骤 5：在隔离数据库执行全部测试**

```bash
export STALLINK_TEST_DATABASE_URL='postgres://postgres@127.0.0.1:5432/stallink_test?sslmode=disable'
go test ./... -count=1
go vet ./...
go test -tags=integration ./... -count=1
```

预期：全部通过。`stallink_test` 必须是专用空测试库，不得使用正在验收的 `stallink` 库。

- [ ] **步骤 6：暂存并给出提交信息**

```bash
git add server/internal/data/integration_test_helpers_test.go \
  server/internal/data/paged_lists_integration_test.go \
  server/internal/data/bundle_update_integration_test.go
# 用户确认后：git commit -m "test(server): 恢复 integration 测试编译"
```

## 任务 2：库存预留全量校验与过期恢复

**文件：**
- 修改：`server/internal/data/reservation_store.go:79-146`
- 创建：`server/internal/data/reservation_store_integration_test.go`
- 修改：`web/src/app/counter/page.tsx:534-562`

- [ ] **步骤 1：先写“只有一条预留过期”的失败集成测试**

fixture 创建一笔包含两个 listing 的待付款预留订单，将其中一条预留改为已过期，另一条保持未来时间：

```go
_, err = pool.Exec(ctx, `
	UPDATE inventory_reservations
	SET expires_at = now() - interval '1 second'
	WHERE order_id = $1 AND listing_id = $2`, orderID, expiredListingID)
```

调用：

```go
_, err = store.PayOrderCommand(ctx, identity, orderID, payCodeID)
if !errors.Is(err, ErrReservationExpired) {
	t.Fatalf("PayOrderCommand error = %v, want ErrReservationExpired", err)
}
```

同时断言订单仍为 `pending`，两个 listing 的 `sold_qty` 均未增加，两个 good 的 `total_stock` 均未减少。

- [ ] **步骤 2：运行测试确认当前实现会暴露部分扣减**

```bash
cd /Users/kelden/go/src/Stallink/server
go test -tags=integration ./internal/data -run TestPayOrderRejectsPartiallyExpiredReservations -v
```

预期：修复前 FAIL；当前实现可能只消耗未过期的一项并继续收款。

- [ ] **步骤 3：实现全有或全无的有效预留核对**

在 `consumeOrderReservations` 读取 active 行后，先调用 `orderReservationLines`，按 listing 汇总 expected 与 active；任何缺失、额外或数量不同都返回 `ErrReservationExpired`，并且必须发生在任何库存 UPDATE 之前：

```go
expected, err := orderReservationLines(ctx, tx, orderID)
if err != nil {
	return err
}
activeQty := make(map[ID]int, len(active))
for _, item := range active {
	activeQty[item.listingID] += item.qty
}
if len(activeQty) != len(expected) {
	return ErrReservationExpired
}
for _, line := range expected {
	if activeQty[line.listingID] != line.qty {
		return ErrReservationExpired
	}
}
```

保留现有 listing/good 锁、`reserved_qty` / `sold_qty` 原子更新和审计逻辑。

- [ ] **步骤 4：补齐完整预留和全部过期两组测试**

完整预留应收款成功且所有预留变为 `consumed`；全部过期应返回 `ErrReservationExpired` 且库存不变。运行：

```bash
go test -tags=integration ./internal/data -run 'TestPayOrder.*Reservation' -v
```

预期：三组测试全部通过。

- [ ] **步骤 5：前端对永久过期提供退出路径**

在 `confirm()` 中，通用失败提示之前增加：

```tsx
if (res.status === 409 && res.data.error === "reservation_expired") {
	clearPendingOrderCommand(scope);
	clearPendingCounterOrder({ accountId: me.id, shopId: me.shopId, eventId });
	setPayOpen(false);
	setOrderId("");
	setReservationExpiresAt("");
	setMessage(`订单 ${orderNumber} 的库存预留已过期，不能继续收款；请核对订单状态后重新开单。`);
	void refreshCatalog();
	return;
}
```

不得继续显示“请重试”，不得自动把订单标为已取消。

- [ ] **步骤 6：回归验证**

```bash
cd /Users/kelden/go/src/Stallink/web
npx tsc --noEmit
npm run build
```

浏览器验收：过期弹窗关闭后不再恢复旧待付款订单，目录库存刷新，用户可重新组货开单。

- [ ] **步骤 7：暂存并给出提交信息**

建议：`fix(orders): 拒绝部分预留消耗并处理过期收款`

## 任务 3：日报一致快照与多币种口径

**文件：**
- 修改：`server/internal/data/report_store.go`
- 修改：`server/internal/data/report_store_test.go`
- 创建：`server/internal/data/report_store_integration_test.go`
- 修改：`web/src/app/admin/reports/page.tsx`

- [ ] **步骤 1：先写多币种活动统计集成测试**

同一活动下建立两笔已付款订单：一笔 `MYR`、一笔 `USD`，各自写入订单行。调用 `BuildDailyReport` 后断言活动明细为两行，而不是合并：

```go
if len(report.Events) != 2 {
	t.Fatalf("events = %+v, want one row per event and currency", report.Events)
}
for _, row := range report.Events {
	if row.Currency == "MYR" && row.RevenueMinor != 10000 {
		t.Fatalf("MYR event = %+v", row)
	}
	if row.Currency == "USD" && row.RevenueMinor != 1000 {
		t.Fatalf("USD event = %+v", row)
	}
}
```

同时断言主币种的 `Orders`、`PaidOrders`、`ItemsSold` 只包含主币种订单；`Currencies` 仍包含两种币种。

- [ ] **步骤 2：运行测试确认当前聚合失败**

```bash
cd /Users/kelden/go/src/Stallink/server
go test -tags=integration ./internal/data -run TestBuildDailyReportSeparatesCurrencies -v
```

预期：修复前活动只有一行，或不同币种金额被合并，测试 FAIL。

- [ ] **步骤 3：扩展活动明细币种字段并修改 SQL**

```go
type DailyReportEvent struct {
	EventID       string `json:"eventId"`
	Name          string `json:"name"`
	Currency      string `json:"currency"`
	Orders        int    `json:"orders"`
	RevenueMinor  int64  `json:"revenueMinor"`
	DiscountMinor int64  `json:"discountMinor"`
}
```

活动查询增加 `o.currency` 的 SELECT、GROUP BY 和 Scan，排序改为活动名、币种；禁止再把不同币种的 minor 单位直接求和。

- [ ] **步骤 4：统一主币种总览口径**

先查询 `Currencies` 并执行 `applyHeadlineCurrency`，再查询总览状态。`applyHeadlineCurrency` 同时复制主币种的 `Orders` 和 `PaidOrders`：

```go
report.Orders = top.Orders
report.PaidOrders = top.PaidOrders
```

总览状态、`ItemsSold` 查询增加 `AND o.currency = $4`；多币种时页面继续通过 `Currencies` 展示其他币种。更新 `report_store_test.go`，验证主币种计数不会被全币种计数污染。

- [ ] **步骤 5：用同一一致读事务包住 BuildDailyReport**

函数开头建立事务，后续所有 Query/QueryRow 改用 `tx`：

```go
tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{
	AccessMode: pgx.ReadOnly,
	IsoLevel:   pgx.RepeatableRead,
})
if err != nil {
	return DailyReport{}, fmt.Errorf("begin daily report: %w", err)
}
defer func() { _ = tx.Rollback(ctx) }()
```

成功返回前 `tx.Commit(ctx)`；任何查询失败直接返回。不得在同一张日报中混用 `s.pool` 与 `tx`。

- [ ] **步骤 6：前端按行币种渲染活动金额**

```tsx
type EventRow = {
	eventId: string; name: string; currency: string;
	orders: number; revenueMinor: number; discountMinor: number;
};
```

活动行 key 改为 `${row.eventId}-${row.currency}`，金额改为 `formatMoney(row.currency, row.revenueMinor)`。读取旧快照时可用 `row.currency ?? report.currency` 兜底，但旧多币种快照必须重新生成后才作为验收数据。

- [ ] **步骤 7：运行完整验证**

```bash
cd /Users/kelden/go/src/Stallink/server
go test ./... -count=1
go test -tags=integration ./internal/data -run 'TestBuildDailyReport|TestApplyHeadline' -v
cd /Users/kelden/go/src/Stallink/web
npx tsc --noEmit
npm run build
```

预期：同一活动不同币种分别显示；总览、活动、商品、收款方式的主币种口径一致。

- [ ] **步骤 8：暂存并给出提交信息**

建议：`fix(reports): 统一日报快照与多币种统计口径`

## 任务 4：消除购物车 Effect 级联并建立前端回归测试

**文件：**
- 修改：`web/package.json`
- 创建：`web/vitest.config.ts`
- 创建：`web/src/test/setup.ts`
- 修改：`web/src/app/counter/page.tsx:125-157、279-349、363-404`
- 创建：`web/src/app/counter/page.test.tsx`
- 创建：`web/src/components/select-menu.test.tsx`

- [ ] **步骤 1：安装并配置测试工具**

```bash
cd /Users/kelden/go/src/Stallink/web
npm install -D vitest @vitejs/plugin-react jsdom @testing-library/react @testing-library/user-event @testing-library/jest-dom
```

`package.json` 增加：

```json
{"scripts":{"test":"vitest run","test:watch":"vitest"}}
```

Vitest 使用 `jsdom` 环境、`src/test/setup.ts` 引入 `@testing-library/jest-dom/vitest`，React 插件复用现有 TypeScript 路径别名。

- [ ] **步骤 2：先写下拉键盘失败/回归用例**

```tsx
it("方向键移动并由 Enter 选中", async () => {
	const user = userEvent.setup();
	const onChange = vi.fn();
	render(<SelectMenu ariaLabel="币种" value="myr" onChange={onChange}
		options={[{value:"myr",label:"MYR"},{value:"usd",label:"USD"}]} />);
	const trigger = screen.getByRole("combobox", { name: "币种" });
	trigger.focus();
	await user.keyboard("{ArrowDown}{ArrowDown}{Enter}");
	expect(onChange).toHaveBeenCalledWith("usd");
	expect(trigger).toHaveFocus();
});
```

再覆盖 Escape 关闭、禁用选项跳过、Portal 中点击选项四条路径。

- [ ] **步骤 3：将目录裁剪提取为纯函数并写测试**

提取 `trimCartToCatalog(listings, bundles, qty, bundleQty)`，返回 `{ nextQty, nextBundleQty, notice }`。测试覆盖：单品超过剩余、套餐售完清空、套餐占用后单品上限、正常数量保持不变。

```tsx
expect(result.nextBundleQty[bundle.bundleListingId]).toBe(0);
expect(result.notice).toContain("已售完");
```

- [ ] **步骤 4：把裁剪调用移到异步目录快照回调**

删除 `page.tsx:279-318` 两个 Effect。初始目录加载和 `refreshCatalog` 都在 Promise 回调中调用纯函数，再分别 `setQty`、`setBundleQty`、`setQuote` 和 `showCatalogNotice`。使用 ref 读取最新购物车，避免把购物车数量加入 `refreshCatalog` 依赖：

```tsx
const cartRef = useRef({ qty, bundleQty });
cartRef.current = { qty, bundleQty };
const refreshCatalog = useCallback(async () => {
	// fetch 成功后：trimCartToCatalog(..., cartRef.current.qty, cartRef.current.bundleQty)
}, [eventId]);
```

目录轮询与报价 Effect 加入稳定的 `refreshCatalog` 依赖，消除 `react-hooks/exhaustive-deps` 警告，不得用 eslint-disable 隐藏本轮错误。

- [ ] **步骤 5：为预留过期恢复写组件测试**

mock `sendOrderCommand` 返回 `{ status: 409, data: { error: "reservation_expired" } }`，断言过期提示可见、旧订单恢复标记被清除、不会显示“确认失败，请重试”。

- [ ] **步骤 6：运行前端完整门禁**

```bash
npm test
npm run lint
npx tsc --noEmit
npm run build
```

预期：测试、0 ESLint error、0 warning、类型检查和构建全部通过。

- [ ] **步骤 7：暂存并给出提交信息**

建议：`test(web): 补齐收银交互回归并修复 Hook 依赖`

## 任务 5：PDF 导出等待资源并移除带外壳 fallback

**文件：**
- 修改：`web/src/app/admin/reports/page.tsx:99-166、179-189`
- 修改：`web/src/app/counter/page.test.tsx` 或新增 `web/src/app/admin/reports/page.test.tsx`

- [ ] **步骤 1：添加资源等待辅助函数**

```tsx
async function waitForPrintAssets(doc: Document) {
	const styles = Array.from(doc.querySelectorAll<HTMLLinkElement>('link[rel="stylesheet"]'));
	await Promise.all(styles.map((link) => link.sheet
		? Promise.resolve()
		: new Promise<void>((resolve, reject) => {
			link.addEventListener("load", () => resolve(), { once: true });
			link.addEventListener("error", () => reject(new Error("stylesheet")), { once: true });
		})));
	await doc.fonts.ready;
}
```

`exportPDF` 改为 `async`，写入 iframe 后等待资源，再调用 `printWindow.print()`。外层用 5 秒超时防止永久等待。

- [ ] **步骤 2：失败时只显示错误，不打印父页面**

```tsx
catch {
	frame.remove();
	setMessage("PDF 样式加载失败，请检查网络后重试。");
}
```

删除两处 `window.print()` fallback，避免再次导出带顶栏、侧栏的页面。

- [ ] **步骤 3：校准按钮预期**

按钮文案改为“导出 PDF”，按钮旁显示短说明“在打印面板选择‘另存为 PDF’”，不只依赖 `title` tooltip。

- [ ] **步骤 4：验证慢资源与正常导出**

浏览器网络设为 Slow 3G 后导出：样式加载完成前不得打开打印面板，失败后无残留 iframe。正常网络下确认标签页数量不变、iframe 无 `.site-header` / `.workspace-sidebar`、A4 仅一页。

- [ ] **步骤 5：暂存并给出提交信息**

建议：`fix(reports): 等待打印资源并移除整页打印回退`

## 任务 6：收款与预览弹窗可访问性

**文件：**
- 修改：`web/src/app/admin/pay-codes/page.tsx:145-158`
- 修改：`web/src/app/counter/page.tsx:740-777`
- 修改：`web/src/app/globals.css:2050-2092`

- [ ] **步骤 1：两个弹窗改用原生 `<dialog>`**

参照 `admin/orders/page.tsx` 的 `dialogRef + showModal()` 模式。打开状态调用 `showModal()`，关闭状态调用 `close()`；`onClose` 同步 React state，`onCancel` 允许 Escape 关闭。

- [ ] **步骤 2：保留点击遮罩关闭和现有视觉结构**

原生 dialog 内保留 `.pay-dialog__panel`，backdrop 点击要求 `event.target === event.currentTarget`。CSS 使用 `dialog.pay-dialog::backdrop` 替代普通 div 的全屏遮罩，面板最大高度继续使用 `calc(100dvh - 2rem)`。

- [ ] **步骤 3：验证键盘闭环**

桌面与 390 各验收一次：打开后焦点进入弹窗，Tab 不进入背景，Escape 关闭，焦点回到触发按钮；收款处理中 `submitting=true` 时不得意外关闭并丢失支付状态。

- [ ] **步骤 4：运行检查**

```bash
cd /Users/kelden/go/src/Stallink/web
npm test
npm run lint
npm run build
```

- [ ] **步骤 5：暂存并给出提交信息**

建议：`fix(web): 使用原生弹窗收口键盘与焦点行为`

## 任务 7：移动端可读性、导航滑轨与最终验收

**文件：**
- 修改：`web/src/app/globals.css:1272、1579-1601、1900-1909、2349-2352`
- 创建：`docs/design-v2/screens/质量复核-日报-390.png`
- 创建：`docs/design-v2/screens/质量复核-收银-390.png`

- [ ] **步骤 1：只提高关键操作信息字号**

日报指标 label/small、订单日期和库存说明调整为 `0.75rem–0.875rem`；非关键 eyebrow 可保留现有小号排印。不得整体放大页面，也不得改变主金额层级。

- [ ] **步骤 2：隐藏移动导航原生滚动轨道但保留滑动**

在移动断点内添加：

```css
.workspace-nav { scrollbar-width: none; }
.workspace-nav::-webkit-scrollbar { display: none; }
```

保留 `overflow-x:auto` 和当前活动项自动可见，390 下不得出现横向页面溢出。

- [ ] **步骤 3：执行全量质量门禁**

```bash
cd /Users/kelden/go/src/Stallink/server
go test ./... -count=1
go vet ./...
go test -tags=integration ./... -count=1
cd /Users/kelden/go/src/Stallink/web
npm test
npm run lint
npx tsc --noEmit
npm run build
cd /Users/kelden/go/src/Stallink
git diff --check
```

预期：全部通过，无空白错误。

- [ ] **步骤 4：完成真实业务验收矩阵**

逐项记录结果：完整预留收款、单条预留过期、全部过期、套餐与单品联合库存、单币种日报、双币种日报、订单日期筛选、下拉键盘操作、二维码预览、PDF 导出、桌面与 390 弹窗。

- [ ] **步骤 5：保存桌面与 390 截图并复核**

检查无布局位移、无多余滚动轨道、无小于 12px 的关键操作信息、无站点外壳进入 PDF。

- [ ] **步骤 6：暂存并给出提交信息**

建议：`fix(web): 提升移动可读性并完成质量验收`

## 回滚策略

- 每个任务独立暂存/提交，按任务反向回滚，避免一次回滚全部功能。
- 本计划不新增数据库迁移，不需要数据库回滚脚本；集成测试的时间修改只发生在隔离 schema。
- 日报前后端必须作为一个回滚单元；旧快照缺少活动币种时仅允许临时回退显示，不能用旧混合金额作为财务验收依据。
- 库存部分消耗修复如出现异常，优先回滚到上一任务提交并停止收款验收，不允许通过跳过全量校验恢复上线。

## 明确不在本轮执行

- 不做全站 CSS 重写或视觉换皮；`globals.css` 的全量模块化属于独立技术债项目，避免与财务、库存正确性修复混在同一变更中。
- 不把 PDF 改为服务端生成，不引入新的 PDF 服务；本轮只保证现有浏览器导出路径稳定、无外壳。
- 不删除历史测试订单，不修改真实店铺库存和收款方式。
