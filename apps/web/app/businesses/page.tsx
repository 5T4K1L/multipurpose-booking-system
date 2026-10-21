"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api, readable } from "../bookings/api";
type Summary = { id: string; name: string; timezone: string; tagline: string };
export default function BusinessDirectory() {
  const [query, setQuery] = useState("");
  const [search, setSearch] = useState("");
  const [version, setVersion] = useState(0);
  return (
    <main className="min-h-screen bg-slate-50 px-4 py-8 text-slate-900 sm:px-8">
      <div className="mx-auto max-w-4xl">
        <p className="eyebrow mt-5">Explore your next booking</p>
        <h1 className="mt-4 text-4xl font-semibold sm:text-5xl">Find a business</h1>
        <p className="my-3">
          Discover services, explore available times, and find what fits your day.
        </p>
        <form
          className="ui-panel my-8 flex flex-wrap items-end gap-3 p-5"
          onSubmit={(event) => {
            event.preventDefault();
            setSearch(query.trim());
            setVersion((v) => v + 1);
          }}
        >
          <label className="w-full min-w-0 sm:w-auto sm:flex-1">
            Business name or introduction
            <input
              className="mt-2 w-full rounded-lg border bg-white p-3"
              value={query}
              maxLength={100}
              onChange={(e) => setQuery(e.target.value)}
            />
          </label>
          <button className="rounded-lg bg-teal-800 px-5 py-3 font-semibold text-white">
            Search businesses
          </button>
        </form>
        <Results key={`${search}:${version}`} search={search} />
      </div>
    </main>
  );
}
function Results({ search }: { search: string }) {
  const [items, setItems] = useState<Summary[]>([]);
  const [cursor, setCursor] = useState("");
  const [busy, setBusy] = useState(true);
  const [message, setMessage] = useState("Loading businesses...");
  const base = `/api/v1/public/businesses?limit=20&q=${encodeURIComponent(search)}`;
  useEffect(() => {
    const controller = new AbortController();
    api<Summary[]>(base, { credentials: "omit", signal: controller.signal })
      .then((result) => {
        setItems(result.data);
        setCursor(result.cursor);
        setMessage(
          result.data.length
            ? ""
            : "No published businesses match your search.",
        );
      })
      .catch((error) => {
        if (!controller.signal.aborted) setMessage(readable(error));
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy(false);
      });
    return () => controller.abort();
  }, [base]);
  async function more() {
    setBusy(true);
    try {
      const result = await api<Summary[]>(
        `${base}&cursor=${encodeURIComponent(cursor)}`,
        { credentials: "omit" },
      );
      setItems((old) => [
        ...old,
        ...result.data.filter(
          (item) => !old.some((existing) => existing.id === item.id),
        ),
      ]);
      setCursor(result.cursor);
      setMessage("");
    } catch {
      setMessage(
        "Could not load more businesses. Retry or search again to refresh the list.",
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <section aria-label="Business search results">
      {message && (
        <p role="status" className="ui-panel my-4 p-5 text-sm">
          {message}
        </p>
      )}
      {busy && !items.length && <div className="grid gap-4 sm:grid-cols-2" aria-hidden="true">{[0,1].map(i=><div className="ui-panel space-y-5 p-6" key={i}><div className="skeleton w-12 !h-12"/><div className="skeleton w-3/4"/><div className="skeleton w-1/2"/></div>)}</div>}
      <ul className="grid gap-4 sm:grid-cols-2">
        {items.map((item) => (
          <li
            key={item.id}
            className="directory-card ui-panel min-w-0 break-words p-6"
          >
            <div className="mb-5 flex h-12 w-12 items-center justify-center rounded-xl bg-teal-50 text-lg font-semibold text-teal-900" aria-hidden="true">{item.name.slice(0,1).toUpperCase()}</div>
            <h2 className="text-xl font-semibold">
              <Link
                className="text-teal-800 hover:underline"
                href={`/businesses/${item.id}`}
              >
                {item.name}
              </Link>
            </h2>
            <p className="my-3">{item.tagline}</p>
            <p className="text-sm text-slate-600">{item.timezone.replaceAll("_", " ")}</p>
          </li>
        ))}
      </ul>
      {cursor && (
        <button
          className="mt-5 rounded-lg border px-4 py-3 disabled:opacity-50"
          disabled={busy}
          onClick={() => void more()}
        >
          Load more businesses
        </button>
      )}
    </section>
  );
}
