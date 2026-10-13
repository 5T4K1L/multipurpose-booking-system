"use client";

import Link from "next/link";
import { useEffect, useState } from "react";

type Account = { id: number; email: string; username: string };

export default function AccountPage() {
  const [account, setAccount] = useState<Account | null>(null);
  const [status, setStatus] = useState("Loading your account...");

  useEffect(() => {
    let active = true;
    const controller = new AbortController();
    const base = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";
    fetch(`${base}/api/v1/auth/me`, {
      credentials: "include", cache: "no-store",
      signal: AbortSignal.any([controller.signal, AbortSignal.timeout(15000)]),
    }).then(async (response) => {
      if (response.status === 401) {
        if (active) setStatus("You are signed out. Sign in to view your account.");
        return;
      }
      if (!response.ok) throw new Error("Account unavailable");
      const data: { user: Account } = await response.json();
      if (active) { setAccount(data.user); setStatus(""); }
    }).catch(() => {
      if (active) setStatus("Your account could not be loaded. Please refresh to try again.");
    });
    return () => { active = false; controller.abort(); };
  }, []);

  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-10">
      <section className="w-full max-w-md" aria-labelledby="account-title">
        <Link href="/" className="text-sm underline underline-offset-4">Booking Platform</Link>
        <h1 id="account-title" className="mt-6 text-3xl font-semibold">Your account</h1>
        <p role="status" className="mt-4">{status}</p>
        {account && <dl className="mt-6 space-y-2 break-words"><dt className="font-semibold">Name</dt><dd>{account.username}</dd><dt className="font-semibold">Email</dt><dd>{account.email}</dd></dl>}
        {account && <p className="mt-6"><Link href="/bookings" className="font-semibold underline underline-offset-4">Open booking workspace</Link></p>}
        <Link href={account ? "/logout" : "/login"} className="mt-6 inline-block underline underline-offset-4">{account ? "Sign out" : "Sign in"}</Link>
      </section>
    </main>
  );
}
