const statusText: Record<string, string> = {
  pending: "待付款",
  paid: "已收款",
  fulfilled: "已交付",
  cancelled: "已取消",
  expired: "预留已过期",
  refund_required: "待退款处理",
};

export function orderStatusLabel(status: string): string {
  return statusText[status] ?? "未知状态，请联系管理员";
}

const actionText: Record<string, string> = {
  "bundle.approve": "审核通过套餐",
  "bundle.create": "创建套餐",
  "bundle.update": "编辑套餐",
  "bundle.reject": "拒绝套餐",
  "good.approve": "审核通过商品",
  "good.reject": "拒绝商品",
  "good.image.add": "添加商品图片",
  "good.image.delete": "删除商品图片",
  "good.image.reorder": "调整商品图片顺序",
  "listing.create": "上架商品",
  "listing.update": "修改商品上架信息",
  "stock.update": "调整商品库存",
  "order.create": "创建订单",
  "order.pay": "确认收款",
  "order.cancel": "取消订单",
  "order.fulfill": "完成交付",
  "order.expire": "订单预留过期",
  "order.refund_required": "收款后取消待退款",
  "order.refund_confirm": "确认退款",
};

export function orderActionLabel(action: string): string {
  return actionText[action] ?? "未知操作，请联系管理员";
}
