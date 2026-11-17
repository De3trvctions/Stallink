/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const ts = require('typescript');
const { webcrypto } = require('node:crypto');

function setup(responses) {
  const values = new Map();
  const requests = [];
  const localStorage = { getItem: (k) => values.get(k) ?? null, setItem: (k, v) => values.set(k, v), removeItem: (k) => values.delete(k) };
  const source = fs.readFileSync('src/lib/order-command.ts', 'utf8');
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
  const exports = {};
  vm.runInNewContext(code, { exports, localStorage, crypto: webcrypto, require: () => ({ browserJSON: async (...args) => { requests.push(args); return responses.shift(); } }) });
  return { ...exports, requests };
}

test('network and timeout retries retain the same command and payload', async () => {
  for (const category of ['network', 'timeout']) {
    const api = setup([{ ok: false, status: 0, category, data: {} }, { ok: true, status: 200, category: 'ok', data: {} }]);
    await api.sendOrderCommand('create', '/orders', { clientOrderId: 'one' });
    assert.ok(api.pendingOrderCommand('create'), `${category} must retain unresolved command`);
    await api.sendOrderCommand('create', '/orders', { clientOrderId: 'one' });
    assert.equal(api.requests[0][3]['Idempotency-Key'], api.requests[1][3]['Idempotency-Key']);
    assert.equal(api.pendingOrderCommand('create'), null);
  }
});

test('a definite client rejection releases the command', async () => {
  const api = setup([{ ok: false, status: 400, category: 'client', data: { error: 'invalid' } }]);
  await api.sendOrderCommand('create', '/orders', {});
  assert.equal(api.pendingOrderCommand('create'), null);
});
