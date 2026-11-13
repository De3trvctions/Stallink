"use client";

import { useEffect, useState } from "react";
import Image from "next/image";
import { useRouter } from "next/navigation";
import { browserJSON } from "@/lib/api";

type PayCode = { id: string; name: string; kind: "qr" | "cash"; isDefault: boolean; isEnabled: boolean; imageURL?: string | null };

export default function PayCodesPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [isDefault, setIsDefault] = useState(true);
  const [codes, setCodes] = useState<PayCode[]>([]);
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [cashPending, setCashPending] = useState(false);
  const [previewCode, setPreviewCode] = useState<PayCode | null>(null);

  const cashCode = codes.find((code) => code.kind === "cash");
  const qrCodes = codes.filter((code) => code.kind !== "cash");
  const cashEnabled = Boolean(cashCode?.isEnabled);

  function load() {
    browserJSON("/api/admin/pay-codes").then((res) => {
      if (res.status === 401) {
        router.replace("/login");
        return;
      }
      if (res.status === 403) {
        router.replace("/counter");
        return;
      }
      setCodes(Array.isArray(res.data) ? res.data as PayCode[] : []);
    });
  }
  useEffect(load, [router]);

  async function toggleCash() {
    setMessage("");
    setCashPending(true);
    try {
      const res = cashCode
        ? await browserJSON(`/api/admin/pay-codes/${cashCode.id}`, "PATCH", { name: cashCode.name, isEnabled: !cashCode.isEnabled, isDefault: cashCode.isDefault })
        : await browserJSON("/api/admin/pay-codes/cash", "POST", { name: "现金", isDefault: false });
      if (!res.ok) {
        setMessage("现金收款方式没有更新成功，请重试。");
        return;
      }
      load();
    } catch {
      setMessage("无法连接服务，请检查网络后重试。");
    } finally {
      setCashPending(false);
    }
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setMessage("");
    if (!file) {
      setMessage("请先选择收款码图片。");
      return;
    }
    const body = new FormData();
    body.set("name", name);
    body.set("isDefault", String(isDefault));
    body.set("image", file);
    setPending(true);
    try {
      const res = await fetch("/api/admin/pay-codes", { method: "POST", body, credentials: "include" });
      if (!res.ok) {
        setMessage("收款码没有上传，请检查图片格式后重试。");
        return;
      }
      setName("");
      setFile(null);
      setMessage("收款码已添加。");
      load();
    } catch {
      setMessage("无法连接服务，请检查网络后重试。");
    } finally {
      setPending(false);
    }
  }

  return (
    <main>
      <p className="eyebrow">现场付款</p>
      <h1>收款方式</h1>
      <p>二维码用于扫码付款；现金是独立收款选项，不需要收款码，开启后收银台会直接提示收取现金。</p>

      <div className="pay-cash-toggle">
        <div>
          <strong>现金收款</strong>
          <span>作为一个收款选项出现在收银台，不显示二维码。</span>
        </div>
        <button type="button" role="switch" aria-checked={cashEnabled} data-active={cashEnabled} disabled={cashPending} onClick={toggleCash}>
          {cashPending ? "正在更新…" : cashEnabled ? "已启用" : "启用现金"}
        </button>
      </div>

      <div className="workspace-list-heading"><h2>二维码收款</h2><span>{qrCodes.length} 种</span></div>
      {qrCodes.length === 0 ? (
        <div className="empty-state empty-state--compact"><span className="empty-state__mark" aria-hidden="true">✳</span><p>还没有二维码收款方式。添加后，销售即可在收银时选择。</p></div>
      ) : (
        <ul className="pay-code-list">
          {qrCodes.map((code) => (
            <li key={code.id} className="pay-code-row">
              <div>
                {code.imageURL ? (
                  <button type="button" className="pay-code-row__thumb" aria-label={`预览 ${code.name} 收款码`} onClick={() => setPreviewCode(code)}>
                    <Image src={code.imageURL} alt="" width={56} height={56} unoptimized />
                  </button>
                ) : null}
                <strong>{code.name}</strong><span className={code.isEnabled ? "account-status" : "account-status account-status--disabled"}><i aria-hidden="true" />{code.isDefault ? "默认" : code.isEnabled ? "已启用" : "已停用"}</span>
              </div>
              <button type="button" className="button button--outline" onClick={async () => {
                const res = await browserJSON(`/api/admin/pay-codes/${code.id}`, "PATCH", { name: code.name, isEnabled: !code.isEnabled, isDefault: code.isDefault && code.isEnabled });
                if (res.ok) load();
                else setMessage("更新失败，请重试。");
              }}>{code.isEnabled ? "停用" : "启用"}</button>
            </li>
          ))}
        </ul>
      )}

      <div className="workspace-list-heading"><h2>添加二维码</h2><span>PNG、JPG 或 WebP</span></div>
      <form className="pay-code-form" onSubmit={submit}>
        <label className="field" htmlFor="pay-code-name">
          <span>显示名称</span>
          <input id="pay-code-name" required value={name} onChange={(event) => setName(event.target.value)} placeholder="例如：DuitNow 或 Touch ’n Go" />
        </label>
        <label className="field" htmlFor="pay-code-image">
          <span>收款码图片</span>
          <input id="pay-code-image" type="file" accept="image/png,image/jpeg,image/webp" onChange={(event) => setFile(event.target.files?.[0] ?? null)} />
        </label>
        {file ? <p className="file-selection">已选择：{file.name}</p> : null}
        <label className="check-field"><input type="checkbox" checked={isDefault} onChange={(event) => setIsDefault(event.target.checked)} /><span>设为默认收款方式</span></label>
        {message ? <p className={`form-message${message.includes("没有") || message.includes("无法") || message.includes("请先") ? " form-message--error" : " form-message--success"}`} role="status">{message}</p> : null}
        <button className="button button--dark" type="submit" disabled={pending}>{pending ? "正在添加…" : "添加收款码"}</button>
      </form>

      {previewCode ? (
        <div className="pay-dialog" role="dialog" aria-modal="true" aria-labelledby="pay-preview-title" onClick={() => setPreviewCode(null)}>
          <div className="pay-dialog__panel" onClick={(event) => event.stopPropagation()}>
            <header className="pay-dialog__head">
              <div><p className="eyebrow">收款码预览</p><h2 id="pay-preview-title">{previewCode.name}</h2></div>
              <button type="button" className="pay-dialog__close" aria-label="关闭预览" onClick={() => setPreviewCode(null)}>×</button>
            </header>
            <div className="pay-dialog__qr">
              {previewCode.imageURL ? <Image src={previewCode.imageURL} alt={`${previewCode.name} 收款码`} width={560} height={560} unoptimized /> : null}
              <span>{previewCode.isDefault ? "默认收款方式" : previewCode.isEnabled ? "已启用" : "已停用"}</span>
            </div>
          </div>
        </div>
      ) : null}
    </main>
  );
}
