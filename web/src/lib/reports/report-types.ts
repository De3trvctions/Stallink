// Shapes of /api/admin/daily-reports snapshots, shared by the page and the PDF export.

export type CurrencyRow = {
  currency: string; orders: number; paidOrders: number; revenueMinor: number;
  discountMinor: number; refundMinor: number; cashOverShortMinor: number; cashTenderedOrders?: number;
};
export type EventRow = { eventId: string; name: string; currency: string; orders: number; revenueMinor: number; discountMinor: number };
export type GoodRow = { currency: string; name: string; quantity: number; revenueMinor: number };
export type PayCodeRow = { currency: string; name: string; orders: number; revenueMinor: number };
export type Report = {
  date: string; generatedAt: string; source: string; currency: string;
  orders: number; paidOrders: number; pendingOrders: number; closedOrders: number; refundOrders: number;
  itemsSold: number; revenueMinor: number; discountMinor: number; refundMinor: number; cashOverShortMinor: number; cashTenderedOrders?: number;
  averageMinor: number; currencies: CurrencyRow[]; events: EventRow[]; topGoods: GoodRow[]; payCodes: PayCodeRow[];
};
