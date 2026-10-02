"use client";

import Link from "next/link";
import { useState } from "react";

export default function LogoutPage() {
  const [pending, setPending] = useState(false);
  const [done, setDone] = useState(false);
  const [error, setError] = useState("");

  async function logout() {
    if (pending) return;
    setPending(true);
    setError("");
    try {
      const base = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";
      const response = await fetch(`${base}/api/v1/auth/logout`, {
        method: "POST",
        credentials: "include",
        signal: AbortSignal.timeout(15000),
      });
      if (response.status === 204) setDone(true);
      else setError(response.status === 429
        ? "Too many attempts. Please wait before trying again."
        : "Sign-out could not be confirmed. Please try again.");
    } catch {
      setError("We could not reach the server. Please retry to finish signing out.");
    } finally {
      setPending(false);
    }
  }

  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-10">
      <section className="w-full max-w-md" aria-labelledby="logout-title" aria-busy={pending}>
        <Link href="/" className="text-sm underline underline-offset-4">Booking Platform</Link>
        <h1 id="logout-title" className="mt-6 text-3xl font-semibold">Sign out</h1>
        <p className="mt-3 text-sm">End your session in this browser. Your other sessions stay signed in.</p>
        {!done && <button onClick={logout} disabled={pending} className="mt-8 min-h-12 w-full rounded-lg bg-blue-700 px-4 py-3 font-semibold text-white hover:bg-blue-800 disabled:opacity-60">{pending ? "Signing out..." : "Sign out"}</button>}
        <p role="status" className="mt-4">{done ? "You are signed out." : ""}</p>
        {error && <p role="alert" className="mt-4">{error}</p>}
        {done && <Link href="/login" className="mt-6 inline-block underline underline-offset-4">Sign in again</Link>}
      </section>
    </main>
  );
}
