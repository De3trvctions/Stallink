const eventInfo = {
  name: "Comic Fiesta",
  venue: "吉隆坡会展中心 Hall 4",
  dates: "12月13–14日",
  currencyLabel: "马币",
  status: "进行中",
};

const goods = {
  zhantai: { id: "zhantai", name: "新刊《站台》", kind: "新刊", price: 3500, tier: "2 件一组 RM 60.00", stock: 40, held: 2, tone: "cobalt", cover: "站台", note: "A5 胶装 64P。只在本场出售。" },
  sezhi: { id: "sezhi", name: "色纸「雨停之前」", kind: "周边", price: 2500, tier: "", stock: 30, held: 0, tone: "paper", cover: "雨停", note: "手绘色纸，无装裱。" },
  lipai: { id: "lipai", name: "亚克力立牌 15cm", kind: "周边", price: 4500, tier: "", stock: 20, held: 1, tone: "ink", cover: "立牌", note: "单面印刷。" },
  sticker: { id: "sticker", name: "和纸贴纸包 8 枚", kind: "周边", price: 1200, tier: "", stock: 80, held: 0, tone: "moss", cover: "贴纸", note: "和纸，八枚一组。" },
  card: { id: "card", name: "明信片套装 5 张", kind: "周边", price: 1800, tier: "", stock: 50, held: 0, tone: "warn", cover: "明信片", note: "不含套餐。" },
  sketch: { id: "sketch", name: "线稿本", kind: "新刊", price: 2800, tier: "", stock: 24, held: 0, tone: "ink", cover: "线稿", note: "空白线稿，可现场署名。" },
  acrylic: { id: "acrylic", name: "亚克力挂件", kind: "周边", price: 4200, tier: "", stock: 16, held: 0, tone: "cobalt", cover: "挂件", note: "南风社本场限定。" },
  artbook: { id: "artbook", name: "夜航画集", kind: "新刊", price: 5500, tier: "", stock: 0, held: 0, tone: "ink", cover: "夜航", note: "这一批已经卖完。" },
  plane: { id: "plane", name: "纸飞机贴纸", kind: "周边", price: 1000, tier: "", stock: 60, held: 4, tone: "paper", cover: "纸飞机", note: "一小袋十枚。" },
};

const bundle = {
  id: "rainset",
  name: "雨季三件套",
  price: 6200,
  parts: ["zhantai", "sezhi", "sticker"],
  single: 7200,
  tone: "cobalt",
  cover: "三件套",
  note: "新刊、色纸、贴纸各一件。库存跟这些单品走。",
};

const booths = [
  { id: "yupeng", no: "D-18", name: "雨棚社", blurb: "新刊和雨季三件套。预留 15 分钟，柜台扫码或现金，不邮寄。", goods: ["zhantai", "sezhi", "lipai", "sticker", "card"], bundle: true },
  { id: "salt", no: "A-02", name: "盐与墨", blurb: "线稿本可以现场署名。", goods: ["sketch"], bundle: false },
  { id: "south", no: "B-11", name: "南风社", blurb: "亚克力挂件，本场现货。", goods: ["acrylic"], bundle: false },
  { id: "night", no: "C-06", name: "夜航线", blurb: "画集已售罄，摊位仍在。", goods: ["artbook"], bundle: false },
  { id: "plane", no: "E-21", name: "纸飞机", blurb: "贴纸按袋卖。", goods: ["plane"], bundle: false },
];

const views = ["directory", "booth", "product", "wishlist", "lookup", "counter", "pay", "refund", "report", "review"];

const state = {
  view: "directory",
  id: null,
  q: "",
  filter: "all",
  wishlist: load("stallink-wish", []),
  cart: [],
  orders: [
    { number: "K7M2QP", status: "pending", booth: "D-18", shop: "雨棚社", total: 3500, lines: ["新刊《站台》 ×1"], pay: "未付款", ref: "", by: "" },
    { number: "M3P7LD", status: "paid", booth: "D-18", shop: "雨棚社", total: 6200, lines: ["雨季三件套 ×1"], pay: "Touch 'n Go", ref: "TNG-88421", by: "阿林" },
    { number: "Q8N4WC", status: "fulfilled", booth: "D-18", shop: "雨棚社", total: 4500, lines: ["亚克力立牌 15cm ×1"], pay: "DuitNow", ref: "", by: "阿林" },
    { number: "R5T1HB", status: "refund_requested", booth: "A-02", shop: "盐与墨", total: 2800, lines: ["线稿本 ×1"], pay: "现金", ref: "", by: "阿林", reason: "封面压痕，顾客不要了", requestedBy: "阿林" },
    { number: "H2K8ND", status: "paid", booth: "D-18", shop: "雨棚社", total: 2500, lines: ["色纸「雨停之前」 ×1"], pay: "现金", ref: "", by: "阿林" },
  ],
  lookup: "",
  lookupTried: false,
  payMethod: "qr",
  payCode: "Touch 'n Go",
  cash: 0,
  ref: "",
  proof: false,
  receipt: null,
  role: "sales",
  refundReason: "",
  counted: "",
  reviewId: "pencil",
  reviews: [
    { id: "pencil", shop: "盐与墨", type: "单品", name: "铅笔稿", price: "RM 15.00", status: "pending", note: "单页铅笔稿，A5。" },
    { id: "pair", shop: "盐与墨", type: "套餐", name: "线稿两件套", price: "RM 40.00", status: "pending", note: "线稿本 + 铅笔稿。铅笔稿还没通过。", blocked: "组成里的铅笔稿还在待审核" },
  ],
  autoApprove: false,
};

function load(key, fallback) {
  try { return JSON.parse(sessionStorage.getItem(key)) || fallback; } catch { return fallback; }
}
function saveWish() { sessionStorage.setItem("stallink-wish", JSON.stringify(state.wishlist)); }
function esc(value) {
  return String(value).replace(/[&<>"']/g, (ch) => ({ "&": "&amp;", "<": "&lt;", ">": "&gt;", '"': "&quot;", "'": "&#39;" }[ch]));
}
function money(minor) {
  const sign = minor < 0 ? "−" : "";
  return `RM ${sign}${(Math.abs(minor) / 100).toFixed(2)}`;
}
function available(item) { return Math.max(0, item.stock - item.held); }
function stockText(item) {
  const left = available(item);
  if (left === 0 && item.held === 0) return "售罄";
  if (item.held > 0) return `余 ${left}，其中 ${item.held} 件待付款`;
  return `余 ${left}`;
}
function boothOfGood(id) {
  return booths.find((booth) => booth.goods.includes(id) || (id === bundle.id && booth.bundle));
}
function bundleLeft() {
  return Math.min(...bundle.parts.map((id) => available(goods[id])));
}

function parseHash() {
  const raw = (location.hash || "#directory").slice(1);
  const [view, id] = raw.split("/");
  state.view = views.includes(view) ? view : "directory";
  state.id = id || null;
}

function go(view, id) {
  const next = id ? `#${view}/${id}` : `#${view}`;
  if (location.hash === next) render();
  else location.hash = next;
}

function header() {
  const staff = ["counter", "pay", "refund", "report", "review"].includes(state.view);
  const wish = state.wishlist.reduce((sum, item) => sum + item.qty, 0);
  const link = (view, label, extra = "") => `<a href="#${view}" ${state.view === view ? 'aria-current="page"' : ""}>${label}${extra}</a>`;
  if (staff) {
    return `<header class="top">
      <a class="brand" href="#directory"><span class="mark">S</span>Stallink</a>
      <p class="note">现场工作台 · ${esc(eventInfo.name)} · D-18 雨棚社</p>
      <nav>
        ${link("counter", "收银")}
        ${link("refund", "退款")}
        ${link("report", "日报")}
        ${link("review", "审核")}
        <a href="#directory">回到本场</a>
      </nav>
    </header>`;
  }
  return `<header class="top">
    <a class="brand" href="#directory"><span class="mark">S</span>Stallink</a>
    <p class="note">独立摊位 · 现场好找</p>
    <nav>
      ${link("directory", "本场")}
      ${link("wishlist", "想买", wish ? `<span class="count-pill">${wish}</span>` : "")}
      ${link("lookup", "订单号")}
      <a class="desk-link" href="#counter">现场工作台</a>
    </nav>
  </header>`;
}

function cover(item, size = "") {
  return `<span class="cover cover--${item.tone} ${size}">${esc(item.cover)}</span>`;
}

function filteredBooths() {
  const q = state.q.trim().toLowerCase();
  return booths.filter((booth) => {
    const names = booth.goods.map((id) => goods[id].name).join(" ");
    const hay = `${booth.no} ${booth.name} ${booth.blurb} ${names}`.toLowerCase();
    if (q && !hay.includes(q)) return false;
    if (state.filter === "book" && !booth.goods.some((id) => goods[id].kind === "新刊")) return false;
    if (state.filter === "bundle" && !booth.bundle) return false;
    if (state.filter === "stock" && booth.goods.every((id) => available(goods[id]) === 0 && goods[id].held === 0)) return false;
    return true;
  });
}

function boothCard(booth) {
  const thumbs = booth.goods.slice(0, 3).map((id) => cover(goods[id])).join("");
  return `<button class="booth-card" type="button" data-go="booth" data-id="${booth.id}">
      <span class="ticket"><span>BOOTH</span><strong>${esc(booth.no)}</strong></span>
      <span><h2>${esc(booth.name)}</h2><p>${esc(booth.blurb)}</p></span>
      <span class="thumbs">${thumbs}</span>
    </button>`;
}
function directory() {
  const list = filteredBooths();
  const chips = [
    ["all", "全部摊位"],
    ["book", "有新刊"],
    ["bundle", "有套餐"],
    ["stock", "只看有货"],
  ].map(([id, label]) => `<button class="chip" type="button" data-filter="${id}" aria-pressed="${state.filter === id}">${label}</button>`).join("");
  const cards = list.map(boothCard).join("");
  return `<section class="wrap">
    <p class="kicker">${esc(eventInfo.name)} · Kuala Lumpur</p>
    <h1 class="display">这场展，按摊位找。</h1>
    <p class="lede">先对上摊号，再看这一摊在卖什么。价格是单件标价，到柜台由店员开单。</p>
    <p class="meta-row"><span>${esc(eventInfo.venue)}</span><span>${esc(eventInfo.dates)}</span><span>${esc(eventInfo.currencyLabel)}</span><span>${booths.length} 个摊位 · ${esc(eventInfo.status)}</span></p>
    <div class="toolbar">
      <label class="search"><span>搜索</span><input id="q" value="${esc(state.q)}" placeholder="摊名、摊号或作品" autocomplete="off"></label>
      <div class="chips" role="group" aria-label="筛选">${chips}</div>
      <span class="result-count" aria-live="polite">${list.length} 个摊位</span>
    </div>
    <div class="booth-list">${cards || `<p class="empty">这场没有符合的摊位。</p>`}</div>
    <p class="foot">原型按 Comic Fiesta 这一场排摊。公开页不能自己付款。</p>
  </section>`;
}

function boothPage() {
  const booth = booths.find((item) => item.id === state.id) || booths[0];
  const cards = booth.goods.map((id) => goodCard(goods[id])).join("");
  const bundleCard = booth.bundle ? goodCard(bundle, true) : "";
  return `<section class="wrap">
    <p class="kicker"><a href="#directory">${esc(eventInfo.name)}</a> · ${esc(booth.no)}</p>
    <div class="hero-shop">
      <div>
        <h1 class="display">${esc(booth.name)}</h1>
        <p class="lede">${esc(booth.blurb)}</p>
      </div>
      <div class="pass"><div><span class="kicker">本场摊位</span><b>${esc(booth.no)}</b><span>${esc(eventInfo.venue)} · ${esc(eventInfo.dates)}</span></div><aside>ADMIT ONE</aside></div>
    </div>
    <ul class="rules">
      <li><span>取货</span>到这个摊号，由店员开单</li>
      <li><span>邮寄</span>不邮寄</li>
      <li><span>预留</span>店员生成待付款单后保留 15 分钟</li>
      <li><span>库存</span>全店共用一批货，各场一起卖</li>
    </ul>
    <div class="toolbar"><span class="tag">全部</span><span class="tag">新刊</span><span class="tag">周边</span><span class="tag">套餐</span></div>
    <div class="shelf">${bundleCard}${cards}</div>
  </section>`;
}

function goodCard(item, isBundle = false) {
  const left = isBundle ? bundleLeft() : available(item);
  const held = isBundle ? 0 : item.held;
  const stock = left === 0 && held === 0 ? "售罄" : `余 ${left}`;
  const tier = !isBundle && item.tier ? `<p class="subprice">${esc(item.tier)}</p>` : "";
  const save = isBundle ? `<p class="subprice">单买 ${money(item.single)}</p>` : tier;
  return `<button class="good-card" type="button" data-go="product" data-id="${item.id}">
    ${cover(item)}
    <span class="tag">${isBundle ? "套餐" : esc(item.kind)}</span>
    <h3>${esc(item.name)}</h3>
    <span class="price">${money(item.price)}</span>
    ${save}
    <p class="stockline">${stock} · 全店共用</p>
  </button>`;
}

function productPage() {
  const isBundle = state.id === bundle.id;
  const item = isBundle ? bundle : goods[state.id];
  if (!item) return directory();
  const booth = boothOfGood(item.id);
  const left = isBundle ? bundleLeft() : available(item);
  const wished = state.wishlist.some((row) => row.id === item.id);
  const parts = isBundle ? `<div class="facts">${bundle.parts.map((id) => `<div><dt>含</dt><dd>${esc(goods[id].name)} · ${money(goods[id].price)}</dd></div>`).join("")}</div>` : "";
  const tier = !isBundle && item.tier ? `<p class="subprice">阶梯价另计：${esc(item.tier)}</p>` : "";
  const save = isBundle ? `<p class="subprice">单买 ${money(bundle.single)}，这套省 ${money(bundle.single - bundle.price)}</p>` : tier;
  const stock = isBundle
    ? `这套还能组 ${left} 套。卖出一套会同时减少新刊、色纸和贴纸。`
    : (left === 0 && item.held === 0 ? "售罄" : `本场可售 ${left}${item.held ? `，其中 ${item.held} 件待付款` : ""}。全店还剩这些，本场和其他场一起卖。`);
  return `<section class="wrap product">
    <div>${cover(item, "cover--lg")}</div>
    <div class="buy">
      <p class="kicker">${esc(booth.no)} · ${esc(booth.name)}</p>
      <p class="price">${money(item.price)}</p>
      <h1>${esc(item.name)}</h1>
      ${save}
      <p>${esc(item.note)}</p>
      <p class="stockline">${stock}</p>
      ${parts}
      <dl class="facts">
        <div><dt>取货</dt><dd>到 ${esc(booth.no)} 现场购买</dd></div>
        <div><dt>邮寄</dt><dd>不邮寄</dd></div>
        <div><dt>开单</dt><dd>店员开单，公开页不收款</dd></div>
        <div><dt>库存</dt><dd>各场共用这一批货</dd></div>
      </dl>
      <div class="actions">
        <a class="btn btn--solid" href="#booth/${booth.id}">到 ${esc(booth.no)} 现场购买</a>
        <button class="btn btn--ghost" type="button" data-wish="${item.id}">${wished ? "已记下" : "记下想买"}</button>
      </div>
      <a class="booth-jump" href="#booth/${booth.id}"><span>${esc(booth.name)}</span><strong>${esc(booth.no)}</strong></a>
    </div>
  </section>`;
}

function wishlistPage() {
  if (!state.wishlist.length) {
    return `<section class="wrap lookup"><p class="kicker">想买</p><h1 class="display">还没有记下。</h1><p class="lede">在商品页点「记下想买」。这份清单留在这台手机上，不占库存，也不收款。</p><p class="actions"><a class="btn btn--solid" href="#directory">去本场逛摊</a></p></section>`;
  }
  const rows = state.wishlist.map((row) => {
    const item = row.id === bundle.id ? bundle : goods[row.id];
    const booth = boothOfGood(row.id);
    return `<article class="line">
      ${cover(item)}
      <div><strong>${esc(item.name)}</strong><p class="stockline">${esc(booth.no)} ${esc(booth.name)} · ${money(item.price)}</p></div>
      <div class="stepper">
        <button type="button" data-wish-step="${row.id}" data-delta="-1" aria-label="减少">−</button>
        <span>${row.qty}</span>
        <button type="button" data-wish-step="${row.id}" data-delta="1" aria-label="增加">+</button>
      </div>
    </article>`;
  }).join("");
  return `<section class="wrap">
    <p class="kicker">想买清单</p>
    <h1 class="display">到柜台出示。</h1>
    <p class="lede">店员按这份清单开单。确认之前不锁库存，也不出现应付金额。</p>
    <div class="list" style="margin-top:22px">${rows}</div>
    <p class="foot">本机保存。换一台手机就看不到。</p>
  </section>`;
}

function reviewStatus(item) {
  if (item.status === "approved") return "已通过";
  if (item.status === "rejected") return "已退回";
  return "待审核";
}
function statusLabel(status) {
  return { pending: "待付款", paid: "已收款", fulfilled: "已取货", refund_requested: "待店主确认", refund_required: "待退款", cancelled: "已取消" }[status] || status;
}

function lookupPage() {
  const found = state.lookupTried ? state.orders.find((order) => order.number.toLowerCase() === state.lookup.trim().toLowerCase()) : null;
  const miss = state.lookupTried && state.lookup.trim() && !found;
  const sheet = found ? `<article class="order-sheet">
      <p class="kicker">${esc(found.shop)}</p>
      <p class="order-no">${esc(found.number)}</p>
      <p><span class="status status--${esc(found.status)}">${statusLabel(found.status)}</span></p>
      <p>摊位 ${esc(found.booth)} · ${esc(found.lines.join("、"))}</p>
      <p class="price">${money(found.total)}</p>
      <p class="hint">到该摊号向店员出示这个订单号。这里不能付款。</p>
    </article>` : "";
  return `<section class="wrap lookup">
    <p class="kicker">订单号</p>
    <h1 class="display">凭六位号回看。</h1>
    <p class="lede">不用登录。试 M3P7LD、K7M2QP、Q8N4WC 或 R5T1HB。</p>
    <form id="lookup-form" class="stack" style="margin-top:22px">
      <label class="field"><span>订单号</span><input id="lookup" maxlength="6" value="${esc(state.lookup)}" placeholder="六位订单号" autocapitalize="characters"></label>
      <button class="btn btn--solid" type="submit">查看</button>
    </form>
    ${miss ? `<p class="banner" style="margin-top:16px">没有这个订单号。</p>` : ""}
    ${sheet}
  </section>`;
}

function cartLines() {
  return state.cart.map((row) => {
    const item = row.id === bundle.id ? bundle : goods[row.id];
    return { ...row, item, line: item.price * row.qty };
  });
}
function cartTotal() { return cartLines().reduce((sum, row) => sum + row.line, 0); }
function looseSetCount() {
  const qty = Object.fromEntries(state.cart.map((row) => [row.id, row.qty]));
  return Math.min(qty.zhantai || 0, qty.sezhi || 0, qty.sticker || 0);
}

function counterPage() {
  const catalog = [bundle, ...booths[0].goods.map((id) => goods[id])];
  const tiles = catalog.map((item) => {
    const left = item.id === bundle.id ? bundleLeft() : available(item);
    const sold = left === 0;
    return `<button class="tile" type="button" data-add="${item.id}" ${sold ? "disabled" : ""}>
      ${cover(item)}
      <strong>${esc(item.name)}</strong>
      <span class="price">${money(item.price)}</span>
      <span class="stockline">${sold ? "售罄" : `可售 ${left}`}</span>
    </button>`;
  }).join("");
  const lines = cartLines();
  const loose = looseSetCount();
  const rows = lines.map((row) => `<div class="ticket-row"><span>${esc(row.item.name)} ×${row.qty}</span><span>${money(row.line)} <button type="button" data-cart-step="${row.id}" data-delta="-1" aria-label="减少">−</button></span></div>`).join("");
  const offer = loose > 0 ? `<p class="hint">这 ${loose} 组可以改成雨季三件套，每套省 ${money(1000)}。<button class="btn" type="button" data-action="make-bundle">改成套餐</button></p>` : "";
  return `<section class="wrap">
    <p class="kicker">收银 · ${esc(eventInfo.name)} · 进行中</p>
    <div class="desk">
      <div>
        <label class="search"><span>本场货</span><input placeholder="只搜雨棚社这一场" disabled value=""></label>
        <div class="grid-goods" style="margin-top:12px">${tiles}</div>
      </div>
      <aside class="ticket-panel">
        <h2>小票</h2>
        <p class="hint">D-18 雨棚社 · 马币 · 顾客称呼可空着</p>
        ${rows || `<p class="empty">点左边的货，加进这一单。</p>`}
        ${offer}
        <div class="total-row"><span>合计</span><strong>${money(cartTotal())}</strong></div>
        <button class="btn btn--solid" type="button" data-go="pay" ${lines.length ? "" : "disabled"}>收款</button>
      </aside>
    </div>
  </section>`;
}

function qrCells() {
  const pattern = "1110111001010111010101100010111";
  return pattern.split("").map((bit) => `<i class="${bit === "1" ? "" : "off"}"></i>`).join("");
}

function payPage() {
  if (!state.cart.length && !state.receipt) {
    return `<section class="wrap lookup"><p class="kicker">出示收款</p><h1 class="display">还没有小票。</h1><p class="actions"><a class="btn btn--solid" href="#counter">回收银</a></p></section>`;
  }
  const total = cartTotal();
  const change = Math.max(0, state.cash - total);
  const short = state.payMethod === "cash" && state.cash < total;
  const qrReady = state.payMethod === "qr" && (state.ref.trim() || state.proof);
  const cashReady = state.payMethod === "cash" && state.cash >= total && total > 0;
  const ready = Boolean(state.cart.length) && (qrReady || cashReady);
  if (state.receipt) {
    return `<section class="wrap lookup">
      <p class="kicker">已收款</p>
      <h1 class="display">把订单号交给客人。</h1>
      <article class="order-sheet">
        <p class="order-no">${esc(state.receipt.number)}</p>
        <p>D-18 雨棚社 · ${money(state.receipt.total)} · ${esc(state.receipt.pay)}</p>
        <p class="hint">库存已扣。客人可以凭这个号在「订单号」页回看。</p>
      </article>
      <p class="actions"><a class="btn btn--solid" href="#lookup">去订单号页</a><a class="btn" href="#counter">继续开单</a></p>
    </section>`;
  }
  const keys = [1000, 2000, 5000].map((minor) => `<button class="money-key" type="button" data-cash="${minor}">${money(minor)}</button>`).join("");
  return `<section class="wrap">
    <p class="kicker">出示收款</p>
    <div class="pay">
      <section class="guest">
        <p class="kicker" style="color:#c9d2f2">请客人看这一面</p>
        <p class="price">${money(total)}</p>
        <p>D-18 雨棚社 · ${esc(state.payCode)}</p>
        <div class="qr" aria-hidden="true">${qrCells()}</div>
        <p>示意收款码。客人扫的是店铺的码，不是订单专属码。</p>
      </section>
      <section class="staff-pane">
        <h2>店员确认</h2>
        <div class="chips" role="group" aria-label="收款方式">
          <button class="chip" type="button" data-pay="qr" aria-pressed="${state.payMethod === "qr"}">扫码</button>
          <button class="chip" type="button" data-pay="cash" aria-pressed="${state.payMethod === "cash"}">现金</button>
        </div>
        ${state.payMethod === "qr" ? `<label class="field" style="margin-top:14px"><span>渠道参考号</span><input id="payref" value="${esc(state.ref)}" placeholder="例如 TNG-10221"></label>
          <label class="hint"><input id="proof" type="checkbox" ${state.proof ? "checked" : ""}> 已贴上付款截图</label>
          <p class="hint">参考号和截图至少要有一个，才能扣库存。</p>` : `<p class="hint">实收面额</p><div class="keys">${keys}<button class="money-key" type="button" data-action="cash-clear">清空</button></div>
          <p>实收 ${money(state.cash)} · 找零 ${money(change)}</p>
          ${short ? `<p class="banner">实收还不够。</p>` : ""}`}
        <div class="actions"><button class="btn btn--solid" type="button" data-action="confirm-pay" ${ready ? "" : "disabled"}>顾客已付款，扣减库存</button></div>
      </section>
    </div>
  </section>`;
}

function refundPage() {
  const roleName = state.role === "sales" ? "销售 阿林" : "店主 陈雨";
  const rows = state.orders.filter((order) => ["paid", "fulfilled", "refund_requested"].includes(order.status)).map((order) => {
    const mine = order.requestedBy === "阿林" && state.role === "sales";
    let action = "";
    if (order.status === "paid" || order.status === "fulfilled") {
      action = state.role === "sales"
        ? `<button class="btn" type="button" data-refund-ask="${order.number}">标成待店主确认</button>`
        : `<span class="hint">等收银员发起</span>`;
    } else if (mine) {
      action = `<button class="btn" type="button" disabled>不能确认自己发起的退款</button>`;
    } else if (state.role === "shop_admin") {
      action = `<button class="btn btn--danger" type="button" data-refund-ok="${order.number}">确认钱已退、货已收回</button>`;
    } else {
      action = `<span class="hint">待店主确认</span>`;
    }
    return `<tr><td>${esc(order.number)}</td><td>${esc(order.shop)} ${esc(order.booth)}</td><td>${money(order.total)}</td><td><span class="status status--${esc(order.status)}">${statusLabel(order.status)}</span><br>${esc(order.reason || "")}</td><td>${action}</td></tr>`;
  }).join("");
  return `<section class="wrap">
    <p class="kicker">退款复核</p>
    <h1 class="display">另一个人确认。</h1>
    <p class="lede">收银员只能发起。店主确认之后，这一单才取消并回补库存。当前身份：${roleName}。</p>
    <label class="field" style="max-width:420px;margin-top:18px"><span>退款原因</span><input id="refund-reason" value="${esc(state.refundReason || "客人不要了")}"></label>
    <p class="actions"><button class="btn" type="button" data-action="toggle-role">换成${state.role === "sales" ? "店主 陈雨" : "销售 阿林"}</button></p>
    <table class="table"><thead><tr><th>订单</th><th>摊位</th><th>金额</th><th>状态</th><th></th></tr></thead><tbody>${rows}</tbody></table>
  </section>`;
}

function varianceText() {
  const expected = state.orders
    .filter((order) => ["paid", "fulfilled"].includes(order.status) && order.pay === "现金")
    .reduce((sum, order) => sum + order.total, 0);
  if (state.counted === "") return "填写实点后才计算";
  const counted = Math.round(Number(state.counted) * 100);
  if (Number.isNaN(counted)) return "实点不是金额";
  const diff = counted - expected;
  if (diff === 0) return "持平";
  return diff > 0 ? `长款 ${money(diff)}` : `短款 ${money(diff)}`;
}
function payReady() {
  const total = cartTotal();
  const qrReady = state.payMethod === "qr" && (state.ref.trim() || state.proof);
  const cashReady = state.payMethod === "cash" && state.cash >= total && total > 0;
  return Boolean(state.cart.length) && (qrReady || cashReady);
}
function reportPage() {
  const done = state.orders.filter((order) => ["paid", "fulfilled"].includes(order.status));
  const cashOrders = done.filter((order) => order.pay === "现金");
  const expected = cashOrders.reduce((sum, order) => sum + order.total, 0);
  const diffText = varianceText();
  const codes = ["Touch 'n Go", "DuitNow", "现金"];
  const codeRows = codes.map((code) => {
    const rows = done.filter((order) => order.pay === code);
    const missing = rows.filter((order) => code !== "现金" && !order.ref).length;
    const sum = rows.reduce((total, order) => total + order.total, 0);
    return `<tr><td>${esc(code)}</td><td>${rows.length}</td><td>${money(sum)}</td><td>${code === "现金" ? "—" : `${missing} 笔缺参考号`}</td></tr>`;
  }).join("");
  return `<section class="wrap">
    <p class="kicker">收摊日报 · 马币 · 不和其他币种相加</p>
    <h1 class="display">先点现金。</h1>
    <div class="metrics" style="margin-top:22px">
      <div class="metric"><span>已收款订单</span><strong>${done.length}</strong></div>
      <div class="metric"><span>现金应有</span><strong>${money(expected)}</strong></div>
      <div class="metric"><span>长短款</span><strong id="diff">${esc(diffText)}</strong></div>
    </div>
    <label class="field" style="max-width:280px;margin-top:18px"><span>现金实点（RM）</span><input id="counted" inputmode="decimal" value="${esc(state.counted)}" placeholder="数完钱箱再填"></label>
    <p class="hint">现金应有是现金订单的金额。实点和应有的差才叫长短款，找零不算长款。</p>
    <table class="table"><thead><tr><th>收款方式</th><th>笔数</th><th>金额</th><th>凭据</th></tr></thead><tbody>${codeRows}</tbody></table>
  </section>`;
}

function reviewPage() {
  const current = state.reviews.find((item) => item.id === state.reviewId) || state.reviews[0];
    const queue = state.reviews.map((item) => `<button type="button" data-review="${item.id}" ${item.id === current.id ? 'aria-current="true"' : ""}>
      <strong>${esc(item.name)}</strong>
      <span class="stockline">${esc(item.shop)} · ${esc(item.type)} · ${esc(reviewStatus(item))}</span>
    </button>`).join("");
  const blocked = Boolean(current.blocked) && current.status === "pending";
  return `<section class="wrap">
    <p class="kicker">平台 · ${esc(eventInfo.name)}</p>
    <div class="split">
      <div>
        <h1 class="display" style="font-size:36px">本场的摊</h1>
        <div class="list" style="margin:14px 0 22px">${booths.map((booth) => `<a class="line" href="#booth/${booth.id}"><span class="ticket"><strong>${esc(booth.no)}</strong></span><span>${esc(booth.name)}</span><span></span></a>`).join("")}</div>
        <div class="queue">${queue}</div>
      </div>
      <article class="staff-pane">
        <p class="tag">${esc(current.type)}</p>
        <h2>${esc(current.name)}</h2>
        <p>${esc(current.shop)} · ${esc(current.price)}</p>
        <p>${esc(current.note)}</p>
        ${blocked ? `<p class="banner">${esc(current.blocked)}。通过按钮不可用。</p>` : ""}
        <div class="actions">
          <button class="btn btn--solid" type="button" data-action="approve" ${current.status !== "pending" || blocked ? "disabled" : ""}>通过</button>
          <button class="btn btn--danger" type="button" data-action="reject" ${current.status !== "pending" ? "disabled" : ""}>退回</button>
        </div>
        <p class="hint">自动通过当前${state.autoApprove ? "开着" : "关着"}。最近一次由平台管理员在 11月20日 09:14 关闭。</p>
        <button class="btn" type="button" data-action="auto">${state.autoApprove ? "关闭自动通过" : "打开自动通过"}</button>
      </article>
    </div>
  </section>`;
}

function viewBody() {
  switch (state.view) {
    case "booth": return boothPage();
    case "product": return productPage();
    case "wishlist": return wishlistPage();
    case "lookup": return lookupPage();
    case "counter": return counterPage();
    case "pay": return payPage();
    case "refund": return refundPage();
    case "report": return reportPage();
    case "review": return reviewPage();
    default: return directory();
  }
}

function render() {
  const focus = document.activeElement;
  const focusId = focus && focus.id;
  const caret = focus && typeof focus.selectionStart === "number" ? focus.selectionStart : null;
  document.getElementById("app").innerHTML = `${header()}<main id="main">${viewBody()}</main><p class="live" aria-live="polite"></p>`;
  if (focusId) {
    const next = document.getElementById(focusId);
    if (next) {
      next.focus();
      if (caret != null && next.setSelectionRange) next.setSelectionRange(caret, caret);
    }
  }
  const titles = { directory: "本场目录", booth: "摊位", product: "商品", wishlist: "想买", lookup: "订单号", counter: "收银", pay: "出示收款", refund: "退款复核", report: "收摊日报", review: "本场审核" };
  document.title = `Stallink · ${titles[state.view] || "Comic Fiesta"}`;
}

function addWish(id) {
  const row = state.wishlist.find((item) => item.id === id);
  if (row) row.qty += 1;
  else state.wishlist.push({ id, qty: 1 });
  saveWish();
}
function stepWish(id, delta) {
  const row = state.wishlist.find((item) => item.id === id);
  if (!row) return;
  row.qty += delta;
  if (row.qty <= 0) state.wishlist = state.wishlist.filter((item) => item.id !== id);
  saveWish();
}
function addCart(id) {
  const item = id === bundle.id ? bundle : goods[id];
  const left = id === bundle.id ? bundleLeft() : available(item);
  const row = state.cart.find((line) => line.id === id);
  const have = row ? row.qty : 0;
  if (have >= left) return;
  if (row) row.qty += 1;
  else state.cart.push({ id, qty: 1 });
  state.receipt = null;
}
function makeBundle() {
  const sets = looseSetCount();
  if (!sets) return;
  for (const id of bundle.parts) {
    const row = state.cart.find((line) => line.id === id);
    row.qty -= sets;
    if (row.qty <= 0) state.cart = state.cart.filter((line) => line.id !== id);
  }
  const existing = state.cart.find((line) => line.id === bundle.id);
  if (existing) existing.qty += sets;
  else state.cart.push({ id: bundle.id, qty: sets });
}
function confirmPay() {
  const total = cartTotal();
  if (!total) return;
  const number = randomNumber();
  const pay = state.payMethod === "cash" ? "现金" : state.payCode;
  for (const row of state.cart) {
    if (row.id === bundle.id) {
      for (const part of bundle.parts) goods[part].stock = Math.max(0, goods[part].stock - row.qty);
    } else {
      goods[row.id].stock = Math.max(0, goods[row.id].stock - row.qty);
    }
  }
  const order = {
    number,
    status: "paid",
    booth: "D-18",
    shop: "雨棚社",
    total,
    lines: cartLines().map((row) => `${row.item.name} ×${row.qty}`),
    pay,
    ref: state.payMethod === "qr" ? state.ref.trim() : "",
    by: "阿林",
  };
  state.orders.unshift(order);
  state.receipt = order;
  state.cart = [];
  state.cash = 0;
  state.ref = "";
  state.proof = false;
  state.lookup = number;
}
function randomNumber() {
  const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789";
  return Array.from({ length: 6 }, () => alphabet[Math.floor(Math.random() * alphabet.length)]).join("");
}

document.getElementById("app").addEventListener("click", (event) => {
  const el = event.target.closest("[data-go],[data-filter],[data-wish],[data-wish-step],[data-add],[data-cart-step],[data-action],[data-pay],[data-cash],[data-refund-ask],[data-refund-ok],[data-review]");
  if (!el) return;
  if (el.dataset.go) go(el.dataset.go, el.dataset.id);
  if (el.dataset.filter) { state.filter = el.dataset.filter; render(); }
  if (el.dataset.wish) { addWish(el.dataset.wish); render(); }
  if (el.dataset.wishStep) { stepWish(el.dataset.wishStep, Number(el.dataset.delta)); render(); }
  if (el.dataset.add) { addCart(el.dataset.add); render(); }
  if (el.dataset.cartStep) {
    const row = state.cart.find((line) => line.id === el.dataset.cartStep);
    if (row) row.qty += Number(el.dataset.delta);
    state.cart = state.cart.filter((line) => line.qty > 0);
    render();
  }
  if (el.dataset.pay) { state.payMethod = el.dataset.pay; render(); }
  if (el.dataset.cash) { state.cash += Number(el.dataset.cash); render(); }
  if (el.dataset.review) { state.reviewId = el.dataset.review; render(); }
  if (el.dataset.refundAsk) {
    const order = state.orders.find((item) => item.number === el.dataset.refundAsk);
    order.status = "refund_requested";
    order.reason = state.refundReason || "客人不要了";
    order.requestedBy = "阿林";
    render();
  }
  if (el.dataset.refundOk) {
    const order = state.orders.find((item) => item.number === el.dataset.refundOk);
    if (order.requestedBy === "阿林" && state.role === "sales") return;
    order.status = "cancelled";
    render();
  }
  const action = el.dataset.action;
  if (action === "make-bundle") { makeBundle(); render(); }
  if (action === "cash-clear") { state.cash = 0; render(); }
  if (action === "confirm-pay") { confirmPay(); render(); }
  if (action === "toggle-role") { state.role = state.role === "sales" ? "shop_admin" : "sales"; render(); }
  if (action === "approve" || action === "reject") {
    const item = state.reviews.find((row) => row.id === state.reviewId);
    if (!item || item.status !== "pending") return;
    if (action === "approve" && item.blocked) return;
    item.status = action === "approve" ? "approved" : "rejected";
    render();
  }
  if (action === "auto") { state.autoApprove = !state.autoApprove; render(); }
});

document.getElementById("app").addEventListener("input", (event) => {
  if (event.target.id === "q") {
    state.q = event.target.value;
    const list = document.querySelector(".booth-list");
    const count = document.querySelector(".result-count");
    if (list && count && state.view === "directory") {
      const found = filteredBooths();
      count.textContent = `${found.length} 个摊位`;
      list.innerHTML = found.map(boothCard).join("") || `<p class="empty">这场没有符合的摊位。</p>`;
    }
  }
  if (event.target.id === "lookup") state.lookup = event.target.value;
  if (event.target.id === "payref") {
    state.ref = event.target.value;
    const button = document.querySelector("[data-action='confirm-pay']");
    if (button) button.disabled = !payReady();
  }
  if (event.target.id === "refund-reason") state.refundReason = event.target.value;
  if (event.target.id === "proof") { state.proof = event.target.checked; render(); }
  if (event.target.id === "counted") {
    state.counted = event.target.value;
    const node = document.getElementById("diff");
    if (node) node.textContent = varianceText();
  }
});

document.getElementById("app").addEventListener("submit", (event) => {
  if (event.target.id !== "lookup-form") return;
  event.preventDefault();
  const input = document.getElementById("lookup");
  if (input) state.lookup = input.value;
  state.lookupTried = true;
  render();
});

window.addEventListener("hashchange", () => { parseHash(); render(); });
parseHash();
if (!location.hash) location.hash = "#directory";
else render();
