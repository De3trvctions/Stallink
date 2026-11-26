/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');

const src = fs.readFileSync('src/components/confirm-dialog.tsx', 'utf8');
const appearance = fs.readFileSync('src/app/platform/appearance/page.tsx', 'utf8');

test('ConfirmDialog accepts optional initialFocus and defaults stay danger-based', () => {
  assert.match(src, /initialFocus\?:\s*"cancel"\s*\|\s*"confirm"/);
  assert.match(src, /initialFocus \?\? \(danger \? "cancel" : "confirm"\)|focusTarget = initialFocus \?\? \(danger \? "cancel" : "confirm"\)/);
  // Default path still keys off danger when initialFocus is omitted.
  assert.match(src, /danger/);
});

test('appearance publish dialog focuses 取消 without changing default danger=false', () => {
  const block = appearance.slice(appearance.indexOf('<ConfirmDialog'), appearance.indexOf('/>', appearance.indexOf('<ConfirmDialog')) + 2);
  assert.match(block, /initialFocus="cancel"/);
  assert.match(block, /danger=\{false\}/);
});
