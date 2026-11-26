/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const globals = fs.readFileSync('src/app/globals.css', 'utf8');
const expo = fs.readFileSync('src/app/exhibition-workspace.css', 'utf8');
const page = fs.readFileSync('src/app/admin/reports/page.tsx', 'utf8');

test('five metric cards, no 现金长短款 (C-2 = A, Kel 13:55)', () => {
  assert.equal((page.match(/className="report-metric( report-metric--lead)?"/g) || []).length, 5);
  assert.doesNotMatch(page, /现金长短款|长短款/);
});

test('D1 guards stay in place (already fixed, prevent regression)', () => {
  assert.match(expo, /--workspace-rail:\s*0px/);
  assert.match(globals, /\.report-metric\s*{\s*min-width:\s*0/);
  assert.match(globals, /\.report-metric strong\s*{[^}]*font-size:\s*clamp\(/);
});

test('narrow metric grid: 2 columns, lead card spans both (already 2 columns)', () => {
  assert.match(globals, /@media \(max-width: 720px\)\s*{\s*\.report-hero\s*{\s*grid-template-columns:\s*1fr 1fr/);
  assert.match(globals, /\.report-metric--lead\s*{[^}]*grid-column:\s*span 2/);
});

test('desktop metric grid is 3 columns, lead spans 2 → 2+1 / 3 (to do)', () => {
  assert.match(globals, /\.report-hero\s*{[^}]*grid-template-columns:\s*repeat\(3,\s*minmax\(0,\s*1fr\)\)/);
});

const reportsCss = fs.readFileSync('src/app/admin/reports/reports.css', 'utf8');

test('C12 eyebrow 经营 above 销售日报', () => {
  assert.match(page, /<p className="eyebrow">经营<\/p>\s*<h1>销售日报<\/h1>/);
});

test('C13 toolbar has filters/actions groups and no viewport-only nowrap', () => {
  assert.equal((page.match(/className="report-toolbar__group"/g) || []).length >= 2, true);
  assert.doesNotMatch(reportsCss, /min-width:\s*70rem/);
  assert.match(reportsCss, /\.report-range-detail-toolbar \.report-toolbar__actions\s*{[^}]*flex-wrap:\s*nowrap/);
});

test('C14 top goods has four cells: index, name, quantity, revenue', () => {
  assert.doesNotMatch(page, /{index \+ 1}\. {good\.name}/);
  assert.match(globals, /\.report-rank li\s*{[^}]*grid-template-columns:[^;]*minmax\(0,\s*1fr\)[^;]*max-content[^;]*max-content/);
});

test('R1 export currency has a visible label 导出币种', () => {
  assert.match(page, />导出币种</);
});

test('issue 2: all-currency hint and filename fallback stay', () => {
  assert.match(page, /CSV 只导出主币种/);
  assert.match(page, /filename="\?\(\[\^";\]\+\)"\?/);
});

test('issue 11: non-primary currency note', () => {
  // 设计说明 §4.5 第 185 行原句：售出件数、待付款、已关闭和退款笔数只按主币种 MYR 统计，不按其他币种拆分。（MYR 取 report.currency）
  assert.match(page, /售出件数、待付款、已关闭和退款笔数只按主币种\s*\{[^}]*currency[^}]*\}\s*统计，不按其他币种拆分。/);
  assert.doesNotMatch(page, /件数仅统计主币种。/, 'old short sentence should be gone from the screen');
});

test('currency summary has its own 5-column template, no 长短款', () => {
  assert.match(reportsCss + globals, /\.report-table--currency\s*{[^}]*grid-template-columns/);
  assert.doesNotMatch(page, /长短款/);
});

test('metric values may wrap; .report-metric strong has no nowrap (UIUX 定稿)', () => {
  const strongRules = [...globals.matchAll(/\.report-metric(?:--lead)? strong\s*{([^}]*)}/g)].map((m) => m[1]);
  assert.ok(strongRules.length >= 1, 'expected .report-metric strong rules');
  for (const body of strongRules) {
    assert.doesNotMatch(body, /white-space:\s*nowrap/, `nowrap still on metric value: ${body}`);
  }
});

test('loading skeleton reserves five metric cells in the same hero grid', () => {
  assert.equal((page.match(/report-metric--skeleton/g) || []).length, 5);
  assert.match(page, /正在读取日报[\s\S]{0,1200}report-metric--lead report-metric--skeleton/);
});

test('currency summary uses a shared 5-column track with 7.25rem revenue column', () => {
  assert.match(reportsCss, /\.report-table--currency(?:\s+li)?\s*{[^}]*7\.25rem/);
  assert.doesNotMatch(page, /cashOverShortMinor/);
});

test('390 currency rows are two lines: currency+revenue, then orders/discount/refund', () => {
  const narrow = reportsCss.split('@media (max-width: 390px)')[1] || '';
  assert.match(narrow, /\.report-table--currency li > :nth-child\(4\)\s*{[^}]*grid-row:\s*2/);
  assert.doesNotMatch(narrow, /\.report-table--currency li > :nth-child\(4\)\s*{[^}]*grid-row:\s*3/);
  assert.match(narrow, /\.report-table--currency li > strong\s*{[^}]*grid-row:\s*1/);
});

test('shop theme keeps the currency grid and does not flex-wrap those rows', () => {
  const shop = fs.readFileSync('src/app/shop.css', 'utf8');
  const flexRules = [...shop.matchAll(/([^{]+)\{([^}]*display:\s*flex[^}]*)\}/g)];
  for (const m of flexRules) {
    const sel = m[1];
    if (!/report-table/.test(sel)) continue;
    assert.match(sel, /:not\(\.report-table--currency\)/, `shop flex rule must exclude currency: ${sel.trim()}`);
    assert.doesNotMatch(sel, /:is\([^)]*\.report-rank/, `shop flex rule still targets rank: ${sel.trim()}`);
  }
});
