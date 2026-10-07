"use client";

import { useEffect, useState } from "react";

export default function OAuthButtons() {
  const [providers, setProviders] = useState<string[]>([]);
  const base = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";
  useEffect(() => {
    const controller = new AbortController();
    fetch(`${base}/api/v1/auth/providers`, { cache: "no-store", signal: AbortSignal.any([controller.signal, AbortSignal.timeout(10000)]) })
      .then(async (response) => {
        if (!response.ok) return;
        const data: { providers?: unknown } = await response.json();
        if (Array.isArray(data.providers)) setProviders(data.providers.filter((p): p is string => p === "google" || p === "apple"));
      }).catch(() => { /* Password login remains available if discovery fails. */ });
    return () => controller.abort();
  }, [base]);
  if (providers.length === 0) return null;
  return (
    <div className="mt-6 space-y-3" aria-label="Other sign-in options">
      {providers.map((provider) => (
        <form key={provider} action={`${base}/api/v1/auth/${provider}`} method="post">
          <button className="min-h-12 w-full rounded-lg border border-neutral-400 px-4 py-3 font-semibold hover:bg-neutral-500/10">Continue with {provider === "google" ? "Google" : "Apple"}</button>
        </form>
      ))}
    </div>
  );
}
