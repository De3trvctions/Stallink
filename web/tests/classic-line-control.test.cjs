/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const read = (p) => fs.readFileSync(p, 'utf8');
const globals = read('src/app/globals.css');

// WCAG 相对亮度和对比度
const lum = (hex) => {
  const c = hex.replace('#', '').match(/../g).map((h) => parseInt(h, 16) / 255)
    .map((v) => (v <= 0.03928 ? v / 12.92 : ((v + 0.055) / 1.055) ** 2.4));
  return 0.2126 * c[0] + 0.7152 * c[1] + 0.0722 * c[2];
};
const ratio = (a, b) => { const [x, y] = [lum(a), lum(b)].sort((m, n) => n - m); return (x + 0.05) / (y + 0.05); };

// 取出所有“选择器 { 声明 }”块（不处理嵌套 @media 的外层，只看最内层）
const blocks = (src) => [...src.matchAll(/([^{}]+)\{([^{}]*)\}/g)].map((m) => ({ sel: m[1].replace(/\/\*[\s\S]*?\*\//g, '').trim(), body: m[2] }));
const rule = (src, sel) => blocks(src).filter((b) => b.sel.split(',').map((s) => s.replace(/^&\s*/, '').trim()).includes(sel));

test('classic light: --line-control is #8f8675 and passes 3:1 on every classic surface', () => {
  assert.equal((globals.match(/--line-control:\s*#8f8675;/g) || []).length, 1, ':root defines --line-control #8f8675 once');
  assert.ok(globals.indexOf('--line-control: #8f8675') < globals.indexOf('prefers-color-scheme: dark'), 'defined in the light :root block');
  assert.match(globals, /--line-strong:\s*#c6bdaa;/, 'divider token unchanged');
  for (const bg of ['#faf8f3', '#fffdf8', '#ffffff', '#fffefa', '#f0ece2']) {
    assert.ok(ratio('#8f8675', bg) >= 3, `#8f8675 on ${bg}: ${ratio('#8f8675', bg).toFixed(2)}`);
  }
});

test('classic dark: both dark token blocks set --line-control #676c78 and pass 3:1', () => {
  assert.equal((globals.match(/--line-control:\s*#676c78;/g) || []).length, 2);
  assert.equal((globals.match(/--line-strong:\s*#40444f;/g) || []).length, 2, 'dark divider token unchanged');
  for (const bg of ['#0e1014', '#16181d', '#1e2129']) {
    assert.ok(ratio('#676c78', bg) >= 3, `#676c78 on ${bg}: ${ratio('#676c78', bg).toFixed(2)}`);
  }
});

const controls = [
  // §3.1（浅色 33）
  '.button--outline', '.field input', '.field textarea', '.field select', '.workspace-toolbar button:not(.button)',
  '.workspace-page select', '.workspace-page form .select-menu__trigger', '.workspace-page li button',
  '.pay-code-form input[type="file"]', '.workspace-page input[type="file"]', '.tier-editor__remove',
  '.workspace-page form button.tier-editor__remove', '.goods-search', '.counter-catalog-search input',
  '.workspace-page form .counter-catalog-filter button', '.quantity-control', '.header-account', '.draft-gallery__upload',
  '.image-picker', '.counter-page .counter-sheet__close', '.pay-dialog__close', '.goods-view-tabs',
  '.good-listing-editor__release', '.bundle-existing-picker__head button', '.workspace-page form .bundle-existing-picker__head button',
  '.bundle-existing-picker input', '.review-switch', '.review-switch__track', '.report-date input', '.select-menu__trigger',
  '.workspace-page .select-menu__trigger', '.slug-input > span', '.catalog__box',
  // §3.2
  '.goods-event-option', '.counter-product--bundle', '.workspace-shell .stock-meter__bar', '.text-link',
  // 附录（--line）
  '.product-gallery__thumbs button', '.bundle-component-draft__images button', '.review-filter button',
  '.counter-catalog-filter button', '.workspace-page form button.counter-product',
];

test('every listed control uses var(--line-control), none keeps --line-strong / --line on its border', () => {
  for (const sel of controls) {
    const hits = rule(globals, sel).filter((b) => /border|text-decoration-color/.test(b.body));
    assert.ok(hits.length, `rule not found: ${sel}`);
    assert.ok(hits.some((b) => /var\(--line-control\)/.test(b.body)), `${sel} should use --line-control`);
    for (const b of hits) {
      assert.doesNotMatch(b.body, /(border(-color)?|text-decoration-color):[^;]*var\(--line(-strong)?\)/, `${sel} still uses --line/--line-strong`);
    }
  }
});

test('dark blocks: 3 control groups use --line-control in both dark blocks (6)', () => {
  const dark = blocks(globals).filter((b) => /^&\s*\.(button--outline|workspace-page form)/.test(b.sel) && /border-color:\s*var\(--line-control\)/.test(b.body));
  assert.ok(dark.length >= 6, `dark control rules using --line-control: ${dark.length}`);
});

test('dividers, disabled and hover keep their original tokens', () => {
  for (const [sel, token] of [['.section-heading', '--line-strong'], ['.report-hero', '--line-strong'], ['.order-status', '--line-strong'],
    ['.confirm-dialog', '--line-strong'], ['.event-card:hover', '--line-strong'], ['.workspace-page form button.counter-product--unavailable', '--line']]) {
    assert.ok(rule(globals, sel).some((b) => b.body.includes(`var(${token})`)), `${sel} must keep ${token}`);
  }
});

test('goods page inline picker border uses --line-control', () => {
  assert.match(read('src/app/admin/goods/page.tsx'), /pickerStyle[^;]*border:\s*"1px solid var\(--line-control\)"/);
});

test('other themes unchanged: every block that redefines --line-strong also defines --line-control', () => {
  for (const file of ['src/app/shop.css', 'src/app/exhibition.css', 'src/app/exhibition-workspace.css']) {
    for (const b of blocks(read(file)).filter((x) => /--line-strong:/.test(x.body))) {
      assert.match(b.body, /--line-control:/, `${file} block "${b.sel.slice(0, 60)}" must define --line-control`);
    }
  }
});

test('E28: exhibition workspace controls read --line-control; card hover keeps --line-strong (Kel 14:40)', () => {
  const ws = read('src/app/exhibition-workspace.css');
  const strong = blocks(ws).filter((b) => /border-color:\s*var\(--line-strong\)/.test(b.body)).map((b) => b.sel.trim());
  // 唯一例外：卡片悬停边框不是控件，Kel 14:40 定保持 --line-strong
  assert.deepEqual(strong, ['.workspace-shell .workspace-card:hover'], 'only the card hover border may stay on --line-strong');
});

test('exhibition workspace E28 form controls use --line-control', () => {
  const expo = read('src/app/exhibition-workspace.css');
  assert.match(
    expo,
    /counter-catalog-filter button\[data-active="true"\][\s\S]*?border-color:\s*var\(--line-control\)/,
    'active filter buttons',
  );
  assert.match(
    expo,
    /counter-product:has\(\.counter-product__count\)\s*\{[^}]*border-color:\s*var\(--line-control\)/,
    'selected counter product',
  );
  assert.match(
    expo,
    /\.workspace-shell \.review-switch__track\s*\{[^}]*border-color:\s*var\(--line-control\)/,
    'review switch track',
  );
});

test('K-Q19: exhibition customer, workspace, and shop set distinct --line-control hex values (not #738b86)', () => {
  const expected = {
    'src/app/exhibition.css': '#857e6e',
    'src/app/exhibition-workspace.css': '#6f8681',
    'src/app/shop.css': '#8f8c8b',
  };
  const surfaces = {
    '#857e6e': ['#f4f0e6', '#fcf9f0', '#e8e2d5', '#eee5d4', '#ffffff'],
    '#6f8681': ['#edf3f1', '#ffffff', '#e2ece8', '#dcebe3', '#d5e6df'],
    '#8f8c8b': ['#f2f4f5', '#ffffff', '#fef3f2', '#fdf4ef'],
  };
  for (const [file, hex] of Object.entries(expected)) {
    const src = read(file);
    const values = [...src.matchAll(/--line-control:\s*([^;]+);/g)].map((m) => m[1].trim());
    assert.ok(values.length, `${file} has no --line-control`);
    for (const v of values) assert.equal(v, hex, `${file} --line-control is ${v}, expected ${hex}`);
    assert.doesNotMatch(src, /--line-control:\s*#738b86/);
    for (const bg of surfaces[hex]) {
      assert.ok(ratio(hex, bg) >= 3, `${hex} on ${bg}: ${ratio(hex, bg).toFixed(3)}`);
    }
  }
});
