export type OrderAction = "fulfill" | "cancel" | "refund";

// 按钮顺序即展示顺序：「标记交付」在前，已收款、已交付也能取消（进入待退款）。
const actionsByStatus: Record<string, OrderAction[]> = {
  pending: ["cancel"],
  paid: ["fulfill", "cancel"],
  fulfilled: ["cancel"],
  refund_required: ["refund"],
};

export function orderActionsFor(status: string): OrderAction[] {
  return actionsByStatus[status] ?? [];
}

export function actionNeedsReason(action: OrderAction): boolean {
  return action === "cancel" || action === "refund";
}

// 前端校验：先 trim，再按 Unicode 码点计数（2–500）。后端目前不接收原因字段，见 orderActionRequestBody。
export const REASON_MIN_LENGTH = 2;
export const REASON_MAX_LENGTH = 500;

export type ReasonResult = { ok: true; reason: string } | { ok: false; error: string };

export function validateActionReason(input: string): ReasonResult {
  const reason = input.trim();
  const length = Array.from(reason).length;
  if (!reason) return { ok: false, error: "请填写原因。" };
  if (length < REASON_MIN_LENGTH) return { ok: false, error: `原因至少 ${REASON_MIN_LENGTH} 个字。` };
  if (length > REASON_MAX_LENGTH) return { ok: false, error: `原因不能超过 ${REASON_MAX_LENGTH} 个字。` };
  return { ok: true, reason };
}

// 取消 / 确认退款 / 交付接口目前不读取请求体，原因无处落库。
// 在后端提供原因字段之前不发送任何臆造字段，保持空对象。
export function orderActionRequestBody(action: OrderAction): Record<string, never> {
  void action;
  return {};
}

export type PayCodeChoice = { id: string; kind: string };

// 收款只提交支付方式；后端不再接收或记录实收与找零（K-Q10、K-Q17）。
export function payCommandBody(code: PayCodeChoice): { payCodeId: string } {
  return { payCodeId: code.id };
}
