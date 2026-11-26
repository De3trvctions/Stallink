/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const shell = fs.readFileSync('src/components/workspace-shell.tsx', 'utf8');
const overviewSrc = fs.readFileSync('src/app/admin/page.tsx', 'utf8');

// spec §5.2 / C8 / C11（12:44 版）：同序、同号、同名、同分组
const business = [
  ['/admin/events', '01', '活动'], ['/admin/goods', '02', '商品'],
  ['/admin/orders', '03', '订单记录'], ['/admin/reports', '04', '销售日报'],
];
const shop = [
  ['/admin/pay-codes', '05', '收款方式'], ['/admin/sales', '06', '销售账号'],
  ['/admin/shop', '07', '店铺资料'], ['/admin/audit', '08', '操作记录'],
];
const managed = [...business, ...shop];

// 取出所有含 href 的对象字面量，读 href / index / label|title
function items(src) {
  return [...src.matchAll(/\{[^{}]*href: "([^"]+)"[^{}]*\}/g)].map((m) => ({
    href: m[1],
    index: (m[0].match(/index: "([^"]*)"/) || [, ''])[1],
    name: (m[0].match(/title: "([^"]*)"/) || m[0].match(/label: "([^"]*)"/) || [, ''])[1], // 卡片用 title，侧栏用 label
    at: m.index,
  }));
}
const shopDesk = shell.slice(shell.indexOf('const cashierGroup'), shell.indexOf('sales: {'));
const nav = items(shopDesk);
const cards = items(overviewSrc.slice(0, overviewSrc.indexOf('export default')));
const find = (list, href) => list.find((i) => i.href === href);

test('sidebar: 8 managed items 01-08 with spec names', () => {
  for (const [href, index, name] of managed) {
    const it = find(nav, href);
    assert.ok(it, `sidebar ${href}`);
    assert.equal(it.index, index, `sidebar index ${href}`);
    assert.equal(it.name, name, `sidebar name ${href}`);
  }
});

test('sidebar: 总览 and cashier group are unnumbered', () => {
  for (const href of ['/admin', '/counter', '/counter/orders']) {
    assert.equal(find(nav, href).index, '', `${href} must have no index`);
  }
  assert.equal(find(nav, '/counter').name, '现场开单');
  assert.equal(find(nav, '/counter/orders').name, '本场订单');
  assert.doesNotMatch(shell, /index: "(POS|ORD|00|09|10)"/);
});

test('sidebar: groups 经营(总览, 01-04) → 现场收银 → 店铺(05-08)', () => {
  const desk = shopDesk.slice(shopDesk.indexOf('groups:'));
  const pos = (s) => { const i = desk.indexOf(s); assert.ok(i >= 0, s); return i; };
  const order = [
    pos('label: "经营"'), pos('href: "/admin",'),
    ...business.map(([h]) => pos(`href: "${h}"`)),
    pos('cashierGroup'), pos('label: "店铺"'),
    ...shop.map(([h]) => pos(`href: "${h}"`)),
  ];
  assert.deepEqual(order, [...order].sort((x, y) => x - y));
  assert.match(shell, /const cashierGroup: NavGroup = \{\s*label: "现场收银"/); // C9：一个常量，不是工厂
});

test('overview: exactly the 8 managed cards, same index and name as sidebar', () => {
  assert.equal(cards.length, 8);
  assert.deepEqual(cards.map((c) => [c.href, c.index, c.name]), managed);
  assert.ok(!cards.some((c) => c.href.startsWith('/counter')), 'no cashier cards');
  assert.doesNotMatch(overviewSrc, /活动与场次|商品与套餐/);
});

test('overview: cards grouped 经营 01-04, 店铺 05-08', () => {
  const head = overviewSrc.slice(0, overviewSrc.indexOf('export default'));
  const b = head.indexOf('"经营"'); const s = head.indexOf('"店铺"');
  assert.ok(b >= 0 && s > b, 'group labels 经营 then 店铺');
  for (const c of cards) {
    const inBusiness = c.at > b && c.at < s;
    assert.equal(inBusiness, Number(c.index) <= 4, `${c.href} group`);
  }
});

test('C10 sidebar foot wraps back link and logout', () => {
  assert.match(shell, /className="workspace-sidebar__foot"/);
});

test('C15 counter H1 is 现场开单 (spec Q5 = A, Kel decided 13:19)', () => {
  const counter = fs.readFileSync('src/app/counter/page.tsx', 'utf8');
  assert.match(counter, /<h1>现场开单<\/h1>/);
});
