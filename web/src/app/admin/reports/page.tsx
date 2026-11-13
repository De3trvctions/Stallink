"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { browserJSON } from "@/lib/api";
import { formatMoney } from "@/lib/pricing";

type CurrencyRow = { currency: string; orders: number; paidOrders: number; revenueMinor: number; discountMinor: number };
type EventRow = { eventId: string; name: string; orders: number; revenueMinor: number };
type GoodRow = { name: string; quantity: number; revenueMinor: number };
type PayCodeRow = { name: string; orders: number; revenueMinor: number };
type Report = {
  date: string; generatedAt: string; source: "manual" | "scheduled";
  currency: string; orders: number; paidOrders: number; pendingOrders: number;
  closedOrders: number; refundOrders: number; itemsSold: number;
  revenueMinor: number; discountMinor: number; averageMinor: number;
  currencies: CurrencyRow[]; events: EventRow[]; topGoods: GoodRow[]; payCodes: PayCodeRow[];
};

function todayInKualaLumpur() {
  return new Intl.DateTimeFormat("en-CA", { timeZone: "Asia/Kuala_Lumpur" }).format(new Date());
}

function displayDate(value: string) {
  const [year, month, day] = value.split("-");
  return `${year} 年 ${Number(month)} 月 ${Number(day)} 日`;
}

function generatedTime(value: string) {
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return new Intl.DateTimeFormat("zh-CN", {
    timeZone: "Asia/Kuala_Lumpur", month: "numeric", day: "numeric",
    hour: "2-digit", minute: "2-digit", hour12: false,
  }).format(date);
}

export default function DailyReportsPage() {
  const router = useRouter();
  const [date, setDate] = useState(todayInKualaLumpur);
  const [report, setReport] = useState<Report | null>(null);
  const [loading, setLoading] = useState(true);
  const [generating, setGenerating] = useState(false);
  const [message, setMessage] = useState("");

  const load = useCallback(async (target: string) => {
    setLoading(true);
    setMessage("");
    try {
      const res = await browserJSON(`/api/admin/daily-reports?date=${encodeURIComponent(target)}`);
      if (res.status === 401 || res.status === 403) {
        router.replace("/login");
        return;
      }
      if (res.status === 404) {
        setReport(null);
        return;
      }
      if (!res.ok) {
        setMessage("日报读取失败，请稍后重试。");
        return;
      }
      setReport(res.data as Report);
    } catch {
      setMessage("无法连接服务，请稍后重试。");
    } finally {
      setLoading(false);
    }
  }, [router]);

  useEffect(() => {
    const timer = window.setTimeout(() => { void load(date); }, 0);
    return () => window.clearTimeout(timer);
  }, [date, load]);

  async function generate() {
    if (generating) return;
    setGenerating(true);
    setMessage("");
    try {
      const res = await browserJSON("/api/admin/daily-reports", "POST", { date });
      if (res.status === 401 || res.status === 403) {
        router.replace("/login");
        return;
      }
      if (!res.ok) {
        setMessage("日报生成失败，请稍后重试。");
        return;
      }
      setReport(res.data as Report);
      setMessage("日报已生成。");
    } catch {
      setMessage("无法连接服务，请稍后重试。");
    } finally {
      setGenerating(false);
    }
  }

  function exportPDF() {
    const sheet = document.querySelector<HTMLElement>(".report-sheet");
    if (!sheet) return;
    const filename = `销售日报-${date}`;
    const pageStyles = Array.from(
      document.querySelectorAll<HTMLLinkElement | HTMLStyleElement>('link[rel="stylesheet"], style'),
    ).map((node) => node.outerHTML).join("\n");
    const printDocument = `<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<base href="${window.location.origin}/">
<title>${filename}</title>
${pageStyles}
<style>
html, body { margin: 0; padding: 0; background: #fff; }
.report-export-document { width: 794px; margin: 0 auto; padding: 12mm 0; }
.report-export-document > h1 {
  margin: 0 0 5mm;
  font-family: var(--font-display);
  font-size: 24pt;
  font-weight: 500;
  letter-spacing: -0.02em;
}
.report-sheet { margin-top: 0; background: #fff; }
@page { size: A4 portrait; margin: 12mm; }
@media print {
  .report-export-document { width: auto; margin: 0; padding: 0; }
}
</style>
</head>
<body>
<main class="report-export-document">
  <h1>销售日报</h1>
  ${sheet.outerHTML}
</main>
</body>
</html>`;
    const frame = document.createElement("iframe");
    frame.title = "销售日报导出";
    frame.setAttribute("aria-hidden", "true");
    frame.style.cssText = "position:fixed;left:-10000px;top:0;width:794px;height:1123px;border:0;opacity:0;pointer-events:none;";
    document.body.appendChild(frame);
    const frameDocument = frame.contentDocument;
    if (!frameDocument) {
      frame.remove();
      window.print();
      return;
    }
    frameDocument.open();
    frameDocument.write(printDocument);
    frameDocument.close();
    window.setTimeout(() => {
      const printWindow = frame.contentWindow;
      if (!printWindow) {
        frame.remove();
        window.print();
        return;
      }
      const cleanup = () => {
        printWindow.removeEventListener("afterprint", cleanup);
        frame.remove();
      };
      printWindow.addEventListener("afterprint", cleanup);
      printWindow.focus();
      printWindow.print();
    }, 800);
  }

  const topCurrency = report && report.currencies.length > 1;
  const discountMinor = report?.discountMinor ?? 0;
  const originalMinor = (report?.revenueMinor ?? 0) + discountMinor;
  const discountRate = originalMinor > 0 ? (discountMinor / originalMinor) * 100 : 0;

  return (
    <main className="report-page">
        <p className="eyebrow">店铺经营</p>
        <h1>销售日报</h1>
        <p>每天 00:10 自动为每家店生成前一天的日报；也可以选日期提前生成，两边用的是同一套统计。</p>

        <div className="report-toolbar">
          <label className="report-date">
            <span>日报日期</span>
            <input type="date" value={date} max={todayInKualaLumpur()} onChange={(change) => setDate(change.target.value)} />
          </label>
          <button className="button button--dark" type="button" onClick={() => void generate()} disabled={generating || loading}>
            {generating ? "正在生成…" : report ? "重新生成这天的日报" : "生成这天的日报"}
          </button>
          <button className="button button--outline" type="button" onClick={() => void load(date)} disabled={loading || generating}>刷新</button>
          <button className="button button--outline" type="button" onClick={exportPDF} disabled={!report || loading || generating} title="在打印窗口选择“另存为 PDF”">下载 PDF</button>
        </div>
        {message ? <p className="form-message" role="status">{message}</p> : null}

        {loading ? <p className="report-loading" role="status">正在读取日报…</p> : !report ? (
          <section className="report-empty">
            <p className="eyebrow">{date}</p>
            <h2>{displayDate(date)} 还没有日报</h2>
            <p>定时任务会在次日 00:10 生成；想现在看，直接生成这一天即可。</p>
            <button className="button button--dark" type="button" onClick={() => void generate()} disabled={generating}>
              {generating ? "正在生成…" : "生成这一天"}
            </button>
          </section>
        ) : (
          <article className="report-sheet">
            <header className="report-sheet__head">
              <div>
                <p className="eyebrow">{report.date} · {report.source === "manual" ? "手动生成" : "定时生成"}</p>
                <h2>{displayDate(report.date)}</h2>
              </div>
              <p className="report-sheet__stamp">{generatedTime(report.generatedAt)} 生成</p>
            </header>

            <section className="report-hero" aria-label="日报总览">
              <div className="report-metric report-metric--lead">
                <span>营业额</span>
                <strong>{formatMoney(report.currency, report.revenueMinor)}</strong>
                <small>{topCurrency ? `主币种 ${report.currency}，其他币种见下方明细` : "已付款与已交付订单"}</small>
              </div>
              <div className="report-metric"><span>已付款订单</span><strong>{report.paidOrders}</strong><small>共 {report.orders} 张单</small></div>
              <div className="report-metric"><span>客单价</span><strong>{formatMoney(report.currency, report.averageMinor)}</strong><small>{topCurrency ? `按主币种 ${report.currency} 的已付款订单` : "按已付款订单"}</small></div>
              <div className="report-metric"><span>售出件数</span><strong>{report.itemsSold}</strong><small>已付款订单内数量</small></div>
            </section>

            <section className="report-discount" aria-label="优惠汇总">
              <div className="report-discount__lead">
                <span className="report-discount__label">共优惠</span>
                <strong className="report-discount__amount">−{formatMoney(report.currency, discountMinor)}</strong>
                <p className="report-discount__note">
                  {topCurrency ? `仅计主币种 ${report.currency} · 已付款订单原价与成交价差额` : "已付款订单 · 原价与成交价差额"}
                </p>
              </div>
              <div className="report-discount__figures">
                <div>
                  <span>原价合计</span>
                  <strong>{formatMoney(report.currency, originalMinor)}</strong>
                </div>
                <div>
                  <span>优惠率</span>
                  <strong>{discountRate.toFixed(2)}%</strong>
                </div>
                <div className="report-discount__meter" aria-hidden="true">
                  <span style={{ width: `${Math.min(100, Math.max(0, discountRate))}%` }} />
                </div>
              </div>
            </section>

            <section className="report-status" aria-label="订单状态">
              <p><span>待付款</span><strong>{report.pendingOrders}</strong></p>
              <p><span>已取消 / 过期</span><strong>{report.closedOrders}</strong></p>
              <p><span>需退款</span><strong>{report.refundOrders}</strong></p>
            </section>

            {topCurrency ? (
              <section className="report-block">
                <header className="report-block__head"><h3>币种</h3><span>按营业额排序</span></header>
                <ul className="report-table report-table--four">
                  {report.currencies.map((row) => (
                    <li key={row.currency}>
                      <span>{row.currency}</span>
                      <span>{row.orders} 单 · 已付 {row.paidOrders}</span>
                      <span className="report-table__discount">优惠 −{formatMoney(row.currency, row.discountMinor ?? 0)}</span>
                      <strong>{formatMoney(row.currency, row.revenueMinor)}</strong>
                    </li>
                  ))}
                </ul>
              </section>
            ) : null}

            <div className="report-columns">
              <section className="report-block">
                <header className="report-block__head"><h3>按活动</h3><span>{report.events.length} 场</span></header>
                {report.events.length === 0 ? <p className="report-block__empty">这天还没有订单。</p> : (
                  <ul className="report-table">
                    {report.events.map((row) => (
                      <li key={row.eventId}>
                        <span>{row.name}</span>
                        <span>{row.orders} 单</span>
                        <strong>{formatMoney(report.currency, row.revenueMinor)}</strong>
                      </li>
                    ))}
                  </ul>
                )}
              </section>
              <section className="report-block">
                <header className="report-block__head"><h3>卖得最好</h3><span>前 5 件</span></header>
                {report.topGoods.length === 0 ? <p className="report-block__empty">暂无已付款商品。</p> : (
                  <ul className="report-rank">
                    {report.topGoods.map((row, index) => (
                      <li key={`${row.name}-${index}`}>
                        <em>{String(index + 1).padStart(2, "0")}</em>
                        <span>{row.name}<small>{row.quantity} 件</small></span>
                        <strong>{formatMoney(report.currency, row.revenueMinor)}</strong>
                      </li>
                    ))}
                  </ul>
                )}
              </section>
            </div>

            <section className="report-block">
              <header className="report-block__head"><h3>收款方式</h3><span>已付款订单</span></header>
              {report.payCodes.length === 0 ? <p className="report-block__empty">暂无已付款订单。</p> : (
                <ul className="report-table">
                  {report.payCodes.map((row) => (
                    <li key={row.name}>
                      <span>{row.name}</span>
                      <span>{row.orders} 单</span>
                      <strong>{formatMoney(report.currency, row.revenueMinor)}</strong>
                    </li>
                  ))}
                </ul>
              )}
            </section>
          </article>
        )}
    </main>
  );
}
