"use client";

import { useCallback, useEffect, useRef, useState, type FormEvent } from "react";
import { browserJSON } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { formatMoney } from "@/lib/pricing";
import { messageClass, useMessage } from "@/lib/message";
import SelectMenu from "@/components/select-menu";
import { inclusiveReportDates, MAX_REPORT_DAYS } from "./report-dates";
import "./reports.css";

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
type ReportDay = {
  date: string;
  report: Report | null;
  error?: string;
};

const REPORT_QUERY_BATCH_SIZE = 4;

function localToday() {
  const now = new Date();
  const month = String(now.getMonth() + 1).padStart(2, "0");
  const day = String(now.getDate()).padStart(2, "0");
  return `${now.getFullYear()}-${month}-${day}`;
}

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

async function readReportDay(date: string): Promise<ReportDay> {
  const query = new URLSearchParams({ date });
  const res = await browserJSON(`/api/admin/daily-reports?${query}`);
  if (res.ok && res.data.date === date) return { date, report: res.data as unknown as Report };
  if (res.status === 404 && res.errorCode === "report_not_generated") return { date, report: null };
  return { date, report: null, error: res.ok ? "日报数据无法读取，请重试。" : describeRequest(res) };
}

export default function AdminReportsPage() {
  const [start, setStart] = useState("");
  const [end, setEnd] = useState("");
  const [selectedDate, setSelectedDate] = useState("");
  const [currency, setCurrency] = useState("");
  const [days, setDays] = useState<ReportDay[]>([]);
  const [loading, setLoading] = useState(true);
  const [exporting, setExporting] = useState(false);
  const [generating, setGenerating] = useState(false);
  const requestSequence = useRef(0);
  const startInput = useRef<HTMLInputElement>(null);
  const endInput = useRef<HTMLInputElement>(null);
  const { message, setError, setSuccess } = useMessage();

  const load = useCallback(async (dates: string[]) => {
    const request = ++requestSequence.current;
    setLoading(true);
    setError(null);
    setCurrency("");
    const nextDays: ReportDay[] = [];
    for (let index = 0; index < dates.length; index += REPORT_QUERY_BATCH_SIZE) {
      const batch = dates.slice(index, index + REPORT_QUERY_BATCH_SIZE);
      const results = await Promise.all(batch.map(async (date) => {
        try {
          return await readReportDay(date);
        } catch {
          return { date, report: null, error: "无法连接服务，请检查网络后重试。" };
        }
      }));
      if (request !== requestSequence.current) return;
      nextDays.push(...results);
    }
    setDays(nextDays);
    setSelectedDate((previous) => {
      if (nextDays.some(({ date }) => date === previous)) return previous;
      return nextDays.find((day) => day.report)?.date ?? dates[0];
    });
    setLoading(false);
  }, [setError]);

  useEffect(() => {
    const sequence = requestSequence;
    const timer = window.setTimeout(() => {
      const today = localToday();
      setStart(today);
      setEnd(today);
      void load([today]);
    }, 0);
    return () => { window.clearTimeout(timer); sequence.current++; };
  }, [load]);

  const selectedDay = days.find((day) => day.date === selectedDate);
  const report = selectedDay?.report ?? null;
  const busy = loading || generating || exporting;
  const range = days.length > 1;

  function search(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const from = startInput.current;
    const to = endInput.current;
    if (busy || !from || !to) return;
    const result = inclusiveReportDates(from.valueAsNumber, to.valueAsNumber);
    if (result.error !== undefined) {
      to.setCustomValidity(result.error);
      to.reportValidity();
      return;
    }
    void load(result.dates);
  }

  function selectDay(date: string) {
    setSelectedDate(date);
    setCurrency("");
    setError(null);
  }

  async function generate() {
    if (!selectedDay || selectedDay.error || busy) return;
    const date = selectedDay.date;
    const request = ++requestSequence.current;
    setGenerating(true);
    setError(null);
    try {
      const res = await browserJSON("/api/admin/daily-reports", "POST", { date });
      if (request !== requestSequence.current) return;
      if (!res.ok) { setError(describeRequest(res)); return; }
      setCurrency("");
      if (res.data.date !== date) { setError("生成的日报数据无法读取，请重新查询。"); return; }
      const generated = res.data as unknown as Report;
      setDays((current) => current.map((day) => day.date === date ? { date, report: generated } : day));
      setSuccess(`${date} 的日报已生成。`);
    } finally {
      setGenerating(false);
    }
  }

  async function exportCsv() {
    if (!report || busy) return;
    const { date } = report;
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
      const filename = match?.[1] ?? `daily-report-${date}-${currency || report.currency}.csv`;
      const url = URL.createObjectURL(blob);
      const anchor = document.createElement("a");
      anchor.href = url;
      anchor.download = filename;
      document.body.appendChild(anchor);
      anchor.click();
      anchor.remove();
      URL.revokeObjectURL(url);
      setSuccess(`${date} 的 CSV 已开始下载。`);
    } catch {
      setError("无法连接服务，导出失败，请重试。");
    } finally {
      setExporting(false);
    }
  }

  const currencyOptions = (report?.currencies ?? []).map((row) => ({ value: row.currency, label: row.currency }));
  const retry = () => void load(days.map(({ date }) => date));

  return (
    <main className="workspace-page report-page">
      <header className="workspace-page__heading">
        <h1>经营日报</h1>
      </header>

      <search className="report-range-search" aria-label="查询日报">
        <form className="report-toolbar report-range-filters" onSubmit={search}>
          <div className="field">
            <label htmlFor="report-start">起始日期</label>
            <input id="report-start" ref={startInput} name="start" type="date" required
              min="0001-01-01" max="9999-12-31" value={start} disabled={busy} aria-describedby="report-date-hint"
              onChange={(event) => { endInput.current?.setCustomValidity(""); setStart(event.target.value); }} />
          </div>
          <div className="field">
            <label htmlFor="report-end">结束日期</label>
            <input id="report-end" ref={endInput} name="end" type="date" required
              min="0001-01-01" max="9999-12-31" value={end} disabled={busy} aria-describedby="report-date-hint"
              onChange={(event) => { event.currentTarget.setCustomValidity(""); setEnd(event.target.value); }} />
          </div>
          <button type="submit" className="button button--dark" disabled={busy}>{loading ? "正在查询…" : "查询日报"}</button>
        </form>
        <p id="report-date-hint" className="field-hint">含起止日期，每次最多查询 {MAX_REPORT_DAYS} 天。</p>
      </search>

      {loading ? <p className="empty-state" role="status">正在读取日报…</p> : null}

      {!loading && range ? (
        <section className="report-range-results" aria-labelledby="report-range-heading">
          <div className="report-range-heading">
            <h2 id="report-range-heading">逐日收款</h2>
            <p role="status">{days[days.length - 1].date} 至 {days[0].date} · {days.filter((day) => day.report).length} / {days.length} 天有日报</p>
          </div>
          <ul className="report-range-days">
            {days.map((day) => (
              <li key={day.date}>
                <button className="report-range-day" type="button" aria-pressed={day.date === selectedDate}
                  aria-controls="report-detail" disabled={busy} onClick={() => selectDay(day.date)}>
                  <time dateTime={day.date}>{day.date}</time>
                  {day.error ? <span>查询失败</span> : !day.report ? <span>未生成</span> : (
                    <span className="report-range-amounts">
                      {day.report.currencies?.length ? day.report.currencies.map((row) => (
                        <span key={row.currency}>{row.currency} {formatMoney(row.currency, row.revenueMinor)}</span>
                      )) : <span>{day.report.currency} {formatMoney(day.report.currency, day.report.revenueMinor)}</span>}
                    </span>
                  )}
                  <span className="report-range-day-action">{day.date === selectedDate ? "当前日报" : "查看"}</span>
                </button>
              </li>
            ))}
          </ul>
        </section>
      ) : null}

      {!loading && selectedDay ? (
        <div id="report-detail">
          <div className="report-toolbar report-range-detail-toolbar">
            <strong><time dateTime={selectedDay.date}>{selectedDay.date}</time> 日报</strong>
            <div className="report-toolbar__actions">
              {!selectedDay.error ? (
                <button type="button" className="button button--outline" disabled={busy} onClick={() => void generate()}>
                  {generating ? "正在生成…" : report ? "重新生成日报" : "生成日报"}
                </button>
              ) : null}
              {report ? (
                <>
                  {currencyOptions.length > 1 ? (
                    <SelectMenu
                      value={currency}
                      onChange={setCurrency}
                      ariaLabel="选择导出明细币种"
                      placeholder="导出主币种明细"
                      options={[{ value: "", label: "导出主币种明细" }, ...currencyOptions]}
                      disabled={busy}
                    />
                  ) : null}
                  <button type="button" className="button button--outline" disabled={busy} onClick={() => void exportCsv()}>
                    {exporting ? "正在导出…" : "导出当日 CSV"}
                  </button>
                </>
              ) : null}
            </div>
          </div>
          {message ? <p className={messageClass(message)} role="status">{message.text}</p> : null}
          {selectedDay.error ? (
            <div className="empty-state">
              <span className="empty-state__mark" aria-hidden="true">!</span>
              <p role="alert">{selectedDay.error}</p>
              <button type="button" className="button button--outline" disabled={busy} onClick={retry}>重试查询</button>
            </div>
          ) : !report ? (
            <p className="empty-state" role="status">这一天还没有生成日报。</p>
          ) : (
            <DailyReportSheet report={report} />
          )}
        </div>
      ) : null}
    </main>
  );
}

function DailyReportSheet({ report }: { report: Report }) {
  return (
    <section className="report-sheet" aria-label="日报内容">
      <div className="report-sheet__head">
        <div>
          <p className="eyebrow">{report.date} · {report.currency}</p>
          <h2>当日订单与收款</h2>
          <p>生成于 {new Date(report.generatedAt).toLocaleString("zh-CN")}
            {report.source === "manual" ? " · 手动生成" : report.source === "scheduled" ? " · 定时生成" : ""}</p>
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

      {(report.currencies?.length ?? 0) > 1 ? (
        <section className="report-block" aria-label="分币种汇总">
          <div className="report-block__head"><h3>分币种汇总</h3><span>各币种独立合计，不跨币种相加</span></div>
          <ul className="report-table">
            {report.currencies?.map((row) => (
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
  );
}
