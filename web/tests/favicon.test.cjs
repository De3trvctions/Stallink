/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');
const path = require('node:path');
const app = 'src/app';
const icons = 'public/icons';

function exists(rel) {
  return fs.existsSync(path.join(rel));
}

function readPngHeader(file) {
  const data = fs.readFileSync(file);
  assert.equal(data.subarray(0, 8).equals(Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])), true, `${file} is not a PNG`);
  const width = data.readUInt32BE(16);
  const height = data.readUInt32BE(20);
  const bitDepth = data[24];
  const colorType = data[25];
  return { width, height, bitDepth, colorType, data };
}

function parseIco(file) {
  const buf = fs.readFileSync(file);
  const reserved = buf.readUInt16LE(0);
  const type = buf.readUInt16LE(2);
  const count = buf.readUInt16LE(4);
  const entries = [];
  for (let i = 0; i < count; i += 1) {
    const o = 6 + i * 16;
    const payloadOffset = buf.readUInt32LE(o + 12);
    const bytes = buf.readUInt32LE(o + 8);
    const payload = buf.subarray(payloadOffset, payloadOffset + bytes);
    entries.push({
      width: buf[o] || 256,
      height: buf[o + 1] || 256,
      payload,
    });
  }
  return { reserved, type, count, entries };
}

test('App Router favicon and icon files from option A are in place', () => {
  assert.equal(exists(`${app}/favicon.ico`), true, 'src/app/favicon.ico');
  assert.equal(exists(`${app}/icon.svg`), true, 'src/app/icon.svg');
  assert.equal(exists(`${app}/apple-icon.png`), true, 'src/app/apple-icon.png');
  assert.equal(exists(`${app}/manifest.ts`), true, 'src/app/manifest.ts');
  for (const name of ['icon-192.png', 'icon-512.png', 'icon-192-maskable.png', 'icon-512-maskable.png']) {
    assert.equal(exists(`${icons}/${name}`), true, `${icons}/${name}`);
  }
});

test('icon.svg is the option A charcoal and lime wordmark, not a design-draft comment copy', () => {
  const svg = fs.readFileSync(`${app}/icon.svg`, 'utf8');
  assert.match(svg, /#262820/);
  assert.match(svg, /#e3f7a2/);
  assert.match(svg, /<title[^>]*>Stallink<\/title>/);
  assert.doesNotMatch(svg, /不是前端用的最终资源|不要复制进 web/);
});

test('favicon.ico contains 16, 32 and 48 PNG images with transparent corners', () => {
  const ico = parseIco(`${app}/favicon.ico`);
  assert.equal(ico.reserved, 0);
  assert.equal(ico.type, 1);
  assert.equal(ico.count, 3);
  assert.deepEqual(ico.entries.map((entry) => entry.width), [16, 32, 48]);
  assert.deepEqual(ico.entries.map((entry) => entry.height), [16, 32, 48]);
  for (const entry of ico.entries) {
    assert.equal(entry.payload.subarray(0, 8).equals(Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])), true, `${entry.width} is not PNG-in-ICO`);
    const colorType = entry.payload[25];
    assert.equal(colorType, 6, `${entry.width} should keep alpha (no flatten)`);
  }
});

test('apple-icon and manifest PNGs are the required sizes and opaque charcoal', () => {
  const files = {
    [`${app}/apple-icon.png`]: 180,
    [`${icons}/icon-192.png`]: 192,
    [`${icons}/icon-512.png`]: 512,
    [`${icons}/icon-192-maskable.png`]: 192,
    [`${icons}/icon-512-maskable.png`]: 512,
  };
  for (const [file, size] of Object.entries(files)) {
    const header = readPngHeader(file);
    assert.equal(header.width, size, file);
    assert.equal(header.height, size, file);
    assert.ok(header.colorType === 2 || header.colorType === 3, `${file} should be opaque (no alpha color type), got ${header.colorType}`);
  }
});

test('manifest uses option A background, has no theme_color, and lists public icon paths', () => {
  const src = fs.readFileSync(`${app}/manifest.ts`, 'utf8');
  assert.match(src, /name:\s*"Stallink"/);
  assert.match(src, /short_name:\s*"Stallink"/);
  assert.match(src, /start_url:\s*"\/"/);
  assert.match(src, /display:\s*"standalone"/);
  assert.match(src, /background_color:\s*"#262820"/);
  assert.doesNotMatch(src, /theme_color/);
  assert.match(src, /\/icons\/icon-192\.png/);
  assert.match(src, /\/icons\/icon-512\.png/);
  assert.match(src, /\/icons\/icon-192-maskable\.png/);
  assert.match(src, /\/icons\/icon-512-maskable\.png/);
  assert.match(src, /purpose:\s*"any"/);
  assert.match(src, /purpose:\s*"maskable"/);
});

test('root layout does not duplicate metadata.icons or set theme-color', () => {
  const layout = fs.readFileSync(`${app}/layout.tsx`, 'utf8');
  assert.doesNotMatch(layout, /icons\s*:/);
  assert.doesNotMatch(layout, /themeColor|theme-color/);
});
