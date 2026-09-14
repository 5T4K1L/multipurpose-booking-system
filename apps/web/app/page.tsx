"use client";

import { useEffect, useState } from "react";

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
    <main className="flex min-h-screen items-center justify-center p-6">
      <div className="text-center">
        <h1 className="text-2xl font-semibold">Booking Platform</h1>

        <p className="mt-4">
          API status: <strong>{apiStatus}</strong>
        </p>
      </div>
    </main>
  );
}
