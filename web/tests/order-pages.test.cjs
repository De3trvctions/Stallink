/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const ts = require('typescript');
const React = require('react');
const { renderToStaticMarkup } = require('react-dom/server');

const order = { id: '1', number: 'A123', status: 'paid', eventName: '现场', boothName: '', currency: 'MYR', totalMinor: 1200, createdAt: '2026-01-01T00:00:00Z', cashReceivedMinor: null, cashChangeMinor: null, lines: [{ goodName: '测试商品', qty: 2, lineTotalMinor: 1200 }] };

function renderPage(path) {
  const source = fs.readFileSync(path, 'utf8');
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX } }).outputText;
  const exports = {};
  const components = {
    'next/link': ({ children, ...props }) => React.createElement('a', props, children),
    '@/components/paged-list': { usePagedList: () => ({ items: [order], loading: false, error: '', refresh() {}, reset() {} }), ListControls: () => null },
    '@/components/confirm-dialog': { ConfirmDialog: () => null },
    '@/components/workspace-shell': { WorkspaceShell: ({ children }) => children },
    '@/components/select-menu': { default: () => null },
    '@/lib/message': { useMessage: () => ({ message: null }), messageClass: () => '' },
    '@/lib/errors': { describeRequest: () => '' },
    '@/lib/pricing': { formatMoney: (_currency, value) => `RM ${(value / 100).toFixed(2)}` },
    '@/lib/orders': { orderStatusLabel: () => '已收款' },
    '@/lib/order-command': {},
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
