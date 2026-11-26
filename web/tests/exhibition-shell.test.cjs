/* eslint-disable @typescript-eslint/no-require-imports */
const { test } = require('node:test');
const assert = require('node:assert/strict');
const fs = require('node:fs');

const css = fs.readFileSync('src/app/exhibition.css', 'utf8');
const nav = fs.readFileSync('src/components/app-nav.tsx', 'utf8');

function mediaBlock(src, query) {
  const start = src.search(new RegExp(`@media \\(max-width: ${query}\\)\\s*\\{`));
  if (start < 0) return '';
  const open = src.indexOf('{', start);
  let depth = 0;
  for (let i = open; i < src.length; i += 1) {
    if (src[i] === '{') depth += 1;
    else if (src[i] === '}') {
      depth -= 1;
      if (depth === 0) return src.slice(open, i + 1);
    }
  }
  return src.slice(open);
}

const narrow = mediaBlock(css, '620px');

test('E29 narrow exhibition rail is 52px', () => {
  assert.match(narrow, /--expo-rail:\s*52px/);
  assert.match(narrow, /\.exhibition-nav\s*{[^}]*padding:\s*\.5rem 0/);
  // K-Q19: do not assert a hex value for --line-control
  assert.doesNotMatch(narrow, /--line-control:\s*#857e6e/);
  assert.doesNotMatch(narrow, /--line-control:\s*#738b86/);
  assert.doesNotMatch(narrow, /--line-control:\s*#8f8c8b/);
});

test('E30 exhibition brand has full and short names', () => {
  assert.match(nav, /className="exhibition-nav__brand-full"/);
  assert.match(nav, /className="exhibition-nav__brand-short" aria-hidden="true"/);
  assert.match(narrow, /exhibition-nav__brand-full[^}]*display:\s*none|exhibition-nav__brand-full\s*{[^}]*display:\s*none/);
  assert.match(narrow, /exhibition-nav__brand-short[^}]*display:\s*(?:block|flex|inline)|exhibition-nav__brand-short\s*{[^}]*display:\s*(?:block|flex|inline)/);
  assert.match(narrow, /exhibition-nav__brand[^{]*{[^}]*font-size:\s*1\.15rem/);
});

test('E22-E24 logout sits at the bottom of the 52px rail', () => {
  assert.match(nav, /railIcons[\s\S]*logout:/);
  assert.match(nav, /<RailIcon name="logout"/);
  assert.match(narrow, /:is\(nav, \.exhibition-nav__settings\) :is\(a, button, select\)\s*{[^}]*width:\s*52px/);
  assert.match(narrow, /:is\(nav, \.exhibition-nav__settings\) :is\(a, button, select\)\s*{[^}]*min-height:\s*44px/);
  assert.match(narrow, /\.exhibition-nav__settings\s*{[^}]*position:\s*sticky/);
  assert.match(narrow, /\.exhibition-nav__settings\s*{[^}]*bottom:\s*0/);
});

test('E22-E24 no 390 top bar logout; rail stays vertical at 390', () => {
  assert.doesNotMatch(css, /--expo-rail:\s*0px/);
  assert.doesNotMatch(css, /@media \(max-width: 390px\)/);
});
