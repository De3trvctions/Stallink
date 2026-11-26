import type { RequestResult } from "@/lib/api";

// Field-level / business error codes mapped to plain Chinese. Unknown codes
// fall back to a generic message; the raw English code is never shown to staff.
const errorCodeText: Record<string, string> = {
  // 入参校验
  invalid_login_name: "登录名需 3–32 个字符，且不能包含空格。",
  invalid_pay_code_name: "收款方式名称需 1–40 个字符。",
  invalid_event: "活动名称、日期或场次信息有误，请检查后重试。",
  event_required: "请选择或新建活动。",
  event_currency_required: "请先在店铺资料里设置默认货币，或为活动选择货币。",
  invalid_profile: "称呼或联系方式格式有误，请检查后重试。",
  invalid_shop: "店铺资料填写有误，请检查后重试。",
  slug_readonly: "店铺地址由平台分配，不能修改",
  invalid_pay_code_currency: "收款码币种需为 MYR、SGD 或 CNY。",
  invalid_currency: "所选币种不受支持，请重新选择。",
  invalid_date: "所选日期无效，请重新选择。",
  invalid_json: "提交内容格式有误，请刷新后重试。",
  // 收款与库存
  not_listed: "商品已下架或未通过审核，不能继续收款。请取消这张订单后重新开单。",
  pay_code_currency_mismatch: "该收款码币种与订单币种不一致，请更换收款方式。",
  pay_code_required: "这家店还没有可用的收款方式，请先在后台配置。",
  event_not_open: "活动尚未开始或已结束，暂时不能开单。",
  not_enough_stock: "本场剩余库存不足，请核对所选数量。",
  reservation_expired: "库存预留已过期，未确认收款，请重新下单。",
  quote_expired: "价格预览已失效，正在重新计算。",
  quote_stale: "价格预览已失效，正在重新计算。",
  quote_invalid: "价格预览已失效，正在重新计算。",
  invalid_cart: "套餐与单品共用库存，请核对所选数量。",
  pricing_complexity_exceeded: "这组商品优惠规则过于复杂，请减少品项后重试。",
  invalid_price_tiers: "优惠包必须比对应数量的单买总价更低。",
  stock_exceeded: "各场可售库存不能超过商品总库存。",
  // 订单状态与并发
  invalid_status: "该订单已不处于待退款状态。",
  command_payload_unresolved: "上一笔操作仍待服务端确认，请重试上一笔操作后再修改购物车。",
  command_in_progress: "上一笔操作仍在处理中，请稍候再试。",
  command_storage_unavailable: "本地存储不可用，操作未能保存，请刷新页面后重试。",
  idempotency_conflict: "重复的操作请求，请刷新后确认结果。",
  conflict: "内容已被其他操作更新，请刷新后重试。",
  checkout_conflict: "订单状态已变化，请刷新后重试。",
  // 账号与审核
  login_taken: "这个账号已经存在，请换一个登录名。",
  invalid_review: "拒绝商品需要填写不超过 500 字的原因。",
  bundle_components_not_eligible: "套餐中的商品需要先通过审核并保持上架。",
  // 图片
  image_limit: "图片数量或大小超出限制：每张最多 6 MiB，每件商品最多 8 张。",
  pay_code_image_limit: "收款码图片无损压缩后仍超过 4 MiB 或尺寸过大，请换一张图片。",
  invalid_image: "图片格式或大小不符合要求，请选择 JPG、PNG 或 WebP 图片。",
  image_storage_failed: "图片存储暂时不可用，请稍后重试。",
  // 报表
  report_not_generated: "这一天还没有生成日报。",
};

export function errorCodeLabel(code: string): string {
  return errorCodeText[code] ?? "";
}

// Turn a classified request result into a staff-facing Chinese message.
export function describeRequest(result: Pick<RequestResult, "category" | "status" | "errorCode">): string {
  switch (result.category) {
    case "timeout":
      return "网络响应超时，请检查网络后重试。";
    case "network":
      return "无法连接服务，请检查网络后重试。";
    case "unauthorized":
      return "登录已过期，请重新登录。";
    case "forbidden":
      return "没有权限执行这个操作。";
    case "server":
      return "服务暂时不可用，请稍后重试。";
    case "client": {
      return errorCodeText[result.errorCode] ?? "操作没有成功，请检查填写内容后重试。";
    }
    default:
      return "操作没有成功，请稍后重试。";
  }
}
