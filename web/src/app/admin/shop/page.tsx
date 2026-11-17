"use client";

import { useEffect, useState } from "react";
import { browserJSON } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { messageClass, useMessage } from "@/lib/message";
import { ConfirmDialog } from "@/components/confirm-dialog";

export default function ShopSettingsPage() {
  const [name, setName] = useState("");
  const [notice, setNotice] = useState("");
  const [slug, setSlug] = useState("");
  const [isEnabled, setIsEnabled] = useState(true);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [copyLabel, setCopyLabel] = useState("复制链接");
  const [confirmDisable, setConfirmDisable] = useState(false);
  const { message, setError, setSuccess } = useMessage();

  useEffect(() => {
    let active = true;
    (async () => {
      try {
        const me = await browserJSON("/api/auth/me");
        if (!active) return;
        if (!me.ok || me.data.role !== "shop_admin") {
          setError("登录已过期，请重新登录后再修改店铺资料。");
          setLoading(false);
          return;
        }
        const slugValue = String(me.data.shopSlug ?? "");
        const shopRes = await browserJSON(`/api/public/shops/${encodeURIComponent(slugValue)}`);
        if (!active) return;
        setSlug(slugValue);
        setName(String(shopRes.data.name ?? ""));
        setNotice(String(shopRes.data.notice ?? ""));
        setIsEnabled(Boolean(shopRes.data.isEnabled ?? true));
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

  async function submit(event: React.FormEvent<HTMLFormElement>) {
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
      const res = await browserJSON("/api/admin/shop", "PATCH", { name, notice, slug, isEnabled });
      if (!res.ok) { setError(describeRequest(res)); return; }
      setSuccess("店铺资料已保存。");
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
            <label className="check-field"><input type="checkbox" checked={isEnabled} onChange={(event) => setIsEnabled(event.target.checked)} /><span>公开展示这家店铺（取消勾选即停用，顾客暂时看不到店铺与在售商品，可随时恢复）</span></label>
            {message ? <p className={messageClass(message)} role="status">{message.text}</p> : null}
            <button className="button button--dark" type="submit" disabled={saving}>{saving ? "正在保存…" : "保存店铺资料"}</button>
          </>
        )}
      </form>
      <ConfirmDialog
        open={confirmDisable}
        title="停用店铺"
        description="停用后公开链接暂时无法访问，在售商品一并隐藏；可随时重新启用，这不是删除店铺。"
        confirmLabel="停用店铺"
        danger
        pending={saving}
        onConfirm={() => { setConfirmDisable(false); void save(); }}
        onCancel={() => { setConfirmDisable(false); setIsEnabled(true); }}
      />
    </main>
  );
}
