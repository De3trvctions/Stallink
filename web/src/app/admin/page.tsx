import Link from "next/link";

const sections = [
  { href: "/admin/events", index: "01", title: "活动与场次", description: "设置时间、场馆与摊位号，决定这一场卖什么。", label: "开摊准备" },
  { href: "/admin/goods", index: "02", title: "商品与套餐", description: "维护说明、价格、活动库存与组合优惠。", label: "目录与库存" },
  { href: "/admin/orders", index: "03", title: "订单记录", description: "核对现场订单、商品明细与收款状态。", label: "现场流水" },
  { href: "/admin/pay-codes", index: "04", title: "收款方式", description: "上传收款码，设定现场默认使用的那一个。", label: "付款配置" },
  { href: "/admin/sales", index: "05", title: "销售账号", description: "给现场同事开账号，限定他们能做的事。", label: "团队与权限" },
  { href: "/admin/shop", index: "06", title: "店铺资料", description: "店名、公告与对外分享链接，顾客看到的就是这些。", label: "公开页面" },
];

export default function AdminHome() {
  return (
    <main className="workspace-overview">
      <header className="workspace-overview__heading">
        <p className="eyebrow">店铺工作台</p>
        <h1>开摊之前，把清单过一遍。</h1>
        <p>活动定场地，商品定货架，收款码定结账方式。三项确认完，现场就能直接开单。</p>
      </header>
      <div className="overview-actions">
        <span><i aria-hidden="true" />本店工作台已连接</span>
        <Link href="/counter" className="button button--dark">打开现场收银 <span aria-hidden="true">→</span></Link>
      </div>
      <section className="workspace-card-grid" aria-label="店铺管理功能">
        {sections.map((section) => (
          <Link href={section.href} key={section.href} className="workspace-card">
            <span className="workspace-card__index">{section.index}</span>
            <h2>{section.title}</h2>
            <span className="workspace-card__arrow" aria-hidden="true">→</span>
            <p>{section.description}</p>
          </Link>
        ))}
      </section>
      <aside className="workspace-tip">
        <span className="workspace-tip__mark" aria-hidden="true">※</span>
        <div><strong>开摊前检查</strong><p>活动时间与摊位号对得上、商品可售数量够卖、收款码能正常打开。</p></div>
        <Link href="/admin/events">先看活动 <span aria-hidden="true">→</span></Link>
      </aside>
    </main>
  );
}
