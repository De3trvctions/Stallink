const SUBJECT_LABELS: Record<string, string> = {
  order: "订单",
  good: "商品",
  listing: "上架",
  bundle: "套餐",
  event: "活动",
  shop: "店铺",
  pay_code: "收款方式",
  account: "账号",
};

export type AuditSubject = {
  subjectType?: string;
  subjectId?: string;
  subjectLabel?: string;
};

/** K-Q5 = C：优先直接显示后端返回的名称；没有名称时用「类型 #id」。 */
export function auditSubjectLabel(entry: AuditSubject): string {
  const type = String(entry.subjectType ?? "").trim();
  const id = String(entry.subjectId ?? "").trim();
  const label = String(entry.subjectLabel ?? "").trim();
  const kind = SUBJECT_LABELS[type] ?? type;
  if (kind && label) return `${kind} · ${label}`;
  if (kind && id) return `${kind} #${id}`;
  if (label) return label;
  if (kind) return kind;
  if (id) return `#${id}`;
  return "—";
}

export function formatAuditTime(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return new Intl.DateTimeFormat("zh-CN", { timeZone: "Asia/Kuala_Lumpur" }).format(date);
}
