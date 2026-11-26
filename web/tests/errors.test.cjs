/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');

const src = fs.readFileSync('src/lib/errors.ts', 'utf8');

test('event_currency_required and event_required have staff-facing copy', () => {
  assert.match(src, /event_currency_required:\s*"请先在店铺资料里设置默认货币，或为活动选择货币。"/);
  assert.match(src, /event_required:\s*"请选择或新建活动。"/);
});

test('invalid_event copy stays unchanged', () => {
  assert.match(src, /invalid_event:\s*"活动名称、日期或场次信息有误，请检查后重试。"/);
});

test('invalid_cash_received copy is removed after S6', () => {
  assert.doesNotMatch(src, /invalid_cash_received/);
});
