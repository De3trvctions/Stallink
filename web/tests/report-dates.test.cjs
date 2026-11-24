/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const ts = require('typescript');

const source = fs.readFileSync('src/app/admin/reports/report-dates.ts', 'utf8');
const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
const moduleExports = {};
vm.runInNewContext(code, { exports: moduleExports });
const { inclusiveReportDates } = moduleExports;

test('report dates include both boundaries and preserve calendar days across daylight saving', () => {
  const result = inclusiveReportDates(Date.UTC(2026, 2, 7), Date.UTC(2026, 2, 10));
  assert.deepEqual(Array.from(result.dates), ['2026-03-10', '2026-03-09', '2026-03-08', '2026-03-07']);
  assert.deepEqual(Array.from(inclusiveReportDates(Date.UTC(2024, 1, 29), Date.UTC(2024, 1, 29)).dates), ['2024-02-29']);
});

test('report date query accepts up to 31 days and rejects reversed or empty dates', () => {
  assert.equal(inclusiveReportDates(Date.UTC(2026, 0, 1), Date.UTC(2026, 0, 31)).dates.length, 31);
  assert.match(inclusiveReportDates(Date.UTC(2026, 0, 1), Date.UTC(2026, 1, 1)).error, /31/);
  assert.match(inclusiveReportDates(Date.UTC(2026, 0, 2), Date.UTC(2026, 0, 1)).error, /起始日期/);
  assert.match(inclusiveReportDates(NaN, Date.UTC(2026, 0, 1)).error, /有效/);
});
