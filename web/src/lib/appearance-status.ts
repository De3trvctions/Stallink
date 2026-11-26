export type AppearancePublisher = { id?: string; name?: string };
export type AppearanceMeta = {
  updatedAt?: string | null;
  updatedBy?: AppearancePublisher | null;
};

export function formatAppearanceTime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  const parts = new Intl.DateTimeFormat("en-US", {
    timeZone: "Asia/Kuala_Lumpur",
    month: "numeric",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    hourCycle: "h23",
  }).formatToParts(date);
  const get = (type: string) => parts.find((part) => part.type === type)?.value ?? "";
  return `${get("month")}/${get("day")} ${get("hour")}:${get("minute")}`;
}

export function appearancePublishLabel(meta: AppearanceMeta | null | undefined, opts?: { justPublished?: boolean }): string {
  if (opts?.justPublished) return "你 · 刚刚";
  if (!meta?.updatedAt) return "尚无记录";
  const name = String(meta.updatedBy?.name ?? "").trim();
  const time = formatAppearanceTime(meta.updatedAt);
  return name ? `上次发布：${name} · ${time}` : `上次发布：${time}`;
}
