/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const ts = require('typescript');

function load() {
  const source = fs.readFileSync('src/lib/audit.ts', 'utf8');
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
  const exports = {};
  vm.runInNewContext(code, { exports });
  return exports;
}

test('auditSubjectLabel uses type and id, and falls back to an em dash', () => {
  const { auditSubjectLabel } = load();
  assert.equal(auditSubjectLabel({ subjectType: 'order', subjectId: '101' }), '订单 #101');
  assert.equal(auditSubjectLabel({ subjectType: 'good', subjectId: '7' }), '商品 #7');
  assert.equal(auditSubjectLabel({ subjectType: 'listing', subjectId: '3' }), '上架 #3');
  assert.equal(auditSubjectLabel({ subjectType: 'bundle', subjectId: '2' }), '套餐 #2');
  assert.equal(auditSubjectLabel({ subjectType: 'custom', subjectId: '9' }), 'custom #9');
  assert.equal(auditSubjectLabel({ subjectType: '', subjectId: '' }), '—');
  assert.equal(auditSubjectLabel({ subjectType: '  ', subjectId: '  ' }), '—');
});

test('audit page renders an object column from subject fields', () => {
  const source = fs.readFileSync('src/app/admin/audit/page.tsx', 'utf8');
  assert.match(source, /auditSubjectLabel/);
  assert.match(source, /audit-row__subject/);
  assert.match(source, />对象</);
});


test('auditSubjectLabel prefers subjectLabel and falls back to type #id', () => {
  const { auditSubjectLabel } = load();
  assert.equal(auditSubjectLabel({ subjectType: 'good', subjectId: '12', subjectLabel: '亚克力立牌·海雾' }), '商品 · 亚克力立牌·海雾');
  assert.equal(auditSubjectLabel({ subjectType: 'good', subjectId: '12', subjectLabel: '' }), '商品 #12');
  assert.equal(auditSubjectLabel({ subjectType: '', subjectId: '', subjectLabel: '' }), '—');
});

test('audit timestamps are formatted in Asia/Kuala_Lumpur from RFC3339', () => {
  const source = fs.readFileSync('src/app/admin/audit/page.tsx', 'utf8');
  const lib = fs.readFileSync('src/lib/audit.ts', 'utf8');
  assert.match(source + lib, /Asia\/Kuala_Lumpur/);
  assert.match(source + lib, /Intl\.DateTimeFormat/);
});
