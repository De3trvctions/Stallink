/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');

test('workspace page titles follow the sidebar and design names', () => {
  const reports = fs.readFileSync('src/app/admin/reports/page.tsx', 'utf8');
  assert.match(reports, /<p className="eyebrow">经营<\/p>\s*<h1>销售日报<\/h1>/);
  assert.doesNotMatch(reports, /经营日报/);

  const adminOrders = fs.readFileSync('src/app/admin/orders/page.tsx', 'utf8');
  assert.match(adminOrders, /<h1>订单记录<\/h1>/);
  assert.doesNotMatch(adminOrders, /核对现场订单/);

  const counterOrders = fs.readFileSync('src/app/counter/orders/page.tsx', 'utf8');
  assert.match(counterOrders, /<h1>本场订单<\/h1>/);

  const shell = fs.readFileSync('src/components/workspace-shell.tsx', 'utf8');
  assert.match(shell, /href: "\/counter\/orders", label: "本场订单"/);
  assert.match(shell, /href: "\/admin\/reports", label: "销售日报"/);
  assert.match(shell, /href: "\/admin\/orders", label: "订单记录"/);
  assert.doesNotMatch(shell, /label: "现场订单"/);
});

test('overview cards follow sidebar order and include sales report and audit', () => {
  const source = fs.readFileSync('src/app/admin/page.tsx', 'utf8');
  const titles = [...source.matchAll(/title: "([^"]+)"/g)].map((match) => match[1]);
  assert.deepEqual(titles, ['活动', '商品', '订单记录', '销售日报', '收款方式', '销售账号', '店铺资料', '操作记录']);
  const indexes = [...source.matchAll(/index: "(\d+)"/g)].map((match) => match[1]);
  assert.deepEqual(indexes, ['01', '02', '03', '04', '05', '06', '07', '08']);
});

test('shop settings does not expose an editable slug and PATCHes contact, booth, and default currency', () => {
  const source = fs.readFileSync('src/app/admin/shop/page.tsx', 'utf8');
  assert.doesNotMatch(source, /id="shop-slug"/);
  assert.doesNotMatch(source, /onChange=\{\(event\) => setSlug/);
  assert.match(source, /由平台分配，不可自行修改/);
  assert.match(source, /id="shop-contact"/);
  assert.match(source, /id="shop-booth"/);
  assert.match(source, /id="shop-currency"/);
  assert.match(source, /打开店铺预览/);
  assert.doesNotMatch(source, /等后端/);
  assert.match(source, /shop-settings-preview/);
  const patch = source.match(/browserJSON\(\s*"\/api\/admin\/shop",\s*"PATCH",\s*\{[^}]+\}\s*\)/);
  assert.ok(patch, 'PATCH call not found');
  assert.match(patch[0], /contact/);
  assert.match(patch[0], /boothNumber/);
  assert.match(patch[0], /defaultCurrency/);
  assert.doesNotMatch(patch[0], /\bslug\b/);
});

test('shop sidebar management items are numbered 01-08 to match overview cards', () => {
  const shell = fs.readFileSync('src/components/workspace-shell.tsx', 'utf8');
  const overview = fs.readFileSync('src/app/admin/page.tsx', 'utf8');

  const itemPattern = /\{\s*href: "([^"]+)",\s*label: "([^"]+)"(?:,\s*index: "([^"]+)")?(?:,\s*icon: "[^"]+")?\s*\}/g;
  function itemsIn(block) {
    return [...block.matchAll(itemPattern)].map((match) => ({
      href: match[1],
      label: match[2],
      index: match[3] || null,
    }));
  }

  const cashierStart = shell.indexOf('const cashierGroup');
  const cashierEnd = shell.indexOf('};', cashierStart);
  const cashierItems = itemsIn(shell.slice(cashierStart, cashierEnd));
  assert.deepEqual(cashierItems.map((item) => item.label), ['现场开单', '本场订单']);
  assert.deepEqual(cashierItems.map((item) => item.index), [null, null]);
  assert.doesNotMatch(shell.slice(cashierStart, cashierEnd), /index: "POS"|index: "ORD"/);

  const shopBlock = shell.slice(shell.indexOf('shop: {'), shell.indexOf('sales: {'));
  const shopItems = itemsIn(shopBlock);
  const overviewLink = shopItems.find((item) => item.href === '/admin');
  assert.ok(overviewLink, 'shop nav includes 总览');
  assert.equal(overviewLink.label, '总览');
  assert.equal(overviewLink.index, null);
  assert.doesNotMatch(shopBlock, /index: "00"/);

  const cardHrefs = [...overview.matchAll(/href: "([^"]+)"/g)].map((match) => match[1]);
  const cardTitles = [...overview.matchAll(/title: "([^"]+)"/g)].map((match) => match[1]);
  const cardIndexes = [...overview.matchAll(/index: "(\d+)"/g)].map((match) => match[1]);
  assert.deepEqual(cardIndexes, ['01', '02', '03', '04', '05', '06', '07', '08']);

  const expectedLabels = ['活动', '商品', '订单记录', '销售日报', '收款方式', '销售账号', '店铺资料', '操作记录'];
  const management = expectedLabels.map((_, index) => {
    const href = cardHrefs[index];
    const item = shopItems.find((entry) => entry.href === href);
    assert.ok(item, `shop nav missing ${href}`);
    return item;
  });
  assert.deepEqual(management.map((item) => item.href), cardHrefs);
  assert.deepEqual(management.map((item) => item.index), cardIndexes);
  assert.deepEqual(management.map((item) => item.label), expectedLabels);
  assert.deepEqual(cardTitles, expectedLabels);
  assert.deepEqual(management.map((item) => item.label), cardTitles);

  const salesBlock = shell.slice(shell.indexOf('sales: {'), shell.indexOf('platform: {'));
  assert.match(salesBlock, /groups: \[cashierGroup\]/);
  assert.doesNotMatch(salesBlock, /index: "/);

  assert.match(shell, /\{item\.index \? <span className="workspace-nav__index"/);
  assert.match(shell, /index\?: string/);
});

test('shop no longer has a collapse toggle and only classic shows sidebar indexes', () => {
  const shell = fs.readFileSync('src/components/workspace-shell.tsx', 'utf8');
  assert.doesNotMatch(shell, /workspace-nav-toggle/);
  assert.doesNotMatch(shell, /openNavigationPath|navigationOpen|setOpenNavigationPath/);
  assert.match(shell, /nav\.scrollLeft \+=/);
  assert.match(shell, /nav\.scrollTop \+=/);
  assert.match(shell, /overflowY === "auto" \|\| style\.overflowY === "scroll"/);
  assert.match(shell, /\[pathname, allowed\]/);

  const shop = fs.readFileSync('src/app/shop.css', 'utf8');
  assert.match(shop, /body:has\(#storefront-shell\[data-storefront-theme="shop"\]\) \.workspace-shell \.workspace-nav__index \{\s*display: none;/);
  assert.doesNotMatch(shop, /workspace-nav-toggle\[aria-expanded/);
  assert.match(shop, /@media \(max-width: 900px\)/);
  assert.match(shop, /overflow-x: auto;/);

  const exhibition = fs.readFileSync('src/app/exhibition-workspace.css', 'utf8');
  assert.match(exhibition, /\.workspace-nav__index \{\s*display: none;/);
});

test('exhibition storefront rail keeps logout at the bottom when signed in', () => {
  const nav = fs.readFileSync('src/components/app-nav.tsx', 'utf8');
  const exhibitionPublic = nav.slice(nav.indexOf('if (storefrontTheme === "exhibition" && !isWorkspace)'), nav.indexOf('return (\n    <header className={`site-header'));
  assert.match(exhibitionPublic, /exhibition-nav__settings/);
  assert.match(exhibitionPublic, /<RailIcon name="logout"/);
  assert.match(exhibitionPublic, /\{loading \? "退出中" : "退出"\}/);
  assert.match(exhibitionPublic, /onClick=\{logout\}/);
  const exhibitionCss = fs.readFileSync('src/app/exhibition.css', 'utf8');
  const narrow = exhibitionCss.split('@media (max-width: 620px)')[1] || '';
  assert.match(narrow, /--expo-rail:\s*52px/);
  assert.match(narrow, /\.exhibition-nav__settings\s*\{[^}]*position:\s*sticky/);
  assert.doesNotMatch(exhibitionCss, /--expo-rail:\s*0px/);
});
