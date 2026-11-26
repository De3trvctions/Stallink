/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const vm = require('node:vm');
const ts = require('typescript');

function load() {
  const source = fs.readFileSync('src/lib/booth.ts', 'utf8');
  const code = ts.transpileModule(source, { compilerOptions: { module: ts.ModuleKind.CommonJS } }).outputText;
  const exports = {};
  vm.runInNewContext(code, { exports });
  return exports;
}

test('customerBoothLabel prefers displayBoothNumber, then boothNumber, else 待公布', () => {
  const { customerBoothLabel } = load();
  assert.equal(customerBoothLabel({ displayBoothNumber: 'A-27', boothNumber: '' }), 'A-27');
  assert.equal(customerBoothLabel({ displayBoothNumber: 'B-01', boothNumber: 'B-01' }), 'B-01');
  assert.equal(customerBoothLabel({ displayBoothNumber: '', boothNumber: '' }), '待公布');
  assert.equal(customerBoothLabel({ boothNumber: 'C-03' }), 'C-03');
});

test('customer pages use customerBoothLabel instead of event.boothNumber || 待公布', () => {
  const files = [
    'src/app/c/[slug]/events/[eventId]/page.tsx',
    'src/app/c/[slug]/listings/[listingId]/page.tsx',
    'src/app/c/[slug]/bundles/[bundleListingId]/page.tsx',
    'src/components/booth-view.tsx',
    'src/components/goods-grid.tsx',
  ];
  for (const file of files) {
    const src = fs.readFileSync(file, 'utf8');
    assert.match(src, /customerBoothLabel/, `${file} should use customerBoothLabel`);
    assert.doesNotMatch(src, /event\.boothNumber\s*\|\|\s*"待公布"/, `${file} still inlines booth fallback`);
  }
});

test('new event forms prefill boothNumber and defaultCurrency from GET /api/admin/shop', () => {
  for (const file of ['src/app/admin/events/page.tsx', 'src/app/admin/goods/page.tsx']) {
    const src = fs.readFileSync(file, 'utf8');
    assert.match(src, /\/api\/admin\/shop/, file);
    assert.match(src, /defaultCurrency/, file);
    assert.match(src, /boothNumber/, file);
  }
});
