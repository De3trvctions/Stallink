/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const globals = fs.readFileSync('src/app/globals.css', 'utf8');

test('C1 shared radius/shadow/motion tokens exist', () => {
  for (const token of ['--radius-control', '--radius-card', '--radius-panel']) {
    assert.match(globals, new RegExp(`${token}:`), `${token} missing`);
  }
});

test('C2-C4 no legacy green left in globals.css', () => {
  assert.doesNotMatch(globals, /rgba\(49,\s*89,\s*66/);
});

test('C5 focus ring uses --focus', () => {
  assert.match(globals, /:focus-visible[^}]*outline:\s*2px solid var\(--focus\)/);
});

test('C6 one global reduced-motion rule', () => {
  assert.match(globals, /@media \(prefers-reduced-motion: reduce\)\s*{\s*\*,\s*\*::before,\s*\*::after/);
});

test('C7 theme toggle only renders for classic', () => {
  const nav = fs.readFileSync('src/components/app-nav.tsx', 'utf8');
  assert.match(nav, /storefrontTheme\s*(!==|===)\s*"classic"/);
});

const expo = fs.readFileSync('src/app/exhibition-workspace.css', 'utf8');

test('E1/E2 exhibition workspace muted and accent-light', () => {
  assert.match(expo, /--muted:\s*#4b6465/);
  assert.match(expo, /--accent-light:\s*#e3f7a2/);
});

test('E3-E7 shadows and selected state use tokens, no hard-coded lime', () => {
  assert.match(expo, /--shadow-rail:/);
  assert.doesNotMatch(expo, /#d8f69a/);
});

test('E13 report page no longer overrides --muted', () => {
  assert.doesNotMatch(expo, /\.report-page[^{]*{[^}]*--muted:/);
});

test('E25 exhibition workspace font and charcoal tokens', () => {
  assert.match(expo, /--expo-charcoal:\s*#262820/);
  assert.match(expo, /--line-control:\s*#6f8681/);
  assert.doesNotMatch(expo, /--line-control:\s*#(?:738b86|857e6e|8f8c8b)/);
  assert.match(expo, /--font-display:\s*"Songti SC"/);
});

test('E27 sidebar logo uses display font and brand mark', () => {
  // Spec names .workspace-brand__mark; markup is .workspace-sidebar__logo (aria-hidden).
  assert.match(expo, /\.workspace-sidebar__logo\s*\{[^}]*var\(--font-display\)/);
  assert.match(expo, /\.workspace-sidebar__logo::after\s*\{[^}]*var\(--brand-mark\)/);
});

test('E26 active filter text uses charcoal', () => {
  assert.match(expo, /counter-catalog-filter button\[data-active="true"\][\s\S]*?color:\s*var\(--expo-charcoal\)/);
});

test('E14 panel radius uses --radius-panel', () => {
  assert.match(expo, /\.workspace-card,[\s\S]*?--radius-panel/);
});

test('E8/E9 exhibition logout uses --touch', () => {
  assert.match(expo, /\.workspace-sidebar__logout\s*{[^}]*min-height:\s*var\(--touch\)/);
  const narrow = expo.split('@media (max-width: 900px)')[1] || '';
  assert.match(narrow, /\.workspace-sidebar__logout\s*{[^}]*min-height:\s*var\(--touch\)/);
});

test('E10 exhibition nav scrolls and foot stays put', () => {
  assert.match(expo, /\.workspace-nav\s*{[^}]*flex:\s*1 1 auto/);
  assert.match(expo, /\.workspace-nav\s*{[^}]*overflow-y:\s*auto/);
  assert.match(expo, /\.workspace-sidebar__foot\s*{[^}]*flex:\s*0 0 auto/);
  assert.match(expo, /\.workspace-sidebar__foot\s*{[^}]*display:\s*grid/);
});

test('E11 exhibition sidebar indexes stay hidden', () => {
  assert.match(expo, /\.workspace-nav__index\s*{[^}]*display:\s*none/);
});

test('D2 exhibition narrow nav still scrolls horizontally', () => {
  const narrow = expo.split('@media (max-width: 900px)')[1] || '';
  assert.match(narrow, /\.workspace-nav\s*{[^}]*overflow-x:\s*auto/);
});

test('E12 exhibition overview drops navy', () => {
  assert.doesNotMatch(expo, /#0e203a|#22467c/i);
  assert.match(expo, /linear-gradient\(115deg,\s*#123740,\s*#215e58\)/);
});

test('E15 exhibition counter products are a single column at all widths', () => {
  const withoutMid = expo.replace(/@media \(min-width: 1100px\) and \(max-width: 1330px\)[^{]*\{[\s\S]*?\n    \}/, '');
  assert.match(withoutMid, /\.counter-products\s*{[^}]*grid-template-columns:\s*minmax\(0,\s*1fr\)/);
});

const exhibitionCss = fs.readFileSync('src/app/exhibition.css', 'utf8');

test('K-Q19 exhibition customer --line-control is #857e6e in both token blocks', () => {
  const values = [...exhibitionCss.matchAll(/--line-control:\s*([^;]+);/g)].map((m) => m[1].trim());
  assert.ok(values.length >= 2, `expected both token blocks, got ${values.join(', ')}`);
  for (const v of values) assert.equal(v, '#857e6e');
  assert.doesNotMatch(exhibitionCss, /--line-control:\s*#(?:738b86|6f8681|8f8c8b)/);
});

test('E16-E20 exhibition product page drops cobalt', () => {
  assert.doesNotMatch(exhibitionCss, /#1e36a8/i);
});

test('E21 exhibition customer tokens include zero radii', () => {
  const block = exhibitionCss.match(/\[data-storefront-theme="exhibition"\]\s*{[^}]+}/);
  assert.ok(block, 'exhibition token block missing');
  assert.match(block[0], /--radius-control:\s*0/);
  assert.match(block[0], /--radius-card:\s*0/);
  assert.match(block[0], /--radius-panel:\s*0/);
  assert.match(block[0], /--shadow-1:\s*none/);
  assert.match(block[0], /--shadow-2:\s*none/);
  assert.match(block[0], /--focus:\s*#262820/);
  assert.match(block[0], /--brand-mark:\s*#8c3c2d/);
  assert.match(block[0], /--warning-soft:\s*#f3e1d8/);
  assert.match(block[0], /--success:\s*#3d5a2a/);
});

const shop = fs.readFileSync('src/app/shop.css', 'utf8');

test('S1-S4 shop tokens', () => {
  assert.match(shop, /--warning:\s*#b42318/);
  assert.match(shop, /--line-strong:\s*#dedfe0/);
  assert.match(shop, /--accent-dark:\s*#4524db/);
  for (const token of ['--radius-control', '--radius-card', '--radius-panel', '--radius-pill']) {
    assert.match(shop, new RegExp(`${token}:`));
  }
  assert.match(shop, /--line-control:\s*#8f8c8b/);
  assert.doesNotMatch(shop, /--line-control:\s*#(?:738b86|6f8681|857e6e)/);
});

test('S5 no Inter webfont', () => {
  assert.doesNotMatch(shop, /font-(display|ui|mono):\s*Inter/);
});

test('S8 shop sidebar nav scrolls, foot stays put', () => {
  assert.match(shop, /\.workspace-sidebar\s*{[^}]*display:\s*flex/);
  assert.match(shop, /\.workspace-sidebar\s*{[^}]*flex-direction:\s*column/);
  assert.doesNotMatch(shop, /\.workspace-sidebar\s*{[^}]*overflow-y:\s*auto/);
  assert.match(shop, /\.workspace-nav\s*{[^}]*flex:\s*1 1 auto/);
  assert.match(shop, /\.workspace-nav\s*{[^}]*min-height:\s*0/);
  assert.match(shop, /\.workspace-nav\s*{[^}]*overflow-y:\s*auto/);
  assert.match(shop, /\.workspace-sidebar__foot\s*{[^}]*flex:\s*0 0 auto/);
});

test('S9 shop nav group spacing is 16px', () => {
  assert.match(shop, /\.workspace-nav__group\s*{[^}]*margin-bottom:\s*16px/);
});

test('S11-S12 no collapse toggle; narrow nav scrolls horizontally', () => {
  assert.doesNotMatch(shop, /\.workspace-nav-toggle\s*\{/);
  const narrow = shop.split('@media (max-width: 900px)')[1] || '';
  assert.match(narrow, /\.workspace-nav\s*{[^}]*overflow-x:\s*auto/);
});

test('S6/S7 shop heading size is 28px', () => {
  assert.match(shop, /\.workspace-page > main > h1[^{]*\.workspace-overview__heading h1\s*{[^}]*font-size:\s*28px/);
  assert.match(shop, /\.admin-overview \.workspace-overview__heading h1\s*{[^}]*font-size:\s*28px/);
});

test('S13 shop narrow header hides about and view-store links', () => {
  const narrow = shop.split('@media (max-width: 900px)')[1] || '';
  assert.match(narrow, /\.header-link\[href="\/about"\]/);
  assert.match(narrow, /display:\s*none/);
});

test('S14 shop counter products are a single column', () => {
  assert.match(shop, /\.counter-products\s*{[^}]*grid-template-columns:\s*minmax\(0,\s*1fr\)/);
});

test('S15 shop counter filters use a pill track', () => {
  assert.match(shop, /\.counter-catalog-filter\s*{[^}]*border-radius:\s*var\(--radius-pill\)/);
});

test('S16 shop disabled primary buttons use muted gray', () => {
  assert.match(shop, /:is\(\.button--dark, \.ed-btn\):disabled\s*{[^}]*background:\s*#ebebeb/);
  assert.match(shop, /:is\(\.button--dark, \.ed-btn\):disabled\s*{[^}]*color:\s*var\(--shop-muted-strong\)/);
});
