"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api, ApiError, Business, readable } from "../bookings/api";
import { DraftProvider } from "./drafts";
import Catalog from "./catalog";
import timezones from "./timezones.json";
import { button, input, panel } from "./styles";

export default function BusinessPage() {
  return (
    <DraftProvider>
      <BusinessWorkspace />
    </DraftProvider>
  );
}
function BusinessWorkspace() {
  const [businesses, setBusinesses] = useState<Business[]>([]);
  const [selected, setSelected] = useState("");
  const [signedIn, setSignedIn] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("Loading your business...");
  const [version, setVersion] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    Promise.all([
      api<{ user: { id: number } }>("/api/v1/auth/me", {
        signal: controller.signal,
      }),
      api<Business[]>("/api/v1/businesses?limit=100", {
        signal: controller.signal,
      }),
    ])
      .then(([, result]) => {
        setSignedIn(true);
        setBusinesses(result.data);
        setSelected((previous) =>
          result.data.some((b) => b.id === previous)
            ? previous
            : ((result.data.find(b => b.role === "BUSINESS_OWNER") ?? result.data[0])?.id ?? ""),
        );
        setMessage("");
      })
      .catch((error) => {
        if (!controller.signal.aborted) {
          setMessage(readable(error));
          if (error instanceof ApiError && [401, 403].includes(error.status)) {
            setSignedIn(false);
            setBusinesses([]);
            setSelected("");
          }
        }
      });
    return () => controller.abort();
  }, [version]);
  async function create(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const form = event.currentTarget;
    const values = new FormData(form);
    const timezone = String(values.get("timezone")).trim();
    try {
      new Intl.DateTimeFormat("en", { timeZone: timezone }).format();
    } catch {
      setMessage("Enter a valid timezone, such as Asia/Shanghai or UTC.");
      return;
    }
    setBusy(true);
    setMessage("");
    try {
      const { data } = await api<Business>("/api/v1/businesses", {
        method: "POST",
        body: JSON.stringify({
          name: String(values.get("name")).trim(),
          timezone,
          profile:{tagline:String(values.get("tagline")??""),description:String(values.get("description")??""),contact_email:String(values.get("contact_email")??""),contact_phone:String(values.get("contact_phone")??""),website:String(values.get("website")??"")},
        }),
      });
      setBusinesses((previous) => [...previous, data]);
      setSelected(data.id);
      window.dispatchEvent(new Event("booking-session-changed"));
      form.reset();
      setMessage("Business created. Add your locations and services below.");
    } catch (error) {
      setMessage(
        `${readable(error)} If the response was interrupted, refresh the business list before submitting again to avoid duplicates.`,
      );
    } finally {
      setBusy(false);
    }
  }
  const business = businesses.find((b) => b.id === selected);
  return (
    <main className="min-h-screen bg-slate-50 px-4 py-8 text-slate-900 sm:px-8">
      <div className="mx-auto max-w-6xl">
        
        <header className="my-8">
          <p className="text-sm font-semibold uppercase tracking-widest text-teal-700">
            Your business, organized
          </p>
          <h1 className="mt-2 text-4xl font-semibold">{signedIn && !businesses.length ? "Set up your business" : "Business settings"}</h1>
          <p className="mt-3 max-w-2xl text-slate-600">
            Set up your services, people and opening hours, then publish your
            business so customers can book.
          </p>
        </header>
        {message && (
          <p
            role="status"
            className="mb-5 rounded-xl border border-teal-200 bg-teal-50 p-4"
          >
            {message}
          </p>
        )}
        <button
          type="button"
          disabled={busy}
          onClick={() => setVersion((v) => v + 1)}
          className="workspace-text-action mb-3"
        >
          Refresh business
        </button>
        {!signedIn ? (
          <p>
            <Link href="/login" className="underline">
              Sign in
            </Link>{" "}
            to open your business workspace.
          </p>
        ) : (
          <>
            <section className={business?"mb-4":panel} aria-label="Business selection">
              {business && <p className="text-sm text-slate-600">Your workspace: <strong>{business.name}</strong>.</p>}
              {!businesses.length && <details className="mt-5" open>
                <summary className="cursor-pointer font-semibold text-teal-800">
                  Create a business
                </summary>
                <form onSubmit={create} className="mt-4">
                  <fieldset
                    disabled={busy}
                    className="grid gap-4 sm:grid-cols-2"
                  >
                    <label>
                      Business name
                      <input
                        name="name"
                        required
                        maxLength={100}
                        className={input}
                      />
                    </label>
                    <label>
                      Business timezone
                      <select
                        aria-label="Business timezone"
                        name="timezone"
                        required
                        defaultValue="Asia/Shanghai"
                        className={input}
                      >
                        {timezones.map((zone) => (
                          <option key={zone} value={zone}>
                            {zone.replaceAll("_", " ")}
                          </option>
                        ))}
                      </select>
                    </label>
                    <label className="sm:col-span-2">Short introduction<input name="tagline" maxLength={120} required className={input} placeholder="What do you offer your customers?"/></label>
                    <label className="sm:col-span-2">About your business<textarea name="description" maxLength={2000} className={input} rows={4}/></label>
                    <label>Business contact email<input name="contact_email" type="email" maxLength={254} required className={input}/></label>
                    <label>Business phone<input name="contact_phone" type="tel" maxLength={40} required className={input}/></label>
                    <label className="sm:col-span-2">Website (optional, https://)<input name="website" type="url" maxLength={500} className={input}/></label>
                    <p className="sm:col-span-2 text-sm text-slate-600">These details form your public profile. You will review and publish it after adding your services and opening hours. Upload a business photo under Profile &amp; publishing.</p>
                    <button className={button}>
                      {busy ? "Saving..." : "Create business"}
                    </button>
                  </fieldset>
                </form>
              </details>}
            </section>
            {business && (
              <Catalog
                key={`${business.id}:${version}`}
                business={business}
                onBusinessUpdated={(updated) =>
                  setBusinesses((previous) =>
                    previous.map((item) =>
                      item.id === updated.id ? updated : item,
                    ),
                  )
                }
              />
            )}
          </>
        )}
      </div>
    </main>
  );
}
