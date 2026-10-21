"use client";

import { formatBookingTime } from "../../bookings/format-booking-time";
import Link from "next/link";
import { useRef, useState } from "react";
import { ApiError, mutate, readable, Slot } from "../../bookings/api";
import { PublicLocation, PublicService } from "../../business/public-profile";

type Receipt = {
  id: string;
  starts_at: string;
  ends_at: string;
  status: string;
  replayed: boolean;
};
export default function BookingConfirmation({
  businessID,
  service,
  location,
  slot,
  availabilityReady,
  onConflict,
  onBusy,
  onBooked,
}: {
  businessID: string;
  service: PublicService;
  location: PublicLocation;
  slot: Slot;
  availabilityReady: boolean;
  onConflict: () => void;
  onBusy: (busy: boolean) => void;
  onBooked: () => void;
}) {
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  const [contact,setContact]=useState({name:"",phone:"",email:"",notes:""});
  const form=useRef<HTMLFormElement>(null);
  const [receipt, setReceipt] = useState<Receipt | null>(null);
  const pending = useRef(false);
  const [uncertain, setUncertain] = useState(false);
  async function book() {
    if (pending.current || receipt || (!availabilityReady && !uncertain))
      return;
    pending.current = true;
    setBusy(true);
    onBusy(true);

    setMessage("Submitting your booking...");
    let submitted = false;
    try {
      submitted = true;
      const result = await mutate<Receipt>(
        0,
        "/api/v1/customer/bookings",
        {
          contact,
          business_id: businessID,
          service_id: service.id,
          location_id: location.id,
          starts_at: slot.starts_at,
          expected_ends_at: slot.ends_at,
          expected_timezone: location.timezone,
        },
      );
      setReceipt(result);
      setMessage("");
      onBooked();
    } catch (error) {
      setUncertain(
        submitted &&
          (!(error instanceof ApiError) ||
            error.status >= 500 ||
            error.status === 429),
      );
      if (error instanceof ApiError && error.status === 409) onConflict();
      if (error instanceof ApiError && error.status === 404)
        setMessage("This business, service or location is no longer available. Refresh the profile.");
      else if (error instanceof ApiError && error.status === 403)
        setMessage(
          "This booking could not be accepted. Refresh availability and try again.",
        );
      else setMessage(readable(error));
    } finally {
      pending.current = false;
      setBusy(false);
      onBusy(false);
    }
  }
  return (
    <section
      aria-label="Confirm your booking"
      className="range-picker booking-confirmation"
    >
      <h3 className="text-lg font-semibold">
        {receipt ? "Booking received" : "Confirm your booking"}
      </h3>
      <p className="mt-2 break-words">
        {service.name} at {location.name}
      </p>
      <p className="mt-2">
        {formatBookingTime(receipt?.starts_at ?? slot.starts_at, receipt?.ends_at ?? slot.ends_at)} (your local time)
      </p>
      {receipt ? (
        <div role="status" className="mt-3 break-words">
          <p>Booking status: {receipt.status}.</p>
          <p>Booking reference: {receipt.id}</p>
          <p className="mt-2">
            {receipt.replayed
              ? "Your previous request was recovered; no duplicate booking was created."
              : "Your booking is saved and visible to this business."}
          </p>
          <p className="mt-2">
            <Link className="underline" href={`/my-bookings/${receipt.id}`}>
              View booking details
            </Link>
          </p>
        </div>
      ) : (
        <>
          <p className="my-3 text-sm">
            No account needed. Add your contact details, then confirm to reserve your time.
          </p>
          <form ref={form} onSubmit={event=>{event.preventDefault();void book();}} className="my-4 grid gap-4 sm:grid-cols-2">
            <fieldset disabled={busy||uncertain} className="contents">
              <label>Your name<input required maxLength={80} autoComplete="name" className="ui-input mt-1 block w-full border p-3" value={contact.name} onChange={e=>setContact({...contact,name:e.target.value})}/></label>
              <label>Contact number<input required type="tel" minLength={5} maxLength={40} autoComplete="tel" className="ui-input mt-1 block w-full border p-3" value={contact.phone} onChange={e=>setContact({...contact,phone:e.target.value})}/></label>
              <label className="sm:col-span-2">Email (optional)<input type="email" maxLength={254} autoComplete="email" className="ui-input mt-1 block w-full border p-3" value={contact.email} onChange={e=>setContact({...contact,email:e.target.value})}/></label>
              <label className="sm:col-span-2">Anything the business should know? (optional)<textarea maxLength={1000} className="ui-input mt-1 block w-full border p-3" value={contact.notes} onChange={e=>setContact({...contact,notes:e.target.value})}/></label>
            </fieldset>
            <p className="text-sm sm:col-span-2">Your contact details are shared with this business to handle your booking. Save your private recovery link from My bookings to access it on another device.</p>
          <button
            type="submit"
            disabled={busy || (!availabilityReady && !uncertain)}
            className="cta cta-primary w-full sm:col-span-2"
          >
            {busy
              ? "Submitting..."
              : uncertain
                ? "Retry booking confirmation"
                : "Confirm booking"}
          </button></form>
          {!availabilityReady && !uncertain && (
            <p className="mt-3 text-sm">
              This time is no longer available or could not be checked. Refresh
              availability or choose another time.
            </p>
          )}
          {uncertain && (
            <p className="mt-3 text-sm">
              We could not confirm the result. Retry to recover this request
              safely before choosing another time.
            </p>
          )}
          {message && (
            <p role="status" className="mt-3">
              {message}
            </p>
          )}

        </>
      )}
    </section>
  );
}
