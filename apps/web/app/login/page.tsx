"use client";

import Link from "next/link";
import { useState, type FormEvent } from "react";

export default function LoginPage() {
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");
  const [failed, setFailed] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending) return;
    const form = event.currentTarget;
    const data = new FormData(form);
    setPending(true);
    setMessage("");
    setFailed(false);
    try {
      const base = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";
      const response = await fetch(`${base}/api/v1/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "include",
        signal: AbortSignal.timeout(15000),
        body: JSON.stringify({ email: data.get("email"), password: data.get("password") }),
      });
      if (response.ok) {
        form.reset();
        setMessage("You’re signed in.");
      } else {
        setFailed(true);
        setMessage(response.status === 401
          ? "Unable to sign in with those credentials."
          : response.status === 429
            ? "Too many attempts. Please wait before trying again."
            : response.status === 400
              ? "Enter your email and password."
              : "Sign-in is unavailable right now. Please try again later.");
      }
    } catch {
      setFailed(true);
      setMessage("We couldn’t reach the server. Please try again.");
    } finally {
      setPending(false);
    }
  }

  const inputClass = "mt-2 w-full rounded-lg border border-neutral-400 bg-transparent px-3 py-3 focus:outline-2 focus:outline-offset-2 focus:outline-blue-600";
  return (
    <main className="flex min-h-screen items-center justify-center px-4 py-10">
      <section className="w-full max-w-md" aria-labelledby="login-title">
        <Link href="/" className="text-sm underline underline-offset-4">Booking Platform</Link>
        <h1 id="login-title" className="mt-6 text-3xl font-semibold">Welcome back</h1>
        <p className="mt-3 text-sm">Sign in with your email and password.</p>
        <form onSubmit={submit} className="mt-8 space-y-5" aria-busy={pending}>
          <div><label htmlFor="email">Email</label><input className={inputClass} id="email" name="email" type="email" autoComplete="username" required maxLength={254} /></div>
          <div>
            <label htmlFor="password">Password</label>
            <input className={inputClass} id="password" name="password" type={showPassword ? "text" : "password"} autoComplete="current-password" required maxLength={512} />
            <button type="button" className="mt-2 min-h-11 text-sm underline" aria-controls="password" aria-pressed={showPassword} onClick={() => setShowPassword(!showPassword)}>{showPassword ? "Hide password" : "Show password"}</button>
          </div>
          <button disabled={pending} className="min-h-12 w-full rounded-lg bg-blue-700 px-4 py-3 font-semibold text-white hover:bg-blue-800 disabled:opacity-60">{pending ? "Signing in…" : "Sign in"}</button>
          <p role={failed ? "alert" : "status"} className="min-h-6 text-sm">{message}</p>
        </form>
        <p className="mt-6 text-sm">New here? <Link href="/register" className="underline underline-offset-4">Create an account</Link></p>
        <p className="mt-4 text-sm"><Link href="/logout" className="underline underline-offset-4">Sign out</Link></p>
        <p className="mt-4 text-sm"><Link href="/account" className="underline underline-offset-4">Your account</Link></p>
      </section>
    </main>
  );
}
