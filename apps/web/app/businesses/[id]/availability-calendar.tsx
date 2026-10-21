"use client";

import { Slot } from "../../bookings/api";
export type CalendarDay = {
  date: string;
  status: "available" | "fully_booked" | "unavailable";
  slots: Slot[];
};
export type CalendarResult = {
  month: string;
  timezone: string;
  advisory: boolean;
  days: CalendarDay[];
};
const labels = {
  available: "Available",
  fully_booked: "Fully booked",
  unavailable: "Unavailable",
};
const colors = {
  available: "border-teal-700 bg-teal-50 text-teal-900",
  fully_booked: "border-amber-300 bg-amber-50 text-amber-900",
  unavailable: "border-slate-200 bg-slate-100 text-slate-500",
};
function changeMonth(month: string, delta: number) {
  const day = new Date(`${month}-01T12:00:00Z`);
  day.setUTCMonth(day.getUTCMonth() + delta);
  return day.toISOString().slice(0, 7);
}
export default function AvailabilityCalendar({
  month,
  result,
  date,
  disabled,
  loading,
  onMonth,
  onDate,
}: {
  month: string;
  result: CalendarResult | null;
  date: string;
  disabled: boolean;
  loading: boolean;
  onMonth: (month: string) => void;
  onDate: (date: string) => void;
}) {
  const first = new Date(`${month}-01T12:00:00Z`);
  const count = new Date(
    Date.UTC(first.getUTCFullYear(), first.getUTCMonth() + 1, 0),
  ).getUTCDate();
  const title = new Intl.DateTimeFormat(undefined, {
    month: "long",
    year: "numeric",
    timeZone: "UTC",
  }).format(first);
  return (
    <section aria-label="Date at this location" className="mt-5 min-w-0">
      <h3 className="font-semibold">Date at this location</h3>
      <div className="my-3 flex items-center justify-between gap-2">
        <button
          type="button"
          aria-label="Previous month"
          disabled={disabled || month <= "2000-01"}
          onClick={() => onMonth(changeMonth(month, -1))}
          className="rounded-lg border px-3 py-2 disabled:opacity-40"
        >
          ‹
        </button>
        <span aria-live="polite" className="text-center font-semibold">
          {title}
        </span>
        <button
          type="button"
          aria-label="Next month"
          disabled={disabled || month >= "2100-12"}
          onClick={() => onMonth(changeMonth(month, 1))}
          className="rounded-lg border px-3 py-2 disabled:opacity-40"
        >
          ›
        </button>
      </div>
      <div
        className="grid grid-cols-7 gap-1 text-center text-xs"
        aria-busy={loading}
      >
        {["Sun", "Mon", "Tue", "Wed", "Thu", "Fri", "Sat"].map((day) => (
          <span key={day} className="py-2 font-medium">
            {day}
          </span>
        ))}
        {Array.from({ length: first.getUTCDay() }, (_, i) => (
          <span key={`blank-${i}`} aria-hidden="true" />
        ))}
        {Array.from({ length: count }, (_, i) => {
          const value = `${month}-${String(i + 1).padStart(2, "0")}`;
          const status = result?.days.find(
            (item) => item.date === value,
          )?.status;
          const label = status
            ? labels[status]
            : loading
              ? "Checking"
              : "Not checked";
          return (
            <button
              key={value}
              type="button"
              disabled={disabled || status !== "available"}
              aria-label={`${value}: ${label}`}
              aria-pressed={date === value}
              title={`${value}: ${label}`}
              onClick={() => onDate(value)}
              className={`min-h-12 min-w-0 rounded-md border py-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-teal-800 ${status ? colors[status] : "border-slate-200 text-slate-400"} ${date === value ? "ring-2 ring-teal-800 ring-offset-1" : ""}`}
            >
              <span className="block text-sm font-semibold">{i + 1}</span>
              <span aria-hidden="true" className="block text-[10px]">
                {status === "available"
                  ? "●"
                  : status === "fully_booked"
                    ? "Full"
                    : status
                      ? "—"
                      : "…"}
              </span>
            </button>
          );
        })}
      </div>
      <div
        className="mt-4 flex flex-wrap gap-x-4 gap-y-2 text-xs"
        aria-label="Calendar legend"
      >
        <span className="text-teal-900">● Available</span>
        <span className="text-amber-900">Full · Fully booked</span>
        <span className="text-slate-600">— Unavailable</span>
      </div>
    </section>
  );
}
