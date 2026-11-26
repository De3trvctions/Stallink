/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');

const shop = fs.readFileSync('src/app/shop.css', 'utf8');

function matchingClose(src, openIdx) {
  let depth = 0;
  for (let i = openIdx; i < src.length; i += 1) {
    if (src[i] === '{') depth += 1;
    else if (src[i] === '}') {
      depth -= 1;
      if (depth === 0) return i;
    }
  }
  return -1;
}

/** Source-order cascade for --shop-header-height, respecting max-width media queries. */
function effectiveShopHeaderHeight(src, width) {
  const text = src.replace(/\/\*[\s\S]*?\*\//g, '');
  const decls = [];
  const re = /--shop-header-height:\s*([^;]+);/g;
  let m;
  while ((m = re.exec(text))) {
    const idx = m.index;
    const before = text.slice(0, idx);
    const enclosing = [];
    const mediaRe = /@media\s*\(\s*max-width:\s*(\d+)px\s*\)\s*\{/g;
    let mm;
    while ((mm = mediaRe.exec(before))) {
      const open = mm.index + mm[0].length - 1;
      const close = matchingClose(text, open);
      if (close > idx) enclosing.push(Number(mm[1]));
    }
    decls.push({ value: m[1].trim(), maxWidths: enclosing, index: idx });
  }
  let value = null;
  for (const decl of decls) {
    if (decl.maxWidths.every((max) => width <= max)) value = decl.value;
  }
  return value;
}

const blocks = (src) => [...src.matchAll(/([^{}]+)\{([^{}]*)\}/g)].map((m) => ({
  sel: m[1].replace(/\/\*[\s\S]*?\*\//g, '').trim(),
  body: m[2],
}));

test('D9 shop narrow logout label sits inside the pill', () => {
  const m900 = shop.match(/@media \(max-width: 900px\)\s*\{([\s\S]*?)\n\}\s*\n@media/);
  assert.ok(m900, '900px shop sidebar media block missing');
  const narrow = m900[1];
  const logout = blocks(narrow).find((b) => /workspace-sidebar__logout/.test(b.sel) && !/sidebar__brand/.test(b.sel) && !/sidebar__back/.test(b.sel));
  assert.ok(logout, '900px .workspace-sidebar__logout rule missing');
  assert.match(logout.body, /display:\s*(inline-flex|flex)/);
  assert.match(logout.body, /align-items:\s*center/);
  assert.match(logout.body, /justify-content:\s*center/);
  assert.match(logout.body, /min-height:\s*44px/);
  assert.match(logout.body, /min-width:\s*44px/);
  assert.match(logout.body, /margin:\s*0 0 0 8px/);
  assert.match(logout.body, /background:\s*var\(--surface\)/);
  assert.match(logout.body, /box-shadow:\s*var\(--shop-shadow-chip\)/);
  assert.match(logout.body, /color:\s*var\(--ink\)/);
  assert.doesNotMatch(logout.body, /margin-top:/, 'margin-top would shift the pill away from the label');
  assert.doesNotMatch(logout.body, /padding-top:/, 'padding-top would shift the pill away from the label');
  // 导航不要占满整行，把「退出登录」挤到下一行、卡片下方空出一块。
  const nav = blocks(narrow).find((b) => /\.workspace-nav\s*$/.test(b.sel.replace(/\s+/g, ' ')) || /workspace-shell \.workspace-nav$/.test(b.sel.replace(/\s+/g, ' ')));
  assert.ok(nav, '900px .workspace-nav rule missing');
  assert.doesNotMatch(nav.body, /flex:\s*1 0 100%/);
});


test('shop sidebar sticky top is 96px on desktop, follows header height at 900/640, and never uses 56px', () => {
  // 只按选择器查吸顶 top 和页头 min-height，不把文件里其它 56px（搜索盒、活动卡片日期列）算进去。
  const sidebarRules = blocks(shop).filter((b) => /workspace-sidebar\s*$/.test(b.sel.replace(/\s+/g, ' ')));
  assert.ok(sidebarRules.some((b) => /top\s*:/.test(b.body)), 'workspace-sidebar top rules missing');
  for (const rule of sidebarRules) {
    assert.doesNotMatch(rule.body, /top:\s*56px/, `${rule.sel} sticky top must not be 56px`);
  }
  const headerInners = blocks(shop).filter((b) => /\.site-header__inner\s*$/.test(b.sel.replace(/\s+/g, ' ')));
  assert.ok(headerInners.length >= 1, '.site-header__inner rule missing');
  for (const rule of headerInners) {
    assert.doesNotMatch(rule.body, /min-height:\s*56px/, `${rule.sel} min-height must not be 56px`);
  }
  assert.match(shop, /--shop-header-height:\s*72px/);
  assert.match(shop, /\.site-header__inner\s*\{[^}]*min-height:\s*var\(--shop-header-height\)/);
  const desktopTops = blocks(shop).filter((b) => /workspace-sidebar\s*$/.test(b.sel.replace(/\s+/g, ' ')) && !/workspace-shell/.test(b.sel) && /top:\s*96px/.test(b.body));
  assert.ok(desktopTops.length >= 1, 'desktop .workspace-sidebar should be top: 96px');
  const m900 = shop.match(/@media \(max-width: 900px\)\s*\{([\s\S]*?)\n\}\s*\n@media/);
  assert.ok(m900, '900px media block missing');
  const sidebar900 = blocks(m900[1]).find((b) => /workspace-shell \.workspace-sidebar$/.test(b.sel.replace(/\s+/g, ' ')));
  assert.ok(sidebar900, '900px workspace-shell sidebar rule missing');
  assert.match(sidebar900.body, /position:\s*sticky/);
  assert.match(sidebar900.body, /top:\s*var\(--shop-header-height\)/);
  const logout = blocks(m900[1]).find((b) => /workspace-sidebar__logout/.test(b.sel) && !/sidebar__brand/.test(b.sel) && !/sidebar__back/.test(b.sel));
  assert.ok(logout, '900px logout rule missing');
  assert.doesNotMatch(logout.body, /display:\s*none/);
  const heightDecls = [];
  {
    const text = shop.replace(/\/\*[\s\S]*?\*\//g, '');
    const re = /--shop-header-height:\s*([^;]+);/g;
    let m;
    while ((m = re.exec(text))) {
      const before = text.slice(0, m.index);
      const enclosing = [];
      const mediaRe = /@media\s*\(\s*max-width:\s*(\d+)px\s*\)\s*\{/g;
      let mm;
      while ((mm = mediaRe.exec(before))) {
        const open = mm.index + mm[0].length - 1;
        const close = matchingClose(text, open);
        if (close > m.index) enclosing.push(Number(mm[1]));
      }
      heightDecls.push({ value: m[1].trim(), maxWidths: enclosing, index: m.index });
    }
  }
  assert.equal(heightDecls.length, 2, '--shop-header-height may only be declared twice (root 72px, 640 64px)');
  assert.deepEqual(heightDecls.map((d) => ({ value: d.value, maxWidths: d.maxWidths })), [
    { value: '72px', maxWidths: [] },
    { value: '64px', maxWidths: [640] },
  ]);
  const decls900 = heightDecls.filter((d) => d.maxWidths.includes(900));
  const decl640 = heightDecls.find((d) => d.maxWidths.includes(640));
  assert.doesNotMatch(m900[1], /--shop-header-height\s*:/, '900 block must not redeclare the header height (it would override the later 640 value)');
  if (decls900.length && decl640) {
    assert.ok(Math.max(...decls900.map((d) => d.index)) < decl640.index, 'if 900 block declares the variable, it must appear before the 640 block');
  }
  assert.equal(effectiveShopHeaderHeight(shop, 1280), '72px');
  assert.equal(effectiveShopHeaderHeight(shop, 900), '72px');
  assert.equal(effectiveShopHeaderHeight(shop, 768), '72px');
  assert.equal(effectiveShopHeaderHeight(shop, 641), '72px');
  assert.equal(effectiveShopHeaderHeight(shop, 640), '64px');
  assert.equal(effectiveShopHeaderHeight(shop, 390), '64px');
  assert.equal(effectiveShopHeaderHeight(shop, 320), '64px');
});

test('shop ≤900 horizontal sidebar is width-constrained so 861–900 does not overflow', () => {
  // 根因：≤900 把侧栏改成横排，但 width:100% 原来只写在 ≤860；再加上
  // .workspace-shell { align-items: start }，width:auto 的侧栏会按内容撑开（实测 900 宽时约 998px）。
  const m900 = shop.match(/@media \(max-width: 900px\)\s*\{([\s\S]*?)\n\}\s*\n@media/);
  assert.ok(m900, '900px media block missing');
  const sidebar900 = blocks(m900[1]).find((b) => /workspace-shell \.workspace-sidebar$/.test(b.sel.replace(/\s+/g, ' ')));
  assert.ok(sidebar900, '900px workspace-shell sidebar rule missing');
  assert.match(sidebar900.body, /width:\s*100%/);
  assert.match(sidebar900.body, /min-width:\s*0/);
  assert.match(sidebar900.body, /box-sizing:\s*border-box/);
});
