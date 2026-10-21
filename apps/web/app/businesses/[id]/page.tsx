"use client";

import Link from "next/link";
import Availability from "./availability";
import { useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { api, ApiError, readable } from "../../bookings/api";
import PublicProfileCard, {
  PublicProfile,
} from "../../business/public-profile";

export default function PublicBusinessPage() {
  const { id } = useParams<{ id: string }>();
  return <Profile key={id} id={id} />;
}
function Profile({ id }: { id: string }) {
  const [bookingBusy, setBookingBusy] = useState(false);
  const [selected, setSelected] = useState("");
  const [profile, setProfile] = useState<PublicProfile | null>(null);
  const [message, setMessage] = useState("Loading business profile...");
  const [busy, setBusy] = useState(true);
  const [version, setVersion] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    api<PublicProfile>(`/api/v1/public/businesses/${encodeURIComponent(id)}`, {
      credentials: "omit",
      signal: controller.signal,
    })
      .then((result) => {
        setProfile(result.data);
        setSelected("");
        setMessage("");
      })
      .catch((error) => {
        if (!controller.signal.aborted) {
          setProfile(null);
          setMessage(
            error instanceof ApiError && [400, 404].includes(error.status)
              ? "This business profile is unavailable. It may not be published."
              : readable(error),
          );
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy(false);
      });
    return () => controller.abort();
  }, [id, version]);
  return (
    <main className="min-h-screen bg-slate-50 px-4 py-8 text-slate-900 sm:px-8">
      <div className="mx-auto max-w-3xl">
        <nav className="mb-8 flex flex-wrap gap-5">
          <Link href="/businesses" className="text-teal-800 underline">
            Find a business
          </Link>
          <Link href="/" className="font-semibold text-teal-800 underline">
            Booking Platform
          </Link>
        </nav>
        <h1 className="mb-5 text-2xl font-semibold">Book with this business</h1>
        {message && (
          <p role="status" className="my-4">
            {message}
          </p>
        )}
        {profile && (
          <>
            <PublicProfileCard profile={profile} />
            <section
              aria-label="Choose a service"
              className="range-picker"
            >
              <h2 className="text-xl font-semibold">Choose a service</h2>
              <p className="my-3 text-sm">
                Choose what you need, then pick a branch and an available date.
              </p>
              {profile.services?.length ? (
                <>
                  <div className="booking-option-grid" role="group" aria-label="Services">{profile.services.map(item=><button type="button" key={item.id} data-service-id={item.id} disabled={bookingBusy} aria-pressed={selected===item.id} className="range-choice booking-option" onClick={()=>setSelected(item.id)}><strong>{item.name}</strong><span>{item.all_day?"Whole day":item.booking_options?.enabled?"Choose start and end":`${item.duration_minutes} minutes`}</span></button>)}</div>
                  {selected && (
                    <p role="status" className="mt-3 break-words text-sm text-slate-600">
                      Selected:{" "}
                      {
                        profile.services.find((item) => item.id === selected)
                          ?.name
                      }
                      . Check availability below.
                    </p>
                  )}
                </>
              ) : (
                <p>No published services are available yet.</p>
              )}
            </section>
            {selected &&
              profile.services.some((item) => item.id === selected) && (
                <Availability
                  onBusy={setBookingBusy}
                  key={selected}
                  businessID={id}
                  service={profile.services.find(
                    (item) => item.id === selected,
                  )!}
                />
              )}
          </>
        )}
        <button
          disabled={busy || bookingBusy}
          className="mt-6 rounded-lg border px-4 py-2 disabled:opacity-50"
          onClick={() => {
            setSelected("");
            setProfile(null);
            setBusy(true);
            setMessage("Refreshing profile...");
            setVersion((v) => v + 1);
          }}
        >
          Refresh profile
        </button>
      </div>
    </main>
  );
}
