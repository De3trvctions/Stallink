"use client";

import { useEffect, useRef, useState } from "react";
import Image from "next/image";
import { SafeImage } from "@/components/safe-image";
import { browserFormData, browserJSON } from "@/lib/api";
import { describeRequest } from "@/lib/errors";
import { messageClass, useMessage } from "@/lib/message";
import { ConfirmDialog } from "@/components/confirm-dialog";

type PayCode = {
  id: string;
  name: string;
  kind: "qr" | "cash";
  currency: string | null;
  isDefault: boolean;
  isEnabled: boolean;
  imageURL: string;
};

const CURRENCY_OPTIONS = ["MYR", "SGD", "CNY"];

export default function AdminPayCodesPage() {
  const [codes, setCodes] = useState<PayCode[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");
  const [adding, setAdding] = useState(false);
  const [form, setForm] = useState({ name: "", kind: "qr" as "qr" | "cash", currency: "", isDefault: false });
  const [imageFile, setImageFile] = useState<File | null>(null);
  const [preview, setPreview] = useState("");
  const [imageError, setImageError] = useState("");
  const [pendingId, setPendingId] = useState("");
  const [confirmTarget, setConfirmTarget] = useState<PayCode | null>(null);
  const { message, setError, setSuccess } = useMessage();
  const fileRef = useRef<HTMLInputElement>(null);

  async function load() {
    try {
      const res = await browserJSON("/api/admin/pay-codes");
      if (!res.ok) { setLoadError(describeRequest(res)); return; }
      setCodes(Array.isArray(res.data) ? res.data as PayCode[] : []);
    } catch {
      setLoadError("无法连接服务，请检查网络后重试。");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    const timer = window.setTimeout(() => { void load(); }, 0);
    return () => window.clearTimeout(timer);
  }, []);

  function selectImage(event: React.ChangeEvent<HTMLInputElement>) {
    const file = event.currentTarget.files?.[0];
    event.currentTarget.value = "";
    setImageError("");
    if (!file) { setPreview(""); return; }
    if (file.size > 6 * 1024 * 1024) {
      setImageError("收款码图片不能超过 6 MiB。");
      return;
    }
    if (!["image/jpeg", "image/png", "image/webp"].includes(file.type)) {
      setImageError("请选择 JPG、PNG 或 WebP 格式的收款码图片。");
      return;
    }
    if (preview) URL.revokeObjectURL(preview);
    setImageFile(file);
    setPreview(URL.createObjectURL(file));
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setError(null);
    if (!form.name.trim()) { setError("请填写收款方式名称。"); return; }
    if (form.kind === "qr" && !imageFile) { setError("请上传收款码图片。"); return; }
    setPendingId("");
    setAdding(true);
    try {
      let res;
      if (form.kind === "cash") {
        res = await browserJSON("/api/admin/pay-codes/cash", "POST", { name: form.name.trim(), isDefault: form.isDefault, ...(form.currency ? { currency: form.currency } : {}) });
      } else {
        const body = new FormData();
        body.set("name", form.name.trim());
        body.set("isDefault", String(form.isDefault));
        body.set("currency", form.currency);
        if (imageFile) body.append("image", imageFile);
        res = await browserFormData("/api/admin/pay-codes", "POST", body);
      }
      if (!res.ok) { setError(describeRequest(res)); return; }
      setSuccess("收款方式已保存。");
      setForm({ name: "", kind: "qr", currency: "", isDefault: false });
      setImageFile(null);
      setImageError("");
      if (fileRef.current) fileRef.current.value = "";
      await load();
    } catch {
      setError("无法连接服务，请检查网络后重试。");
    } finally {
      setAdding(false);
    }
  }

  async function toggleEnabled(code: PayCode, nextEnabled: boolean) {
    setPendingId(code.id);
    setError(null);
    try {
      const res = await browserJSON(`/api/admin/pay-codes/${code.id}`, "PATCH", {
        name: code.name,
        isEnabled: nextEnabled,
        isDefault: code.isDefault,
        currency: code.currency,
      });
      if (!res.ok) { setError(describeRequest(res)); return; }
      setSuccess(nextEnabled ? "收款方式已启用。" : "收款方式已停用。");
      await load();
    } catch {
      setError("无法连接服务，请检查网络后重试。");
    } finally {
      setPendingId("");
      setConfirmTarget(null);
    }
  }

  async function makeDefault(code: PayCode) {
    setPendingId(code.id);
    setError(null);
    try {
      const res = await browserJSON(`/api/admin/pay-codes/${code.id}`, "PATCH", {
        name: code.name,
        isEnabled: code.isEnabled,
        isDefault: true,
        currency: code.currency,
      });
      if (!res.ok) { setError(describeRequest(res)); return; }
      setSuccess(`已把「${code.name}」设为默认收款方式。`);
      await load();
    } catch {
      setError("无法连接服务，请检查网络后重试。");
    } finally {
      setPendingId("");
    }
  }

  return (
    <main className="workspace-page workspace-subpage">
      <header className="workspace-page__heading">
        <p className="eyebrow">付款配置</p>
        <h1>收款方式</h1>
        <p>上传收款码或新增现金收款。币种留空表示跟随活动币种；与订单币种不一致的收款码无法使用。</p>
      </header>

      {message ? <p className={messageClass(message)} role="status">{message.text}</p> : null}

      {loading ? <p className="empty-state">正在读取收款方式…</p> : loadError ? (
        <p className="empty-state" role="alert">
          <span className="empty-state__mark" aria-hidden="true">!</span>
          <span>{loadError}</span>
          <button type="button" className="button button--outline" onClick={() => void load()}>重试</button>
        </p>
      ) : codes.length === 0 ? (
        <p className="empty-state"><span className="empty-state__mark" aria-hidden="true">○</span><span>还没有收款方式，先在下方添加。</span></p>
      ) : (
        <ul className="pay-code-list">
          {codes.map((code) => (
            <li className="pay-code-row" key={code.id}>
              <div>
                {code.imageURL ? <SafeImage className="pay-code-row__thumb" src={code.imageURL} alt={`${code.name} 收款码`} width={56} height={56} unoptimized fallback={<span className="pay-code-row__thumb pay-code-row__thumb--empty" aria-hidden="true">现金</span>} /> : <span className="pay-code-row__thumb pay-code-row__thumb--empty" aria-hidden="true">现金</span>}
                <div>
                  <strong>{code.name}</strong>
                  <span className="pay-code-row__meta">
                    {code.kind === "cash" ? "现金" : "扫码"}
                    {code.currency ? ` · ${code.currency}` : " · 跟随活动币种"}
                    {code.isDefault ? " · 默认" : ""}
                  </span>
                  <span className={`account-status${code.isEnabled ? "" : " account-status--disabled"}`}><i aria-hidden="true" />{code.isEnabled ? "启用中" : "已停用"}</span>
                </div>
              </div>
              <div className="pay-code-row__actions">
                {!code.isDefault ? <button type="button" className="button button--outline" disabled={pendingId === code.id} onClick={() => void makeDefault(code)}>设为默认</button> : null}
                {code.isEnabled
                  ? <button type="button" className="button button--danger" disabled={pendingId === code.id} onClick={() => setConfirmTarget(code)}>停用</button>
                  : <button type="button" className="button button--outline" disabled={pendingId === code.id} onClick={() => void toggleEnabled(code, true)}>启用</button>}
              </div>
            </li>
          ))}
        </ul>
      )}

      <form className="pay-code-form" onSubmit={submit}>
        <h2>新增收款方式</h2>
        <div className="pay-code-form__row">
          <label className="field"><span>名称</span><input value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} placeholder="例如：微信收款 / 支付宝 / 现金" /></label>
          <label className="field"><span>类型</span>
            <select value={form.kind} onChange={(event) => setForm({ ...form, kind: event.target.value as "qr" | "cash" })}>
              <option value="qr">扫码收款码</option>
              <option value="cash">现金收款</option>
            </select>
          </label>
          <label className="field"><span>币种 <small>留空跟随活动</small></span>
            <select value={form.currency} onChange={(event) => setForm({ ...form, currency: event.target.value })}>
              <option value="">跟随活动币种</option>
              {CURRENCY_OPTIONS.map((currency) => <option key={currency} value={currency}>{currency}</option>)}
            </select>
          </label>
        </div>
        {form.kind === "qr" ? (
          <label className="field"><span>收款码图片</span>
            <input ref={fileRef} type="file" accept="image/jpeg,image/png,image/webp" onChange={selectImage} />
            {preview ? <Image className="pay-code-form__preview" src={preview} alt="收款码预览" width={120} height={120} unoptimized /> : null}
            {imageError ? <span className="field__error" role="alert">{imageError}</span> : null}
          </label>
        ) : null}
        <label className="field field--checkbox">
          <input type="checkbox" checked={form.isDefault} onChange={(event) => setForm({ ...form, isDefault: event.target.checked })} />
          <span>设为默认收款方式</span>
        </label>
        <button className="button button--dark" type="submit" disabled={adding}>{adding ? "正在保存…" : "保存收款方式"}</button>
      </form>

      <ConfirmDialog
        open={confirmTarget !== null}
        title="停用收款方式"
        description={`停用后「${confirmTarget?.name ?? ""}」不再出现在收银台，可随时重新启用；这不是删除。`}
        confirmLabel="停用"
        danger
        pending={pendingId !== ""}
        onConfirm={() => confirmTarget && void toggleEnabled(confirmTarget, false)}
        onCancel={() => setConfirmTarget(null)}
      />
    </main>
  );
}
