/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const ts = require('typescript');

function load() {
  const source = fs.readFileSync('src/lib/order-actions.ts', 'utf8');
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
  const exports = {};
  vm.runInNewContext(code, { exports });
  return exports;
}

test('order actions put fulfil before cancel and let paid or fulfilled orders be cancelled', () => {
  const { orderActionsFor } = load();
  assert.deepEqual(Array.from(orderActionsFor('pending')), ['cancel']);
  assert.deepEqual(Array.from(orderActionsFor('paid')), ['fulfill', 'cancel']);
  assert.deepEqual(Array.from(orderActionsFor('fulfilled')), ['cancel']);
  assert.deepEqual(Array.from(orderActionsFor('refund_required')), ['refund']);
  for (const status of ['cancelled', 'expired', 'unknown']) {
    assert.deepEqual(Array.from(orderActionsFor(status)), [], status);
  }
});

test('cancel and refund confirmations require a reason, fulfil does not', () => {
  const { actionNeedsReason } = load();
  assert.equal(actionNeedsReason('cancel'), true);
  assert.equal(actionNeedsReason('refund'), true);
  assert.equal(actionNeedsReason('fulfill'), false);
});

test('reason validation requires 2 to 500 trimmed Unicode characters', () => {
  const { validateActionReason, REASON_MIN_LENGTH, REASON_MAX_LENGTH } = load();
  assert.equal(REASON_MIN_LENGTH, 2);
  assert.equal(REASON_MAX_LENGTH, 500);
  assert.equal(validateActionReason('').ok, false);
  assert.match(validateActionReason('   ').error, /原因/);

  const one = validateActionReason('退');
  assert.equal(one.ok, false);
  assert.match(one.error, /2/);

  assert.deepEqual({ ...validateActionReason('顾客') }, { ok: true, reason: '顾客' });
  assert.deepEqual({ ...validateActionReason('  顾客退货  ') }, { ok: true, reason: '顾客退货' });

  const trimmedShort = validateActionReason('  退  ');
  assert.equal(trimmedShort.ok, false);
  assert.match(trimmedShort.error, /2/);

  assert.equal(validateActionReason('退'.repeat(500)).ok, true);
  const tooLong = validateActionReason('退'.repeat(501));
  assert.equal(tooLong.ok, false);
  assert.match(tooLong.error, /500/);

  // 代理对按码点计为 1 字：emoji 与生僻字都不能用 UTF-16 length。
  assert.equal(validateActionReason('🙂').ok, false);
  assert.deepEqual({ ...validateActionReason('🙂👍') }, { ok: true, reason: '🙂👍' });
  assert.deepEqual({ ...validateActionReason('𠮷好') }, { ok: true, reason: '𠮷好' });
  assert.equal(validateActionReason('🙂'.repeat(500)).ok, true);
  const tooManyEmoji = validateActionReason('🙂'.repeat(501));
  assert.equal(tooManyEmoji.ok, false);
  assert.match(tooManyEmoji.error, /500/);
});

test('cancelling an unpaid order still requires a reason', () => {
  const { orderActionsFor, actionNeedsReason } = load();
  assert.deepEqual(Array.from(orderActionsFor('pending')), ['cancel']);
  assert.equal(actionNeedsReason('cancel'), true);
});

test('order action requests do not send a reason field the API does not accept', () => {
  const { orderActionRequestBody } = load();
  for (const action of ['cancel', 'refund', 'fulfill']) {
    assert.deepEqual({ ...orderActionRequestBody(action) }, {}, action);
  }
});

test('payCommandBody only returns payCodeId for cash and qr, with no cashReceivedMinor', () => {
  const { payCommandBody } = load();
  assert.deepEqual({ ...payCommandBody({ id: '7', kind: 'cash' }) }, { payCodeId: '7' });
  assert.deepEqual({ ...payCommandBody({ id: '8', kind: 'qr' }) }, { payCodeId: '8' });
  const cash = payCommandBody({ id: '7', kind: 'cash' });
  const qr = payCommandBody({ id: '8', kind: 'qr' });
  assert.equal('cashReceivedMinor' in cash, false);
  assert.equal('cashReceivedMinor' in qr, false);
});
