import { formatMoney } from "../pricing";
import type { CurrencyRow, Report } from "./report-types";

// A library-neutral description of what the sales-report PDF shows. It mirrors
// the on-screen DailyReportSheet; renderers only lay it out. A single-day
// export is a range of one day, so a multi-day PDF only passes more days.

export const ALL_CURRENCIES_LABEL = "全部币种";

export type ReportDayInput = { date: string; report: Report | null };

export type ReportMetric = { label: string; value: string; note: string; lead: boolean };
export type ReportRow = { cells: string[]; amount: string };
export type ReportGroup = { currency: string; rows: ReportRow[] };
export type ReportBlock = { title: string; caption: string; groups: ReportGroup[]; empty: string };
export type ReportSummary = { title: string; caption: string; rows: ReportRow[] };

export type ReportDaySection =
  | { date: string; missing: true; message: string }
  | {
    date: string;
    missing: false;
    eyebrow: string;
    heading: string;
    generated: string;
    stamp: string;
    metrics: ReportMetric[];
    currencySummary: ReportSummary | null;
    blocks: ReportBlock[];
  };

export type ReportDocument = {
  title: string;
  /** Currency the dropdown selected; empty means all currencies. */
  currencyFilter: string;
  /** Label used in the file name and heading: a code or 全部币种. */
  currencyLabel: string;
  dates: string[];
  days: ReportDaySection[];
};

export type BuildReportDocumentInput = {
  days: ReportDayInput[];
  currency: string;
  formatGeneratedAt?: (iso: string) => string;
};

function defaultGeneratedAt(iso: string): string {
  const date = new Date(iso);
  return Number.isNaN(date.getTime()) ? iso : date.toLocaleString("zh-CN");
}

export function overShortLabel(currency: string, minor: number, cashTenderedOrders?: number): string {
  // K-Q18 = A：有字段且为 0 时显示「—（未录实收）」；大于 0 时照常显示长款 / 短款 / 持平。
  // 字段不存在（旧快照兜底）按旧逻辑显示数值，不当成 0。
  if (cashTenderedOrders === 0) return "—（未录实收）";
  if (minor === 0) return `${formatMoney(currency, 0)} 持平`;
  if (minor > 0) return `长款 ${formatMoney(currency, minor)}`;
  return `短款 ${formatMoney(currency, -minor)}`;
}

function groupByCurrency<T extends { currency: string }>(rows: T[], currency: string): [string, T[]][] {
  const map = new Map<string, T[]>();
  rows.forEach((row) => {
    if (currency && row.currency !== currency) return;
    const list = map.get(row.currency) ?? [];
    list.push(row);
    map.set(row.currency, list);
  });
  return [...map.entries()];
}

function zeroRow(currency: string): CurrencyRow {
  return { currency, orders: 0, paidOrders: 0, revenueMinor: 0, discountMinor: 0, refundMinor: 0, cashOverShortMinor: 0, cashTenderedOrders: 0 };
}

function metricsFor(report: Report, currency: string): ReportMetric[] {
  const headline = report.currency;
  const target = currency || headline;
  if (target === headline) {
    return [
      { label: `收款总额（${headline}）`, value: formatMoney(headline, report.revenueMinor), note: `${report.paidOrders} 笔已收款 · ${report.pendingOrders} 笔待付款`, lead: true },
      { label: "订单数", value: String(report.orders), note: `已关闭 ${report.closedOrders} · 退款 ${report.refundOrders}`, lead: false },
      { label: "售出件数", value: String(report.itemsSold), note: `平均每单 ${formatMoney(headline, report.averageMinor)}`, lead: false },
      { label: `优惠金额（${headline}）`, value: formatMoney(headline, report.discountMinor), note: "数量价与套餐优惠合计", lead: false },
      { label: `待退金额（${headline}）`, value: formatMoney(headline, report.refundMinor), note: "待确认退款的金额", lead: false },
      { label: `现金长短款（${headline}）`, value: overShortLabel(headline, report.cashOverShortMinor, report.cashTenderedOrders), note: "正数长款 · 负数短款", lead: false },
    ];
  }
  // Snapshots keep per-currency money and order counts, but status counts and
  // items sold only for the headline currency. Never borrow those numbers.
  const row = report.currencies?.find((entry) => entry.currency === target) ?? zeroRow(target);
  const average = row.paidOrders > 0 ? Math.round(row.revenueMinor / row.paidOrders) : 0;
  return [
    { label: `收款总额（${target}）`, value: formatMoney(target, row.revenueMinor), note: `${row.paidOrders} 笔已收款`, lead: true },
    { label: "订单数", value: String(row.orders), note: `按 ${target} 统计`, lead: false },
    { label: "售出件数", value: "—", note: `件数仅统计主币种 ${headline} · 平均每单 ${formatMoney(target, average)}`, lead: false },
    { label: `优惠金额（${target}）`, value: formatMoney(target, row.discountMinor), note: "数量价与套餐优惠合计", lead: false },
    { label: `待退金额（${target}）`, value: formatMoney(target, row.refundMinor), note: "待确认退款的金额", lead: false },
    { label: `现金长短款（${target}）`, value: overShortLabel(target, row.cashOverShortMinor, row.cashTenderedOrders), note: "正数长款 · 负数短款", lead: false },
  ];
}

function summaryFor(report: Report): ReportSummary | null {
  const rows = report.currencies ?? [];
  if (rows.length <= 1) return null;
  // Kel: the per-currency summary always lists every currency, filter or not.
  return {
    title: "分币种汇总",
    caption: "各币种独立合计，不跨币种相加",
    rows: rows.map((row) => ({
      cells: [row.currency, `${row.orders} 笔 · 已收款 ${row.paidOrders}`, `退款 ${formatMoney(row.currency, row.refundMinor)}`, overShortLabel(row.currency, row.cashOverShortMinor, row.cashTenderedOrders)],
      amount: formatMoney(row.currency, row.revenueMinor),
    })),
  };
}

function blocksFor(report: Report, currency: string): ReportBlock[] {
  return [
    {
      title: "活动与场次",
      caption: "按币种分组",
      empty: "这一天没有活动成交。",
      groups: groupByCurrency(report.events ?? [], currency).map(([code, rows]) => ({
        currency: code,
        rows: rows.map((event) => ({ cells: [event.name, `${event.orders} 笔`, `优惠 ${formatMoney(code, event.discountMinor)}`], amount: formatMoney(code, event.revenueMinor) })),
      })),
    },
    {
      title: "热销商品",
      caption: "按币种分组",
      empty: "这一天没有售出商品。",
      groups: groupByCurrency(report.topGoods ?? [], currency).map(([code, rows]) => ({
        currency: code,
        rows: rows.map((good, index) => ({ cells: [`${index + 1}. ${good.name}`, `${good.quantity} 件`], amount: formatMoney(code, good.revenueMinor) })),
      })),
    },
    {
      title: "收款方式",
      caption: "按币种分组",
      empty: "这一天没有收款记录。",
      groups: groupByCurrency(report.payCodes ?? [], currency).map(([code, rows]) => ({
        currency: code,
        rows: rows.map((payCode) => ({ cells: [payCode.name, `${payCode.orders} 笔`], amount: formatMoney(code, payCode.revenueMinor) })),
      })),
    },
  ];
}

function sourceLabel(source: string): string {
  if (source === "manual") return " · 手动生成";
  if (source === "scheduled") return " · 定时生成";
  return "";
}

function currencyLabelFor(currency: string, days: ReportDayInput[]): string {
  if (currency) return currency;
  const codes = new Set<string>();
  days.forEach(({ report }) => {
    if (!report) return;
    if (report.currencies?.length) report.currencies.forEach((row) => codes.add(row.currency));
    else codes.add(report.currency);
  });
  return codes.size === 1 ? [...codes][0] : ALL_CURRENCIES_LABEL;
}

export function buildReportDocument({ days, currency, formatGeneratedAt = defaultGeneratedAt }: BuildReportDocumentInput): ReportDocument {
  if (days.length === 0) throw new Error("导出日报至少需要一天。");
  const sorted = [...days].sort((a, b) => a.date.localeCompare(b.date));
  for (let index = 1; index < sorted.length; index++) {
    if (sorted[index].date === sorted[index - 1].date) throw new Error(`导出日期重复：${sorted[index].date}`);
  }
  const dates = sorted.map(({ date }) => date);
  const first = dates[0];
  const last = dates[dates.length - 1];
  return {
    title: first === last ? `销售日报 · ${first}` : `销售日报 · ${first} 至 ${last}`,
    currencyFilter: currency,
    currencyLabel: currencyLabelFor(currency, sorted),
    dates,
    days: sorted.map(({ date, report }): ReportDaySection => {
      if (!report) return { date, missing: true, message: "这一天还没有生成日报。" };
      return {
        date,
        missing: false,
        eyebrow: `${report.date} · ${currency || report.currency}`,
        heading: "当日订单与收款",
        generated: `生成于 ${formatGeneratedAt(report.generatedAt)}${sourceLabel(report.source)}`,
        stamp: report.refundOrders > 0 ? "含退款订单" : "无退款",
        metrics: metricsFor(report, currency),
        currencySummary: summaryFor(report),
        blocks: blocksFor(report, currency),
      };
    }),
  };
}
