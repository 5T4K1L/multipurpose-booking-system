"use client";
import { useEffect, useRef, useState } from "react";
import { api, ApiError, readable, Slot } from "../../bookings/api";
import { localDateTime, localToInstant } from "../../bookings/local-time";
import { PublicLocation, PublicService } from "../../business/public-profile";
import BookingConfirmation from "./booking-confirmation";

export default function BookingRange({
  businessID,
  service,
  location,
  slots,
  onBusy,
  onRefresh,
}: {
  businessID: string;
  service: PublicService;
  location: PublicLocation;
  slots: Slot[];
  onBusy: (busy: boolean) => void;
  onRefresh: () => void;
}) {
  const [start, setStart] = useState(slots[0]?.starts_at ?? "");
  const [end, setEnd] = useState(
    slots[0] ? localDateTime(slots[0].ends_at, Intl.DateTimeFormat().resolvedOptions().timeZone) : "",
  );
  const [zone] = useState(() => Intl.DateTimeFormat().resolvedOptions().timeZone);
  const [checked, setChecked] = useState<Slot | null>(null),
    [available, setAvailable] = useState(false),
    [message, setMessage] = useState(""),
    [checking, setChecking] = useState(false),
    [booking, setBooking] = useState(false),
    [booked, setBooked] = useState(false);
  const generation = useRef(0);
  const options = service.booking_options!;
  const format = (value: string) =>
    new Intl.DateTimeFormat(undefined, {
      timeZone: zone,
      dateStyle: "medium",
      timeStyle: "short",
    }).format(new Date(value));
  function clear() {
    generation.current++;
    setChecked(null);
    setAvailable(false);
    setMessage("");
    setBooked(false);
    setChecking(false);
  }
  async function check() {
    const revision = ++generation.current;
    try {
      if (!start || !slots.some((s) => s.starts_at === start)) throw new Error("Choose an available start time.");
      const endsAt = localToInstant(end, zone),
        minutes = (Date.parse(endsAt) - Date.parse(start)) / 60000;
      if (
        minutes < options.minimum_minutes ||
        minutes > options.maximum_minutes
      )
        throw new Error(
          `Choose an end time between ${options.minimum_minutes} and ${options.maximum_minutes} minutes after the start.`,
        );
      setChecking(true);
      setAvailable(false);
      setMessage("");
      const query = new URLSearchParams({
        location_id: location.id,
        starts_at: start,
        ends_at: endsAt,
      });
      const response = await api<{ slots: Slot[] }>(
        `/api/v1/public/businesses/${businessID}/services/${service.id}/range?${query}`,
        { credentials: "omit" },
      );
      if (revision !== generation.current) return;
      const slot = response.data.slots[0];
      setAvailable(!!slot);
      if (slot) setChecked(slot);
      setMessage(
        slot
          ? "The full date and time range is available. Confirm below to book."
          : "This full range is unavailable. Try a shorter range or another start/end time. Opening hours, staff, resources and existing bookings must allow the whole interval.",
      );
    } catch (e) {
      if (revision === generation.current) {
        setAvailable(false);
        setMessage(
          e instanceof ApiError
            ? readable(e)
            : e instanceof Error
              ? e.message
              : readable(e),
        );
      }
    } finally {
      if (revision === generation.current) setChecking(false);
    }
  }
  useEffect(() => {
    if (!checked || booking || booked) return;
    let active = true,
      pending = false;
    const controller = new AbortController();
    const refresh = async () => {
      if (document.hidden || pending) return;
      pending = true;
      try {
        const query = new URLSearchParams({
          location_id: location.id,
          starts_at: checked.starts_at,
          ends_at: checked.ends_at,
        });
        const r = await api<{ slots: Slot[] }>(
          `/api/v1/public/businesses/${businessID}/services/${service.id}/range?${query}`,
          {
            credentials: "omit",
            signal: AbortSignal.any([
              controller.signal,
              AbortSignal.timeout(15000),
            ]),
          },
        );
        if (active) setAvailable(r.data.slots.length > 0);
      } catch {
        if (active) setAvailable(false);
      } finally {
        pending = false;
      }
    };
    const timer = window.setInterval(() => void refresh(), 30000),
      focus = () => void refresh();
    window.addEventListener("focus", focus);
    document.addEventListener("visibilitychange", focus);
    return () => {
      active = false;
      controller.abort();
      clearInterval(timer);
      window.removeEventListener("focus", focus);
      document.removeEventListener("visibilitychange", focus);
    };
  }, [checked, booking, booked, businessID, service.id, location.id]);
  useEffect(
    () => () => {
      generation.current++;
    },
    [],
  );
  const lengthLabel = (minutes: number) => {
    const days = Math.floor(minutes / 1440), hours = Math.floor((minutes % 1440) / 60), mins = minutes % 60;
    return [days ? `${days} day${days===1?"":"s"}` : "", hours ? `${hours} hr` : "", mins ? `${mins} min` : ""].filter(Boolean).join(" ");
  };
  const lengths = [...new Set([options.minimum_minutes,60,120,240,1440,options.maximum_minutes])].filter(value=>value>=options.minimum_minutes&&value<=options.maximum_minutes).sort((a,b)=>a-b);
  const minimumEnd = start ? localDateTime(new Date(Date.parse(start)+options.minimum_minutes*60000).toISOString(),zone) : "";
  const maximumEnd = start ? localDateTime(new Date(Date.parse(start)+options.maximum_minutes*60000).toISOString(),zone) : "";
  let duration = 0;
  try { duration = (Date.parse(localToInstant(end,zone))-Date.parse(start))/60000; } catch { /* Incomplete input is explained on check. */ }
  const validLength = Number.isFinite(duration)&&duration>=options.minimum_minutes&&duration<=options.maximum_minutes;
  return (
    <section aria-label="Choose booking range" className="range-picker">
      <header><p className="eyebrow">Your booking</p><h3 className="text-xl font-semibold">Choose your start and end</h3><p className="mt-2 text-sm text-slate-600">Choose a start time, then when you want to finish. All times are shown in {zone.replaceAll("_"," ")}.</p></header>
      <form onSubmit={e=>{e.preventDefault();void check();}}>
        <fieldset disabled={booking||checking} className="min-w-0">
          <div className="range-step">
            <h4><span className="range-step-number" aria-hidden="true">1</span>Start time</h4>
            <p className="my-2 text-sm text-slate-600">{start?new Intl.DateTimeFormat(undefined,{timeZone:zone,dateStyle:"full"}).format(new Date(start)):"Choose an available date in the calendar above."}</p>
            <div className="range-time-grid" role="group" aria-label="Available start times">
              {slots.map(slot=><button key={slot.starts_at} type="button" className="range-choice" aria-pressed={start===slot.starts_at} aria-label={format(slot.starts_at)} onClick={()=>{setStart(slot.starts_at);setEnd(localDateTime(new Date(Date.parse(slot.starts_at)+options.minimum_minutes*60000).toISOString(),zone));clear();}}>{new Intl.DateTimeFormat(undefined,{timeZone:zone,timeStyle:"short"}).format(new Date(slot.starts_at))}</button>)}
            </div>
            {!slots.length&&<p role="status" className="my-3 text-sm">No start times remain. Choose another date above.</p>}
            {start&&!slots.some(slot=>slot.starts_at===start)&&<p role="status" className="my-3 text-sm">Your selected start is no longer available. Choose another time.</p>}
          </div>
          <div className="range-step">
            <h4><span className="range-step-number" aria-hidden="true">2</span>When should it end?</h4>
            <p className="my-2 text-sm text-slate-600">Book for {lengthLabel(options.minimum_minutes)} to {lengthLabel(options.maximum_minutes)}. Choose a length or enter your own end date and time.</p>
            <div className="range-lengths" role="group" aria-label="Quick booking lengths">{lengths.map(value=><button type="button" key={value} disabled={!start} className="range-choice" aria-pressed={duration===value} onClick={()=>{setEnd(localDateTime(new Date(Date.parse(start)+value*60000).toISOString(),zone));clear();}}>{lengthLabel(value)}</button>)}</div>
            <div className="range-end-fields">
              <label>End date<input className="ui-input mt-2 w-full" type="date" required min={minimumEnd.slice(0,10)||"2000-01-01"} max={maximumEnd.slice(0,10)||"2100-12-31"} value={end.split("T")[0]??""} onChange={e=>{setEnd(`${e.target.value}T${end.split("T")[1]??""}`);clear();}}/></label>
              <label>End time<input className="ui-input mt-2 w-full" type="time" required step={60} value={end.split("T")[1]??""} onChange={e=>{setEnd(`${end.split("T")[0]??""}T${e.target.value}`);clear();}}/></label>
            </div>
          </div>
          <div className="range-summary" aria-live="polite"><span className="text-sm text-slate-600">Your selection</span><strong>{start?format(start):"Choose a start"} <span aria-hidden="true">&rarr;</span> {validLength?format(localToInstant(end,zone)):"Choose a valid end"}</strong>{validLength&&<span className="text-sm">{lengthLabel(duration)} total</span>}</div>
          <p className="my-3 text-sm text-slate-600">We will check the full time against opening hours and existing bookings. Choosing a length does not reserve it.</p>
          <button type="submit" disabled={checking||!slots.some(slot=>slot.starts_at===start)} className="cta cta-primary w-full">{checking?"Checking availability...":"Check availability"}</button>
        </fieldset>
      </form>
      {message && !booked && (
        <p role="status" className="mt-3 text-sm">
          {message}
        </p>
      )}
      {checked && (
        <BookingConfirmation
          key={`${checked.starts_at}:${checked.ends_at}`}
          businessID={businessID}
          service={service}
          location={location}
          slot={checked}
          availabilityReady={available && slots.some((s) => s.starts_at === checked.starts_at)}
          onConflict={() => {
            setAvailable(false);
            void check();
            onRefresh();
          }}
          onBusy={(value) => {
            setBooking(value);
            onBusy(value);
          }}
          onBooked={() => {
            setBooked(true);
            setMessage("");
            onRefresh();
          }}
        />
      )}
    </section>
  );
}
