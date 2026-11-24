export const appearancePath = "/api/platform/appearance";

export type StorefrontTheme = "classic" | "exhibition" | "shop";

export function isStorefrontTheme(value: unknown): value is StorefrontTheme {
  return value === "classic" || value === "exhibition" || value === "shop";
}
