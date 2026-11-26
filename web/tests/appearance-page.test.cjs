/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const page = fs.readFileSync('src/app/platform/appearance/page.tsx', 'utf8');
const css = fs.readFileSync('src/app/platform/appearance/appearance.css', 'utf8');

test('A2/A3 publish goes through a confirm dialog', () => {
  assert.match(page, /confirmOpen/);
  assert.match(page, /ConfirmDialog/);
});
test('A5 offline banner listens to online/offline', () => {
  assert.match(page, /addEventListener\("offline"/);
  assert.match(page, /addEventListener\("online"/);
});
test('A6 401 and 403 have their own blocks', () => {
  assert.match(page, /登录已过期/);
  assert.match(page, /只有平台管理员可以修改平台外观/);
});
test('A7 revert link and A11 two tags', () => {
  assert.match(page, /还原为当前风格/);
  assert.match(page, /已选 · 未发布/);
});
test('A8 view storefront opens in a new tab', () => {
  assert.match(page, /target="_blank" rel="noopener"/);
});
test('A10/A17/A18 status bar, message blocks, reduced motion', () => {
  assert.match(page, /appearance-status/);
  assert.match(css, /\.appearance-offline/);
  assert.match(css, /prefers-reduced-motion: reduce/);
});
test('A1 options include swatches and swatchLabel', () => {
  assert.match(page, /swatches:/);
  assert.match(page, /swatchLabel:/);
});
test('A13-A16 preview tokens, two faces, and tags', () => {
  assert.doesNotMatch(css, /#5433eb/i);
  const hexes = [...css.matchAll(/#[0-9a-fA-F]{3,8}/g)].map((m) => m[0]);
  for (const hex of hexes) {
    const idx = css.indexOf(hex);
    const before = css.slice(Math.max(0, idx - 400), idx);
    assert.match(before, /--sample-/, `${hex} must be declared as a --sample-* token`);
  }
  assert.match(css, /\.appearance-preview\s*{[^}]*grid-template-columns:\s*1\.15fr 1fr/);
  assert.doesNotMatch(css, /\.appearance-label small\s*{[^}]*color:\s*var\(--accent\)/);
  assert.match(css, /\.appearance-tag--live/);
  assert.match(css, /\.appearance-tag--draft/);
});

test('publish confirm dialog focuses 取消 via initialFocus', () => {
  const block = page.slice(page.indexOf('<ConfirmDialog'), page.indexOf('/>', page.indexOf('<ConfirmDialog')) + 2);
  assert.match(block, /initialFocus="cancel"/);
  assert.match(block, /danger=\{false\}/);
});


test('A19 publish status uses updatedAt/updatedBy and 尚无记录 / 你 · 刚刚', () => {
  const helpers = fs.readFileSync('src/lib/appearance-status.ts', 'utf8');
  assert.match(page, /appearancePublishLabel|formatAppearanceStatus/);
  assert.match(page + helpers, /尚无记录/);
  assert.match(page, /你 · 刚刚/);
});
