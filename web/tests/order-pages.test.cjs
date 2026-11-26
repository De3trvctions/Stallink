/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const ts = require('typescript');
const React = require('react');
const { renderToStaticMarkup } = require('react-dom/server');

const order = { id: '1', number: 'A123', status: 'paid', eventName: '现场', boothName: '', currency: 'MYR', totalMinor: 1200, createdAt: '2026-01-01T00:00:00Z', cashReceivedMinor: null, cashChangeMinor: null, lines: [{ goodName: '测试商品', qty: 2, lineTotalMinor: 1200 }] };

function loadModule(path) {
  const source = fs.readFileSync(path, 'utf8');
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
  const exports = {};
  vm.runInNewContext(code, { exports });
  return exports;
}

function renderPage(path, rows = [order]) {
  const source = fs.readFileSync(path, 'utf8');
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX } }).outputText;
  const exports = {};
  const components = {
    'next/link': ({ children, ...props }) => React.createElement('a', props, children),
    '@/components/paged-list': { usePagedList: () => ({ items: rows, loading: false, error: '', refresh() {}, reset() {} }), ListControls: () => null },
    '@/components/confirm-dialog': { ConfirmDialog: () => null },
    '@/components/order-action-dialog': { OrderActionDialog: () => null },
    '@/components/workspace-shell': { WorkspaceShell: ({ children }) => children },
    '@/components/select-menu': { default: () => null },
    '@/lib/message': { useMessage: () => ({ message: null }), messageClass: () => '' },
    '@/lib/errors': { describeRequest: () => '' },
    '@/lib/pricing': { formatMoney: (_currency, value) => `RM ${(value / 100).toFixed(2)}` },
    '@/lib/orders': { orderStatusLabel: () => '已收款' },
    '@/lib/order-command': {},
    '@/lib/order-actions': loadModule('src/lib/order-actions.ts'),
  };
  vm.runInNewContext(code, { exports, require: (name) => components[name] ?? require(name), Date, encodeURIComponent });
  return renderToStaticMarkup(React.createElement(exports.default));
}

for (const path of ['src/app/counter/orders/page.tsx', 'src/app/admin/orders/page.tsx']) {
  test(`${path} displays order lines returned by the API`, () => {
    const html = renderPage(path);
    assert.match(html, /A123/);
    assert.match(html, /测试商品 × 2/);
    assert.match(html, /RM 12\.00/);
  });
}

for (const path of ['src/app/counter/orders/page.tsx', 'src/app/admin/orders/page.tsx']) {
  test(`${path} lists fulfil before cancel for paid orders`, () => {
    const html = renderPage(path, [{ ...order, status: 'paid' }]);
    assert.ok(html.indexOf('标记交付') !== -1 && html.indexOf('标记交付') < html.indexOf('取消订单'), html);
  });

  test(`${path} lets fulfilled orders be cancelled but not fulfilled again`, () => {
    const html = renderPage(path, [{ ...order, status: 'fulfilled' }]);
    assert.match(html, /取消订单/);
    assert.doesNotMatch(html, /标记交付/);
  });

  test(`${path} does not send a cancel reason field the API ignores`, () => {
    const source = fs.readFileSync(path, 'utf8');
    assert.doesNotMatch(source, /reason:\s*"/);
    assert.match(source, /orderActionRequestBody/);
  });

  test(`${path} does not display cash received or change on the order row`, () => {
    const html = renderPage(path, [{ ...order, status: 'paid', cashReceivedMinor: 5000, cashChangeMinor: 0 }]);
    assert.doesNotMatch(html, /实收/);
    assert.doesNotMatch(html, /找零/);
  });
}
