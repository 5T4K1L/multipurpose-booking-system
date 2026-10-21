"use client";

import { useEffect, useState } from "react";
import Link from "next/link";

export default function Home() {
  const [apiStatus, setApiStatus] = useState("Checking API...");

  useEffect(() => {
    // Request the API health status from the Go backend.
    fetch("http://localhost:8080/health")
      .then((response) => {
        if (!response.ok) {
          throw new Error("API request failed");
        }

        return response.json();
      })
      .then((data: { status: string }) => {
        setApiStatus(data.status);
      })
      .catch(() => {
        // Show a simple failure state if the API cannot be reached.
        setApiStatus("unreachable");
      });
  }, []);

  return (
    <main className="landing">
      <section className="hero" aria-labelledby="home-title">
        <div>
          <p className="eyebrow">Make time for what matters</p>
          <h1 id="home-title">Your next plan.<br/><span>All booked.</span></h1>
          <p className="hero-copy">Find the right business, choose a time that works, and keep everything in one place. A simpler way to make it happen.</p>
          <div className="hero-actions"><Link className="cta cta-primary" href="/businesses">Find a business <span aria-hidden="true">&#8594;</span></Link><Link className="cta cta-secondary" href="/my-bookings">My bookings</Link></div>
          <p className="hero-note">Business owner? <Link className="underline" href="/register">Create an owner account</Link> or <Link className="underline" href="/login">sign in</Link>.</p>
        </div>
        <div className="journey">
          <div className="journey-label"><span>A little less back and forth</span><span>Your time, simplified</span></div>
          <h2>From finding to booked.</h2>
          <ol><li><div><strong>Find your fit</strong><p>Explore businesses and the services they offer.</p></div></li><li><div><strong>Make room in your day</strong><p>Choose a branch, date, and available time.</p></div></li><li><div><strong>Keep your plans close</strong><p>View and manage your bookings in one place.</p></div></li></ol>
          <p className="journey-bottom"><span aria-hidden="true">&#10003;</span> Clear choices, from start to finish.</p>
        </div>
      </section>
      <section className="business-invite" aria-labelledby="business-title"><div><p className="eyebrow">For the people behind the business</p><h2 id="business-title">More organized. More in control.</h2><p>Bring your services, availability, team, and bookings together in your business workspace.</p></div><Link href="/business" className="cta cta-secondary">Open your workspace <span aria-hidden="true">&#8599;</span></Link></section>
      <p className="connection-status" data-online={apiStatus !== "unreachable" && apiStatus !== "Checking API..."} role="status">{apiStatus === "Checking API..." ? "Checking service availability..." : apiStatus === "unreachable" ? "Booking service is currently unavailable. Please try again shortly." : "Booking service is available"}</p>
    </main>
  );
}
