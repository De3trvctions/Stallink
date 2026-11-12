"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { browserJSON } from "@/lib/api";

type Sales = { id: string; loginName: string; enabled: boolean };

export default function SalesPage() {
  const router = useRouter();
  const [loginName, setLoginName] = useState("");
  const [password, setPassword] = useState("");
  const [people, setPeople] = useState<Sales[]>([]);
  const [resetTarget, setResetTarget] = useState("");
  const [resetPassword, setResetPassword] = useState("");
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);

  function load() {
    browserJSON("/api/admin/sales").then((res) => {
      if (res.status === 403) {
        router.replace("/counter");
        return;
      }
      setPeople(Array.isArray(res.data) ? res.data as Sales[] : []);
    });
  }
  useEffect(load, [router]);

  async function createSales(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    try {
      const res = await browserJSON("/api/admin/sales", "POST", { loginName, password });
      if (!res.ok) {
        setMessage(res.data.error === "login_taken" ? "这个账号已经存在。" : "账号创建失败，请检查登录名和至少 8 位的初始密码。");
        return;
      }
      setLoginName("");
      setPassword("");
      setMessage("销售账号已创建。");
      load();
    } catch {
      setMessage("无法连接服务，请检查网络后重试。");
    } finally {
      setPending(false);
    }
  }

  async function savePassword(event: React.FormEvent<HTMLFormElement>, person: Sales) {
    event.preventDefault();
    setMessage("");
    try {
      const res = await browserJSON(`/api/admin/sales/${person.id}/password`, "POST", { password: resetPassword });
      if (!res.ok) {
        setMessage("密码没有更新，请使用至少 8 位字符。");
        return;
      }
      setMessage(`${person.loginName} 的密码已更新。`);
      setResetTarget("");
      setResetPassword("");
    } catch {
      setMessage("无法连接服务，请检查网络后重试。");
    }
  }

  return (
    <main>
      <p className="eyebrow">团队与权限</p>
      <h1>销售账号</h1>
      <p>为现场工作人员开设独立账号。每位销售只能处理自己店铺的现场订单。</p>
      <form className="sales-create-form" onSubmit={createSales}>
        <label className="field" htmlFor="sales-login">
          <span>登录名</span>
          <input id="sales-login" autoComplete="off" required value={loginName} onChange={(event) => setLoginName(event.target.value)} placeholder="例如：booth-helper" />
        </label>
        <label className="field" htmlFor="sales-password">
          <span>初始密码 <small>至少 8 位</small></span>
          <input id="sales-password" type="password" autoComplete="new-password" minLength={8} maxLength={72} required value={password} onChange={(event) => setPassword(event.target.value)} placeholder="设置临时密码" />
        </label>
        <button className="button button--dark" type="submit" disabled={pending}>{pending ? "正在创建…" : "创建销售账号"}</button>
      </form>
      {message ? <p className={`form-message${message.includes("失败") || message.includes("没有") || message.includes("无法") ? " form-message--error" : " form-message--success"}`} role="status">{message}</p> : null}

      <div className="workspace-list-heading"><h2>现有账号</h2><span>{people.length} 个</span></div>
      {people.length === 0 ? (
        <div className="empty-state empty-state--compact"><span className="empty-state__mark" aria-hidden="true">✳</span><p>还没有销售账号。</p></div>
      ) : (
        <ul className="sales-list">
          {people.map((person) => (
            <li key={person.id} className="sales-row">
              <div className="sales-row__identity">
                <span className="sales-row__avatar" aria-hidden="true">{person.loginName.slice(0, 1).toUpperCase()}</span>
                <div><strong>{person.loginName}</strong><span className={person.enabled ? "account-status" : "account-status account-status--disabled"}><i aria-hidden="true" />{person.enabled ? "已启用" : "已停用"}</span></div>
              </div>
              <div className="sales-row__actions">
                <button type="button" className="button button--outline" onClick={async () => { await browserJSON(`/api/admin/sales/${person.id}/${person.enabled ? "disable" : "enable"}`, "POST", {}); load(); }}>{person.enabled ? "停用" : "启用"}</button>
                <button type="button" className="text-link" onClick={() => { setMessage(""); setResetPassword(""); setResetTarget(resetTarget === person.id ? "" : person.id); }}>{resetTarget === person.id ? "取消" : "重置密码"}</button>
              </div>
              {resetTarget === person.id ? (
                <form className="sales-reset" onSubmit={(event) => savePassword(event, person)}>
                  <label className="field" htmlFor={`reset-${person.id}`}>
                    <span>为 {person.loginName} 设置新密码</span>
                    <input id={`reset-${person.id}`} type="password" autoComplete="new-password" minLength={8} maxLength={72} required value={resetPassword} onChange={(event) => setResetPassword(event.target.value)} placeholder="至少 8 位" />
                  </label>
                  <button className="button button--dark" type="submit">保存新密码</button>
                </form>
              ) : null}
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
