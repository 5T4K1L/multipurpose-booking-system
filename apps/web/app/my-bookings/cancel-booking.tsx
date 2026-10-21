"use client";
import { useRef, useState } from "react";
import { ApiError, mutate, readable } from "../bookings/api";

export default function CancelBooking({
  id,
  disabled,
  onBusy,
  onCancelled,
}: {
  id: string;
  disabled: boolean;
  onBusy: (busy: boolean) => void;
  onCancelled: () => void;
}) {
  const [confirm, setConfirm] = useState(false),
    [busy, setBusy] = useState(false),
    [message, setMessage] = useState("");
  const pending = useRef(false);
  async function cancel() {
    if (pending.current) return;
    pending.current = true;
    setBusy(true);
    onBusy(true);
    setMessage("");
    try {
      const data={user:{id:0}};
      await mutate(data.user.id, `/api/v1/customer/bookings/${id}/cancel`, {});
      onCancelled();
    } catch (e) {
      setMessage(
        e instanceof ApiError && e.status === 409
          ? "This booking can no longer be cancelled here. Refresh its details to see its current status."
          : e instanceof ApiError && e.status === 404
            ? "This booking is unavailable or does not belong to your current account. Refresh its details."
            : readable(e),
      );
    } finally {
      pending.current = false;
      setBusy(false);
      onBusy(false);
    }
  }
  return (
    <section
      aria-label="Cancel booking"
      className="mt-5 rounded-lg border border-slate-300 p-4"
    >
      {!confirm ? (
        <button
          type="button"
          disabled={disabled}
          className="rounded-lg border border-red-800 px-4 py-2 text-red-900"
          onClick={() => setConfirm(true)}
        >
          Cancel booking
        </button>
      ) : (
        <>
          <h3 className="font-semibold">Cancel this booking?</h3>
          <p className="my-3 text-sm">
            Your reserved time will be released. The booking will stay in your
            history as cancelled.
          </p>
          <div className="flex flex-wrap gap-3">
            <button
              type="button"
              disabled={busy || disabled}
              className="rounded-lg bg-red-800 px-4 py-2 text-white disabled:opacity-50"
              onClick={() => void cancel()}
            >
              {busy ? "Cancelling..." : "Confirm cancellation"}
            </button>
            <button
              type="button"
              disabled={busy || disabled}
              className="rounded-lg border px-4 py-2"
              onClick={() => {
                setConfirm(false);
                setMessage("");
              }}
            >
              Keep booking
            </button>
          </div>
        </>
      )}
      {message && (
        <p role="status" className="mt-3 text-sm">
          {message}
        </p>
      )}
    </section>
  );
}
