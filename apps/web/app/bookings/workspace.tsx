"use client";

import BookingOrganizer from "../components/booking-organizer";
import {formatBookingTime} from "./format-booking-time";
import { useEffect, useState } from "react";
import {
  api,
  Booking,
  Business,
  CatalogItem,
  catalogAll,
  dateInZone,
  mutate,
  readable,
  Slot,
} from "./api";

export default function Workspace({
  business,
  actor,
  management = false,
}: {
  business: Business;
  actor: number;
  management?: boolean;
}) {
  const [catalog, setCatalog] = useState<Record<string, CatalogItem[]>>({
    locations: [],
    services: [],
    staff: [],
    resources: [],
  });
  const [location, setLocation] = useState("");
  const [service, setService] = useState("");
  const [staff, setStaff] = useState("");
  const [resource, setResource] = useState("");
  const [date, setDate] = useState("");
  const [slots, setSlots] = useState<Slot[] | null>(null);
  const [selected, setSelected] = useState("");
  const [rows, setRows] = useState<Booking[]>([]);
  const [cursor, setCursor] = useState("");
  const [detailAnchor, setDetailAnchor] = useState("");
  const [detail, setDetail] = useState<Booking | null>(null);
  const [moving, setMoving] = useState<Booking | null>(null);
  const [cancelling, setCancelling] = useState<Booking | null>(null);
  const [deleting,setDeleting]=useState<string|null>(null);
  const [completing, setCompleting] = useState<Booking | null>(null);
  const [clock, setClock] = useState(() => Date.now());
  useEffect(() => {
    const timer = setInterval(() => setClock(Date.now()), 30000);
    return () => clearInterval(timer);
  }, []);
  const [busy, setBusy] = useState(true);
  const [message, setMessage] = useState("Loading workspace...");
  const root = `/api/v1/businesses/${business.id}`;
  const canWrite = ["BUSINESS_OWNER", "BUSINESS_ADMIN"].includes(business.role);
  const zone =
    management && !moving
      ? business.timezone
      : (catalog.locations.find((item) => item.id === location)?.timezone ??
        business.timezone);
  const format = (value: string, full = false) =>
    new Intl.DateTimeFormat("en", {
      timeZone: zone,
      ...(full
        ? {
            year: "numeric" as const,
            month: "short" as const,
            day: "numeric" as const,
          }
        : {}),
      hour: "numeric",
      minute: "2-digit",
      timeZoneName: "short",
    }).format(new Date(value));
  const name = (kind: string, id: string | null) =>
    catalog[kind].find((item) => item.id === id)?.name ??
    (id ? "Archived item" : "None selected");
  useEffect(() => {
    let active = true;
    Promise.all([
      Promise.all(
        ["locations", "services", "staff", "resources"].map(
          async (kind) =>
            [kind, await catalogAll(`${root}/catalog/${kind}`)] as const,
        ),
      ),
      api<Booking[]>(`${root}/bookings?limit=25`),
    ])
      .then(([items, bookings]) => {
        if (!active) return;
        const loaded = Object.fromEntries(items);
        setCatalog(loaded);
        setLocation(loaded.locations[0]?.id ?? "");
        setService(loaded.services[0]?.id ?? "");
        setStaff(loaded.staff[0]?.id ?? "");
        setResource(
          loaded.resources.find(
            (item) => item.location_id === loaded.locations[0]?.id,
          )?.id ?? "",
        );
        setRows(bookings.data);
        setCursor(bookings.cursor);
        setDate(
          dateInZone(
            new Date(Date.now() + 86400000),
            loaded.locations[0]?.timezone ?? business.timezone,
          ),
        );
        setMessage("");
      })
      .catch((error) => {
        if (active) setMessage(readable(error));
      })
      .finally(() => {
        if (active) setBusy(false);
      });
    return () => {
      active = false;
    };
  }, [root, business.timezone]);
  useEffect(() => {
    if (cancelling) document.getElementById("cancel-title")?.focus();
  }, [cancelling]);
  useEffect(() => {
    if (completing) document.getElementById("complete-title")?.focus();
  }, [completing]);
  const clearSlots = () => {
    setSlots(null);
    setSelected("");
  };
  async function loadSlots() {
    if (!service || !location || !date) {
      setMessage("Choose a service, location and date first.");
      return;
    }
    setBusy(true);
    setMessage("");
    clearSlots();
    try {
      const query = new URLSearchParams({
        date,
        location_id: location,
        ...(staff ? { staff_id: staff } : {}),
        ...(resource ? { resource_id: resource } : {}),
      });
      const result = await api<{ slots: Slot[] }>(
        `${root}/catalog/services/${service}/slots?${query}`,
      );
      setSlots(result.data.slots);
    } catch (error) {
      setMessage(readable(error));
    } finally {
      setBusy(false);
    }
  }
  async function refresh(append = false) {
    const result = await api<Booking[]>(
      `${root}/bookings?limit=25${append && cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
    );
    setRows((previous) =>
      append
        ? [
            ...previous,
            ...result.data.filter(
              (item) => !previous.some((old) => old.id === item.id),
            ),
          ]
        : result.data,
    );
    setCursor(result.cursor);
  }
  async function refreshClick(append = false) {
    setBusy(true);
    try {
      await refresh(append);
      setMessage("");
    } catch (error) {
      setMessage(readable(error));
    } finally {
      setBusy(false);
    }
  }
  async function book() {
    if (!selected) return;
    setBusy(true);
    setMessage("");
    try {
      const path = moving
        ? `${root}/bookings/${moving.id}/reschedule`
        : `${root}/bookings`;
      const body = moving
        ? { starts_at: selected }
        : {
            service_id: service,
            location_id: location,
            staff_id: staff || null,
            resource_id: resource || null,
            starts_at: selected,
          };
      const result = await mutate<{
        id?: string;
        replacement?: { id: string };
      }>(actor, path, body);
      const id = result.replacement?.id ?? result.id;
      setMoving(null);
      clearSlots();
      setMessage(
        "Booking saved. Refresh available times to see the remaining slots.",
      );
      if (id) {setDetailAnchor(id);setDetail((await api<Booking>(`${root}/bookings/${id}`)).data);}
      await refresh();
    } catch (error) {
      setMessage(readable(error));
    } finally {
      setBusy(false);
    }
  }
  async function cancelBooking() {
    if (!cancelling) return;
    setBusy(true);
    try {
      await mutate(actor, `${root}/bookings/${cancelling.id}/cancel`, {});
      setCancelling(null);
      setDetail(null);
      clearSlots();
      await refresh();
      setMessage("Booking cancelled. Its capacity is available again.");
    } catch (error) {
      setMessage(readable(error));
    } finally {
      setBusy(false);
    }
  }
  async function deleteCancelled(id:string){setBusy(true);setMessage("");try{await api(`${root}/bookings/${id}`,{method:"DELETE"});setDeleting(null);setDetail(null);await refresh();setMessage("Cancelled booking removed from your business list. Customer history is kept.");}catch(error){setMessage(readable(error));}finally{setBusy(false);}}
  async function completeBooking() {
    if (!completing) return;
    setBusy(true);
    try {
      await mutate(actor, `${root}/bookings/${completing.id}/complete`, {});
      setCompleting(null);
      setDetail(null);
      await refresh();
      setMessage("Booking marked completed. Its history is retained.");
    } catch (error) {
      setMessage(readable(error));
    } finally {
      setBusy(false);
    }
  }
  async function inspect(id: string, anchor = id) {
    setDetailAnchor(anchor);setMoving(null);setCancelling(null);setCompleting(null);
    setBusy(true);
    try {
      setDetail((await api<Booking>(`${root}/bookings/${id}`)).data);
    } catch (error) {
      setMessage(readable(error));
    } finally {
      setBusy(false);
    }
  }
  function beginMove(booking: Booking) {
    setMoving(booking);setDetail(null);setCancelling(null);setCompleting(null);
    setLocation(booking.location_id);
    setService(booking.service_id);
    setStaff(booking.staff_id ?? "");
    setResource(booking.resource_id ?? "");
    setDate(
      dateInZone(
        new Date(booking.starts_at),
        catalog.locations.find((item) => item.id === booking.location_id)
          ?.timezone ?? business.timezone,
      ),
    );
    clearSlots();
    setMessage(
      "Choose another available time below this booking, then confirm the move. Times overlapping the original reservation are hidden.",
    );

  }
  const inputClass =
    "mt-1 block w-full rounded-lg border border-slate-300 bg-white p-3 disabled:bg-slate-100";
  const buttonClass =
    "rounded-lg border border-slate-300 px-4 py-2 text-sm font-semibold disabled:opacity-40";
  const availabilityPanel = (
        <section
          aria-labelledby="availability-title"
          className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm sm:p-6"
        >
          <div className="flex flex-wrap justify-between gap-3">
            <h2 id="availability-title" className="text-xl font-semibold">
              {moving ? "Move your booking" : "Find an available time"}
            </h2>
            <span className="text-sm text-slate-500">Times in {zone}</span>
          </div>
          {moving && (
            <div className="mt-3 flex flex-wrap items-center justify-between gap-3 rounded-lg bg-amber-50 p-3">
              <p>
                Moving the booking from {format(moving.starts_at, true)}. Its
                selections stay the same.
              </p>
              <button
                onClick={() => {
                  setMoving(null);
                  clearSlots();
                }}
                disabled={busy}
                className={buttonClass}
              >
                Stop moving
              </button>
            </div>
          )}
          <fieldset
            disabled={busy}
            className="mt-5 grid gap-4 sm:grid-cols-2 lg:grid-cols-3"
          >
            <label className="text-sm font-medium">
              Service
              <select
                disabled={!!moving}
                value={service}
                onChange={(event) => {
                  setService(event.target.value);
                  clearSlots();
                }}
                className={inputClass}
              >
                <option value="">Select service</option>
                {catalog.services.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="text-sm font-medium">
              Location
              <select
                disabled={!!moving}
                value={location}
                onChange={(event) => {
                  setLocation(event.target.value);
                  setResource("");
                  clearSlots();
                }}
                className={inputClass}
              >
                <option value="">Select location</option>
                {catalog.locations.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="text-sm font-medium">
              Date
              <input
                type="date"
                required
                value={date}
                onChange={(event) => {
                  setDate(event.target.value);
                  clearSlots();
                }}
                className={inputClass}
              />
            </label>
            <label className="text-sm font-medium">
              Staff
              <select
                disabled={!!moving}
                value={staff}
                onChange={(event) => {
                  setStaff(event.target.value);
                  clearSlots();
                }}
                className={inputClass}
              >
                <option value="">No staff reservation</option>
                {catalog.staff.map((item) => (
                  <option key={item.id} value={item.id}>
                    {item.name}
                  </option>
                ))}
              </select>
            </label>
            <label className="text-sm font-medium">
              Resource
              <select
                disabled={!!moving}
                value={resource}
                onChange={(event) => {
                  setResource(event.target.value);
                  clearSlots();
                }}
                className={inputClass}
              >
                <option value="">No resource reservation</option>
                {catalog.resources
                  .filter((item) => item.location_id === location)
                  .map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name} (capacity {item.capacity})
                    </option>
                  ))}
              </select>
            </label>
            <button
              onClick={loadSlots}
              className="self-end rounded-lg bg-teal-800 px-5 py-3 font-semibold text-white disabled:opacity-50"
            >
              {busy ? "Working..." : "Find available times"}
            </button>
          </fieldset>
          <p className="mt-3 text-sm text-slate-500">
            Only selected staff and resources are reserved. Times can change
            until your booking is saved.
          </p>
          {slots && (
            <div className="mt-5">
              <p role="status" className="mb-3 text-sm text-slate-600">
                {slots.length
                  ? `${slots.length} available times`
                  : "No available times. Try another date or selection. Demo availability lasts seven days."}
              </p>
              <div className="grid max-h-72 grid-cols-3 gap-2 overflow-y-auto sm:grid-cols-5 lg:grid-cols-7">
                {slots.map((slot) => (
                  <button
                    key={slot.starts_at}
                    disabled={busy || !canWrite}
                    aria-pressed={selected === slot.starts_at}
                    onClick={() => setSelected(slot.starts_at)}
                    className={`rounded-lg border px-2 py-3 text-sm disabled:opacity-50 ${selected === slot.starts_at ? "border-teal-800 bg-teal-800 text-white" : "border-slate-200 hover:border-teal-700"}`}
                    title={slot.starts_at}
                  >
                    {format(slot.starts_at)}
                  </button>
                ))}
              </div>
            </div>
          )}
          {selected && (
            <div className="mt-5 flex flex-wrap items-center justify-between gap-3 rounded-xl bg-teal-50 p-4">
              <p>
                {format(selected, true)} / {name("services", service)}
              </p>
              <button
                disabled={busy || !canWrite}
                onClick={book}
                className="rounded-lg bg-teal-800 px-5 py-3 font-semibold text-white disabled:opacity-50"
              >
                {moving ? "Confirm new time" : "Confirm booking"}
              </button>
            </div>
          )}
        </section>
      );
  return (
    <div className="space-y-6">
      {message && (
        <p
          role="status"
          className="rounded-xl border border-teal-200 bg-teal-50 p-4"
        >
          {message}
        </p>
      )}
      {!canWrite && (
        <p className="rounded-xl bg-slate-100 p-4">
          View-only access. An owner or administrator can create or change
          bookings.
        </p>
      )}
      {!management && !moving && availabilityPanel}
      <section
        aria-labelledby="booking-list-title"
        className="rounded-2xl border border-slate-200 bg-white p-5 shadow-sm sm:p-6"
      >
        <div className="flex justify-between gap-3">
          <h2 id="booking-list-title" className="text-xl font-semibold">
            Bookings & history
          </h2>
          <button
            onClick={() => refreshClick()}
            disabled={busy}
            className={buttonClass}
          >
            Refresh bookings
          </button>
        </div>
        <p className="mt-2 text-sm text-slate-500">
          Booking times use {business.timezone}. Open a booking to manage its time or status.
          Cancelled, rescheduled and completed bookings stay in history.
        </p>
        {rows.length === 0 && (
          <p className="py-8 text-center text-slate-500">
            No bookings recorded for this business yet.
          </p>
        )}
        <BookingOrganizer rows={rows} busy={busy} paginated={!!cursor} actor={actor} onBusy={setBusy}
          actions={canWrite?[
            {key:"cancel",label:"Cancel selected",status:"cancelled",eligible:(row:Booking)=>row.status==="confirmed"&&Date.now()<Date.parse(row.busy_starts_at),path:(row:Booking)=>`${root}/bookings/${row.id}/cancel`},
            {key:"complete",label:"Complete selected",status:"completed",eligible:(row:Booking)=>row.status==="confirmed"&&Date.now()>=Date.parse(row.busy_ends_at),path:(row:Booking)=>`${root}/bookings/${row.id}/complete`}
          ]:[]} onUpdated={(id,status)=>{setRows(previous=>previous.map(row=>row.id===id?{...row,status}:row));setDetail(null);setMoving(null);setCancelling(null);setCompleting(null);clearSlots();}}>
          {(row) => (
            <article
              aria-label={`Booking ${row.id}`}
              className="booking-card"
            >
              <div className="booking-card-heading">
                <p className="font-semibold">
                  {name("services", row.service_id)}
                </p>
                <p className="mt-1 text-sm text-slate-600">
                  {formatBookingTime(row.starts_at,row.ends_at,undefined,business.timezone)} /{" "}
                  {name("locations", row.location_id)}
                </p>
                {(row.staff_id || row.resource_id) && <p className="mt-1 text-sm text-slate-500">
                  {[row.staff_id ? name("staff",row.staff_id) : null,row.resource_id ? name("resources",row.resource_id) : null].filter(Boolean).join(" / ")}
                </p>}
                <p className="booking-reference mt-2">Reference: {row.id}</p>
                {row.customer_name&&<p className="mt-2 text-sm">Customer: {row.customer_name}</p>}
                <span
                  className={`mt-2 inline-block rounded-full px-3 py-1 text-xs font-semibold ${row.status === "confirmed" ? "bg-teal-50 text-teal-900" : "bg-slate-100 text-slate-600"}`}
                >
                  {row.status === "rescheduled" ? "Moved" : row.status}
                </span>
              </div>
              <div className="booking-actions">
                {canWrite&&row.status==="cancelled"&&<button disabled={busy} className={buttonClass} onClick={()=>setDeleting(row.id)}>Delete from business list</button>}
                <button
                  disabled={busy}
                  onClick={() => inspect(row.id)}
                  aria-expanded={!!detail && detailAnchor===row.id}
                  className={buttonClass}
                >
                  Details
                </button>
                {canWrite &&
                  row.status === "confirmed" &&
                  clock < Date.parse(row.busy_starts_at) && (
                    <>
                      <button
                        disabled={busy}
                        onClick={() => beginMove(row)}
                        className={buttonClass}
                      >
                        Reschedule
                      </button>
                      <button
                        disabled={busy}
                        onClick={() => {setCancelling(row);setCompleting(null);setMoving(null);setDetail(null);}}
                        className={buttonClass}
                      >
                        Cancel booking
                      </button>
                    </>
                  )}
                {canWrite &&
                  row.status === "confirmed" &&
                  clock >= Date.parse(row.busy_ends_at) && (
                    <button
                      disabled={busy}
                      onClick={() => {setCompleting(row);setCancelling(null);setMoving(null);setDetail(null);}}
                      className={buttonClass}
                    >
                      Mark completed
                    </button>
                  )}
              </div>
            {deleting===row.id&&<section className="booking-delete-confirm" aria-label="Confirm booking deletion"><p>Remove this cancelled booking from your business list? The customer&apos;s history is kept.</p><div className="booking-actions"><button disabled={busy} className={buttonClass} onClick={()=>void deleteCancelled(row.id)}>Confirm delete</button><button disabled={busy} className={buttonClass} onClick={()=>setDeleting(null)}>Keep booking</button></div></section>}
            {moving?.id === row.id && availabilityPanel}
      {completing?.id === row.id && (
        <section
          role="alertdialog"
          aria-labelledby="complete-title"
          className="rounded-2xl border border-teal-300 bg-teal-50 p-5"
        >
          <h2 id="complete-title" tabIndex={-1} className="font-semibold">
            Mark this booking completed?
          </h2>
          <p className="my-3">
            {name("services", completing.service_id)} /{" "}
            {format(completing.starts_at, true)}. Confirm that the service was
            delivered. Completion is final; the booking stays in history.
          </p>
          <div className="flex flex-wrap gap-3">
            <button
              disabled={busy}
              onClick={completeBooking}
              className={buttonClass}
            >
              Confirm completion
            </button>
            <button
              disabled={busy}
              onClick={() => setCompleting(null)}
              className={buttonClass}
            >
              Keep confirmed
            </button>
          </div>
        </section>
      )}
      {cancelling?.id === row.id && (
        <section
          role="alertdialog"
          aria-labelledby="cancel-title"
          className="rounded-2xl border border-amber-300 bg-amber-50 p-5"
        >
          <h2 id="cancel-title" tabIndex={-1} className="font-semibold">
            Cancel this booking?
          </h2>
          <p className="my-3">
            {format(cancelling.starts_at, true)} /{" "}
            {name("services", cancelling.service_id)}. Its reserved capacity
            will be released.
          </p>
          <div className="flex gap-3">
            <button
              disabled={busy}
              onClick={cancelBooking}
              className="rounded-lg bg-red-800 px-4 py-2 font-semibold text-white disabled:opacity-50"
            >
              Confirm cancellation
            </button>
            <button
              disabled={busy}
              onClick={() => setCancelling(null)}
              className={buttonClass}
            >
              Keep booking
            </button>
          </div>
        </section>
      )}
      {detail && detailAnchor === row.id && (
        <section
          aria-labelledby="details-title"
          className="rounded-2xl border border-slate-200 bg-white p-5"
        >
          <div className="flex justify-between">
            <h2 id="details-title" className="font-semibold">
              Current booking details
            </h2>
            <button onClick={() => setDetail(null)} className={buttonClass}>
              Close details
            </button>
          </div>
          <p className="mt-3">
            {format(detail.starts_at, true)} / {format(detail.ends_at)} /{" "}
            {detail.status}
          </p>
          <p className="mt-2 text-sm text-slate-600">
            Preparation starts {format(detail.busy_starts_at, true)}. Capacity
            is held until {format(detail.busy_ends_at, true)} for confirmed
            bookings.
          </p>
          <dl className="customer-detail-grid">{[["Customer",detail.customer_name],["Contact number",detail.customer_phone],["Email",detail.customer_email],["Additional information",detail.customer_notes]].map(([label,value])=><div key={label}><dt>{label}</dt><dd>{value||"Not provided"}</dd></div>)}</dl>
          <p className="mt-3 break-all text-xs text-slate-500">
            Booking reference: {detail.id}
          </p>
          <div className="mt-3 flex gap-3">
            {detail.replacement_id && (
              <button
                disabled={busy}
                onClick={() => inspect(detail.replacement_id!, detailAnchor)}
                className={buttonClass}
              >
                View replacement
              </button>
            )}
            {detail.supersedes_booking_id && (
              <button
                disabled={busy}
                onClick={() => inspect(detail.supersedes_booking_id!, detailAnchor)}
                className={buttonClass}
              >
                View original
              </button>
            )}
          </div>
        </section>
      )}
            </article>
          )}
        </BookingOrganizer>
        {cursor && (
          <button
            onClick={() => refreshClick(true)}
            disabled={busy}
            className={`mt-4 ${buttonClass}`}
          >
            Load more bookings
          </button>
        )}
      </section>

    </div>
  );
}
