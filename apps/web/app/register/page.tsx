"use client";

import { useState, type FormEvent } from "react";
import Link from "next/link";

export default function RegisterPage() {
  const [pending, setPending] = useState(false);
  const [message, setMessage] = useState("");
  const [failed, setFailed] = useState(false);
  const [showPassword, setShowPassword] = useState(false);

  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    if (pending) return;
    const form = event.currentTarget;
    const data = new FormData(form);
    const password = String(data.get("password") ?? "");
    if (Array.from(password).length < 15 || Array.from(password).length > 128) {
      setFailed(true);
      setMessage("Use a password of 15–128 characters.");
      return;
    }
    setPending(true);
    setMessage("");
    setFailed(false);
    try {
      const base = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";
      const response = await fetch(`${base}/api/v1/auth/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        credentials: "omit",
        signal: AbortSignal.timeout(15000),
        body: JSON.stringify({ email: data.get("email"), username: data.get("username"), password }),
      });
      if (response.status === 202) {
        form.reset();
        setMessage("Registration request received.");
      } else {
        setFailed(true);
        setMessage(response.status === 429
          ? "Too many attempts. Please wait before trying again."
          : response.status === 400
            ? "Check your email, name, and password, then try again."
            : "Registration is unavailable right now. Please try again later.");
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
      <section className="w-full max-w-md" aria-labelledby="register-title">
        <Link href="/" className="text-sm underline underline-offset-4">Booking Platform</Link>
        <h1 id="register-title" className="mt-6 text-3xl font-semibold">Create your account</h1>
        <p className="mt-3 text-sm">Choose a long, unique password to protect your account.</p>
        <form onSubmit={submit} className="mt-8 space-y-5" aria-busy={pending}>
          <div><label htmlFor="username">Name</label><input className={inputClass} id="username" name="username" autoComplete="name" required maxLength={80} /></div>
          <div><label htmlFor="email">Email</label><input className={inputClass} id="email" name="email" type="email" autoComplete="email" required maxLength={254} /></div>
          <div>
            <label htmlFor="password">Password</label>
            <input className={inputClass} id="password" name="password" type={showPassword ? "text" : "password"} autoComplete="new-password" required aria-describedby="password-help" />
            <p id="password-help" className="mt-2 text-sm">15–128 characters. Spaces are welcome.</p>
            <button type="button" className="mt-2 min-h-11 text-sm underline" aria-controls="password" aria-pressed={showPassword} onClick={() => setShowPassword(!showPassword)}>{showPassword ? "Hide password" : "Show password"}</button>
          </div>
          <button disabled={pending} className="min-h-12 w-full rounded-lg bg-blue-700 px-4 py-3 font-semibold text-white hover:bg-blue-800 disabled:opacity-60">{pending ? "Submitting…" : "Create account"}</button>
          <p role={failed ? "alert" : "status"} className="min-h-6 text-sm">{message}</p>
        </form>
      </section>
    </main>
  );
}
