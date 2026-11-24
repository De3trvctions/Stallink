import Link from "next/link";

export default function AboutPage() {
  return (
    <main className="ed-frame ed-about">
      <div className="ed-crumb">
        <span className="ed-mono">关于</span>
        <span className="ed-mono ed-sep" aria-hidden="true">/</span>
        <span className="ed-mono" style={{ color: "var(--ink)" }}>Stallink</span>
      </div>

      <section className="ed-hero ed-hero--compact classic-intro">
        <div>
          <div className="ed-kicker">
            <span className="ed-dash" aria-hidden="true" />
            <span className="ed-mono">为漫展与市集摊位而生</span>
          </div>
          <h1 className="ed-title">一家摊，<br />一个链接。</h1>
          <p className="ed-desc">Stallink 帮独立创作者把这一场活动的作品、库存和摊位号装进一条链接：展前线上先逛，开展现场再见，收摊即可复盘。</p>
        </div>
      </section>

      <section className="exhibition-only exhibition-about-intro">
        <div>
          <span className="exhibition-kicker" lang="en">About Stallink</span>
          <h1>在展场，<br />遇见创作者。</h1>
          <p>展前浏览独立创作者的作品与价格，开展后按活动与摊位号找到他们。</p>
          <Link className="exhibition-action" href="/#catalog">逛展场作品 <span aria-hidden="true">↗</span></Link>
        </div>
      </section>

      <section className="exhibition-only exhibition-visit">
        <h2>从作品目录，到摊位现场</h2>
        <ol role="list">
          <li><h3>先逛作品</h3><p>在作品目录搜索作品、摊位或活动，查看价格和余量。</p></li>
          <li><h3>找到摊位</h3><p>打开作品的活动页，查看日期、场馆和摊位号。</p></li>
          <li><h3>现场购买</h3><p>到摊位选购，由工作人员开单并当场取货。</p></li>
        </ol>
      </section>

      <section className="ed-band classic-intro" aria-label="我们解决什么">
        <div className="ed-frame ed-band__in">
          <div>
            <h2 className="ed-h2">从布货到收银，<br />一条链路打通。</h2>
            <p className="ed-band__note">不再靠群里接龙、手抄订单和临时算账。商品、报价、收款与库存扣减都在同一个工作台里完成。</p>
          </div>
          <div className="ed-facts">
            <div><b>链接</b><span>每家摊位独立入口</span></div>
            <div><b>库存</b><span>共享池实时余量</span></div>
            <div><b>开单</b><span>现场收款即扣减</span></div>
            <div><b>套餐</b><span>组合优惠自动算价</span></div>
          </div>
        </div>
      </section>

      <section className="ed-steps classic-intro" aria-label="使用流程">
        <h2>从线上到收银</h2>
        <div className="ed-step-list">
          <div className="ed-step">
            <span className="ed-step__n">01</span>
            <div>
              <strong>展前，上传这一场的货</strong>
              <p>按活动建库存、上传图片与阶梯价，只展示这一场真正带来的作品。</p>
            </div>
          </div>
          <div className="ed-step">
            <span className="ed-step__n">02</span>
            <div>
              <strong>开展，粉丝按摊位号找到你</strong>
              <p>活动现场扫码进店，先看图和价、查余量，再走到摊位。</p>
            </div>
          </div>
          <div className="ed-step">
            <span className="ed-step__n">03</span>
            <div>
              <strong>现场，收银台开单交付</strong>
              <p>记录订单与收款，库存同步扣减；订单、预占与流水可追溯，收摊即可复盘。</p>
            </div>
          </div>
        </div>
      </section>

      <section className="ed-shelf classic-intro" aria-label="面向谁">
        <div className="ed-shelf__head">
          <h2>为两类人准备</h2>
        </div>
        <div className="ed-about__cols">
          <article className="ed-about__card">
            <h3>给摊主 / 创作者</h3>
            <p>一个工作台管理多场活动的商品、套餐与收款；共享库存避免超卖，订单状态与预占余量一目了然。</p>
          </article>
          <article className="ed-about__card">
            <h3>给逛展顾客</h3>
            <p>画册式摊位页，按活动看商品与价格，到现场扫码即可核对商品和库存。</p>
          </article>
        </div>
      </section>

      <section className="ed-about__cta classic-intro">
        <h2>准备好摆出你的摊位了吗？</h2>
        <div className="ed-actions">
          <Link href="/register" className="ed-btn">注册顾客账号<svg viewBox="0 0 16 16" aria-hidden="true"><path d="M3 13L13 3M6 3h7v7" /></svg></Link>
          <Link href="/login" className="ed-link">已有账号，登录</Link>
        </div>
      </section>

      <footer className="ed-frame ed-foot classic-intro">
        <div className="ed-foot__mark">STALLINK</div>
        <div className="ed-foot__meta">
          <div><span>顾客端</span><span>画册式摊位页</span></div>
          <div><span>店员端</span><span>票据式工作台</span></div>
          <div><span>入口</span><span>每家摊一条独立链接</span></div>
        </div>
      </footer>
    </main>
  );
}
