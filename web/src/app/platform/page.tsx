"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import Link from "next/link";
import { browserJSON } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { messageClass, useMessage } from "@/lib/message";
import { appearancePath, isStorefrontTheme, type StorefrontTheme } from "@/lib/storefront-theme";
import { ConfirmDialog } from "@/components/confirm-dialog";

type Shop = { id: string; name: string; slug: string; isEnabled: boolean };

export default function PlatformPage() {
  const router = useRouter();
  const [form, setForm] = useState({ name: "", slug: "", notice: "", adminLogin: "", adminPassword: "" });
  const [shops, setShops] = useState<Shop[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [pending, setPending] = useState(false);
  const [confirmTarget, setConfirmTarget] = useState<Shop | null>(null);
  const [theme, setTheme] = useState<StorefrontTheme | null>(null);
  const { message, setError, setSuccess } = useMessage();

  const load = useCallback(() => {
    setLoadError("");
    browserJSON(appearancePath).then((res) => {
      if (isStorefrontTheme(res.data.theme)) setTheme(res.data.theme);
    }).catch(() => {});
    browserJSON("/api/platform/shops").then((res) => {
      if (res.category === "unauthorized" || res.category === "forbidden") {
        router.replace("/login");
        return;
      }
      if (!res.ok) { setLoadError(describeRequest(res)); return; }
      setShops(Array.isArray(res.data) ? res.data as Shop[] : []);
    }).catch(() => setLoadError("无法连接服务，请检查网络后重试。"))
      .finally(() => setLoading(false));
  }, [router]);
  useEffect(() => {
    const timer = window.setTimeout(load, 0);
    return () => window.clearTimeout(timer);
  }, [router, load]);

  async function createShop(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setError(null);
    try {
      const res = await browserJSON("/api/platform/shops", "POST", form);
      if (!res.ok) {
        setError(res.errorCode === "login_taken" ? "管理员账号已被使用，请换一个。" : describeRequest(res));
        return;
      }
      setSuccess(`店铺已创建：/c/${form.slug}`);
      setForm({ name: "", slug: "", notice: "", adminLogin: "", adminPassword: "" });
      load();
    } catch {
      setError("无法连接服务，请检查网络后重试。");
    } finally {
      setPending(false);
    }
  }

  async function toggleShop(shop: Shop, nextEnabled: boolean) {
    setPending(true);
    setError(null);
    try {
      const res = await browserJSON(`/api/platform/shops/${shop.id}`, "PATCH", { isEnabled: nextEnabled });
      if (!res.ok) { setError(describeRequest(res)); return; }
      setSuccess(nextEnabled ? `店铺「${shop.name}」已重新启用。` : `店铺「${shop.name}」已停用。`);
      load();
    } catch {
      setError("无法连接服务，请检查网络后重试。");
    } finally {
      setPending(false);
      setConfirmTarget(null);
    }
  }

  return (
    <>
      <main>
        <p className="eyebrow">创作者入驻</p>
        <h1>开设店铺</h1>
        <p>为一位创作者建立独立摊位链接和第一位店铺管理员。</p>
        <section className="platform-appearance">
          <p>
            <span className="eyebrow">平台外观</span>
            <strong>{theme === "exhibition" ? "展场风格" : theme === "shop" ? "店面风格" : theme === "classic" ? "现有风格" : "读取中…"}</strong>
            <span className="platform-appearance__note">· 顾客与工作台共用一套界面，切换后全站立即生效。</span>
          </p>
          <Link className="button button--outline" href="/platform/appearance">切换界面风格 ↗</Link>
        </section>
        <form className="platform-create-form" onSubmit={createShop}>
          <label className="field" htmlFor="platform-shop-name"><span>店铺名称</span><input id="platform-shop-name" required value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} placeholder="例如：Mika 的作品摊" data-error="platform-shop-name-error" /><span id="platform-shop-name-error" className="field__validity">错误：请填写店铺名称。</span></label>
          <label className="field" htmlFor="platform-shop-slug"><span>专属链接 <small>3-40 位小写字母、数字或连字符</small></span><input id="platform-shop-slug" required minLength={3} maxLength={40} pattern="[a-z0-9][a-z0-9-]{1,38}[a-z0-9]" value={form.slug} onChange={(event) => setForm({ ...form, slug: event.target.value.toLowerCase() })} placeholder="例如：mika-art" data-error="platform-shop-slug-error" /><span id="platform-shop-slug-error" className="field__validity">错误：专属链接需要 3 到 40 位小写字母、数字或连字符。</span></label>
          <label className="field" htmlFor="platform-shop-notice"><span>店铺公告 <small>选填</small></span><textarea id="platform-shop-notice" rows={3} value={form.notice} onChange={(event) => setForm({ ...form, notice: event.target.value })} placeholder="介绍作品或近期出摊安排" /></label>
          <div className="platform-form-divider"><span>店铺管理员</span><i /></div>
          <label className="field" htmlFor="platform-admin-login"><span>管理员账号 <small>3-32 位，不含空格</small></span><input id="platform-admin-login" autoComplete="off" required minLength={3} maxLength={32} pattern="\S{3,32}" value={form.adminLogin} onChange={(event) => setForm({ ...form, adminLogin: event.target.value })} placeholder="为店主创建登录名" data-error="platform-admin-login-error" /><span id="platform-admin-login-error" className="field__validity">错误：管理员账号需要 3 到 32 个字符，且不能包含空格。</span></label>
          <label className="field" htmlFor="platform-admin-password"><span>初始密码 <small>至少 8 位</small></span><input id="platform-admin-password" type="password" autoComplete="new-password" required minLength={8} maxLength={72} value={form.adminPassword} onChange={(event) => setForm({ ...form, adminPassword: event.target.value })} placeholder="设置临时密码" data-error="platform-admin-password-error" /><span id="platform-admin-password-error" className="field__validity">错误：初始密码至少需要 8 个字符。</span></label>
          {message ? <p className={messageClass(message)} role="status">{message.text}</p> : null}
          <button className="button button--dark" type="submit" disabled={pending}>{pending ? "正在创建…" : "创建店铺"}</button>
        </form>

        <div className="workspace-list-heading"><h2>已开设店铺</h2><span>{shops.length} 家</span></div>
        {loading ? <p className="empty-state">正在读取店铺…</p> : loadError ? (
          <p className="empty-state" role="alert">
            <span className="empty-state__mark" aria-hidden="true">!</span>
            <span>{loadError}</span>
            <button type="button" className="button button--outline" onClick={load}>重试</button>
          </p>
        ) : shops.length === 0 ? (
          <div className="empty-state empty-state--compact"><span className="empty-state__mark" aria-hidden="true">✳</span><p>还没有店铺。创建后，店主可以管理自己的商品和活动。</p></div>
        ) : (
          <ul className="platform-shop-list">
            {shops.map((shop) => (
              <li key={shop.id}>
                <div><strong>{shop.name}</strong><span>/c/{shop.slug}</span></div>
                <span className={shop.isEnabled ? "account-status" : "account-status account-status--disabled"}><i aria-hidden="true" />{shop.isEnabled ? "公开中" : "已停用"}</span>
                {shop.isEnabled ? (
                  <button type="button" className="button button--danger" disabled={pending} onClick={() => setConfirmTarget(shop)}>停用店铺</button>
                ) : (
                  <button type="button" className="button button--outline" disabled={pending} onClick={() => void toggleShop(shop, true)}>重新启用</button>
                )}
              </li>
            ))}
          </ul>
        )}
      </main>
      <ConfirmDialog
        open={confirmTarget !== null}
        title="停用店铺"
        description={`停用后「${confirmTarget?.name ?? ""}」的公开链接暂时无法访问，在售商品一并隐藏；可随时重新启用，这不是删除店铺。`}
        confirmLabel="停用店铺"
        danger
        pending={pending}
        onConfirm={() => confirmTarget && void toggleShop(confirmTarget, false)}
        onCancel={() => setConfirmTarget(null)}
      />
    </>
  );
}
