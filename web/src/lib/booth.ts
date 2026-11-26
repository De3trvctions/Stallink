export type CustomerBoothEvent = {
  displayBoothNumber?: string;
  boothNumber?: string;
};

/** 顾客端摊位号：只用后端算好的 displayBoothNumber，旧接口没有该字段时回退 boothNumber。 */
export function customerBoothLabel(event: CustomerBoothEvent): string {
  return event.displayBoothNumber || event.boothNumber || "待公布";
}
