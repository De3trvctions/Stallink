// Shop owners type product names freely, so the PDF font subset can miss
// characters (rare hanzi, kana, emoji). Each missing character becomes □ and
// the PDF footer explains why.

export const MISSING_GLYPH = "□";

export type CoverageRanges = ReadonlyArray<readonly [number, number]>;
export type GlyphCoverage = (codepoint: number) => boolean;

export function createGlyphCoverage(ranges: CoverageRanges): GlyphCoverage {
  const sorted = [...ranges].sort((a, b) => a[0] - b[0]);
  return (codepoint) => {
    let low = 0;
    let high = sorted.length - 1;
    while (low <= high) {
      const middle = (low + high) >> 1;
      const [start, end] = sorted[middle];
      if (codepoint < start) high = middle - 1;
      else if (codepoint > end) low = middle + 1;
      else return true;
    }
    return false;
  };
}

// Joiners, variation selectors and skin-tone modifiers only change how the
// neighbouring glyph looks; dropping them leaves one □ per visible symbol.
function isInvisibleModifier(codepoint: number): boolean {
  return codepoint === 0x200c || codepoint === 0x200d
    || (codepoint >= 0xfe00 && codepoint <= 0xfe0f)
    || (codepoint >= 0xe0100 && codepoint <= 0xe01ef)
    || (codepoint >= 0x1f3fb && codepoint <= 0x1f3ff);
}

export function replaceMissingGlyphs(text: string, covered: GlyphCoverage): { text: string; replaced: number } {
  let out = "";
  let replaced = 0;
  for (const character of text) {
    const codepoint = character.codePointAt(0) ?? 0;
    if (character === "\n" || character === "\t" || covered(codepoint)) {
      out += character;
    } else if (!isInvisibleModifier(codepoint)) {
      out += MISSING_GLYPH;
      replaced++;
    }
  }
  return { text: out, replaced };
}

export function applyGlyphCoverage<T>(value: T, covered: GlyphCoverage): { value: T; replaced: number } {
  let replaced = 0;
  const walk = (node: unknown): unknown => {
    if (typeof node === "string") {
      const result = replaceMissingGlyphs(node, covered);
      replaced += result.replaced;
      return result.text;
    }
    if (Array.isArray(node)) return node.map(walk);
    if (node && typeof node === "object") {
      return Object.fromEntries(Object.entries(node).map(([key, child]) => [key, walk(child)]));
    }
    return node;
  };
  return { value: walk(value) as T, replaced };
}
