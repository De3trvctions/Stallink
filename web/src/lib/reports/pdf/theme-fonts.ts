import type { StorefrontTheme } from "../../storefront-theme";

// The only place that maps a storefront theme to its PDF font. Files live in
// public/fonts/report/ and are built by scripts/subset-report-fonts.py; after
// regenerating, copy bytes/sha256 from coverage.json here (tests compare them).

export type ReportFontKey = "serif" | "sans";

export const REPORT_THEME_FONTS: Readonly<Record<StorefrontTheme, ReportFontKey>> = {
  classic: "serif",
  exhibition: "serif",
  shop: "sans",
};

const FACES = {
  serif: {
    family: "Noto Serif SC",
    file: "noto-serif-sc-regular.subset.ttf",
    bytes: 3709948,
    sha256: "8c8433de4694e3561701716ca59b09060db7829b909b3d725f852b55c96f3148",
  },
  sans: {
    family: "Noto Sans SC",
    file: "noto-sans-sc-regular.subset.ttf",
    bytes: 2688676,
    sha256: "e18bfbb6b92ec36b06a93c6bdb8aea488cceabfa0c3c5f5828407f41e5717a4a",
  },
} as const;

export type ReportFontFace = {
  key: ReportFontKey;
  family: string;
  file: string;
  /** Versioned so a regenerated subset never comes from a stale cache. */
  url: string;
  bytes: number;
  sha256: string;
};

/** Code points both subsets cover; versioned by the file's own SHA-256. */
export const REPORT_COVERAGE_URL = "/fonts/report/coverage.json?v=4f43d89038dc";

function themeKey(theme: string): ReportFontKey {
  return Object.hasOwn(REPORT_THEME_FONTS, theme) ? REPORT_THEME_FONTS[theme as StorefrontTheme] : "serif";
}

export function reportFontFor(theme: string): ReportFontFace {
  const key = themeKey(theme);
  const face = FACES[key];
  return { key, ...face, url: `/fonts/report/${face.file}?v=${face.sha256.slice(0, 12)}` };
}

export type ReportColors = { ink: string; muted: string; line: string; accent: string };

// Matches each theme's on-screen workspace palette (globals.css,
// exhibition-workspace.css, shop.css); kept here so the PDF needs no DOM.
const COLORS: Record<StorefrontTheme, ReportColors> = {
  classic: { ink: "#16171a", muted: "#6b685f", line: "#dad3c3", accent: "#1e36a8" },
  exhibition: { ink: "#142e32", muted: "#526c6d", line: "#d0dfda", accent: "#215e58" },
  shop: { ink: "#000000", muted: "#787574", line: "#ebebeb", accent: "#5433eb" },
};

export function reportColorsFor(theme: string): ReportColors {
  return Object.hasOwn(COLORS, theme) ? COLORS[theme as StorefrontTheme] : COLORS.classic;
}
