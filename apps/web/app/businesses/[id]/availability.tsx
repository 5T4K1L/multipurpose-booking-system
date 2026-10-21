"use client";

import { formatBookingTime } from "../../bookings/format-booking-time";
import BookingRange from "./booking-range";
import BookingConfirmation from "./booking-confirmation";
import AvailabilityCalendar, { CalendarResult } from "./availability-calendar";
import { useEffect, useState } from "react";
import { api, ApiError, dateInZone, readable, Slot } from "../../bookings/api";
import { PublicService } from "../../business/public-profile";

export default function Availability({
  businessID,
  service,
  onBusy,
}: {
  businessID: string;
  service: PublicService;
  onBusy: (busy: boolean) => void;
}) {
  const places = service.locations ?? [];
  const [bookingBusy, setBookingBusy] = useState(false);
  const [booked, setBooked] = useState(false);
  const [location, setLocation] = useState(places[0]?.id ?? "");
  const place = places.find((item) => item.id === location);
  const [month, setMonth] = useState(() =>
    dateInZone(new Date(), places[0]?.timezone ?? "UTC").slice(0, 7),
  );
  const [date, setDate] = useState("");
  const [result, setResult] = useState<CalendarResult | null>(null);
  const [selected, setSelected] = useState<Slot | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [version, setVersion] = useState(0);
  const [updated, setUpdated] = useState("");
  const slots = result?.days.find((day) => day.date === date)?.slots ?? [];

  useEffect(() => {
    if (!location || bookingBusy) return;
    const controller = new AbortController();
    let pending = false;
    let retryAt = 0;
    async function refresh() {
      if (pending || document.hidden || Date.now() < retryAt) return;
      pending = true;
      setBusy(true);
      try {
        const response = await api<CalendarResult>(
          `/api/v1/public/businesses/${businessID}/services/${service.id}/calendar?location_id=${encodeURIComponent(location)}&month=${month}`,
          {
            credentials: "omit",
            signal: AbortSignal.any([
              controller.signal,
              AbortSignal.timeout(15000),
            ]),
          },
        );
        if (controller.signal.aborted) return;
        setResult(response.data);
        setUpdated(new Date().toLocaleTimeString());
        setMessage("");
      } catch (error) {
        if (controller.signal.aborted) return;
        setResult(null);
        setUpdated("");
        if (error instanceof ApiError && error.status === 429)
          retryAt = Date.now() + 900000;
        setMessage(
          error instanceof ApiError && error.status === 404
            ? "This service or location is no longer available. Refresh the business profile."
            : readable(error),
        );
      } finally {
        pending = false;
        if (!controller.signal.aborted) setBusy(false);
      }
    }
    void refresh();
    const timer = window.setInterval(() => void refresh(), 30000);
    const focus = () => void refresh();
    window.addEventListener("focus", focus);
    document.addEventListener("visibilitychange", focus);
    return () => {
      controller.abort();
      window.clearInterval(timer);
      window.removeEventListener("focus", focus);
      document.removeEventListener("visibilitychange", focus);
    };
  }, [businessID, service.id, location, month, bookingBusy, booked, version]);

  function clear() {
    setResult(null);
    setDate("");
    setSelected(null);
    setBooked(false);
    setMessage("");
    setUpdated("");
  }
  return (
    <section
      aria-label="Service availability"
      className="range-picker"
    >
      <h2 className="text-xl font-semibold">Available dates and times</h2>
      <p className="my-3 text-sm">
        Choose a date with available times for this service and location. Your
        time is reserved only after you confirm the booking.
      </p>
      {!places.length ? (
        <p>
          Online booking is not available for this service yet. Please contact
          the business for help.
        </p>
      ) : (
        <>
          <div className="range-step"><h3 className="font-semibold">Choose a branch</h3><div className="booking-option-grid mt-3" role="group" aria-label="Branches">{places.map(item=><button type="button" key={item.id} data-location-id={item.id} disabled={bookingBusy} aria-pressed={location===item.id} className="range-choice booking-option" onClick={()=>{if(location===item.id)return;setLocation(item.id);setMonth(dateInZone(new Date(),item.timezone).slice(0,7));clear();}}><strong>{item.name}</strong></button>)}</div></div>
          <AvailabilityCalendar
            month={month}
            result={result}
            date={date}
            disabled={bookingBusy}
            loading={busy}
            onMonth={(value) => {
              setMonth(value);
              clear();
            }}
            onDate={(value) => {
              setDate(value);
              setSelected(null);
              setBooked(false);
            }}
          />
          <p className="mt-3 text-sm">
            Calendar dates follow the branch ({place?.timezone}). Time choices and
            confirmations automatically use your device local time, including
            the local date. Availability updates every 30 seconds and is checked
            again when you book.
          </p>
          <div className="mt-3 flex flex-wrap items-center gap-3">
            <button
              type="button"
              disabled={busy || bookingBusy}
              className="rounded-lg border px-4 py-2 disabled:opacity-50"
              onClick={() => setVersion((v) => v + 1)}
            >
              {busy ? "Checking availability..." : "Refresh availability"}
            </button>
            {updated && (
              <span className="text-xs text-slate-600">
                Last checked {updated}
              </span>
            )}
          </div>
        </>
      )}
      {message && (
        <p role="status" className="my-4 text-sm">
          {message}
        </p>
      )}
      {result && !result.days.some((day) => day.status === "available") && (
        <p className="my-4 text-sm">
          No available start dates this month. Try another month or contact the
          business to ask about its availability.
        </p>
      )}
      {date && place && service.booking_options?.enabled && (
        <BookingRange
          key={`${location}:${date}`}
          businessID={businessID}
          service={service}
          location={place}
          slots={slots}
          onBusy={(value) => {
            setBookingBusy(value);
            onBusy(value);
          }}
          onRefresh={() => setVersion((v) => v + 1)}
        />
      )}
      {result && date && !booked && !service.booking_options?.enabled && (
        <section className="range-picker" aria-label="Choose booking time"><p className="eyebrow">Your booking</p><h3 className="text-xl font-semibold">Choose your time</h3><p className="my-3 text-sm text-slate-600">{service.all_day?"Whole day: opening to closing on your chosen date.":`${service.duration_minutes} minutes per booking.`} Times below use your device local time.</p>
          <div className="range-time-grid" role="group" aria-label="Available start times">{slots.map(slot=><button type="button" key={slot.starts_at} disabled={bookingBusy} aria-pressed={selected?.starts_at===slot.starts_at} aria-label={formatBookingTime(slot.starts_at,slot.ends_at)} className="range-choice" onClick={()=>setSelected(slot)}>{new Intl.DateTimeFormat(undefined,{timeStyle:"short"}).format(new Date(slot.starts_at))}</button>)}</div>
          {!slots.length&&<p role="status" className="my-3 text-sm">No remaining times. Choose another available date.</p>}
          {selected&&<div className="range-summary" aria-live="polite"><span className="text-sm text-slate-600">Your selection</span><strong>{formatBookingTime(selected.starts_at,selected.ends_at)}</strong><span className="text-sm">{service.all_day?"Whole day":`${service.duration_minutes} minutes total`}</span></div>}
        </section>
      )}
      {selected && place && (
        <BookingConfirmation
          key={`${location}:${selected.starts_at}`}
          businessID={businessID}
          service={service}
          location={place}
          slot={selected}
          availabilityReady={
            !!result &&
            slots.some(
              (slot) =>
                slot.starts_at === selected.starts_at &&
                slot.ends_at === selected.ends_at,
            )
          }
          onConflict={() => setVersion((v) => v + 1)}
          onBusy={(value) => {
            setBookingBusy(value);
            onBusy(value);
          }}
          onBooked={() => {
            setBooked(true);
            setMessage("");
          }}
        />
      )}
    </section>
  );
}
