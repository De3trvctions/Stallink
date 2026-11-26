/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const ts = require('typescript');

function load() {
  const source = fs.readFileSync('src/lib/appearance-status.ts', 'utf8');
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
  const exports = {};
  vm.runInNewContext(code, { exports });
  return exports;
}

test('appearancePublishLabel formats publisher and time, null metadata, and just-published copy', () => {
  const { appearancePublishLabel } = load();
  assert.equal(
    appearancePublishLabel({ updatedAt: '2026-11-26T21:30:00+08:00', updatedBy: { id: '7', name: '运营 Kel' } }),
    '上次发布：运营 Kel · 11/26 21:30',
  );
  assert.equal(appearancePublishLabel({ updatedAt: null }), '尚无记录');
  assert.equal(appearancePublishLabel({}, { justPublished: true }), '你 · 刚刚');
});
