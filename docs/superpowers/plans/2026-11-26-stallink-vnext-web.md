# Stallink vNext 前端（Next.js 16 + React 19）实现计划

> **面向 AI 代理的工作者：** 必需子技能：使用 `superpowers:subagent-driven-development`（推荐）或 `superpowers:executing-plans` 逐任务实现此计划。步骤使用复选框（`- [ ]`）语法来跟踪进度。

**目标：** 基于 `docs/superpowers/specs/2026-11-26-stallink-prd-vnext.md` 与 `2026-11-26-stallink-feature-list.md`，升级 Stallink Web 前端：支持口径 2 活动限售上限/净锁定保底、503 退避与资格失效自动清挂单、~~现金实收与找零~~（W5 已作废，Kel 2026-11-26）、同店 `sales`/`shop_admin` 取消及部分退款/报损弹窗、跨日退款日报、真离线现金开单（4h 连续断网风控 + PWA/IndexedDB + 超卖处置台）、全站目录 Opt-in、顾客收藏/预选清单（不占库存）/电子取货凭证夹与中英双语 i18n。

**架构：** 保持 Next.js App Router + `/api/[...path]` 代理模式，先补充 `vitest` 单元测试底座，抽取纯逻辑模块（限售保底校验、503 命令重试、~~现金找零~~（W5 已作废）、部分退款校验、4h 断网风控、离线同步队列、i18n 字典）实现 TDD；再对接 Service Worker + IndexedDB 实现断网收现与恢复自动同步。

**技术栈：** Next.js 16.4.6 · React 19.3.0 · TypeScript 5 · Tailwind CSS 4 · Vitest（新增单测） · IndexedDB + Service Worker

---

## 一、与并行修复 Plan 的协调边界 & 阶段并行矩阵

- **协调边界**：后端迁移 `014`（`payment_ref`、`audit_logs.reason` 与 `migrations/down/` 回滚目录，**任务 3**）、`cancel` / `refund-confirm` / 一步全额 `cancel-refund`（`pending` 取消同样必填，原因统一 `2–500` Unicode 字符：`reason_required` / `reason_too_short` / `reason_too_long`，**P1-1，任务 7**）、分布式/降级限流与可信代理 IP（**任务 8**）等前置项由 `docs/superpowers/plans/2026-11-26-stallink-review-fixes.md` 承担；前端在既有 `order-actions.ts`（`REASON_MAX_LENGTH = 500`、最小 `2` 字符、新增 `cancelRefund` 默认入口）与 `Task W6` 对接对应接口，在 `Task W18` 透传 `paymentRef` 至 `014` 字段。
- **拆分布局说明**：**Phase 1（P0，Tasks W0–W10）** 为完整 TDD 代码级计划；**Phase 2（P1，Tasks W11–W18）** 与 **Phase 3（P2，Task W19）** 为清晰任务级拆解。
- **环境红线**：本地调试与验收**仅连接本机 `docker-compose` + 本机 `make dev`（`127.0.0.1:8080`）**。**⚠️ 需要 Kel 手动确认**：任何将 Next 代理或测试环境变量指向非本机云端 API/远程数据库的联调步骤，须由 Kel 手动确认。

| 前端任务 | 对应功能 | 优先级 | 前置依赖 | 可并行组 | 对应后端任务（见 `vnext-server.md`） |
|---|---|---|---|---|---|
| **W0** 引入 Vitest 单测基建 | 全端单测 | P0 | 无 | 组 A（最优先） | 无 |
| **W1** 上架限售上限与净保底 UI | F-01 | P0 | W0 | 组 B（可先 Mock 并行） | S1, S2 |
| **W2** 503 退避重试 & 资格失效清挂单 | F-03, F-04 | P0 | W0 | 组 B | S3, S4 |
| **W3** 套餐 1 图裁剪 & 错误文案更新 | F-06 | P0 | W0 | 组 B | S5 |
| **W4** 活动级时区 & 禁用本地时钟判结束 | F-10, F-16 | P0 | W0 | 组 C | S6, S14 |
| ~~**W5** 现金实收输入框与找零计算器~~ **已作废（Kel 2026-11-26）** | F-07 | — | W0 | 不做：和这一轮去掉实收、找零 UI 的方向相反 | 后端不再接收 `cashReceivedMinor`（K-Q10 = A、K-Q17 = A，Kel 2026-11-26） |
| **W6** 取消必填原因 + 部分退款/报损弹窗 | F-08 | P0 | W0 | 组 D | S7（+修复 plan 任务 7 / P1-1） |
| **W7** 跨日退款日报与导出视图 | F-09 | P0 | W0 | 组 D | S8 |
| **W8** PWA/IndexedDB + 4h 签名与单调时钟防篡改 + 离线开单 | F-11 | P0 | W0（原依赖 W5，W5 已作废，去掉） | 组 E | S9a, S9b |
| **W9** 离线自动同步（带租约验签） + `oversold` 处置台 | F-11, F-12 | P0 | W6, W8 | 组 E | S9a, S9b |
| **W10** Playwright 断网自动验收（P0 压轴关口） | F-11, F-12 | P0 | W8, W9 | 验收关口 | S9a, S9b, S12 |
| **W11–W18** P1 任务（收录/收藏/清单/凭证/双语/审计/SSE） | F-15~F-24 | P1 | Phase 1 | 并行分组 | S13–S19（+修复 plan 任务 3 `014` / 任务 10） |
| **W19** P2 多币种并列汇总 & 残留文案清理 | F-27 | P2 | W7, W15 | 独立 | 无 |

---

## 二、Phase 1：P0 核心实现任务（完整 TDD 规格）

### Task W0：配置 Vitest 单元测试基建

**文件：**
- 修改：`web/package.json`
- 创建：`web/vitest.config.ts`
- 测试：`web/src/lib/__tests__/smoke.test.ts`

- [ ] **步骤 1：创建 Vitest 配置与冒烟测试**

```typescript
// web/vitest.config.ts
import { defineConfig } from "vitest/config";
import path from "node:path";

export default defineConfig({
  test: { environment: "jsdom" },
  resolve: { alias: { "@": path.resolve(__dirname, "src") } },
});
```

```typescript
// web/src/lib/__tests__/smoke.test.ts
import { describe, expect, it } from "vitest";
import { errorCodeLabel } from "@/lib/errors";

describe("errors mapping", () => {
  it("maps not_enough_stock to user message", () => {
    expect(errorCodeLabel("not_enough_stock")).toContain("库存不足");
  });
});
```

- [ ] **步骤 2：在 `web/package.json` 安装 `vitest`、`jsdom` 并增加 `"test": "vitest run"`，运行验证**

运行：`cd web && npm test`
预期：PASS（1 passed）。

- [ ] **步骤 3：Commit**

```bash
git add web/package.json web/package-lock.json web/vitest.config.ts web/src/lib/__tests__/smoke.test.ts
git commit -m "chore(web): set up vitest unit testing harness"
```

---

### Task W1：活动上架限售上限（`maxQty`）与净锁定保底（`minReserveQty`）UI（F-01）

> 对应后端：`Task S1`、`Task S2`

**文件：**
- 创建：`web/src/lib/listing-quota.ts`
- 测试：`web/src/lib/__tests__/listing-quota.test.ts`
- 修改：`web/src/app/admin/goods/page.tsx`、`web/src/components/goods-grid.tsx`、`web/src/lib/errors.ts`

- [ ] **步骤 1：编写失败测试**

```typescript
// web/src/lib/__tests__/listing-quota.test.ts
import { describe, expect, it } from "vitest";
import { validateListingQuotaInput } from "@/lib/listing-quota";

describe("validateListingQuotaInput", () => {
  it("rejects minReserveQty exceeding maxQty remaining cap", () => {
    expect(validateListingQuotaInput({ maxQty: 8, soldQty: 5, reservedQty: 0, minReserveQty: 4 })).toBe("invalid_listing_quota");
  });
  it("accepts unbounded maxQty with valid minReserveQty", () => {
    expect(validateListingQuotaInput({ maxQty: null, soldQty: 0, reservedQty: 0, minReserveQty: 3 })).toBeNull();
  });
});
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd web && npm test -- src/lib/__tests__/listing-quota.test.ts`
预期：FAIL，模块 `@/lib/listing-quota` 不存在。

- [ ] **步骤 3：实现 `listing-quota.ts` 并接入 `admin/goods/page.tsx` 与 `errors.ts`**

```typescript
// web/src/lib/listing-quota.ts
export function validateListingQuotaInput(input: {
  maxQty: number | null;
  soldQty: number;
  reservedQty: number;
  minReserveQty: number;
}): string | null {
  if (!Number.isInteger(input.minReserveQty) || input.minReserveQty < 0) return "invalid_listing_quota";
  if (input.maxQty !== null) {
    if (!Number.isInteger(input.maxQty) || input.maxQty <= 0) return "invalid_listing_quota";
    if (input.soldQty + input.reservedQty + input.minReserveQty > input.maxQty) return "invalid_listing_quota";
  }
  return null;
}
```
在 `admin/goods/page.tsx` 上架表单增加“本场最多卖（选填）”与“从设定时刻起净保底预留（默认 0）”；在 `errors.ts` 将 `stock_exceeded` 更新为 `"各有效活动净保底与待付预留之和超过商品物理总库存。"` 并新增 `invalid_listing_quota`。

- [ ] **步骤 4：运行测试与类型检查**

运行：`cd web && npm test -- src/lib/__tests__/listing-quota.test.ts && npx tsc --noEmit`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add web/src/lib/listing-quota.ts web/src/lib/__tests__/listing-quota.test.ts web/src/app/admin/goods/page.tsx web/src/components/goods-grid.tsx web/src/lib/errors.ts
git commit -m "feat(web): support event maxQty cap and net minReserveQty floor in admin goods"
```

---

### Task W2：503 退避自动重试 & 付款前资格失效自动清挂单（F-03, F-04）

> 对应后端：`Task S3`、`Task S4`

**文件：**
- 修改：`web/src/lib/api.ts`、`web/src/lib/order-command.ts`、`web/src/app/counter/page.tsx`、`web/src/lib/errors.ts`
- 测试：`web/src/lib/__tests__/order-command-retry.test.ts`

- [ ] **步骤 1：编写失败测试**

```typescript
// web/src/lib/__tests__/order-command-retry.test.ts
import { beforeEach, describe, expect, it, vi } from "vitest";
import { sendOrderCommand, pendingOrderCommand } from "@/lib/order-command";

describe("sendOrderCommand retry and eligibility loss", () => {
  beforeEach(() => localStorage.clear());

  it("retries 503 with same Idempotency-Key and clears on order_cancelled_eligibility_lost", async () => {
    const fetchMock = vi.fn()
      .mockResolvedValueOnce(new Response(JSON.stringify({ error: "command_in_progress" }), { status: 503, headers: { "Retry-After": "0" } }))
      .mockResolvedValueOnce(new Response(JSON.stringify({ error: "order_cancelled_eligibility_lost" }), { status: 409 }));
    vi.stubGlobal("fetch", fetchMock);

    const res = await sendOrderCommand("pay:1:1:99", "/api/staff/orders/99/pay", { payCodeId: "1" });
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(res.errorCode).toBe("order_cancelled_eligibility_lost");
    expect(pendingOrderCommand("pay:1:1:99")).toBeNull();
  });
});
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd web && npm test -- src/lib/__tests__/order-command-retry.test.ts`
预期：FAIL。

- [ ] **步骤 3：修改 `order-command.ts` 支持 503 `Retry-After` 最多重试 2 次；在 `counter/page.tsx` 收到 `order_cancelled_eligibility_lost` 时自动清除当前挂单并刷新可售目录**

- [ ] **步骤 4：运行测试验证通过**

运行：`cd web && npm test -- src/lib/__tests__/order-command-retry.test.ts && npx tsc --noEmit`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add web/src/lib/api.ts web/src/lib/order-command.ts web/src/app/counter/page.tsx web/src/lib/errors.ts web/src/lib/__tests__/order-command-retry.test.ts
git commit -m "fix(web): auto-retry 503 command_in_progress and clear pending order on eligibility loss"
```

---

### Task W3：固定套餐表单裁剪对齐（1 图、无阶梯价）与错误文案修复（F-06）

> 对应后端：`Task S5`

**文件：**
- 修改：`web/src/app/admin/goods/page.tsx`、`web/src/lib/errors.ts`
- 测试：`web/src/lib/__tests__/bundle-validation.test.ts`

- [ ] **步骤 1：编写失败测试**

```typescript
// web/src/lib/__tests__/bundle-validation.test.ts
import { describe, expect, it } from "vitest";
import { errorCodeLabel } from "@/lib/errors";

describe("bundle error labels", () => {
  it("explains bundle graph limit and single-image limit", () => {
    expect(errorCodeLabel("bundle_pricing_group_limit_exceeded")).toContain("20");
    expect(errorCodeLabel("bundle_image_limit")).toContain("1");
  });
});
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd web && npm test -- src/lib/__tests__/bundle-validation.test.ts`
预期：FAIL。

- [ ] **步骤 3：补全错误映射并在 `admin/goods/page.tsx` 显式标注“套餐限 1 张主图，固定一口价”**

- [ ] **步骤 4：运行测试验证通过**

运行：`cd web && npm test -- src/lib/__tests__/bundle-validation.test.ts && npx tsc --noEmit`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add web/src/app/admin/goods/page.tsx web/src/lib/errors.ts web/src/lib/__tests__/bundle-validation.test.ts
git commit -m "fix(web): align bundle single-image UI and graph limit error messaging"
```

---

### Task W4：活动级 IANA 时区配置 & 统一采用服务端活动开闭判定（F-10, F-16）

> 对应后端：`Task S6`、`Task S14`

**文件：**
- 创建：`web/src/lib/event-status.ts`
- 测试：`web/src/lib/__tests__/event-status.test.ts`
- 修改：`web/src/app/admin/goods/page.tsx`、`web/src/app/counter/page.tsx`、`web/src/app/c/[slug]/events/[eventId]/page.tsx`

- [ ] **步骤 1：编写失败测试**

```typescript
// web/src/lib/__tests__/event-status.test.ts
import { describe, expect, it } from "vitest";
import { resolveEventFlags } from "@/lib/event-status";

describe("resolveEventFlags", () => {
  it("relies on server event clock flags instead of browser local time", () => {
    const flags = resolveEventFlags({ status: "scheduled", isEndedNow: false, isOpenForOrder: false, allowPresale: false });
    expect(flags.canCreateOrder).toBe(false);
    expect(flags.isEnded).toBe(false);
  });
});
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd web && npm test -- src/lib/__tests__/event-status.test.ts`
预期：FAIL。

- [ ] **步骤 3：实现 `event-status.ts`，并在活动表单加入 `timezone` 下拉与 `allowPresale` 开关，替换前后端分裂的本地时钟判断**

```typescript
// web/src/lib/event-status.ts
export function resolveEventFlags(event: {
  status: string;
  isEndedNow?: boolean;
  isOpenForOrder?: boolean;
  allowPresale?: boolean;
}) {
  const isEnded = Boolean(event.isEndedNow ?? (event.status === "ended"));
  const canCreateOrder = Boolean(event.isOpenForOrder ?? (!isEnded && (event.status === "live" || (event.status === "scheduled" && event.allowPresale))));
  return { isEnded, canCreateOrder };
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd web && npm test -- src/lib/__tests__/event-status.test.ts && npx tsc --noEmit`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add web/src/lib/event-status.ts web/src/lib/__tests__/event-status.test.ts web/src/app/counter/page.tsx web/src/app/c/[slug]/events/[eventId]/page.tsx web/src/app/admin/
git commit -m "feat(web): add event timezone/allowPresale controls and use server event clock flags"
```

---

### Task W5：现场现金收款实收金额输入与实时找零弹窗（F-07）——已作废（Kel 2026-11-26）

> **已作废（Kel 2026-11-26）：** Kel 决定作废 W5。原因：和这一轮工作台打磨（`2026-11-26-stallink-workspace-polish.md`）去掉收银台实收、找零 UI 的方向相反。本任务下面的步骤保留作记录，**不执行**；`web/src/lib/cash-tender.ts` 不创建。~~后端继续接收 `cashReceivedMinor`（S9b 离线现金同步要用），这一条不变。~~ **已改（Kel 2026-11-26 13:55，K-Q17 = A）：** 在线收款和离线同步都不再传 `cashReceivedMinor`，后端忽略该字段。受影响的任务：W8（去掉对 W5 的依赖）、W10（断网验收场景 1 去掉“实收现金”输入）。

**文件：**
- 创建：`web/src/lib/cash-tender.ts`
- 测试：`web/src/lib/__tests__/cash-tender.test.ts`
- 修改：`web/src/app/counter/page.tsx`、`web/src/app/counter/orders/page.tsx`

- [ ] **步骤 1：编写失败测试**

```typescript
// web/src/lib/__tests__/cash-tender.test.ts
import { describe, expect, it } from "vitest";
import { parseCashTender } from "@/lib/cash-tender";

describe("parseCashTender", () => {
  it("computes change minor accurately", () => {
    const res = parseCashTender("50.00", 3500);
    expect(res.ok).toBe(true);
    if (res.ok) {
      expect(res.receivedMinor).toBe(5000);
      expect(res.changeMinor).toBe(1500);
    }
  });
  it("rejects cash received below order total", () => {
    expect(parseCashTender("30.00", 3500).ok).toBe(false);
  });
});
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd web && npm test -- src/lib/__tests__/cash-tender.test.ts`
预期：FAIL。

- [ ] **步骤 3：实现 `cash-tender.ts` 并在收款弹窗接入实收输入框、快捷面额与实时找零**

```typescript
// web/src/lib/cash-tender.ts
export function parseCashTender(rawMajor: string, totalMinor: number):
  | { ok: true; receivedMinor: number; changeMinor: number }
  | { ok: false; errorCode: "invalid_cash_received" } {
  const trimmed = rawMajor.trim();
  if (!/^\d+(?:\.\d{1,2})?$/.test(trimmed)) return { ok: false, errorCode: "invalid_cash_received" };
  const [whole, frac = ""] = trimmed.split(".");
  const receivedMinor = Number(whole) * 100 + Number(frac.padEnd(2, "0"));
  if (!Number.isSafeInteger(receivedMinor) || receivedMinor < totalMinor) {
    return { ok: false, errorCode: "invalid_cash_received" };
  }
  return { ok: true, receivedMinor, changeMinor: receivedMinor - totalMinor };
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd web && npm test -- src/lib/__tests__/cash-tender.test.ts && npx tsc --noEmit`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add web/src/lib/cash-tender.ts web/src/lib/__tests__/cash-tender.test.ts web/src/app/counter/page.tsx web/src/app/counter/orders/page.tsx
git commit -m "feat(pos): add cash received input, quick tender presets, and change calculation"
```

---

### Task W6：取消原因必填、部分退款、报损拆分与重算弹窗（F-08）

> 对应后端：`Task S7`（部分退款/报损）与修复 plan 任务 7（`P1-1`：整单一步 `/cancel-refund` + `cancel`/`refund-confirm` 均校验同店 `sales`/`shop_admin` 与 `2–500` Unicode 字符原因）

**文件：**
- 创建：`web/src/components/refund-dialog.tsx`、`web/src/lib/refund-form.ts`
- 测试：`web/src/lib/__tests__/refund-form.test.ts`
- 修改：`web/src/app/counter/orders/page.tsx`、`web/src/app/admin/orders/page.tsx`

- [ ] **步骤 1：编写失败测试**

```typescript
// web/src/lib/__tests__/refund-form.test.ts
import { describe, expect, it } from "vitest";
import { validateRefundSubmission } from "@/lib/refund-form";

describe("validateRefundSubmission", () => {
  it("enforces reason 2..500 Unicode chars and valid restock+damaged split", () => {
    expect(validateRefundSubmission({ reason: "   ", items: [{ orderLineId: "1", restockQty: 1, damagedQty: 0, maxQty: 1 }] })).toBe("reason_required");
    expect(validateRefundSubmission({ reason: "退", items: [{ orderLineId: "1", restockQty: 1, damagedQty: 0, maxQty: 1 }] })).toBe("reason_too_short");
    expect(validateRefundSubmission({ reason: "退".repeat(501), items: [{ orderLineId: "1", restockQty: 1, damagedQty: 0, maxQty: 1 }] })).toBe("reason_too_long");
    expect(validateRefundSubmission({ reason: "拆封损坏", items: [{ orderLineId: "1", restockQty: 0, damagedQty: 1, maxQty: 1 }] })).toBeNull();
  });
});
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd web && npm test -- src/lib/__tests__/refund-form.test.ts`
预期：FAIL。

- [ ] **步骤 3：实现 `refund-form.ts` 与 `RefundDialog`（支持 `sales` 与 `shop_admin` 取消已付单与部分退款、强制填原因、选择 `restockQty` 与 `damagedQty`、实时预览套餐重算后最大可退额）**

```typescript
// web/src/lib/refund-form.ts
export function validateRefundSubmission(input: {
  reason: string;
  items: { orderLineId: string; restockQty: number; damagedQty: number; maxQty: number }[];
}): string | null {
  const reason = input.reason.trim();
  const count = Array.from(reason).length;
  if (count === 0) return "reason_required";
  if (count < 2) return "reason_too_short";
  if (count > 500) return "reason_too_long";
  let totalQty = 0;
  for (const item of input.items) {
    if (item.restockQty < 0 || item.damagedQty < 0) return "invalid_refund_qty";
    const sum = item.restockQty + item.damagedQty;
    if (sum > item.maxQty) return "invalid_refund_qty";
    totalQty += sum;
  }
  return totalQty > 0 ? null : "invalid_refund_qty";
}
```

- [ ] **步骤 4：运行测试验证通过**

运行：`cd web && npm test -- src/lib/__tests__/refund-form.test.ts && npx tsc --noEmit`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add web/src/lib/refund-form.ts web/src/lib/__tests__/refund-form.test.ts web/src/components/refund-dialog.tsx web/src/app/counter/orders/page.tsx web/src/app/admin/orders/page.tsx
git commit -m "feat(web): add mandatory cancel reason, partial refund preview, and damage split dialog"
```

---

### Task W7：跨日退款日报与导出视图（F-09）

> 对应后端：`Task S8`

**文件：**
- 创建：`web/src/lib/report-summary.ts`
- 测试：`web/src/lib/__tests__/report-summary.test.ts`
- 修改：`web/src/app/admin/reports/page.tsx`

- [ ] **步骤 1：编写失败测试**

```typescript
// web/src/lib/__tests__/report-summary.test.ts
import { describe, expect, it } from "vitest";
import { formatReportCashflow } from "@/lib/report-summary";

describe("formatReportCashflow", () => {
  it("formats gross paid, refund offset, and net received", () => {
    const out = formatReportCashflow({ currency: "MYR", grossPaidMinor: 10000, refundOffsetMinor: 4000, netReceivedMinor: 6000 });
    expect(out.netLabel).toContain("60.00");
    expect(out.offsetLabel).toContain("40.00");
  });
});
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd web && npm test -- src/lib/__tests__/report-summary.test.ts`
预期：FAIL。

- [ ] **步骤 3：实现 `report-summary.ts` 并更新 `admin/reports/page.tsx` 展示当日退款冲减项及原单号/原因明细**

- [ ] **步骤 4：运行测试验证通过**

运行：`cd web && npm test -- src/lib/__tests__/report-summary.test.ts && npx tsc --noEmit`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add web/src/lib/report-summary.ts web/src/lib/__tests__/report-summary.test.ts web/src/app/admin/reports/page.tsx
git commit -m "feat(reports): display gross paid, cross-day refund offsets, and net received"
```

---

### Task W8：PWA/IndexedDB + 4 小时单调时钟与签名租约防篡改 + 真离线现金开单（F-11）

> 对应后端：`Task S9a`（`POST /api/staff/offline-lease`）、`Task S9b`

> **依赖调整（Kel 2026-11-26）：** 原依赖 W5，W5 已作废，本任务不再依赖它。离线现金单不展示实收、找零输入。**K-Q17 = A（Kel 已定 2026-11-26 13:55）：离线订单 payload 和同步请求都不带 `cashReceivedMinor`**，后端按订单总额入账、忽略这个字段。W9 的同步请求同样不带；测试断言 IndexedDB 里存的离线单和 `flushOfflineQueue` 发出的请求体都没有 `cashReceivedMinor`。

**文件：**
- 创建：`web/public/sw.js`、`web/src/lib/offline-pos.ts`
- 测试：`web/src/lib/__tests__/offline-pos.test.ts`
- 修改：`web/src/app/counter/page.tsx`

- [ ] **步骤 1：编写失败测试（验证 `OL` 10 位单号、4 小时限、单调时钟 `performance.now()` 回拨/漂移拦截、缺失或异常租约签名按已超时处理）**

```typescript
// web/src/lib/__tests__/offline-pos.test.ts
import { describe, expect, it } from "vitest";
import { evaluateOfflineLease, generateOfflineOrderNumber, type OfflineLeaseRecord } from "@/lib/offline-pos";

describe("offline POS anti-tamper & rules", () => {
  it("generates 10-char OL number matching backend CHECK constraint", () => {
    expect(generateOfflineOrderNumber("K4", 12)).toMatch(/^OL[ABCDEFGHJKLMNPQRSTUVWXYZ23456789]{8}$/);
  });

  it("allows offline orders within 4h when signature and monotonic clock are valid", () => {
    const syncMs = 1_700_000_000_000;
    const lease: OfflineLeaseRecord = {
      shopId: "1",
      terminalId: "POS-1",
      lastOnlineSyncAt: new Date(syncMs).toISOString(),
      expiresAt: new Date(syncMs + 4 * 3600_000).toISOString(),
      signature: "a".repeat(64),
      localSealOk: true,
      anchorWallMs: syncMs,
      anchorPerfMs: 10_000,
    };
    const ok = evaluateOfflineLease(lease, {
      expectedShopId: "1",
      expectedTerminalId: "POS-1",
      nowWallMs: syncMs + 3 * 3600_000,
      nowPerfMs: 10_000 + 3 * 3600_000,
    });
    expect(ok.allowed).toBe(true);
  });

  it("blocks when signature missing/unverified or system clock rolled back vs performance.now()", () => {
    const syncMs = 1_700_000_000_000;
    const base: OfflineLeaseRecord = {
      shopId: "1",
      terminalId: "POS-1",
      lastOnlineSyncAt: new Date(syncMs).toISOString(),
      expiresAt: new Date(syncMs + 4 * 3600_000).toISOString(),
      signature: "a".repeat(64),
      localSealOk: true,
      anchorWallMs: syncMs,
      anchorPerfMs: 10_000,
    };
    expect(evaluateOfflineLease({ ...base, localSealOk: false }, {
      expectedShopId: "1", expectedTerminalId: "POS-1",
      nowWallMs: syncMs + 1000, nowPerfMs: 11_000,
    })).toEqual({ allowed: false, reason: "expired_or_tampered" });

    // 系统时钟被回调 2 小时，但单调时钟 performance.now() 已走过 1 小时
    expect(evaluateOfflineLease(base, {
      expectedShopId: "1", expectedTerminalId: "POS-1",
      nowWallMs: syncMs - 2 * 3600_000, nowPerfMs: 10_000 + 3600_000,
    })).toEqual({ allowed: false, reason: "expired_or_tampered" });

    // 即使页面未刷新，performance.now() 显示已连续运行 > 4 小时（壁钟停滞伪造）
    expect(evaluateOfflineLease(base, {
      expectedShopId: "1", expectedTerminalId: "POS-1",
      nowWallMs: syncMs + 10_000, nowPerfMs: 10_000 + 4 * 3600_000 + 1,
    })).toEqual({ allowed: false, reason: "expired_or_tampered" });
  });
});
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd web && npm test -- src/lib/__tests__/offline-pos.test.ts`
预期：FAIL，提示模块或 `evaluateOfflineLease` 不存在。

- [ ] **步骤 3：实现 `offline-pos.ts`、`sw.js` 与 `counter/page.tsx` 集成**

```typescript
// web/src/lib/offline-pos.ts（核心防篡改判定 + WebCrypto 本地封印 + OL 单号生成）
const FOUR_HOURS_MS = 4 * 60 * 60 * 1000;
const CLOCK_DRIFT_TOLERANCE_MS = 60 * 1000;
const ALPHABET = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789";

export type OfflineLeaseRecord = {
  shopId: string;
  terminalId: string;
  lastOnlineSyncAt: string;
  expiresAt: string;
  signature: string;
  localSealOk: boolean;
  anchorWallMs: number;
  anchorPerfMs: number | null;
};

export function generateOfflineOrderNumber(terminalShortCode: string, seq: number): string {
  const code = terminalShortCode.toUpperCase().replace(/[^A-Z2-9]/g, "K").padEnd(2, "K").slice(0, 2);
  let body = "";
  let n = Math.max(0, seq);
  for (let i = 0; i < 4; i++) {
    body = ALPHABET[n % ALPHABET.length] + body;
    n = Math.floor(n / ALPHABET.length);
  }
  const raw = `OL${code}${body}`;
  let sum = 0;
  for (let i = 0; i < raw.length; i++) sum += raw.charCodeAt(i) * (i + 1);
  return `${raw}${ALPHABET[sum % ALPHABET.length]}${ALPHABET[(sum * 7) % ALPHABET.length]}`;
}

export function evaluateOfflineLease(
  lease: OfflineLeaseRecord | null,
  ctx: { expectedShopId: string; expectedTerminalId: string; nowWallMs: number; nowPerfMs: number },
): { allowed: true } | { allowed: false; reason: "expired_or_tampered" } {
  if (!lease || !lease.localSealOk || !/^[0-9a-f]{64}$/i.test(lease.signature)) {
    return { allowed: false, reason: "expired_or_tampered" };
  }
  if (lease.shopId !== ctx.expectedShopId || lease.terminalId !== ctx.expectedTerminalId) {
    return { allowed: false, reason: "expired_or_tampered" };
  }
  const syncMs = Date.parse(lease.lastOnlineSyncAt);
  const expMs = Date.parse(lease.expiresAt);
  if (!Number.isFinite(syncMs) || !Number.isFinite(expMs) || expMs - syncMs > FOUR_HOURS_MS + 1000) {
    return { allowed: false, reason: "expired_or_tampered" };
  }
  if (ctx.nowWallMs < syncMs - CLOCK_DRIFT_TOLERANCE_MS || ctx.nowWallMs > expMs) {
    return { allowed: false, reason: "expired_or_tampered" };
  }
  if (lease.anchorPerfMs !== null && ctx.nowPerfMs >= lease.anchorPerfMs) {
    const perfElapsed = ctx.nowPerfMs - lease.anchorPerfMs;
    const wallElapsed = ctx.nowWallMs - lease.anchorWallMs;
    if (perfElapsed > FOUR_HOURS_MS || Math.abs(wallElapsed - perfElapsed) > CLOCK_DRIFT_TOLERANCE_MS) {
      return { allowed: false, reason: "expired_or_tampered" };
    }
  }
  return { allowed: true };
}
```
同时在 `offline-pos.ts` 封装 IndexedDB 存储与 WebCrypto（`crypto.subtle.generateKey(..., false, ["sign","verify"])` 不可导出 HMAC 密钥保存在 IndexedDB 中）：每次联网拿到后端 `POST /api/staff/offline-lease` 返回的 HMAC 签名租约后，前端追加一份 WebCrypto 不可导出密钥的本地封印并记录 `anchorWallMs = Date.now()`、`anchorPerfMs = performance.now()`；任何本地改表行为都会导致 `localSealOk = false`，一律按“已超时”拦截离线开单。

- [ ] **步骤 4：运行测试验证通过**

运行：`cd web && npm test -- src/lib/__tests__/offline-pos.test.ts && npx tsc --noEmit`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add web/public/sw.js web/src/lib/offline-pos.ts web/src/lib/__tests__/offline-pos.test.ts web/src/app/counter/page.tsx
git commit -m "feat(offline): add PWA/IndexedDB offline cash POS with HMAC lease and monotonic clock guard"
```

---

### Task W9：联网异步同步队列（附租约验签刷新） & 超卖（`oversold`）处置台（F-11, F-12）

> 对应后端：`Task S9a`、`Task S9b`

**文件：**
- 创建：`web/src/lib/offline-sync.ts`、`web/src/app/admin/oversold/page.tsx`
- 测试：`web/src/lib/__tests__/offline-sync.test.ts`
- 修改：`web/src/components/workspace-shell.tsx`、`web/src/app/counter/page.tsx`

- [ ] **步骤 1：编写失败测试**

```typescript
// web/src/lib/__tests__/offline-sync.test.ts
import { describe, expect, it, vi } from "vitest";
import { flushOfflineQueue } from "@/lib/offline-sync";

describe("flushOfflineQueue", () => {
  it("syncs chronologically with offlineLease and tracks oversold orders", async () => {
    const syncFn = vi.fn().mockResolvedValue({
      ok: true,
      data: { status: "paid", oversoldStatus: "pending_resolution", oversoldQty: 2 },
    });
    const result = await flushOfflineQueue(
      [
        { clientOrderId: "b", offlineCreatedAt: "2026-11-26T10:02:00Z", payload: {} },
        { clientOrderId: "a", offlineCreatedAt: "2026-11-26T10:01:00Z", payload: {} },
      ],
      syncFn,
    );
    expect(syncFn.mock.calls[0][0].clientOrderId).toBe("a");
    expect(result.oversoldDetected).toBe(1);
  });
});
```

- [ ] **步骤 2：运行测试验证失败**

运行：`cd web && npm test -- src/lib/__tests__/offline-sync.test.ts`
预期：FAIL。

- [ ] **步骤 3：实现 `offline-sync.ts`（同步时附带 `offlineLease`，成功后刷新保存 `refreshedLease`）、全局 `oversold` 横幅告警及 `/admin/oversold` 处置台**

- [ ] **步骤 4：运行单元测试验证通过**

运行：`cd web && npm test && npm run lint && npx tsc --noEmit`
预期：PASS。

- [ ] **步骤 5：Commit**

```bash
git add web/src/lib/offline-sync.ts web/src/lib/__tests__/offline-sync.test.ts web/src/app/admin/oversold/page.tsx web/src/components/workspace-shell.tsx web/src/app/counter/page.tsx
git commit -m "feat(offline): implement chronological offline sync with lease verify and oversold UI"
```

---

### Task W10：Playwright 断网自动验收套件（P0 压轴验收，覆盖 5 大断网核心场景）

> **目标**：补齐 Playwright 基础设施，针对本地完整栈（Docker Compose + 本地 Go API + 本地 Web）实现真实浏览器断网端到端自动化验收，覆盖：**① 断网开现金单 → ② 4 小时超时拦截 → ③ 时钟/签名篡改拦截 → ④ 恢复联网幂等同步 → ⑤ 超卖（`oversold`）告警与处置**。

**文件：**
- 修改：`web/package.json`（新增 `@playwright/test` 依赖及 `"test:e2e": "playwright test"`）
- 创建：`web/playwright.config.ts`
- 创建：`web/e2e/offline-pos.spec.ts`

- [ ] **步骤 1：配置 Playwright 与编写 5 场景端到端验收测试（先写测试脚本）**

```typescript
// web/playwright.config.ts
import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  timeout: 60_000,
  use: {
    baseURL: process.env.E2E_BASE_URL || "http://127.0.0.1:3000",
    trace: "on-first-retry",
  },
});
```

```typescript
// web/e2e/offline-pos.spec.ts
import { expect, test } from "@playwright/test";

test.describe("Offline Cash POS E2E", () => {
  test("1) offline cash order, 2) 4h expiry block, 3) clock/signature tamper block, 4) idempotent sync, 5) oversold alert", async ({ page, context }) => {
    await page.clock.install({ time: new Date("2026-11-26T10:00:00.000Z") });
    // 登录演示收银账号并进入收银台（预先拉取签名租约与商品缓存）
    await page.goto("/login");
    await page.getByLabel("登录名").fill("alpha-sales");
    await page.getByLabel("密码").fill(process.env.STALLINK_LOCAL_SEED_PASSWORD || "demo-password");
    await page.getByRole("button", { name: /登录/ }).click();
    await page.waitForURL("**/counter");
    await expect(page.getByText(/离线就绪/)).toBeVisible();

    // 场景 1：断网后开现金单（生成 OL... 单号）
    // 已调整（Kel 2026-11-26 作废 W5）：不再有“实收现金”输入框，下面原有的 fill("50.00") 一行删除，不执行
    await context.setOffline(true);
    await page.locator(".counter-products button").first().click();
    await page.getByRole("button", { name: "生成订单" }).click();
    // await page.getByLabel("实收现金").fill("50.00");  // W5 已作废，删除
    await page.getByRole("button", { name: /顾客已付款/ }).click();
    await expect(page.locator(".pay-receipt__number")).toHaveText(/^OL[A-Z0-9]{8}$/);
    await page.getByRole("button", { name: "关闭凭证" }).click();

    // 场景 3a：篡改本地时钟（向回拨 2 小时）-> performance.now() 校验不通过，按已超时拦截禁止继续开单
    await page.clock.setFixedTime(new Date("2026-11-26T08:00:00.000Z"));
    await expect(page.getByText(/租约失效或已超时|禁止继续离线开单/)).toBeVisible();
    await expect(page.getByRole("button", { name: "生成订单" })).toBeDisabled();

    // 场景 3b：恢复时间并篡改本地签名 -> 按已超时锁止禁止继续离线开单
    await page.evaluate(() => {
      const raw = localStorage.getItem("stallink.offline-lease.v1");
      if (raw) {
        const parsed = JSON.parse(raw);
        parsed.signature = "0".repeat(64);
        localStorage.setItem("stallink.offline-lease.v1", JSON.stringify(parsed));
      }
    });
    await page.reload();
    await expect(page.getByText(/租约失效或已超时|禁止继续离线开单/)).toBeVisible();

    // 恢复联网同步获取新租约后，验证场景 2：单调时钟推进 > 4 小时拦截
    await context.setOffline(false);
    await expect(page.getByText(/已同步/)).toBeVisible();
    await context.setOffline(true);
    await page.clock.fastForward("04:01:00");
    await expect(page.getByText(/连续离线已超 4 小时/)).toBeVisible();
    await expect(page.getByRole("button", { name: "生成订单" })).toBeDisabled();

    // 场景 4 & 5：再次恢复联网，幂等同步离线单并触发 oversold 告警（可先在另端/Seed 下单占满余量，或校验后端返回 pending_resolution 后全局横幅与处置台联动）
    let syncCalled = 0;
    await page.route("**/api/staff/orders/offline-sync", async (route) => {
      syncCalled += 1;
      const req = route.request();
      expect(req.headers()["idempotency-key"]).toBeTruthy();
      const postData = req.postDataJSON();
      expect(postData.offlineLease?.signature).toMatch(/^[0-9a-f]{64}$/i);
      await route.continue();
    });
    await context.setOffline(false);
    await expect.poll(() => syncCalled).toBeGreaterThan(0);
    await expect(page.getByText(/待同步 0 笔/)).toBeVisible();
    // 校验超卖全局告警与 /admin/oversold 处置台可正常渲染处置
    await page.goto("/admin/oversold");
    await expect(page.getByRole("heading", { name: /超卖处置台/ })).toBeVisible();
  });
});
```

- [ ] **步骤 2：在 `web/package.json` 补全 `@playwright/test`、`"test:e2e": "playwright test"`，并执行本地启动 + E2E 运行命令**

本地运行前置准备与执行命令（仅限本机 `127.0.0.1` Docker Compose 环境）：
```bash
# 1. 拉起本机数据库与服务依赖并初始化演示数据
make compose
export STALLINK_NACOS_USERNAME=nacos
export STALLINK_NACOS_PASSWORD=nacos
export STALLINK_DATABASE_SOURCE='postgres://postgres:stallink@127.0.0.1:5432/stallink?sslmode=disable'
export STALLINK_PLATFORM_ADMIN_PASSWORD='local-admin-password'
export STALLINK_LOCAL_SEED_PASSWORD='local-seed-password'
export STALLINK_OFFLINE_HMAC_SECRET="$(openssl rand -hex 32)"  # 仅注入本机服务端运行进程，切勿写入仓库文件
make seed-local

# 2. 启动本机 Go API（http://127.0.0.1:8080）与 Web（http://127.0.0.1:3000）
make dev   # 终端 1
make web   # 终端 2

# 3. 安装 Playwright 浏览器内核（首次）并运行断网自动验收
cd web && npx playwright install chromium && npm run test:e2e
```
> **⚠️ 需要 Kel 手动确认**：严禁针对非本机云端环境或生产数据库运行 E2E 测试。

预期输出：`1 passed`（全部 5 个断网与超卖验收断言通过），且 `cd web && npm test && npm run lint && npx tsc --noEmit && npm run build` 全部通过。

- [ ] **步骤 3：Commit**

```bash
git add web/package.json web/package-lock.json web/playwright.config.ts web/e2e/offline-pos.spec.ts
git commit -m "test(e2e): add Playwright offline POS, 4h guard, tamper, sync, and oversold verification"
```

---

## 三、Phase 2：P1 任务拆解（任务级 + 交叉引用，对应 `F-15`~`F-24`）

- **Task W11（F-15，对应 S13）**：`web/src/app/admin/page.tsx` 增加“是否允许全站目录收录（`allowPublicCatalog`，默认关闭）”开关；`web/src/components/catalog-explorer.tsx` 增加 `page`/`pageSize` 分页器与排序切换（`live_first`）。
- **Task W12（F-17，对应 S15）**：`web/src/components/favorite-button.tsx`、`web/src/app/me/favorites/page.tsx` —— 顾客收藏摊位/单品/套餐并查看实时库存。
- **Task W13（F-18，对应 S15）**：`web/src/app/c/[slug]/wishlist/page.tsx`、`web/src/app/counter/page.tsx` —— 顾客创建不占库存的预选清单（生成 6 位码/QR），收银台调用 `GET /api/staff/wishlists/{shareCode}` 一键导入并实时报价。
- **Task W14（F-19，对应 S15）**：`web/src/app/me/orders/page.tsx` —— 顾客电子取货凭证夹（查看取货号 `number`、状态与成交明细）。
- **Task W15（F-20，对应 S16）**：新建 `web/src/lib/i18n/`（`zh-CN.ts`、`en.ts` 与 `useLocale()`），覆盖全站导航、收银台、后台、小票凭证与日报 CSV 导出表头中英切换，同步 `PATCH /api/me`。
- **Task W16（F-21，对应 S16）**：`web/src/app/admin/audit/page.tsx` —— 增加操作类型/操作员筛选与 `beforeJson`/`afterJson`/操作原因/金额对比抽屉。
- **Task W17（F-22，对应 S17）**：`web/src/app/counter/orders/page.tsx`、`web/src/app/admin/orders/page.tsx` —— 对 `expired` 订单增加“延迟到账人工恢复收款”弹窗（调用 `/recover-expired`，必填原因）。
- **Task W18（F-23, F-24，对应 S18, S19 + 修复 plan 任务 3 迁移 `014`）**：`web/src/app/counter/page.tsx` —— 终端命名（`POS-1`）、QR 流水参考号 `paymentRef` 输入框（落入修复 plan 任务 3 迁移 `014` 字段）、订阅 `/api/staff/events/{eventID}/stream` SSE 实时刷新，以及 `422 pricing_complexity_exceeded` 引导拆单或降级按单品结算。

---

## 四、Phase 3：P2 任务拆解（`F-27`）

- **Task W19（F-27）**：`web/src/app/admin/reports/page.tsx` —— 多币种日报顶部按币种并列展示总览；清理 `web/src/lib/errors.ts` 遗留文案。

---

## 五、本地全链路验收步骤

1. 启动本机依赖：`make compose && make seed-local`
2. 启动本机 Go API：`cd server && make dev`（含 `STALLINK_OFFLINE_HMAC_SECRET`）
3. 启动本机 Web：`cd web && npm run dev`
4. 运行全量自动化验证：`cd web && npm test && npm run lint && npx tsc --noEmit && npm run build && npm run test:e2e`
> **⚠️ 需要 Kel 手动确认**：严禁将代理或环境配置指向非本机云端 API/远程数据库进行测试；如需对接远程预发布环境，必须先取得 Kel 手动确认。
