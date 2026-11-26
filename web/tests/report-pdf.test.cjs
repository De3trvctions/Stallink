/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const crypto = require('node:crypto');
const ts = require('typescript');

// Loads a TypeScript module and its relative / "@/..." imports in this realm,
// so deepStrictEqual compares plain arrays and objects without realm quirks.
const moduleCache = new Map();
function resolveSource(specifier, fromDir) {
  const base = specifier.startsWith('@/') ? path.resolve('src', specifier.slice(2)) : path.resolve(fromDir, specifier);
  for (const candidate of [base, `${base}.ts`, `${base}.tsx`, path.join(base, 'index.ts')]) {
    if (fs.existsSync(candidate) && fs.statSync(candidate).isFile()) return candidate;
  }
  throw new Error(`cannot resolve ${specifier} from ${fromDir}`);
}
function load(file) {
  const resolved = path.resolve(file);
  if (moduleCache.has(resolved)) return moduleCache.get(resolved).exports;
  const loaded = { exports: {} };
  moduleCache.set(resolved, loaded);
  const source = fs.readFileSync(resolved, 'utf8');
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, target: ts.ScriptTarget.ES2020 } }).outputText;
  const localRequire = (specifier) => (specifier.startsWith('.') || specifier.startsWith('@/'))
    ? load(resolveSource(specifier, path.dirname(resolved)))
    : require(specifier);
  new Function('exports', 'require', 'module', code)(loaded.exports, localRequire, loaded);
  return loaded.exports;
}

const model = () => load('src/lib/reports/document-model.ts');
const naming = () => load('src/lib/reports/report-filename.ts');
const glyphs = () => load('src/lib/reports/glyph-coverage.ts');
const themeFonts = () => load('src/lib/reports/pdf/theme-fonts.ts');
const pdfDefinition = () => load('src/lib/reports/pdf/pdf-definition.ts');

const stamp = (iso) => `at ${iso}`;

function report(overrides = {}) {
  return {
    date: '2026-11-13', generatedAt: '2026-11-13T05:55:00Z', source: 'manual', currency: 'MYR',
    orders: 5, paidOrders: 2, pendingOrders: 0, closedOrders: 3, refundOrders: 1,
    itemsSold: 36, revenueMinor: 56218, discountMinor: 14932, refundMinor: 1200, cashOverShortMinor: -50,
    averageMinor: 28109,
    currencies: [
      { currency: 'MYR', orders: 5, paidOrders: 2, revenueMinor: 56218, discountMinor: 14932, refundMinor: 1200, cashOverShortMinor: -50 },
      { currency: 'SGD', orders: 3, paidOrders: 2, revenueMinor: 4000, discountMinor: 300, refundMinor: 0, cashOverShortMinor: 20 },
    ],
    events: [
      { eventId: '1', name: 'Comic Fiesta 2026', currency: 'MYR', orders: 5, revenueMinor: 56218, discountMinor: 14932 },
      { eventId: '1', name: 'Comic Fiesta 2026', currency: 'SGD', orders: 3, revenueMinor: 4000, discountMinor: 300 },
    ],
    topGoods: [
      { currency: 'MYR', name: '马口铁徽章', quantity: 14, revenueMinor: 19486 },
      { currency: 'SGD', name: '闪膜明信片', quantity: 7, revenueMinor: 4000 },
      { currency: 'MYR', name: '亚克力立牌', quantity: 2, revenueMinor: 3865 },
    ],
    payCodes: [
      { currency: 'MYR', name: '现金', orders: 2, revenueMinor: 56218 },
      { currency: 'SGD', name: 'PayNow', orders: 2, revenueMinor: 4000 },
    ],
    ...overrides,
  };
}

function singleCurrencyReport() {
  const base = report();
  return report({
    currencies: [base.currencies[0]],
    events: [base.events[0]],
    topGoods: base.topGoods.filter((row) => row.currency === 'MYR'),
    payCodes: [base.payCodes[0]],
  });
}

const values = (section) => section.metrics.map((metric) => metric.value);
const groupCurrencies = (block) => block.groups.map((group) => group.currency);

test('document model mirrors the on-screen sheet: six metrics and blocks grouped by currency', () => {
  const doc = model().buildReportDocument({ days: [{ date: '2026-11-13', report: report() }], currency: '', formatGeneratedAt: stamp });
  assert.equal(doc.title, '销售日报 · 2026-11-13');
  assert.equal(doc.currencyLabel, '全部币种');
  assert.deepEqual(doc.dates, ['2026-11-13']);
  const [day] = doc.days;
  assert.equal(day.missing, false);
  assert.equal(day.eyebrow, '2026-11-13 · MYR');
  assert.equal(day.heading, '当日订单与收款');
  assert.equal(day.generated, '生成于 at 2026-11-13T05:55:00Z · 手动生成');
  assert.equal(day.stamp, '含退款订单');
  assert.deepEqual(day.metrics.map((metric) => metric.label), [
    '收款总额（MYR）', '订单数', '售出件数', '优惠金额（MYR）', '待退金额（MYR）', '现金长短款（MYR）',
  ]);
  assert.deepEqual(values(day), ['RM 562.18', '5', '36', 'RM 149.32', 'RM 12.00', '短款 RM 0.50']);
  assert.deepEqual(day.metrics.map((metric) => metric.note), [
    '2 笔已收款 · 0 笔待付款', '已关闭 3 · 退款 1', '平均每单 RM 281.09', '数量价与套餐优惠合计', '待确认退款的金额', '正数长款 · 负数短款',
  ]);
  assert.equal(day.metrics[0].lead, true);
  assert.deepEqual(day.blocks.map((block) => [block.title, block.caption]), [
    ['活动与场次', '按币种分组'], ['热销商品', '按币种分组'], ['收款方式', '按币种分组'],
  ]);
  for (const block of day.blocks) assert.deepEqual(groupCurrencies(block), ['MYR', 'SGD']);
  const goods = day.blocks[1].groups[0].rows;
  assert.deepEqual(goods.map((row) => [...row.cells, row.amount]), [
    ['1. 马口铁徽章', '14 件', 'RM 194.86'],
    ['2. 亚克力立牌', '2 件', 'RM 38.65'],
  ]);
  assert.deepEqual(day.blocks[0].groups[1].rows.map((row) => [...row.cells, row.amount]), [
    ['Comic Fiesta 2026', '3 笔', '优惠 S$ 3.00', 'S$ 40.00'],
  ]);
});

test('PDF currency summary keeps overShortLabel output including 短款', () => {
  const { buildReportDocument, overShortLabel } = model();
  assert.match(overShortLabel('MYR', -50), /短款/);
  const doc = buildReportDocument({ days: [{ date: '2026-11-13', report: report() }], currency: '', formatGeneratedAt: stamp });
  const cells = doc.days[0].currencySummary.rows.flatMap((row) => row.cells);
  assert.equal(cells.some((cell) => String(cell).includes('短款')), true);
  assert.match(cells.join(' '), /短款/);
});


test('PDF over-short shows —（未录实收） when cashTenderedOrders is 0, and missing field keeps the numeric label', () => {
  const { buildReportDocument, overShortLabel } = model();
  assert.equal(overShortLabel('MYR', 0, 0), '—（未录实收）');
  assert.match(overShortLabel('MYR', 0, 2), /持平/);
  assert.match(overShortLabel('MYR', -50, 1), /短款/);
  // 旧快照缺字段：不当成 0，仍显示数值（K-Q18 兜底）。
  assert.match(overShortLabel('MYR', 0, undefined), /持平/);
  assert.equal(overShortLabel('MYR', -50, 0), '—（未录实收）');
  const mixed = report({
    cashOverShortMinor: 0,
    cashTenderedOrders: 0,
    currencies: [
      { currency: 'MYR', orders: 5, paidOrders: 2, revenueMinor: 56218, discountMinor: 14932, refundMinor: 1200, cashOverShortMinor: 0, cashTenderedOrders: 0 },
      { currency: 'SGD', orders: 3, paidOrders: 2, revenueMinor: 4000, discountMinor: 300, refundMinor: 0, cashOverShortMinor: -50, cashTenderedOrders: 1 },
    ],
  });
  const doc = buildReportDocument({ days: [{ date: '2026-11-13', report: mixed }], currency: '', formatGeneratedAt: stamp });
  const metric = doc.days[0].metrics.find((item) => item.label === '现金长短款（MYR）');
  assert.equal(metric.value, '—（未录实收）');
  const cells = doc.days[0].currencySummary.rows.map((row) => row.cells);
  assert.equal(cells[0][3], '—（未录实收）');
  assert.match(String(cells[1][3]), /短款/);
});

test('exporting a non-headline currency uses that currency row for over-short and 未录实收', () => {
  const { buildReportDocument } = model();
  const mixed = report({
    cashOverShortMinor: 0,
    cashTenderedOrders: 0,
    currencies: [
      { currency: 'MYR', orders: 5, paidOrders: 2, revenueMinor: 56218, discountMinor: 14932, refundMinor: 1200, cashOverShortMinor: 0, cashTenderedOrders: 0 },
      { currency: 'SGD', orders: 3, paidOrders: 2, revenueMinor: 4000, discountMinor: 300, refundMinor: 0, cashOverShortMinor: -50, cashTenderedOrders: 1 },
    ],
  });
  const sgd = buildReportDocument({ days: [{ date: '2026-11-13', report: mixed }], currency: 'SGD', formatGeneratedAt: stamp });
  const sgdMetric = sgd.days[0].metrics.find((item) => item.label === '现金长短款（SGD）');
  assert.equal(sgdMetric.value, '短款 S$ 0.50');
  const inverse = report({
    cashOverShortMinor: -50,
    cashTenderedOrders: 2,
    currencies: [
      { currency: 'MYR', orders: 5, paidOrders: 2, revenueMinor: 56218, discountMinor: 14932, refundMinor: 1200, cashOverShortMinor: -50, cashTenderedOrders: 2 },
      { currency: 'SGD', orders: 3, paidOrders: 2, revenueMinor: 4000, discountMinor: 300, refundMinor: 0, cashOverShortMinor: 0, cashTenderedOrders: 0 },
    ],
  });
  const sgdEmpty = buildReportDocument({ days: [{ date: '2026-11-13', report: inverse }], currency: 'SGD', formatGeneratedAt: stamp });
  const emptyMetric = sgdEmpty.days[0].metrics.find((item) => item.label === '现金长短款（SGD）');
  assert.equal(emptyMetric.value, '—（未录实收）');
});

test('currency summary appears only for multi-currency reports and never sums across currencies', () => {
  const multi = model().buildReportDocument({ days: [{ date: '2026-11-13', report: report() }], currency: '', formatGeneratedAt: stamp });
  assert.equal(multi.days[0].currencySummary.title, '分币种汇总');
  assert.equal(multi.days[0].currencySummary.caption, '各币种独立合计，不跨币种相加');
  assert.deepEqual(multi.days[0].currencySummary.rows.map((row) => [...row.cells, row.amount]), [
    ['MYR', '5 笔 · 已收款 2', '退款 RM 12.00', '短款 RM 0.50', 'RM 562.18'],
    ['SGD', '3 笔 · 已收款 2', '退款 S$ 0.00', '长款 S$ 0.20', 'S$ 40.00'],
  ]);
  const single = model().buildReportDocument({ days: [{ date: '2026-11-13', report: singleCurrencyReport() }], currency: '', formatGeneratedAt: stamp });
  assert.equal(single.days[0].currencySummary, null);
});

test('a currency filter narrows headline and detail blocks but keeps every currency in the summary', () => {
  const doc = model().buildReportDocument({ days: [{ date: '2026-11-13', report: report() }], currency: 'SGD', formatGeneratedAt: stamp });
  const [day] = doc.days;
  assert.equal(doc.currencyLabel, 'SGD');
  assert.equal(doc.currencyFilter, 'SGD');
  assert.equal(day.eyebrow, '2026-11-13 · SGD');
  assert.deepEqual(day.metrics.map((metric) => metric.label), [
    '收款总额（SGD）', '订单数', '售出件数', '优惠金额（SGD）', '待退金额（SGD）', '现金长短款（SGD）',
  ]);
  // Status counts, items sold and average are only stored for the headline
  // currency, so a non-headline filter marks them instead of borrowing MYR.
  assert.deepEqual(values(day), ['S$ 40.00', '3', '—', 'S$ 3.00', 'S$ 0.00', '长款 S$ 0.20']);
  assert.equal(day.metrics[0].note, '2 笔已收款');
  assert.equal(day.metrics[1].note, '按 SGD 统计');
  assert.equal(day.metrics[2].note, '件数仅统计主币种 MYR · 平均每单 S$ 20.00');
  assert.deepEqual(day.currencySummary.rows.map((row) => row.cells[0]), ['MYR', 'SGD']);
  for (const block of day.blocks) assert.deepEqual(groupCurrencies(block), ['SGD']);
  assert.deepEqual(day.blocks[1].groups[0].rows.map((row) => row.cells[0]), ['1. 闪膜明信片']);
});

test('filtering on the headline currency keeps the headline metrics shown on screen', () => {
  const doc = model().buildReportDocument({ days: [{ date: '2026-11-13', report: report() }], currency: 'MYR', formatGeneratedAt: stamp });
  const [day] = doc.days;
  assert.deepEqual(values(day), ['RM 562.18', '5', '36', 'RM 149.32', 'RM 12.00', '短款 RM 0.50']);
  assert.deepEqual(day.currencySummary.rows.map((row) => row.cells[0]), ['MYR', 'SGD']);
  for (const block of day.blocks) assert.deepEqual(groupCurrencies(block), ['MYR']);
});

test('filtering on a currency the day lacks shows zeros and empty blocks rather than other currencies', () => {
  const doc = model().buildReportDocument({ days: [{ date: '2026-11-13', report: report() }], currency: 'CNY', formatGeneratedAt: stamp });
  const [day] = doc.days;
  assert.deepEqual(values(day), ['¥ 0.00', '0', '—', '¥ 0.00', '¥ 0.00', '—（未录实收）']);
  for (const block of day.blocks) assert.deepEqual(block.groups, []);
  assert.deepEqual(day.blocks.map((block) => block.empty), ['这一天没有活动成交。', '这一天没有售出商品。', '这一天没有收款记录。']);
});

test('a single-currency report without a filter is labelled with its own currency', () => {
  const doc = model().buildReportDocument({ days: [{ date: '2026-11-13', report: singleCurrencyReport() }], currency: '', formatGeneratedAt: stamp });
  assert.equal(doc.currencyLabel, 'MYR');
  assert.equal(doc.currencyFilter, '');
});

test('a date range builds one ascending section per day, including days without a report', () => {
  const later = report({ date: '2026-11-14', source: 'scheduled', refundOrders: 0 });
  const doc = model().buildReportDocument({
    days: [
      { date: '2026-11-14', report: later },
      { date: '2026-11-12', report: null },
      { date: '2026-11-13', report: report() },
    ],
    currency: '',
    formatGeneratedAt: stamp,
  });
  assert.equal(doc.title, '销售日报 · 2026-11-12 至 2026-11-14');
  assert.deepEqual(doc.dates, ['2026-11-12', '2026-11-13', '2026-11-14']);
  assert.deepEqual(doc.days.map((day) => [day.date, day.missing]), [['2026-11-12', true], ['2026-11-13', false], ['2026-11-14', false]]);
  assert.equal(doc.days[0].message, '这一天还没有生成日报。');
  assert.equal(doc.days[2].generated, '生成于 at 2026-11-13T05:55:00Z · 定时生成');
  assert.equal(doc.days[2].stamp, '无退款');
  assert.throws(() => model().buildReportDocument({ days: [], currency: '' }), /至少/);
  assert.throws(() => model().buildReportDocument({
    days: [{ date: '2026-11-13', report: report() }, { date: '2026-11-13', report: report() }], currency: '',
  }), /重复/);
});

test('pdf file names always carry the currency label, and ranges use both ends', () => {
  const { reportPdfFilename } = naming();
  assert.equal(reportPdfFilename(['2026-11-13'], '全部币种'), '销售日报-2026-11-13-全部币种.pdf');
  assert.equal(reportPdfFilename(['2026-11-13'], 'MYR'), '销售日报-2026-11-13-MYR.pdf');
  assert.equal(reportPdfFilename(['2026-11-14', '2026-11-12', '2026-11-13'], 'SGD'), '销售日报-2026-11-12至2026-11-14-SGD.pdf');
  assert.equal(reportPdfFilename(['2026-11-13'], 'my/r'), '销售日报-2026-11-13-全部币种.pdf');
  assert.throws(() => reportPdfFilename([], 'MYR'), /日期/);
  assert.throws(() => reportPdfFilename(['2026/11/13'], 'MYR'), /日期/);
});

test('model and file name agree on the label for all-currency and filtered exports', () => {
  const { reportPdfFilename } = naming();
  const all = model().buildReportDocument({ days: [{ date: '2026-11-13', report: report() }], currency: '' });
  const sgd = model().buildReportDocument({ days: [{ date: '2026-11-13', report: report() }], currency: 'SGD' });
  assert.equal(reportPdfFilename(all.dates, all.currencyLabel), '销售日报-2026-11-13-全部币种.pdf');
  assert.equal(reportPdfFilename(sgd.dates, sgd.currencyLabel), '销售日报-2026-11-13-SGD.pdf');
});

test('characters outside the font subset become □ and invisible joiners are dropped', () => {
  const { createGlyphCoverage, replaceMissingGlyphs, MISSING_GLYPH } = glyphs();
  const covered = createGlyphCoverage([[0x20, 0x7e], [0x4e00, 0x4e00], [0x5fbd, 0x5fbd], [0x7ae0, 0x7ae0], [0x25a1, 0x25a1], [0x20bb7, 0x20bb7]]);
  assert.equal(MISSING_GLYPH, '□');
  assert.deepEqual(replaceMissingGlyphs('徽章 A-1', covered), { text: '徽章 A-1', replaced: 0 });
  assert.deepEqual(replaceMissingGlyphs('徽章龘', covered), { text: '徽章□', replaced: 1 });
  assert.deepEqual(replaceMissingGlyphs('𠮷一', covered), { text: '𠮷一', replaced: 0 });
  assert.deepEqual(replaceMissingGlyphs('👍🏻 一', covered), { text: '□ 一', replaced: 1 });
  assert.deepEqual(replaceMissingGlyphs('👨‍👩‍👧', covered), { text: '□□□', replaced: 3 });
  assert.deepEqual(replaceMissingGlyphs('ｶ\n一', covered), { text: '□\n一', replaced: 1 });
});

test('glyph coverage walks the whole document and reports how many characters were replaced', () => {
  const { createGlyphCoverage, applyGlyphCoverage } = glyphs();
  const covered = createGlyphCoverage([[0x20, 0x7e], [0x4e00, 0x9fff], [0xff08, 0xff09], [0x3001, 0x3002]]);
  const input = { title: '销售日报', days: [{ rows: [{ cells: ['ラムネ', 'RM 1.00'] }], lead: true, count: 3 }] };
  const { value, replaced } = applyGlyphCoverage(input, covered);
  assert.equal(replaced, 3);
  assert.deepEqual(value, { title: '销售日报', days: [{ rows: [{ cells: ['□□□', 'RM 1.00'] }], lead: true, count: 3 }] });
  assert.equal(input.days[0].rows[0].cells[0], 'ラムネ');
});

test('theme fonts live in one map: serif for classic and exhibition, sans for shop', () => {
  const { reportFontFor, REPORT_THEME_FONTS, REPORT_COVERAGE_URL } = themeFonts();
  assert.deepEqual({ ...REPORT_THEME_FONTS }, { classic: 'serif', exhibition: 'serif', shop: 'sans' });
  assert.equal(reportFontFor('classic').family, 'Noto Serif SC');
  assert.equal(reportFontFor('exhibition').family, 'Noto Serif SC');
  assert.equal(reportFontFor('shop').family, 'Noto Sans SC');
  assert.equal(reportFontFor('unknown').family, 'Noto Serif SC');
  assert.match(REPORT_COVERAGE_URL, /^\/fonts\/report\/coverage\.json\?v=[0-9a-f]{12}$/);
  const coverage = JSON.parse(fs.readFileSync('public/fonts/report/coverage.json', 'utf8'));
  assert.equal(REPORT_COVERAGE_URL.split('=')[1], crypto.createHash('sha256').update(fs.readFileSync('public/fonts/report/coverage.json')).digest('hex').slice(0, 12));
  assert.equal(coverage.characterTable.count, 8105);
  for (const theme of ['classic', 'exhibition', 'shop']) {
    const face = reportFontFor(theme);
    const bytes = fs.readFileSync(path.join('public', face.url.split('?')[0]));
    const digest = crypto.createHash('sha256').update(bytes).digest('hex');
    assert.equal(bytes.length, face.bytes, `${theme} byte count`);
    assert.equal(digest, face.sha256, `${theme} digest`);
    assert.equal(coverage.fonts[face.key].sha256, digest, `${theme} manifest digest`);
    assert.equal(face.url, `/fonts/report/${face.file}?v=${digest.slice(0, 12)}`);
    assert.ok(bytes.length < 4_000_000, `${theme} font stays under 4 MB`);
  }
  for (const file of ['OFL.txt', 'COPYRIGHT.txt', 'README.md', 'UNICODE-LICENSE.txt']) {
    assert.ok(fs.statSync(path.join('public/fonts/report', file)).size > 0, file);
  }
});

test('pdf definition embeds only the report font, keeps metadata generic and breaks pages between days', () => {
  const { buildPdfDefinition, REPORT_PDF_FONT } = pdfDefinition();
  const { reportColorsFor } = themeFonts();
  const doc = model().buildReportDocument({
    days: [{ date: '2026-11-13', report: report() }, { date: '2026-11-14', report: null }],
    currency: '', formatGeneratedAt: stamp,
  });
  const definition = buildPdfDefinition(doc, { colors: reportColorsFor('shop'), missingGlyphs: true });
  assert.equal(definition.defaultStyle.font, REPORT_PDF_FONT);
  assert.equal(definition.pageSize, 'A4');
  assert.deepEqual(Object.keys(definition.info).sort(), ['creator', 'producer', 'title']);
  assert.equal(definition.info.title, doc.title);
  const json = JSON.stringify(definition.content);
  assert.match(json, /收款总额（MYR）/);
  assert.match(json, /分币种汇总/);
  assert.match(json, /这一天还没有生成日报/);
  const breaks = (json.match(/"pageBreak":"before"/g) ?? []).length;
  assert.equal(breaks, 1);
  const footer = definition.footer(1, 2);
  assert.match(JSON.stringify(footer), /已显示为 □/);
  assert.match(JSON.stringify(footer), /第 1 \/ 2 页/);
  const clean = buildPdfDefinition(doc, { colors: reportColorsFor('classic'), missingGlyphs: false });
  assert.doesNotMatch(JSON.stringify(clean.footer(1, 1)), /□/);
  assert.doesNotMatch(json, /"image"|"link"|"attachment"/);
});
