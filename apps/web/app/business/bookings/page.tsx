"use client";

import Link from "next/link";
import { useEffect, useState } from "react";
import { api, Business, readable } from "../../bookings/api";
import Workspace from "../../bookings/workspace";

export default function BusinessBookings() {
  const [actor, setActor] = useState<number | null>(null);
  const [businesses, setBusinesses] = useState<Business[]>([]);
  const [selected, setSelected] = useState("");
  const [message, setMessage] = useState("Loading your businesses...");
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
      .then(([me, list]) => {
        setActor(me.data.user.id);
        setBusinesses(list.data);
        setSelected((list.data.find(b => b.role === "BUSINESS_OWNER") ?? list.data[0])?.id ?? "");
        setMessage("");
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
        
        <h1 className="mt-8 text-3xl font-semibold">Business bookings</h1>
        <p className="my-4 text-slate-600">
          Manage bookings received by your business. Reschedule or cancel before
          preparation starts, and mark delivered services completed after their
          reserved time ends.
        </p>
        {message && (
          <p role="status" className="my-4">
            {message}
          </p>
        )}
        {!actor ? (
          <Link href="/login" className="underline">
            Sign in to continue
          </Link>
        ) : (
          <>
            {business && <p className="mb-6 text-sm text-slate-600">Your business: <strong>{business.name}</strong></p>}
    
            {!businesses.length && (
              <p>
                No business memberships yet.{" "}
                <Link href="/business" className="underline">
                  Open business setup
                </Link>
                .
              </p>
            )}
            {business && (
              <Workspace
                key={business.id}
                business={business}
                actor={actor}
                management
              />
            )}
          </>
        )}
      </div>
    </main>
  );
}
