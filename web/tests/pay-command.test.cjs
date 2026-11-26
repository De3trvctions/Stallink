/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');

const api = fs.readFileSync('src/lib/api.ts', 'utf8');
const counter = fs.readFileSync('src/app/counter/page.tsx', 'utf8');
const errors = fs.readFileSync('src/lib/errors.ts', 'utf8');

test('StaffOrder no longer declares cashReceivedMinor or cashChangeMinor', () => {
  const type = api.slice(api.indexOf('export type StaffOrder'), api.indexOf('export type Good'));
  assert.doesNotMatch(type, /cashReceivedMinor/);
  assert.doesNotMatch(type, /cashChangeMinor/);
});

test('cashier pay commands no longer carry cashReceivedMinor', () => {
  assert.doesNotMatch(counter, /cashReceivedMinor/);
});

test('idempotency_conflict rechecks the order, treats paid as success, otherwise prompts and always clears the saved command', () => {
  const confirm = counter.slice(counter.indexOf('async function confirm()'), counter.indexOf('async function fulfillReceipt()'));
  assert.match(confirm, /idempotency_conflict/);
  assert.match(confirm, /\/api\/staff\/orders\/\$\{orderId\}/);
  assert.match(confirm, /clearPendingOrderCommand\(scope\)/);
  assert.match(confirm, /这笔收款可能已经提交，请到订单记录里核对/);
  assert.match(confirm, /status.*paid|=== ["']paid["']/);
});

test('errors.ts no longer maps invalid_cash_received', () => {
  assert.doesNotMatch(errors, /invalid_cash_received/);
});
