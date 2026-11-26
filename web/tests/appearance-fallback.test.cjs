/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');

const api = fs.readFileSync('src/lib/api.ts', 'utf8');
const layout = fs.readFileSync('src/app/layout.tsx', 'utf8');
const frame = fs.readFileSync('src/components/site-frame.tsx', 'utf8');
const shell = fs.readFileSync('src/components/workspace-shell.tsx', 'utf8');
const globals = fs.readFileSync('src/app/globals.css', 'utf8');

test('A20 getStorefrontTheme returns classic fallback on failure or timeout', () => {
  assert.match(api, /AbortSignal\.timeout\(2500\)/);
  assert.match(api, /fallback:\s*true/);
  assert.match(api, /fallback:\s*false/);
  assert.match(api, /theme:\s*"classic"/);
  assert.match(api, /Promise<\{\s*theme:\s*StorefrontTheme;\s*fallback:\s*boolean\s*\}>/);
});

test('A20 layout passes fallback to SiteFrame; public pages get no banner', () => {
  assert.match(layout, /fallback/);
  assert.match(layout, /<SiteFrame[^>]*fallback=\{fallback\}/);
  assert.doesNotMatch(layout, /platform-theme-fallback/);
  assert.match(frame, /data-theme-fallback/);
  assert.doesNotMatch(frame, /platform-theme-fallback/);
});

test('A21 only platform admins see the fallback reminder and can refresh', () => {
  assert.match(shell, /platform-theme-fallback/);
  assert.match(shell, /fallback && account\.role === "platform_admin"|themeFallback && account\?\.role === "platform_admin"/);
  assert.match(shell, /router\.refresh\(\)/);
  assert.match(shell, /重新检查/);
});

test('A22 fallback reminder reuses error-message styling and a 44px button', () => {
  assert.match(globals, /\.platform-theme-fallback/);
  assert.match(globals, /\.platform-theme-fallback[\s\S]*?min-height:\s*44px/);
  assert.match(globals, /\.platform-theme-fallback[\s\S]*?(--warning-soft|#e6b5a6)/);
});
