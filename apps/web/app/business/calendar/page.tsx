"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import {
  api,
  Booking,
  Business,
  dateInZone,
  readable,
} from "../../bookings/api";
import { button, input, panel } from "../styles";

type Entry = Booking & {
  service_name: string;
  location_name: string;
  staff_name: string;
  resource_name: string;
};
type Calendar = {
  from: string;
  to: string;
  timezone: string;
  counts: Record<string, number>;
  items: Entry[];
  next_cursor: string;
};
const statuses = [
  ["total", "All records"],
  ["confirmed", "Confirmed"],
  ["completed", "Completed"],
  ["cancelled", "Cancelled"],
  ["rescheduled", "Moved"],
];
function addDays(date: string, count: number) {
  const value = new Date(`${date}T12:00:00Z`);
  value.setUTCDate(value.getUTCDate() + count);
  return value.toISOString().slice(0, 10);
}
export default function CalendarPage() {
  const [businesses, setBusinesses] = useState<Business[]>([]);
  const [selected, setSelected] = useState("");
  const [message, setMessage] = useState("Loading businesses...");
  useEffect(() => {
    const controller = new AbortController();
    api<Business[]>("/api/v1/businesses?limit=100", {
      signal: controller.signal,
    })
      .then((result) => {
        setBusinesses(result.data);
        setSelected((result.data.find(b => b.role === "BUSINESS_OWNER") ?? result.data[0])?.id ?? "");
        setMessage(result.data.length ? "" : "No business memberships yet.");
      })
      .catch((error) => {
        if (!controller.signal.aborted) setMessage(readable(error));
      });
    return () => controller.abort();
  }, []);
  const business = businesses.find((item) => item.id === selected);
  return (
    <main className="min-h-screen bg-slate-50 px-4 py-8 text-slate-900 sm:px-8">
      <div className="mx-auto max-w-6xl">
        
        <h1 className="mt-8 text-3xl font-semibold">Booking calendar</h1>
        <p className="my-4 text-slate-600">
          Review your business bookings by date. Counts include every record
          starting in the selected period.
        </p>
        {message && (
          <p role="status" className="my-4">
            {message}
          </p>
        )}
        {business && <p className="mb-6 text-sm text-slate-600">Your business: <strong>{business.name}</strong></p>}

        {business && <CalendarView key={business.id} business={business} />}
        {!businesses.length && (
          <Link className="mt-4 inline-block underline" href="/login">
            Sign in
          </Link>
        )}
      </div>
    </main>
  );
}
function CalendarView({ business }: { business: Business }) {
  const [from, setFrom] = useState(() =>
    dateInZone(new Date(), business.timezone),
  );
  const [to, setTo] = useState(() =>
    addDays(dateInZone(new Date(), business.timezone), 6),
  );
  const [range, setRange] = useState({ from, to, version: 0 });
  const [data, setData] = useState<Calendar | null>(null);
  const [busy, setBusy] = useState(true);
  const [message, setMessage] = useState("Loading calendar...");
  const root = `/api/v1/businesses/${business.id}/bookings/calendar`;
  useEffect(() => {
    const controller = new AbortController();
    api<Calendar>(`${root}?from=${range.from}&to=${range.to}&limit=50`, {
      signal: controller.signal,
    })
      .then((result) => {
        setData(result.data);
        setMessage("");
      })
      .catch((error) => {
        if (!controller.signal.aborted) {
          setData(null);
          setMessage(readable(error));
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy(false);
      });
    return () => controller.abort();
  }, [root, range]);
  function load(start: string, end: string) {
    if (
      !start ||
      !end ||
      end < start ||
      (Date.parse(end) - Date.parse(start)) / 86400000 > 30
    ) {
      setMessage("Choose an ordered range of at most 31 days.");
      return;
    }
    setFrom(start);
    setTo(end);
    setData(null);
    setBusy(true);
    setMessage("Loading calendar...");
    setRange((previous) => ({
      from: start,
      to: end,
      version: previous.version + 1,
    }));
  }
  async function more() {
    if (!data?.next_cursor) return;
    setBusy(true);
    try {
      const result = await api<Calendar>(
        `${root}?from=${range.from}&to=${range.to}&limit=50&cursor=${encodeURIComponent(data.next_cursor)}`,
      );
      setData((previous) =>
        previous
          ? {
              ...result.data,
              items: [
                ...previous.items,
                ...result.data.items.filter(
                  (item) => !previous.items.some((old) => old.id === item.id),
                ),
              ],
            }
          : result.data,
      );
      setMessage("");
    } catch (error) {
      setMessage(readable(error));
    } finally {
      setBusy(false);
    }
  }
  const zone = data?.timezone ?? business.timezone;
  const format = (value: string) =>
    new Intl.DateTimeFormat("en", {
      timeZone: zone,
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "numeric",
      minute: "2-digit",
      timeZoneName: "short",
    }).format(new Date(value));
  const groups = new Map<string, Entry[]>();
  for (const item of data?.items ?? []) {
    const day = dateInZone(new Date(item.starts_at), zone);
    groups.set(day, [...(groups.get(day) ?? []), item]);
  }
  return (
    <div className="mt-6">
      <section className={panel}>
        <h2 className="text-xl font-semibold">Date range</h2>
        <p className="mt-2 text-sm">
          Dates and times use {zone}. Bookings belong to the day they start,
          even if they end the following day.
        </p>
        <form
          onSubmit={(e) => {
            e.preventDefault();
            load(from, to);
          }}
          className="mt-4"
        >
          <fieldset
            disabled={busy}
            className="grid min-w-0 items-end gap-4 sm:grid-cols-3"
          >
            <label>
              From date
              <input
                className={input}
                type="date"
                min="2000-01-01"
                max="2100-12-31"
                required
                value={from}
                onChange={(e) => setFrom(e.target.value)}
              />
            </label>
            <label>
              Through date
              <input
                className={input}
                type="date"
                min="2000-01-01"
                max="2100-12-31"
                required
                value={to}
                onChange={(e) => setTo(e.target.value)}
              />
            </label>
            <button className={button}>Apply dates</button>
          </fieldset>
        </form>
        <div className="mt-4 flex flex-wrap gap-3">
          <button
            disabled={busy}
            className="workspace-choice"
            onClick={() => {
              const today = dateInZone(new Date(), zone);
              load(today, today);
            }}
          >
            Today
          </button>
          <button
            disabled={busy}
            className="workspace-choice"
            onClick={() => {
              const today = dateInZone(new Date(), zone);
              load(today, addDays(today, 6));
            }}
          >
            Next 7 days
          </button>
          <button
            disabled={busy}
            className="workspace-choice"
            onClick={() => load(range.from, range.to)}
          >
            Refresh calendar
          </button>
        </div>
      </section>
      {message && (
        <p role="status" className="my-4 rounded-lg bg-teal-50 p-4">
          {message}
        </p>
      )}
      {data && (
        <>
          <section
            aria-label="Booking totals"
            className="mb-6 grid grid-cols-2 gap-3 sm:grid-cols-5"
          >
            {statuses.map(([key, label]) => (
              <div key={key} className="rounded-xl border bg-white p-4">
                <p className="text-sm text-slate-600">{label}</p>
                <p
                  className="mt-2 text-3xl font-semibold"
                  aria-label={`${label} count`}
                >
                  {data.counts[key]}
                </p>
              </div>
            ))}
          </section>
          <p className="mb-4 text-sm text-slate-600">
            Totals cover {data.from} through {data.to}, including all pages.
            Moved originals and their replacements are separate historical
            records. These are booking counts, not revenue or unique customers.
          </p>
          <section aria-label="Calendar agenda" className={panel}>
            <h2 className="text-xl font-semibold">Calendar agenda</h2>
            <p className="mt-2 text-sm text-slate-500">
              {data.items.length} records loaded. Ordered by start time. Use
              Refresh dashboard for current changes.
            </p>
            {!data.items.length && (
              <p className="my-8">No bookings start in this period.</p>
            )}
            {[...groups].map(([day, items]) => (
              <section key={day} className="mt-6">
                <h3 className="rounded-lg bg-slate-100 p-3 font-semibold">
                  {day}
                </h3>
                <ul className="divide-y">
                  {items.map((item) => (
                    <li key={item.id} className="py-4">
                      <div className="flex flex-wrap justify-between gap-3">
                        <p className="break-words font-semibold">
                          {item.service_name}
                        </p>
                        <span className="rounded-full bg-teal-50 px-3 py-1 text-sm">
                          {item.status === "rescheduled"
                            ? "Moved"
                            : item.status}
                        </span>
                      </div>
                      <p className="mt-2 text-sm">
                        {format(item.starts_at)} to {format(item.ends_at)}
                      </p>
                      <p className="mt-2 break-words text-sm text-slate-600">
                        {[item.location_name,item.staff_name,item.resource_name].filter(Boolean).join(" / ")}
                      </p>
                      <details className="mt-2 text-sm">
                        <summary className="cursor-pointer">
                          Booking details
                        </summary>
                        <p className="mt-2 break-all">Reference: {item.id}</p>
                        <p>
                          Reserved: {format(item.busy_starts_at)} to{" "}
                          {format(item.busy_ends_at)}
                        </p>
                      </details>
                    </li>
                  ))}
                </ul>
              </section>
            ))}
            {data.next_cursor && (
              <button
                disabled={busy}
                className={`${button} mt-4`}
                onClick={more}
              >
                Load more bookings
              </button>
            )}
          </section>
          <Link href="/business/bookings" className="underline">
            Manage bookings: reschedule, cancel or complete
          </Link>
        </>
      )}
    </div>
  );
}
