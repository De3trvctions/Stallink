import { ALL_CURRENCIES_LABEL } from "./document-model";

const DATE = /^\d{4}-\d{2}-\d{2}$/;
const CURRENCY = /^[A-Z]{3}$/;

// 销售日报-2026-11-13-MYR.pdf / 销售日报-2026-11-13-全部币种.pdf; a range uses
// both ends: 销售日报-2026-11-01至2026-11-07-MYR.pdf.
export function reportPdfFilename(dates: string[], currencyLabel: string): string {
  if (dates.length === 0 || dates.some((date) => !DATE.test(date))) throw new Error("日报日期无效，无法生成文件名。");
  const sorted = [...dates].sort();
  const first = sorted[0];
  const last = sorted[sorted.length - 1];
  const span = first === last ? first : `${first}至${last}`;
  const label = CURRENCY.test(currencyLabel) ? currencyLabel : ALL_CURRENCIES_LABEL;
  return `销售日报-${span}-${label}.pdf`;
}
