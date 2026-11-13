"use client";

import { useEffect, useState } from "react";
import { browserJSON } from "@/lib/api";
import SelectMenu from "@/components/select-menu";

const empty = { name: "", startsOn: "", endsOn: "", city: "", venue: "", boothNumber: "", currency: "MYR", status: "scheduled" };
const statusLabel: Record<string, string> = { scheduled: "即将开始", live: "进行中", ended: "已结束" };
type EventSummary = { id: string; name: string; status: string; boothNumber: string; startsOn?: string; endsOn?: string; city?: string; venue?: string; currency?: string };

function dateRange(event: EventSummary) {
  if (!event.startsOn) return "";
  const short = (value: string) => value.slice(5).replace("-", ".");
  return event.endsOn && event.endsOn !== event.startsOn ? `${short(event.startsOn)}–${short(event.endsOn)}` : short(event.startsOn);
}

export default function EventsPage() {
  const [form, setForm] = useState(empty);
  const [events, setEvents] = useState<EventSummary[]>([]);
  const [message, setMessage] = useState("");
  const [pending, setPending] = useState(false);
  const [editingId, setEditingId] = useState("");
  const [featuredId, setFeaturedId] = useState("");

  function load() {
    browserJSON("/api/staff/events").then((res) => setEvents(Array.isArray(res.data) ? res.data as EventSummary[] : []));
  }

  useEffect(() => {
    load();
    let active = true;
    void browserJSON("/api/auth/me")
      .then((me) => {
        const slug = (me.data as { shopSlug?: string } | null)?.shopSlug;
        if (!slug) return null;
        return fetch(`/api/public/shops/${encodeURIComponent(slug)}`);
      })
      .then(async (response) => {
        if (!response?.ok) return null;
        return response.json() as Promise<{ featuredEventId?: string }>;
      })
      .then((data) => {
        if (active && data) setFeaturedId(typeof data.featuredEventId === "string" ? data.featuredEventId : "");
      })
      .catch(() => {
        // The public storefront can still work if the featured event is unavailable.
      });
    return () => { active = false; };
  }, []);

  async function toggleFeatured(item: EventSummary) {
    const next = featuredId === item.id ? "" : item.id;
    setPending(true);
    setMessage("");
    try {
      const res = await browserJSON("/api/admin/shop/featured-event", "PATCH", { eventId: next });
      if (!res.ok) {
        setMessage("主推活动没有保存，请稍后重试。");
        return;
      }
      setFeaturedId(next);
      setMessage(next ? "已把这场活动设为店铺主推，顾客会先看到这一场的商品。" : "已取消主推，店铺按进行中的活动顺序展示。");
    } catch {
      setMessage("无法连接服务，请稍后重试。");
    } finally {
      setPending(false);
    }
  }

  function startEdit(item: EventSummary) {
    setEditingId(item.id);
    setForm({
      name: item.name,
      startsOn: item.startsOn ?? "",
      endsOn: item.endsOn ?? "",
      city: item.city ?? "",
      venue: item.venue ?? "",
      boothNumber: item.boothNumber ?? "",
      currency: item.currency ?? "MYR",
      status: item.status ?? "scheduled",
    });
    setMessage("");
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  function cancelEdit() {
    setEditingId("");
    setForm(empty);
    setMessage("");
  }

  async function submit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setPending(true);
    setMessage("");
    try {
      const res = editingId
        ? await browserJSON(`/api/staff/events/${editingId}`, "PATCH", form)
        : await browserJSON("/api/staff/events", "POST", form);
      if (!res.ok) {
        setMessage("活动没有保存，请检查活动名称和日期后重试。");
        return;
      }
      setMessage(editingId ? "活动已更新，现场收银会立即使用新的场次信息。" : "活动已创建，可以继续为它上架商品。");
      setEditingId("");
      setForm(empty);
      load();
    } catch {
      setMessage("无法连接服务，请检查网络后重试。");
    } finally {
      setPending(false);
    }
  }

  return (
    <main>
      <p className="eyebrow">{editingId ? "活动与场次 · 编辑" : "活动与场次"}</p>
      <h1>{editingId ? "编辑活动" : "活动管理"}</h1>
      <p>活动连接商品、库存和摊位信息。商品可以暂时为空，之后再上架。</p>
      <form className="event-create-form" onSubmit={submit}>
        <label className="field" htmlFor="event-name"><span>活动名称</span><input id="event-name" required value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} placeholder="漫展或活动名称" /></label>
        <div className="goods-form__dates">
          <label className="field" htmlFor="event-start"><span>开始日期</span><input id="event-start" type="date" required value={form.startsOn} onChange={(event) => setForm({ ...form, startsOn: event.target.value })} /></label>
          <label className="field" htmlFor="event-end"><span>结束日期</span><input id="event-end" type="date" required value={form.endsOn} onChange={(event) => setForm({ ...form, endsOn: event.target.value })} /></label>
        </div>
        <div className="goods-form__dates">
          <label className="field" htmlFor="event-city"><span>城市</span><input id="event-city" value={form.city} onChange={(event) => setForm({ ...form, city: event.target.value })} placeholder="城市" /></label>
          <label className="field" htmlFor="event-venue"><span>场馆</span><input id="event-venue" value={form.venue} onChange={(event) => setForm({ ...form, venue: event.target.value })} placeholder="场馆名称" /></label>
        </div>
        <div className="goods-form__dates">
          <label className="field" htmlFor="event-booth"><span>摊位号</span><input id="event-booth" value={form.boothNumber} onChange={(event) => setForm({ ...form, boothNumber: event.target.value })} placeholder="例如：A-12" /></label>
          <label className="field" htmlFor="event-currency"><span>结算币种</span><SelectMenu id="event-currency" value={form.currency} options={[{ value: "MYR", label: "MYR" }, { value: "SGD", label: "SGD" }, { value: "CNY", label: "CNY" }]} onChange={(currency) => setForm({ ...form, currency })} /></label>
        </div>
        <label className="field" htmlFor="event-status"><span>活动状态</span>
          <SelectMenu id="event-status" value={form.status} options={[{ value: "scheduled", label: "即将开始" }, { value: "live", label: "进行中" }, { value: "ended", label: "已结束" }]} onChange={(status) => setForm({ ...form, status })} />
        </label>
        {message ? <p className={`form-message${message.includes("没有") || message.includes("无法") || message.includes("检查") ? " form-message--error" : " form-message--success"}`} role="status">{message}</p> : null}
        <div className="event-form-actions">
          <button className="button button--dark" type="submit" disabled={pending}>{pending ? "正在保存…" : editingId ? "保存活动修改" : "创建活动"}</button>
          {editingId ? <button className="button button--outline" type="button" disabled={pending} onClick={cancelEdit}>取消编辑</button> : null}
        </div>
      </form>

      <div className="workspace-list-heading"><h2>已有活动</h2><span>{events.length} 场</span></div>
      {events.length === 0 ? (
        <div className="empty-state empty-state--compact"><span className="empty-state__mark" aria-hidden="true">✳</span><p>还没有活动。创建后就可以为活动安排商品和库存。</p></div>
      ) : (
        <ul className="workspace-rows">
          {events.map((event) => (
            <li key={event.id}>
              <div><strong>{event.name}</strong><span>摊位 {event.boothNumber || "待公布"}{dateRange(event) ? ` · ${dateRange(event)}` : ""}</span></div>
              <div className="workspace-row__meta">
                {featuredId === event.id ? <span className="event-featured">主推</span> : null}
                <span className={`event-status event-status--${event.status}`}><i aria-hidden="true" />{statusLabel[event.status] ?? event.status}</span>
                <button type="button" className="button button--outline event-feature-button" disabled={pending || event.status === "ended"} onClick={() => void toggleFeatured(event)}>{featuredId === event.id ? "取消主推" : "设为主推"}</button>
                <button type="button" className="button button--outline event-edit-button" onClick={() => startEdit(event)}>{editingId === event.id ? "正在编辑" : "编辑"}</button>
              </div>
            </li>
          ))}
        </ul>
      )}
    </main>
  );
}
