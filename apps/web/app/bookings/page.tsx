"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api, Business, CatalogItem, readable } from "./api";
import Workspace from "./workspace";

export default function BookingsPage() {
 const [user, setUser] = useState<{ id: number; username: string } | null>(null);
 const [businesses, setBusinesses] = useState<Business[]>([]);
 const [selected, setSelected] = useState("");
 const [cursor, setCursor] = useState("");
 const [message, setMessage] = useState("Loading your workspaces...");
 const [busy, setBusy] = useState(false);
 const [name, setName] = useState("My booking demo");
 const [zone, setZone] = useState("Asia/Shanghai");
 const [closure, setClosure] = useState("");
 useEffect(() => {
  const controller = new AbortController();
  Promise.all([api<{ user: { id: number; username: string } }>("/api/v1/auth/me", { signal: controller.signal }), api<Business[]>("/api/v1/businesses?limit=100", { signal: controller.signal })])
   .then(([account, list]) => { setUser(account.data.user); setBusinesses(list.data); setCursor(list.cursor); setSelected(list.data[0]?.id ?? ""); setZone(Intl.DateTimeFormat().resolvedOptions().timeZone); setMessage(""); })
   .catch(error => { if (!controller.signal.aborted) setMessage(readable(error)); });
  return () => controller.abort();
 }, []);
 async function moreBusinesses() {
  setBusy(true); try { const next = await api<Business[]>(`/api/v1/businesses?limit=100&cursor=${encodeURIComponent(cursor)}`); setBusinesses(previous => [...previous, ...next.data]); setCursor(next.cursor); } catch (error) { setMessage(readable(error)); } finally { setBusy(false); }
 }
 async function createDemo(event: React.FormEvent<HTMLFormElement>) {
  event.preventDefault(); setBusy(true); setMessage("Creating your demo workspace...");
  let saved: Business | undefined;
  try {
   new Intl.DateTimeFormat("en", { timeZone: zone }).format();
   const post = async <T,>(path: string, body: object) => (await api<T>(path, { method: "POST", body: JSON.stringify(body) })).data;
   const put = async (path: string, body?: object) => api(path, { method: "PUT", ...(body ? { body: JSON.stringify(body) } : {}) });
   saved = await post<Business>("/api/v1/businesses", { name, timezone: zone });
   const root = `/api/v1/businesses/${saved.id}/catalog`;
   const location = await post<CatalogItem>(`${root}/locations`, { name: "Demo location", timezone: zone });
   const service = await post<CatalogItem>(`${root}/services`, { name: "30-minute appointment", duration_minutes: 30 });
   const staff = await post<CatalogItem>(`${root}/staff`, { name: "Demo specialist" });
   const resource = await post<CatalogItem>(`${root}/resources`, { name: "Demo room", location_id: location.id, capacity: 2 });
   for (const path of [`services/${service.id}/locations/${location.id}`, `services/${service.id}/staff/${staff.id}`, `services/${service.id}/resources/${resource.id}`, `staff/${staff.id}/locations/${location.id}`]) await put(`${root}/${path}`);
   await put(`${root}/locations/${location.id}/schedule`, { weekly_hours: Array.from({ length: 7 }, (_, weekday) => ({ weekday, start_minute: 540, end_minute: 1020 })), closed_dates: closure ? [closure] : [] });
   await put(`${root}/services/${service.id}/scheduling-rules`, { buffer_before_minutes: 15, buffer_after_minutes: 15, minimum_lead_minutes: 30 });
   const start = new Date(); start.setUTCHours(0, 0, 0, 0);
   const end = new Date(start.getTime() + 7 * 86400000);
   const windows = { windows: [{ starts_at: start.toISOString(), ends_at: end.toISOString() }] };
   await put(`${root}/staff/${staff.id}/availability-windows`, windows);
   await put(`${root}/resources/${resource.id}/availability-windows`, windows);
   setBusinesses(previous => [...previous, saved!]); setSelected(saved.id); setMessage("Demo ready. Choose a date and an available time below.");
  } catch (error) {
   if (saved) { setBusinesses(previous => [...previous, saved!]); setSelected(saved.id); setMessage("Setup did not finish. Your workspace was saved; create a fresh demo to try again."); }
   else setMessage(error instanceof RangeError ? "Enter a valid timezone, for example Asia/Shanghai." : readable(error));
  } finally { setBusy(false); }
 }
 const business = businesses.find(item => item.id === selected);
 return <main className="min-h-screen bg-slate-50 px-4 py-8 text-slate-900 sm:px-8">
  <div className="mx-auto max-w-6xl">
   <nav className="flex flex-wrap justify-between gap-3 text-sm"><Link href="/" className="font-semibold text-teal-800">Booking Platform</Link><div className="flex gap-5"><Link href="/account" className="underline">Your account</Link><Link href={user ? "/logout" : "/login"} className="underline">{user ? "Sign out" : "Sign in"}</Link></div></nav>
   <header className="my-8"><p className="text-sm font-semibold uppercase tracking-widest text-teal-700">Plan. Book. Adjust.</p><h1 className="mt-2 text-4xl font-semibold tracking-tight">Booking workspace</h1><p className="mt-3 max-w-2xl text-slate-600">Try a booking, move it to another time, or cancel it. Start with a demo workspace to explore the full flow.</p></header>
   {message && <p role="status" className="mb-5 rounded-xl border border-teal-200 bg-teal-50 p-4">{message}</p>}
   {!user ? <p><Link href="/login" className="font-semibold text-teal-800 underline">Sign in</Link> to manage your workspaces.</p> : <>
    <section aria-label="Workspace selection" className="mb-6 rounded-2xl border border-slate-200 bg-white p-5 shadow-sm">
     <div className="flex flex-wrap items-end gap-4"><label className="min-w-60 flex-1 text-sm font-medium">Your workspace<select aria-label="Your workspace" value={selected} disabled={busy} onChange={event => setSelected(event.target.value)} className="mt-2 block w-full rounded-lg border border-slate-300 p-3"><option value="">Choose a workspace</option>{businesses.map(item => <option key={item.id} value={item.id}>{item.name}</option>)}</select></label>{cursor && <button onClick={moreBusinesses} disabled={busy} className="rounded-lg border px-4 py-3">Load more workspaces</button>}</div>
     <details className="mt-5" open={businesses.length === 0}><summary className="cursor-pointer font-semibold text-teal-800">Create a demo workspace</summary>
      <p className="my-3 text-sm text-slate-600">A new workspace with a location, a specialist and a room for two. Appointments last 30 minutes, with 15-minute buffers and 30-minute notice. Open daily 09:00 to 17:00; availability lasts seven days. This creates saved test data.</p>
      <form onSubmit={createDemo}><fieldset disabled={busy} className="grid gap-4 sm:grid-cols-3">
       <label className="text-sm">Workspace name<input required maxLength={100} value={name} onChange={event => setName(event.target.value)} className="mt-1 block w-full rounded-lg border p-3" /></label>
       <label className="text-sm">Location timezone<input required maxLength={100} value={zone} onChange={event => setZone(event.target.value)} className="mt-1 block w-full rounded-lg border p-3" /></label>
       <label className="text-sm">Closed date (optional)<input type="date" value={closure} onChange={event => setClosure(event.target.value)} className="mt-1 block w-full rounded-lg border p-3" /></label>
       <button className="rounded-lg bg-teal-800 px-5 py-3 font-semibold text-white disabled:opacity-50">{busy ? "Working..." : "Create demo workspace"}</button>
      </fieldset></form>
     </details>
    </section>
    {business && <Workspace key={business.id} business={business} actor={user.id} />}
   </>}
  </div>
 </main>;
}
