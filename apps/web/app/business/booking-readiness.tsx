"use client";
import { useEffect, useState } from "react";
import { api, catalogAll, CatalogItem, readable } from "../bookings/api";
import { button, input } from "./styles";
type Item = {
  kind: "locations" | "services" | "staff" | "resources";
  id: string;
  name: string;
  detail: string;
  needs_attention: boolean;
};
export default function BookingReadiness({
  businessID,
  onConfigure,
}: {
  businessID: string;
  onConfigure: (kind: Item["kind"], id: string) => void;
}) {
  const [services, setServices] = useState<CatalogItem[]>([]),
    [selected, setSelected] = useState(""),
    [items, setItems] = useState<Item[]>([]),
    [message, setMessage] = useState(""),
    [version, setVersion] = useState(0);
  useEffect(() => {
    let active = true;
    catalogAll(`/api/v1/businesses/${businessID}/catalog/services`)
      .then((rows) => {
        if (active) {
          setServices(rows);
          setSelected((old) => old || rows[0]?.id || "");
        }
      })
      .catch((e) => {
        if (active) setMessage(readable(e));
      });
    return () => {
      active = false;
    };
  }, [businessID, version]);
  useEffect(() => {
    if (!selected) return;
    const c = new AbortController();
    api<Item[]>(
      `/api/v1/businesses/${businessID}/catalog/services/${selected}/booking-readiness`,
      { signal: c.signal },
    )
      .then((r) => {
        setItems(r.data);
        setMessage("");
      })
      .catch((e) => {
        if (!c.signal.aborted) {
          setItems([]);
          setMessage(readable(e));
        }
      });
    return () => c.abort();
  }, [businessID, selected, version]);
  return (
    <section
      aria-label="Booking setup checklist"
      className="my-5 rounded-xl border border-teal-200 bg-teal-50 p-4"
    >
      <h3 className="text-lg font-semibold">Why customers can’t book</h3>
      <p className="my-2 text-sm">
        Set location opening hours and service rules. Staff and resources follow those hours. Unavailable dates have no bookable times; fully booked dates have no capacity left.</p>
      <label>
        Check a service
        <select
          className={input}
          value={selected}
          onChange={(e) => {
            if (e.target.value !== selected) {
              setSelected(e.target.value);
              setItems([]);
            }
          }}
        >
          <option value="">Choose a service</option>
          {services.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </select>
      </label>
      <button
        type="button"
        className={`${button} mt-3`}
        onClick={() => setVersion((v) => v + 1)}
      >
        Refresh booking setup
      </button>
      {message && <p role="status">{message}</p>}
      <ul className="mt-3 divide-y">
        {items.map((item, i) => (
          <li key={`${item.kind}:${item.id}:${i}`} className="py-3">
            <p className="font-medium">
              {item.name} · {item.needs_attention ? "Needs attention" : "Saved"}
            </p>
            <p className="my-1 text-sm">{item.detail}</p>
            <button
              type="button"
              className="text-sm text-teal-900 underline"
              onClick={() => onConfigure(item.kind, item.id)}
            >
              Edit {item.name} scheduling
            </button>
          </li>
        ))}
      </ul>
      <p className="mt-2 text-xs">
        This checks saved setup, not availability for a specific date. At least
        one eligible person/resource combination must cover the full booking.
        Save changes, then refresh this checklist.
      </p>
    </section>
  );
}
