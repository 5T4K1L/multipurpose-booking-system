"use client";

import Link from "next/link";
import {guestRecoveryLink} from "./guest-access";
import GuestContactEditor from "./guest-contact-editor";
import BookingOrganizer from "../components/booking-organizer";
import RescheduleBooking from "./reschedule-booking";
import {formatBookingTime} from "../bookings/format-booking-time";
import CancelBooking from "./cancel-booking";
import { useEffect, useRef, useState } from "react";
import { api, ApiError, readable } from "../bookings/api";

type Booking = {
  business_name: string;
  service_name: string;
  location_name: string;
  can_cancel?: boolean;
  can_reschedule?: boolean;
  cancel_before?: string;
  id: string;
  business_id: string;
  service_id: string;
  location_id: string;
  starts_at: string;
  ends_at: string;
  status: string;
  supersedes_booking_id: string | null;
  replacement_id: string | null;
};
const format = (value: string) =>
  new Intl.DateTimeFormat(undefined, {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
export default function CustomerBookings({ id }: { id?: string }) {
  const mutating = useRef(false);
  const [recovery,setRecovery]=useState("");
  const [expanded,setExpanded]=useState<string[]>([]);
  const [actionBusy, setActionBusy] = useState(false);
  const [items, setItems] = useState<Booking[]>([]);
  const [cursor, setCursor] = useState("");
  const [message, setMessage] = useState("Loading your bookings...");
  const [busy, setBusy] = useState(true);
  const [signedOut, setSignedOut] = useState(false);
  const [version, setVersion] = useState(0);
  const request = useRef<AbortController | null>(null);
  const base = "/api/v1/customer/bookings";
  useEffect(() => {
    const controller = new AbortController();
    request.current?.abort();
    request.current = controller;
    api<Booking | Booking[]>(
      id ? `${base}/${encodeURIComponent(id)}` : `${base}?limit=20`,
      { signal: controller.signal },
    )
      .then(({ data, cursor }) => {
        if (controller.signal.aborted) return;
        const rows = Array.isArray(data) ? data : [data];
        setItems(rows);
        setCursor(cursor);
        setSignedOut(false);
        setMessage(rows.length ? "" : "No bookings saved on this device yet. Use your recovery link if you booked elsewhere.");
      })
      .catch((error) => {
        if (controller.signal.aborted) return;
        setItems([]);
        setCursor("");
        setSignedOut(error instanceof ApiError && error.status === 401);
        setMessage(
          error instanceof ApiError && error.status === 404
            ? "This booking is unavailable or does not belong to your account."
            : readable(error),
        );
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy(false);
      });
    return () => controller.abort();
  }, [id, version]);
  // Clear sensitive results before refreshing on tab return/account changes.
  useEffect(() => {
    const refresh = () => {
      if (mutating.current) return;
      request.current?.abort();
      setItems([]);
      setCursor("");
      setBusy(true);
      setMessage("Refreshing your bookings...");
      setVersion((v) => v + 1);
    };
    window.addEventListener("focus", refresh);
    return () => {
      window.removeEventListener("focus", refresh);
      request.current?.abort();
    };
  }, []);
  async function more() {
    if (busy || !cursor) return;
    const controller = new AbortController();
    request.current?.abort();
    request.current = controller;
    setBusy(true);
    setMessage("");
    try {
      const result = await api<Booking[]>(
        `${base}?limit=20&cursor=${encodeURIComponent(cursor)}`,
        { signal: controller.signal },
      );
      if (!controller.signal.aborted) {
        setItems((previous) => [
          ...previous,
          ...result.data.filter(
            (item) => !previous.some((old) => old.id === item.id),
          ),
        ]);
        setCursor(result.cursor);
      }
    } catch (error) {
      if (!controller.signal.aborted) {
        setItems([]);
        setCursor("");
        setSignedOut(error instanceof ApiError && error.status === 401);
        setMessage(readable(error));
      }
    } finally {
      if (!controller.signal.aborted) setBusy(false);
    }
  }
  async function loadDetails(ids:string[]) {
    if(mutating.current)return;
    mutating.current=true;setActionBusy(true);
    try {
      const details=await Promise.all(ids.map(async value=>(await api<Booking>(`${base}/${value}`)).data));
      setItems(previous=>previous.map(row=>details.find(detail=>detail.id===row.id)??row));
      setExpanded(previous=>[...new Set([...previous,...ids])]);
    } catch(error) {setMessage(readable(error));}
    finally {mutating.current=false;setActionBusy(false);}
  }
  function refresh() {
    if (mutating.current) return;
    request.current?.abort();
    setItems([]);
    setCursor("");
    setBusy(true);
    setMessage("Refreshing your bookings...");
    setVersion((v) => v + 1);
  }
  return (
    <main className="min-h-screen bg-slate-50 px-4 py-8 text-slate-900 sm:px-8">
      <div className="mx-auto max-w-5xl">
        
        <h1 className="mt-7 text-3xl font-semibold">
          {id ? "Booking details" : "My bookings"}
        </h1>
        <p className="my-4">
          {id
            ? "Current saved booking details."
            : "Your plans, all in one place. Choose a status to browse upcoming bookings and history."}{" "}
          Times are shown in your device local time. Refresh to see updates from the business.
        </p>
        <button
          disabled={busy || actionBusy}
          className="rounded-lg border bg-white px-4 py-2 disabled:opacity-50"
          onClick={refresh}
        >
          Refresh bookings
        </button>
        {message && (
          <p role="status" className="my-5">
            {message}
          </p>
        )}
        {signedOut && <p className="my-4">Use your private booking recovery link or return on the device where you booked. No owner sign-in is needed.</p>}
        <div className="ui-panel my-5 p-5"><h2 className="font-semibold">Your private booking space</h2><p className="my-2 text-sm text-slate-600">Bookings are saved for this browser. A private recovery link opens all bookings saved here on another device. Keep it private.</p><button className="cta cta-secondary" onClick={()=>{try{setRecovery(guestRecoveryLink());}catch{setMessage("Allow browser storage to keep access to your bookings.")}}}>Show my recovery link</button>{recovery&&<label className="mt-3 block">Private link<input className="ui-input mt-1 block w-full border p-3" readOnly value={recovery} onFocus={e=>e.target.select()}/></label>}</div>
        <BookingOrganizer rows={items} busy={busy||actionBusy} paginated={!!cursor}
          onBusy={value=>{mutating.current=value;setActionBusy(value);}} onReview={loadDetails}
          onUpdated={(id,status)=>setItems(previous=>previous.map(row=>row.id===id?{...row,status,can_cancel:false,can_reschedule:false}:row))}
          actions={[{key:"cancel",label:"Cancel selected",status:"cancelled",eligible:(row:Booking)=>row.status==="confirmed"&&(row.can_cancel??Date.parse(row.starts_at)>Date.now()),path:(row:Booking)=>`${base}/${row.id}/cancel`}]}> 
          {(item,review) => {
            const showDetails=!!id||expanded.includes(item.id)||review;
            return (
            <article
              key={item.id}
              className="booking-card"
              aria-label={`Booking ${item.id}`}
            >
              <h2 className="text-lg font-semibold">
                {item.service_name || "Booked service"}
              </h2>
              <p className="mt-2 text-sm"><span className="text-slate-500">Business: </span>{item.business_name || "Business name unavailable"}</p>
              <p className="mt-1 text-sm"><span className="text-slate-500">Location / branch: </span>{item.location_name || "Branch name unavailable"}</p>
              <p className="booking-status" data-status={item.status}>Status: {item.status}</p>
              <p className="booking-reference">Booking reference: {item.id}</p>
              {!id && !review && <button className="booking-detail-toggle" disabled={actionBusy} aria-expanded={showDetails} onClick={()=>{if(showDetails)setExpanded(ids=>ids.filter(value=>value!==item.id));else void loadDetails([item.id]);}}>{showDetails?"Hide details":"Details"}</button>}
              {showDetails && (
                <div className="customer-booking-details">
                  <dl className="booking-facts">
                    <div><dt>Business</dt><dd>{item.business_name || "Business name unavailable"}</dd></div>
                    <div><dt>Location / branch</dt><dd>{item.location_name || "Branch name unavailable"}</dd></div>
                    <div className="booking-facts-time"><dt>Date &amp; time</dt><dd>{formatBookingTime(item.starts_at,item.ends_at)}</dd><dd className="booking-hint">Your device local time</dd></div>
                  </dl>
                  <dl className="booking-reference booking-reference-grid" aria-label="Booking IDs">
                    <dt>Business ID</dt><dd>{item.business_id}</dd>
                    <dt>Service ID</dt><dd>{item.service_id}</dd>
                    <dt>Location ID</dt><dd>{item.location_id}</dd>
                  </dl>
                  {<Link className="mt-4 inline-block text-sm underline" href={`/businesses/${item.business_id}`}>View business public profile</Link>}
                </div>
              )}
              {showDetails && (item.can_reschedule || item.status === "rescheduled") && (
                <RescheduleBooking id={item.id} canMove={!!item.can_reschedule} disabled={actionBusy} onBusy={value=>{mutating.current=value;setActionBusy(value);}} onMoved={replacement=>{setItems(previous=>previous.map(row=>row.id===item.id?{...row,status:"rescheduled",can_cancel:false,can_reschedule:false,replacement_id:replacement}:row));}} />
              )}
              {showDetails && item.can_cancel && <GuestContactEditor id={item.id} disabled={actionBusy} onBusy={value=>{mutating.current=value;setActionBusy(value)}}/>}
              {showDetails && ["cancelled","completed","rescheduled"].includes(item.status) && <button className="booking-detail-toggle" disabled={actionBusy} onClick={async()=>{if(!window.confirm("Remove this booking from your list? The business keeps its history."))return;mutating.current=true;setActionBusy(true);try{await api(`${base}/${item.id}`,{method:"DELETE"});setItems(rows=>rows.filter(row=>row.id!==item.id));setMessage("Booking removed from your list.")}catch(e){setMessage(readable(e))}finally{mutating.current=false;setActionBusy(false)}}}>Delete from my list</button>}
              {showDetails && item.can_cancel && (
                <>
                  <p className="mt-4 text-sm">
                    Cancel before{" "}
                    {item.cancel_before
                      ? format(item.cancel_before)
                      : format(item.starts_at)}{" "}
                    local time, when preparation for your booking begins.
                  </p>
                  <CancelBooking
                    id={item.id}
                    disabled={actionBusy}
                    onBusy={(value) => {
                      mutating.current = value;
                      setActionBusy(value);
                    }}
                    onCancelled={() => {
                      setItems((previous) =>
                        previous.map((row) =>
                          row.id === item.id
                            ? { ...row, status: "cancelled", can_cancel: false, can_reschedule: false }
                            : row,
                        ),
                      );
                      setMessage(
                        "Your booking is cancelled. The business can see the update, and your reserved time has been released.",
                      );
                    }}
                  />
                </>
              )}
              {showDetails &&
                item.status === "confirmed" &&
                item.can_cancel === false && (
                  <p className="mt-4 text-sm">
                    The cancellation deadline has passed. Contact the business
                    for help.
                  </p>
                )}
              {item.supersedes_booking_id && (
                <p className="mt-3">
                  <Link
                    className="underline"
                    href={`/my-bookings/${item.supersedes_booking_id}`}
                  >
                    View original booking
                  </Link>
                </p>
              )}
              {item.replacement_id && (
                <p className="mt-3">
                  <Link
                    className="underline"
                    href={`/my-bookings/${item.replacement_id}`}
                  >
                    View replacement booking
                  </Link>
                </p>
              )}
            </article>
          );}}
        </BookingOrganizer>
        {cursor && (
          <button
            disabled={busy || actionBusy}
            className="mt-5 rounded-lg bg-teal-800 px-5 py-3 text-white disabled:opacity-50"
            onClick={more}
          >
            {busy ? "Loading..." : "Load more bookings"}
          </button>
        )}
        <p className="mt-6 text-sm">
          Open booking details to cancel or reschedule before preparation begins.
        </p>
      </div>
    </main>
  );
}
