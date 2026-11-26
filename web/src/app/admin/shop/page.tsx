"use client";

import { useEffect, useState, type FormEvent } from "react";
import { browserJSON } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { messageClass, useMessage } from "@/lib/message";
import { ConfirmDialog } from "@/components/confirm-dialog";

type FeaturedEvent = {
  id: string;
  name: string;
  boothNumber?: string;
  currency?: string;
  status?: string;
};

const CURRENCY_OPTIONS = [
  { value: "MYR", label: "MYR · RM" },
  { value: "CNY", label: "CNY · ¥" },
  { value: "SGD", label: "SGD · S$" },
];

const DISABLE_COPY = "停用后，店里的账号登录后进不了工作台，需要平台管理员重新启用。";

function formatShopUpdatedAt(iso: string): string {
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) return iso;
  return new Intl.DateTimeFormat("zh-CN", { timeZone: "Asia/Kuala_Lumpur" }).format(date);
}

export default function ShopSettingsPage() {
  const [name, setName] = useState("");
  const [notice, setNotice] = useState("");
  const [slug, setSlug] = useState("");
  const [isEnabled, setIsEnabled] = useState(true);
  const [contact, setContact] = useState("");
  const [boothNumber, setBoothNumber] = useState("");
  const [defaultCurrency, setDefaultCurrency] = useState("");
  const [updatedAt, setUpdatedAt] = useState("");
  const [featured, setFeatured] = useState<FeaturedEvent | null>(null);
  const [listedCount, setListedCount] = useState<number | null>(null);
  const [boothSpaceHint, setBoothSpaceHint] = useState(false);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [copyLabel, setCopyLabel] = useState("复制链接");
  const [confirmDisable, setConfirmDisable] = useState(false);
  const { message, setError, setSuccess } = useMessage();

  function applyProfile(data: Record<string, unknown>) {
    setSlug(String(data.slug ?? ""));
    setName(String(data.name ?? ""));
    setNotice(String(data.notice ?? ""));
    setIsEnabled(Boolean(data.isEnabled ?? true));
    setContact(String(data.contact ?? ""));
    setBoothNumber(String(data.boothNumber ?? ""));
    setDefaultCurrency(data.defaultCurrency == null || data.defaultCurrency === "" ? "" : String(data.defaultCurrency));
    setUpdatedAt(typeof data.updatedAt === "string" ? data.updatedAt : "");
    setListedCount(typeof data.onSaleCount === "number" ? data.onSaleCount : null);
    setBoothSpaceHint(false);
    const event = data.featuredEvent;
    if (event && typeof event === "object") {
      const row = event as Record<string, unknown>;
      setFeatured({
        id: String(row.id ?? ""),
        name: String(row.name ?? ""),
        boothNumber: String(row.boothNumber ?? ""),
        currency: String(row.currency ?? ""),
        status: String(row.status ?? ""),
      });
    } else {
      setFeatured(null);
    }
  }

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const shopRes = await browserJSON("/api/admin/shop");
        if (!active) return;
        if (!shopRes.ok) {
          setError(shopRes.category === "unauthorized" || shopRes.status === 401
            ? "登录已过期，请重新登录后再修改店铺资料。"
            : describeRequest(shopRes));
          setLoading(false);
          return;
        }
        applyProfile(shopRes.data);
        setLoading(false);
      } catch {
        if (active) {
          setError("店铺资料暂时无法读取。");
          setLoading(false);
        }
      }
    })();
    return () => { active = false; };
  }, [setError]);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (!isEnabled) {
      setConfirmDisable(true);
      return;
    }
    await save();
  }

  async function save() {
    setSaving(true);
    setError(null);
    try {
      const res = await browserJSON("/api/admin/shop", "PATCH", {
        name,
        notice,
        isEnabled,
        contact,
        boothNumber,
        defaultCurrency: defaultCurrency || null,
      });
      if (!res.ok) { setError(describeRequest(res)); return; }
      applyProfile(res.data);
      setSuccess(isEnabled ? "店铺资料已保存。" : DISABLE_COPY);
    } catch {
      setError("无法连接服务，请检查网络后重试。");
    } finally {
      setSaving(false);
    }
  }

  async function copyLink() {
    const link = `${window.location.origin}/c/${slug}`;
    try {
      await navigator.clipboard.writeText(link);
      setCopyLabel("已复制");
    } catch {
      setCopyLabel("请手动复制");
    }
  }

  const eventLabel = featured ? featured.name : "—";
  const listedLabel = listedCount == null ? "—" : `${listedCount} 个`;

  return (
    <main className="workspace-subpage">
      <p className="eyebrow">公开页面</p>
      <h1>店铺资料</h1>
      <p>这些信息会显示在顾客打开的摊位链接里。</p>
      <div className="shop-settings">
        <form className="shop-settings-form" onSubmit={submit}>
          {loading ? <div className="profile-loading" role="status"><span className="loading-mark" />正在读取店铺资料…</div> : (
            <>
              <div className="shop-settings-form__head"><h2>公开信息</h2><span>顾客可见</span></div>
              <label className="field" htmlFor="shop-name"><span>店铺名称</span><input id="shop-name" required value={name} onChange={(event) => setName(event.target.value)} placeholder="你的摊位名称" /></label>
              <label className="field" htmlFor="shop-notice"><span>店铺公告 <small>选填</small></span><textarea id="shop-notice" rows={4} value={notice} onChange={(event) => setNotice(event.target.value)} placeholder="介绍作品、出摊安排或给顾客的提醒" /></label>
              <label className="field" htmlFor="shop-contact">
                <span>客服联系</span>
                <input id="shop-contact" value={contact} maxLength={120} onChange={(event) => setContact(event.target.value)} placeholder="例如电话、微信或邮箱" />
              </label>
              <label className="field" htmlFor="shop-booth">
                <span>默认摊位号</span>
                <input
                  id="shop-booth"
                  value={boothNumber}
                  onChange={(event) => { setBoothNumber(event.target.value); setBoothSpaceHint(false); }}
                  onBlur={() => setBoothSpaceHint(boothNumber !== boothNumber.trim())}
                  placeholder="例如：A-12"
                />
                <small className="field-hint">
                  新建活动时自动带入；顾客端优先显示活动自己的摊位号，活动没填才用这里的
                  {boothSpaceHint ? "。首尾不能有空格" : ""}
                </small>
              </label>
              <label className="field" htmlFor="shop-currency">
                <span>默认货币</span>
                <select id="shop-currency" value={defaultCurrency} aria-label="默认货币" onChange={(event) => setDefaultCurrency(event.target.value)}>
                  <option value="">未设置</option>
                  {CURRENCY_OPTIONS.map((option) => (
                    <option key={option.value} value={option.value}>{option.label}</option>
                  ))}
                </select>
                <small className="field-hint">新建活动时自动带入</small>
              </label>
              <div className="field">
                <span>店铺地址</span>
                <p className="shop-settings-slug"><strong>/c/{slug || "—"}</strong></p>
                <small className="field-hint">店铺地址 /c/{slug || "…"} 由平台分配，不可自行修改。</small>
              </div>
              <label className="check-field"><input type="checkbox" checked={isEnabled} onChange={(event) => setIsEnabled(event.target.checked)} /><span>公开展示这家店铺（取消勾选即停用，顾客暂时看不到店铺与在售商品，可随时恢复）</span></label>
              {message ? <p className={messageClass(message)} role="status">{message.text}</p> : null}
              <button className="button button--dark" type="submit" disabled={saving}>{saving ? "正在保存…" : "保存修改"}</button>
            </>
          )}
        </form>
        <aside className="shop-settings-preview">
          <div className="shop-settings-preview__panel">
            <div className="shop-settings-form__head"><h2>展示状态</h2><span>预览</span></div>
            <div className="shop-settings-kv"><span>顾客端入口</span><strong>/c/{slug || "—"}</strong></div>
            <div className="shop-settings-kv"><span>当前活动</span><strong>{loading ? "…" : eventLabel}</strong></div>
            <div className="shop-settings-kv"><span>在售商品</span><strong>{loading ? "…" : listedLabel}</strong></div>
            <div className="shop-settings-kv"><span>最近更新</span><strong>{updatedAt ? formatShopUpdatedAt(updatedAt) : "—"}</strong></div>
          </div>
          <div className="shop-link-preview">
            <div><span>顾客访问链接</span><strong>/c/{slug || "—"}</strong></div>
            <button type="button" className="button button--outline" onClick={copyLink} disabled={!slug}>{copyLabel}</button>
          </div>
          {slug ? (
            <a className="button button--outline" href={`/c/${slug}`} target="_blank" rel="noreferrer">打开店铺预览</a>
          ) : null}
        </aside>
      </div>
      <ConfirmDialog
        open={confirmDisable}
        title="停用店铺"
        description={<>停用后，店里的账号<strong>登录后进不了工作台</strong>，需要平台管理员重新启用。</>}
        confirmLabel="停用店铺"
        danger
        pending={saving}
        onConfirm={() => { setConfirmDisable(false); void save(); }}
        onCancel={() => { setConfirmDisable(false); setIsEnabled(true); }}
      />
    </main>
  );
}
