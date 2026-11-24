export const appearancePath = "/api/platform/appearance";

export type StorefrontTheme = "classic" | "exhibition";

export function isStorefrontTheme(value: unknown): value is StorefrontTheme {
  return value === "classic" || value === "exhibition";
}
