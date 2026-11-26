/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const ts = require('typescript');
const React = require('react');
const { renderToStaticMarkup } = require('react-dom/server');

function loadLanding() {
  const source = fs.readFileSync('src/components/shop-landing.tsx', 'utf8');
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS, jsx: ts.JsxEmit.ReactJSX } }).outputText;
  const exports = {};
  const components = {
    'next/link': { __esModule: true, default: ({ children, ...props }) => React.createElement('a', props, children) },
    '@/components/safe-image': { SafeImage: ({ alt }) => React.createElement('img', { alt }) },
  };
  vm.runInNewContext(code, { exports, require: (name) => components[name] ?? require(name) });
  return exports;
}

const good = (overrides = {}) => ({ listingId: '1', name: '徽章', shopName: '小摊', shopSlug: 'stall', eventId: 'e1', eventName: '漫展', priceLabel: 'RM 10', remaining: 3, soldOut: false, imageURL: '/img.png', ...overrides });

test('spotlight shows photo goods when some have images', () => {
  const { ShopSpotlight } = loadLanding();
  const html = renderToStaticMarkup(React.createElement(ShopSpotlight, { goods: [good()] }));
  assert.match(html, /摊位在售好物/);
  assert.match(html, /徽章/);
  assert.doesNotMatch(html, /empty-state/);
});

test('spotlight keeps its heading and shows an empty state when no goods have images', () => {
  const { ShopSpotlight } = loadLanding();
  const html = renderToStaticMarkup(React.createElement(ShopSpotlight, { goods: [good({ imageURL: '' })] }));
  assert.match(html, /摊位在售好物/);
  assert.match(html, /class="empty-state empty-state--compact"/);
  assert.match(html, /还没有带图片的在售好物/);
});

test('spotlight stays hidden when the catalogue failed to load, so the catalogue error is the only message', () => {
  const { ShopSpotlight } = loadLanding();
  const html = renderToStaticMarkup(React.createElement(ShopSpotlight, { goods: [], unavailable: true }));
  assert.equal(html, '');
});
