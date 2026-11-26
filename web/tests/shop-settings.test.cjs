/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');

const page = fs.readFileSync('src/app/admin/shop/page.tsx', 'utf8');
const errors = fs.readFileSync('src/lib/errors.ts', 'utf8');

test('shop settings read GET /api/admin/shop and no longer read public shops', () => {
  assert.match(page, /\/api\/admin\/shop/);
  assert.doesNotMatch(page, /\/api\/public\/shops\//);
});

test('PATCH body does not include slug; slug is read-only with 由平台分配 copy', () => {
  const patch = page.match(/browserJSON\(\s*"\/api\/admin\/shop",\s*"PATCH",\s*\{[^}]+\}\s*\)/);
  assert.ok(patch, 'PATCH call not found');
  assert.doesNotMatch(patch[0], /\bslug\b/);
  assert.match(page, /由平台分配/);
  assert.doesNotMatch(page, /<input[^>]*id="shop-slug"/);
});

test('contact, default booth number, and default currency are editable and no longer say 等后端', () => {
  assert.doesNotMatch(page, /等后端/);
  assert.doesNotMatch(page, /id="shop-contact"[^>]*disabled/);
  assert.doesNotMatch(page, /id="shop-booth"[^>]*disabled/);
  assert.doesNotMatch(page, /id="shop-currency"[^>]*disabled/);
  assert.match(page, /maxLength=\{120\}/);
  assert.match(page, /默认摊位号/);
  assert.match(page, /新建活动时自动带入；顾客端优先显示活动自己的摊位号，活动没填才用这里的/);
  assert.match(page, /新建活动时自动带入/);
  assert.match(page, /首尾不能有空格/);
});

test('page displays updatedAt from the admin shop profile', () => {
  assert.match(page, /updatedAt/);
  assert.doesNotMatch(page, /最近更新等后端提供时间字段后再显示/);
});

test('disable-shop copy warns staff cannot enter the workspace after login', () => {
  assert.match(page, /登录后进不了工作台/);
  assert.match(page, /需要平台管理员重新启用/);
  assert.doesNotMatch(page, /登录被拒绝/);
});

test('errors.ts maps slug_readonly and invalid_shop', () => {
  assert.match(errors, /slug_readonly:\s*"店铺地址由平台分配，不能修改"/);
  assert.match(errors, /invalid_shop:\s*"/);
});
