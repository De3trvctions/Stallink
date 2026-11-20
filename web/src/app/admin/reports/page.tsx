"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { browserJSON } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { formatMoney } from "@/lib/pricing";
import { messageClass, useMessage } from "@/lib/message";
import SelectMenu from "@/components/select-menu";

type CurrencyRow = {
  currency: string; orders: number; paidOrders: number; revenueMinor: number;
  discountMinor: number; refundMinor: number; cashOverShortMinor: number;
};
type EventRow = { eventId: string; name: string; currency: string; orders: number; revenueMinor: number; discountMinor: number };
type GoodRow = { currency: string; name: string; quantity: number; revenueMinor: number };
type PayCodeRow = { currency: string; name: string; orders: number; revenueMinor: number };
type Report = {
  date: string; generatedAt: string; source: string; currency: string;
  orders: number; paidOrders: number; pendingOrders: number; closedOrders: number; refundOrders: number;
  itemsSold: number; revenueMinor: number; discountMinor: number; refundMinor: number; cashOverShortMinor: number;
  averageMinor: number; currencies: CurrencyRow[]; events: EventRow[]; topGoods: GoodRow[]; payCodes: PayCodeRow[];
};

function groupByCurrency<T extends { currency: string }>(rows: T[]): [string, T[]][] {
  const map = new Map<string, T[]>();
  rows.forEach((row) => {
    const list = map.get(row.currency) ?? [];
    list.push(row);
    map.set(row.currency, list);
  });
  return [...map.entries()];
}

function overShortLabel(currency: string, minor: number): string {
  if (minor === 0) return `${formatMoney(currency, 0)} 持平`;
  if (minor > 0) return `长款 ${formatMoney(currency, minor)}`;
  return `短款 ${formatMoney(currency, -minor)}`;
}

function todayString() {
  return new Intl.DateTimeFormat("en-CA", { year: "numeric", month: "2-digit", day: "2-digit" }).format(new Date());
}

export default function AdminReportsPage() {
  // 默认取客户端本地当天；服务端与客户端若跨时区/午夜边界出现差异，
  // 由下方日期输入框的 suppressHydrationWarning 单独容忍这一属性，不影响其余树。
  const [date, setDate] = useState(() => todayString());
  const [currency, setCurrency] = useState("");
  const [report, setReport] = useState<Report | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [exporting, setExporting] = useState(false);
  const [generating, setGenerating] = useState(false);
  const requestSequence = useRef(0);
  const { message, setError, setSuccess } = useMessage();

  const load = useCallback(async () => {
    const request = ++requestSequence.current;
    setLoading(true);
    setLoadError("");
    setReport(null);
    try {
      const query = new URLSearchParams({ date });
      const res = await browserJSON(`/api/admin/daily-reports?${query.toString()}`);
      if (request !== requestSequence.current) return;
      if (!res.ok) { setLoadError(describeRequest(res)); return; }
      setReport(res.data as unknown as Report);
    } catch {
      if (request === requestSequence.current) setLoadError("无法连接服务，请检查网络后重试。");
    } finally {
      if (request === requestSequence.current) setLoading(false);
    }
  }, [date]);

  useEffect(() => {
    const sequence = requestSequence;
    const timer = window.setTimeout(() => { void load(); }, 0);
    return () => { window.clearTimeout(timer); sequence.current++; };
  }, [load]);

  async function generate() {
    const request = ++requestSequence.current;
    setGenerating(true);
    setError(null);
    try {
      const res = await browserJSON("/api/admin/daily-reports", "POST", { date });
      if (request !== requestSequence.current) return;
      if (!res.ok) { setError(describeRequest(res)); return; }
      setCurrency("");
      setReport(res.data as unknown as Report);
      setLoadError("");
      setSuccess("日报已生成。");
    } finally {
      setGenerating(false);
    }
  }

  async function exportCsv() {
    if (!report || exporting) return;
    setExporting(true);
    setError(null);
    try {
      const query = new URLSearchParams({ date });
      if (currency || report.currency) query.set("currency", currency || report.currency);
      const res = await fetch(`/api/admin/daily-reports/export?${query.toString()}`, { credentials: "include" });
      if (!res.ok) {
        const text = await res.text().catch(() => "");
        let code = "";
        try { code = String(JSON.parse(text).error ?? ""); } catch { /* 非 JSON 响应 */ }
        setError(code ? describeRequest({ category: res.status >= 500 ? "server" : "client", status: res.status, errorCode: code }) : "导出失败，请稍后重试。");
        return;
      }
      const blob = await res.blob();
      const disposition = res.headers.get("content-disposition") ?? "";
      const match = /filename="?([^";]+)"?/.exec(disposition);
      const filename = match?.[1] ?? `daily-report-${date}-${report.currency}.csv`;
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = filename;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
      setSuccess("CSV 已开始下载。");
    } catch {
      setError("无法连接服务，导出失败，请重试。");
    } finally {
      setExporting(false);
    }
  }

  const currencyOptions = report?.currencies.map((row) => ({ value: row.currency, label: row.currency })) ?? [];

  return (
    <main className="workspace-page report-page">
      <header className="workspace-page__heading">
        <p className="eyebrow">日报</p>
        <h1>当日经营汇总</h1>
        <p>按币种分别统计，不同币种之间不做合计。</p>
      </header>

      <div className="report-toolbar">
        <label className="field"><span>日期</span><input type="date" value={date} suppressHydrationWarning disabled={generating} onChange={(event) => { requestSequence.current++; setCurrency(""); setDate(event.target.value); }} /></label>
        <SelectMenu
          value={currency}
          onChange={setCurrency}
          ariaLabel="选择导出币种"
          placeholder="导出主币种"
          options={[{ value: "", label: "导出主币种" }, ...currencyOptions]}
        />
        <div className="report-toolbar__actions">
          <button type="button" className="button button--outline" disabled={generating} onClick={() => void load()}>刷新</button>
          <button type="button" className="button button--outline" disabled={generating || loading} onClick={() => void generate()}>{generating ? "正在生成…" : "生成日报"}</button>
          <button type="button" className="button button--dark" disabled={loading || !report || exporting} onClick={() => void exportCsv()}>
            {exporting ? "正在导出…" : "导出 CSV"}
          </button>
        </div>
      </div>

      {message ? <p className={messageClass(message)} role="status">{message.text}</p> : null}

      {loading ? <p className="empty-state">正在读取日报…</p> : loadError ? (
        <p className="empty-state" role="alert">
          <span className="empty-state__mark" aria-hidden="true">!</span>
          <span>{loadError}</span>
          <button type="button" className="button button--outline" onClick={() => void load()}>重试</button>
        </p>
      ) : !report ? (
        <p className="empty-state"><span className="empty-state__mark" aria-hidden="true">○</span><span>这一天还没有生成日报。</span></p>
      ) : (
        <section className="report-sheet" aria-label="日报内容">
          <div className="report-sheet__head">
            <div>
              <p className="eyebrow">{report.date} · {report.currency}</p>
              <h2>当日订单与收款</h2>
              <p>生成于 {new Date(report.generatedAt).toLocaleString("zh-CN")}，来源：{report.source === "live" ? "现场流水" : "历史归档"}。</p>
            </div>
            <span className="report-sheet__stamp">{report.refundOrders > 0 ? "含退款订单" : "无退款"}</span>
          </div>

          <div className="report-hero">
            <div className="report-metric report-metric--lead">
              <span>收款总额（{report.currency}）</span>
              <strong>{formatMoney(report.currency, report.revenueMinor)}</strong>
              <small>{report.paidOrders} 笔已收款 · {report.pendingOrders} 笔待付款</small>
            </div>
            <div className="report-metric">
              <span>订单数</span>
              <strong>{report.orders}</strong>
              <small>已关闭 {report.closedOrders} · 退款 {report.refundOrders}</small>
            </div>
            <div className="report-metric">
              <span>售出件数</span>
              <strong>{report.itemsSold}</strong>
              <small>平均每单 {formatMoney(report.currency, report.averageMinor)}</small>
            </div>
            <div className="report-metric">
              <span>优惠金额（{report.currency}）</span>
              <strong>{formatMoney(report.currency, report.discountMinor)}</strong>
              <small>数量价与套餐优惠合计</small>
            </div>
            <div className="report-metric">
              <span>待退金额（{report.currency}）</span>
              <strong>{formatMoney(report.currency, report.refundMinor)}</strong>
              <small>待确认退款的金额</small>
            </div>
            <div className="report-metric">
              <span>现金长短款（{report.currency}）</span>
              <strong>{overShortLabel(report.currency, report.cashOverShortMinor)}</strong>
              <small>正数长款 · 负数短款</small>
            </div>
          </div>

          {report.currencies.length > 1 ? (
            <section className="report-block" aria-label="分币种汇总">
              <div className="report-block__head"><h3>分币种汇总</h3><span>各币种独立合计，不跨币种相加</span></div>
              <ul className="report-table">
                {report.currencies.map((row) => (
                  <li key={row.currency}>
                    <span>{row.currency}</span>
                    <span>{row.orders} 笔 · 已收款 {row.paidOrders}</span>
                    <span>退款 {formatMoney(row.currency, row.refundMinor)}</span>
                    <span>{overShortLabel(row.currency, row.cashOverShortMinor)}</span>
                    <strong>{formatMoney(row.currency, row.revenueMinor)}</strong>
                  </li>
                ))}
              </ul>
            </section>
          ) : null}

          <div className="report-columns">
            <section className="report-block" aria-label="活动与场次">
              <div className="report-block__head"><h3>活动与场次</h3><span>按币种分组</span></div>
              {groupByCurrency(report.events).map(([currency, rows]) => (
                <div className="report-currency-group" key={currency}>
                  <p className="report-currency-group__label">{currency}</p>
                  <ul className="report-table">
                    {rows.map((event) => (
                      <li key={event.eventId}>
                        <span>{event.name}</span>
                        <span>{event.orders} 笔</span>
                        <span>优惠 {formatMoney(currency, event.discountMinor)}</span>
                        <strong>{formatMoney(currency, event.revenueMinor)}</strong>
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
              {report.events.length === 0 ? <p className="report-empty">这一天没有活动成交。</p> : null}
            </section>

            <section className="report-block" aria-label="热销商品">
              <div className="report-block__head"><h3>热销商品</h3><span>按币种分组</span></div>
              {groupByCurrency(report.topGoods).map(([currency, rows]) => (
                <div className="report-currency-group" key={currency}>
                  <p className="report-currency-group__label">{currency}</p>
                  <ul className="report-rank">
                    {rows.map((good, index) => (
                      <li key={`${currency}-${good.name}-${index}`}>
                        <span>{index + 1}. {good.name}</span>
                        <span>{good.quantity} 件</span>
                        <strong>{formatMoney(currency, good.revenueMinor)}</strong>
                      </li>
                    ))}
                  </ul>
                </div>
              ))}
              {report.topGoods.length === 0 ? <p className="report-empty">这一天没有售出商品。</p> : null}
            </section>
          </div>

          <section className="report-block" aria-label="收款方式">
            <div className="report-block__head"><h3>收款方式</h3><span>按币种分组</span></div>
            {groupByCurrency(report.payCodes).map(([currency, rows]) => (
              <div className="report-currency-group" key={currency}>
                <p className="report-currency-group__label">{currency}</p>
                <ul className="report-table">
                  {rows.map((code) => (
                    <li key={`${currency}-${code.name}`}>
                      <span>{code.name}</span>
                      <span>{code.orders} 笔</span>
                      <strong>{formatMoney(currency, code.revenueMinor)}</strong>
                    </li>
                  ))}
                </ul>
              </div>
            ))}
            {report.payCodes.length === 0 ? <p className="report-empty">这一天没有收款记录。</p> : null}
          </section>
        </section>
      )}
    </main>
  );
}
