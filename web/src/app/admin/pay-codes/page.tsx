"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { browserJSON } from "@/lib/api";

type PayCode = { id: string; name: string; isDefault: boolean; isEnabled: boolean };

export default function PayCodesPage() {
  const router = useRouter();
  const [name, setName] = useState("");
  const [file, setFile] = useState<File | null>(null);
  const [isDefault, setIsDefault] = useState(true);
  const [codes, setCodes] = useState<PayCode[]>([]);
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

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
      setMessage("收款方式已添加。");
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
      <p>上传顾客现场付款时使用的收款码。你可以设置一个默认方式。</p>
      <form className="pay-code-form" onSubmit={submit}>
        <label className="field" htmlFor="pay-code-name">
          <span>显示名称</span>
          <input id="pay-code-name" required value={name} onChange={(event) => setName(event.target.value)} placeholder="例如：DuitNow 或 Touch ’n Go" />
        </label>
        <label className="field" htmlFor="pay-code-image">
          <span>收款码图片 <small>PNG、JPG 或 WebP</small></span>
          <input id="pay-code-image" type="file" accept="image/png,image/jpeg,image/webp" onChange={(event) => setFile(event.target.files?.[0] ?? null)} />
        </label>
        {file ? <p className="file-selection">已选择：{file.name}</p> : null}
        <label className="check-field"><input type="checkbox" checked={isDefault} onChange={(event) => setIsDefault(event.target.checked)} /><span>设为默认收款方式</span></label>
        {message ? <p className={`form-message${message.includes("没有") || message.includes("无法") || message.includes("请先") ? " form-message--error" : " form-message--success"}`} role="status">{message}</p> : null}
        <button className="button button--dark" type="submit" disabled={pending}>{pending ? "正在上传…" : "添加收款方式"}</button>
      </form>

      <div className="workspace-list-heading"><h2>已添加</h2><span>{codes.length} 种</span></div>
      {codes.length === 0 ? (
        <div className="empty-state empty-state--compact"><span className="empty-state__mark" aria-hidden="true">✳</span><p>还没有收款方式。添加后，销售即可在收银时选择。</p></div>
      ) : (
        <ul className="pay-code-list">
          {codes.map((code) => (
            <li key={code.id} className="pay-code-row">
              <div><strong>{code.name}</strong><span className={code.isEnabled ? "account-status" : "account-status account-status--disabled"}><i aria-hidden="true" />{code.isDefault ? "默认" : code.isEnabled ? "已启用" : "已停用"}</span></div>
              <button type="button" className="button button--outline" onClick={async () => {
                const res = await browserJSON(`/api/admin/pay-codes/${code.id}`, "PATCH", { name: code.name, isEnabled: !code.isEnabled, isDefault: code.isDefault && code.isEnabled });
                if (res.ok) load();
                else setMessage("更新失败，请重试。");
              }}>{code.isEnabled ? "停用" : "启用"}</button>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
