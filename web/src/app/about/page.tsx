import Link from "next/link";
import Image from "next/image";

const shopSteps = [
  { when: "展前", title: "整理这场要卖的作品", copy: "摊主按活动上架商品和套餐，设置价格，填好场馆与摊位号。" },
  { when: "开展", title: "看好作品，再去摊位", copy: "顾客从目录或摊位链接查看价格和余量，按活动地点与摊位号找到创作者。" },
  { when: "现场", title: "开单、收款和交付", copy: "工作人员在收银台记录订单，库存随交易更新，收摊后可查订单和销售日报。" },
];

function ShopAbout() {
  return (
    <div className="shop-only shop-about">
      <header className="shop-about__hero">
        <div className="shop-about__copy">
          <p className="shop-about__meta">关于 Stallink</p>
          <h1>一家摊，一个链接。</h1>
          <p>Stallink 是给漫展和市集摊位用的店铺和收银工具。顾客可以先看作品、价格和余量，摊主可以用同一个工作台管理活动、库存与订单。</p>
          <Link className="button button--dark" href="/#catalog">浏览商品 <span aria-hidden="true">→</span></Link>
        </div>
        <Image className="shop-about__photo" src="/images/artist-alley-booth.webp" alt="创作者在市集摊位陈列印刷作品" width={1440} height={960} sizes="(max-width: 760px) 100vw, 50vw" />
      </header>

      <section aria-labelledby="shop-about-steps">
        <h2 id="shop-about-steps">从展前上架到现场交付</h2>
        <ol className="shop-about__steps" role="list">
          {shopSteps.map(({ when, title, copy }) => (
            <li key={when} className="shop-about__card">
              <p className="shop-about__meta">{when}</p>
              <h3>{title}</h3>
              <p>{copy}</p>
            </li>
          ))}
        </ol>
      </section>

      <section aria-labelledby="shop-about-entry">
        <h2 id="shop-about-entry">顾客与摊主入口</h2>
        <div className="shop-about__entries">
          <article className="shop-about__card">
            <h3>逛展顾客</h3>
            <p>打开摊位链接即可浏览商品。创建顾客账号后，可以保存你的称呼和联系方式。</p>
            <Link className="button button--outline" href="/register">注册顾客账号 <span aria-hidden="true">→</span></Link>
          </article>
          <article className="shop-about__card">
            <h3>摊主与工作人员</h3>
            <p>使用你的工作账号登录，进入店铺管理或现场收银。</p>
            <Link className="button button--outline" href="/login">登录工作台 <span aria-hidden="true">→</span></Link>
          </article>
        </div>
      </section>
    </div>
  );
}

export default function AboutPage() {
  return (
    <main className="ed-frame ed-about">
      <ShopAbout />
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
