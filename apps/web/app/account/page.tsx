"use client";

import {useNavigationSession} from "../components/navigation-session";
import Link from "next/link";
import { useEffect, useState } from "react";

type Account = { id: number; email: string; username: string };

export default function AccountPage() {
  const navigation=useNavigationSession();
  const [account, setAccount] = useState<Account | null>(null);
  const [status, setStatus] = useState("Loading your account...");

  useEffect(() => {
    let active = true;
    const controller = new AbortController();
    const base =
      process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";
    fetch(`${base}/api/v1/auth/me`, {
      credentials: "include",
      cache: "no-store",
      signal: AbortSignal.any([controller.signal, AbortSignal.timeout(15000)]),
    })
      .then(async (response) => {
        if (response.status === 401) {
          if (active)
            setStatus("You are signed out. Sign in to view your account.");
          return;
        }
        if (!response.ok) throw new Error("Account unavailable");
        const data: { user: Account } = await response.json();
        if (active) {
          setAccount(data.user);
          setStatus("");
        }
      })
      .catch(() => {
        if (active)
          setStatus(
            "Your account could not be loaded. Please refresh to try again.",
          );
      });
  return () => { active=false; controller.abort(); };
  }, []);

  const business = navigation.businesses.find(b => b.role === "BUSINESS_OWNER") ?? navigation.businesses[0];
  const displayName = business?.name ?? (navigation.ready ? "Set up your business" : "Loading business...");
  return (
    <main className="account-page">
      <header className="account-heading"><p className="eyebrow">Your business account</p><h1 id="account-title">Account</h1><p>Your business profile and account details in one place.</p></header>
      {status && <p role="status" className="ui-panel p-5">{status}</p>}
      {account && <div className="account-grid">
        <section className="ui-panel account-profile" aria-label="Account profile">
          <div className="account-avatar" aria-hidden="true">{business?.name.trim().slice(0,1).toUpperCase()||"B"}</div>
          <h2>{displayName}</h2><p className="account-email">{account.email}</p>
          <div className="account-profile-fields"><h3>Account information</h3><dl><dt>Business name</dt><dd>{displayName}</dd><dt>Email address</dt><dd>{account.email}</dd></dl></div>
          <details className="account-reference"><summary>Account reference</summary><p>Account ID: {account.id}</p></details>
        </section>
        <div className="account-destinations">
          <section className="ui-panel"><p className="eyebrow">Plans</p><h2>Make time for what matters.</h2><p>Review your bookings, change an available time, or find your next service.</p><div className="hero-actions"><Link className="cta cta-primary" href="/my-bookings">My bookings</Link><Link className="cta cta-secondary" href="/businesses">Explore businesses</Link></div></section>
          <section className="ui-panel"><p className="eyebrow">Business</p><h2>{navigation.businesses.length?"Your workspace":"Have a business?"}</h2><p>{navigation.businesses.length?"Keep customer bookings, your team and business settings together.":"Set up a business profile when you are ready to accept bookings."}</p><Link className="cta cta-secondary" href="/business">{navigation.businesses.length?"Open workspace":"Set up a business"}</Link></section>
          <section className="account-signout"><div><h2>Signed in on this device</h2><p>Sign out when you have finished on a shared device.</p></div><Link className="cta cta-secondary" href="/logout">Sign out</Link></section>
        </div>
      </div>}
      {!account&&status&&!status.startsWith("Loading")&&<Link className="cta cta-primary mt-6" href="/login">Sign in</Link>}
    </main>
  );
}
