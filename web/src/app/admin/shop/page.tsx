"use client";

import { useEffect, useState } from "react";
import { browserJSON } from "@/lib/api";

export default function ShopSettingsPage() {
  const [name, setName] = useState("");
  const [notice, setNotice] = useState("");
  const [slug, setSlug] = useState("");
  const [isEnabled, setIsEnabled] = useState(true);
  const [message, setMessage] = useState("");
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [copyLabel, setCopyLabel] = useState("复制链接");

  useEffect(() => {
    let active = true;
    browserJSON("/api/auth/me").then(async (me) => {
      if (!me.ok || me.data.role !== "shop_admin") {
        if (active) {
          setMessage("登录已过期，请重新登录后再修改店铺资料。");
          setLoading(false);
        }
        return;
      }
      const slugValue = String(me.data.shopSlug ?? "");
      const res = await fetch(`/api/public/shops/${encodeURIComponent(slugValue)}`);
      const shop = res.ok ? await res.json() : {};
      if (!active) return;
      setSlug(slugValue);
      setName(String(shop.name ?? ""));
      setNotice(String(shop.notice ?? ""));
      setLoading(false);
    }).catch(() => {
      if (active) {
        setMessage("店铺资料暂时无法读取。");
        setLoading(false);
      }
    });
    return () => { active = false; };
  }, []);

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    setSaving(true);
    try {
      const res = await browserJSON("/api/admin/shop", "PATCH", { name, notice, slug, isEnabled });
      setMessage(res.ok ? "店铺资料已保存。" : "保存失败，请检查店名和链接后重试。");
    } catch {
      setMessage("无法连接服务，请检查网络后重试。");
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

  return (
    <main>
      <p className="eyebrow">公开页面</p>
      <h1>店铺资料</h1>
      <p>这些信息会显示在顾客打开的摊位链接里。</p>
      <form className="shop-settings-form" onSubmit={submit}>
        {loading ? <div className="profile-loading" role="status"><span className="loading-mark" />正在读取店铺资料…</div> : (
          <>
            <label className="field" htmlFor="shop-name"><span>店铺名称</span><input id="shop-name" required value={name} onChange={(event) => setName(event.target.value)} placeholder="你的摊位名称" /></label>
            <label className="field" htmlFor="shop-notice"><span>店铺公告 <small>选填</small></span><textarea id="shop-notice" rows={4} value={notice} onChange={(event) => setNotice(event.target.value)} placeholder="介绍作品、出摊安排或给顾客的提醒" /></label>
            <label className="field" htmlFor="shop-slug"><span>专属链接</span><span className="slug-input"><span>/c/</span><input id="shop-slug" required value={slug} onChange={(event) => setSlug(event.target.value)} /></span></label>
            <div className="shop-link-preview"><div><span>顾客访问链接</span><strong>/c/{slug}</strong></div><button type="button" className="button button--outline" onClick={copyLink}>{copyLabel}</button></div>
            <label className="check-field"><input type="checkbox" checked={isEnabled} onChange={(event) => setIsEnabled(event.target.checked)} /><span>公开展示这家店铺</span></label>
            {message ? <p className={`form-message${message.includes("失败") || message.includes("无法") || message.includes("暂时") ? " form-message--error" : " form-message--success"}`} role="status">{message}</p> : null}
            <button className="button button--dark" type="submit" disabled={saving}>{saving ? "正在保存…" : "保存店铺资料"}</button>
          </>
        )}
      </form>
    </main>
  );
}
