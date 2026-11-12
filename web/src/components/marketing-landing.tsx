import Link from "next/link";
import Image from "next/image";

export function MarketingLanding() {
  return (
    <main>
      <section className="ed-frame ed-hero">
        <div>
          <div className="ed-kicker">
            <span className="ed-dash" aria-hidden="true" />
            <span className="ed-mono">独立摊位 / 一场一会</span>
          </div>
          <h1 className="ed-title">你的作品，<br />值得被看见。</h1>
          <p className="ed-desc">一个链接装下这一摊的全部：作品、库存、摊位号。展前线上先逛，开展现场再见。</p>
          <div className="ed-actions">
            <Link href="/login" className="ed-btn">
              登录 Stallink
              <svg viewBox="0 0 16 16" aria-hidden="true"><path d="M3 13L13 3M6 3h7v7" /></svg>
            </Link>
            <Link href="/register" className="ed-link">注册顾客账号</Link>
          </div>
          <div className="ed-meta">
            <span className="ed-mono">东南亚漫展 / 摊位独立链接</span>
            <span className="ed-mono">轻量 / 开箱即用</span>
          </div>
        </div>
        <figure className="ed-figure">
          <div className="ed-figure__frame">
            <Image
              src="/images/creator-market-hero.png"
              alt="摊位上陈列的插画印刷品、贴纸与手作周边"
              fill
              sizes="(max-width: 960px) 100vw, 45vw"
              priority
            />
          </div>
          <figcaption className="ed-placard">
            <strong>摊位氛围 / 布面印刷与手作周边</strong>
            <span>2026</span>
          </figcaption>
        </figure>
      </section>

      <section className="ed-band" aria-label="Stallink 功能">
        <div className="ed-frame ed-band__in">
          <div>
            <h2 className="ed-h2">一家摊，<br />一个链接。</h2>
            <p className="ed-band__note">每家摊位有自己的地址，不混进别人的货架。展前把这一场带什么放上来，粉丝按摊位号就能找到你。</p>
          </div>
          <div className="ed-facts">
            <div><b>链接</b><span>独立摊位入口</span></div>
            <div><b>库存</b><span>按活动隔离余量</span></div>
            <div><b>开单</b><span>现场收款即扣减</span></div>
          </div>
        </div>
      </section>

      <section className="ed-frame ed-steps" aria-label="从线上到收银">
        <h2>从线上到收银</h2>
        <div className="ed-step-list">
          <div className="ed-step">
            <span className="ed-step__n">01</span>
            <div>
              <strong>展前，上传这一场的货</strong>
              <p>按活动建库存，只展示这一场真正带来的作品。</p>
            </div>
          </div>
          <div className="ed-step">
            <span className="ed-step__n">02</span>
            <div>
              <strong>开展，粉丝按摊位号找到你</strong>
              <p>活动现场扫码进店，先看图和价，再走到摊位。</p>
            </div>
          </div>
          <div className="ed-step">
            <span className="ed-step__n">03</span>
            <div>
              <strong>现场，收银台开单交付</strong>
              <p>记录订单与收款，库存同步扣减，收摊即可复盘。</p>
            </div>
          </div>
        </div>
      </section>

      <footer className="ed-frame ed-foot">
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
