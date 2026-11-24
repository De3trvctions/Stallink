export const MAX_REPORT_DAYS = 31;

// Native date inputs expose UTC midnight through valueAsNumber. Stepping in
// UTC keeps each date on the same calendar day across daylight-saving changes.
const DAY_MS = 86_400_000;

export function inclusiveReportDates(start: number, end: number):
  { dates: string[]; error?: never } | { dates?: never; error: string } {
  const count = (end - start) / DAY_MS + 1;
  if (!Number.isInteger(count)) return { error: "请选择有效的起始日期和结束日期。" };
  if (count < 1) return { error: "结束日期须在起始日期当天或之后。" };
  if (count > MAX_REPORT_DAYS) return { error: `每次最多查询 ${MAX_REPORT_DAYS} 天，请缩短日期区间。` };
  const dates = Array.from({ length: count }, (_, index) =>
    new Date(end - index * DAY_MS).toISOString().slice(0, 10));
  return { dates };
}
