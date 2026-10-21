"use client";
import { useEffect, useRef, useState } from "react";
import { api, readable } from "../bookings/api";
import { usePendingSave } from "./drafts";
import { button, input } from "./styles";
export type BookingOptions = {
  enabled: boolean;
  minimum_minutes: number;
  maximum_minutes: number;
};
export default function BookingOptionsEditor({
  path,
  writable,
  onDirty,
}: {
  path: string;
  writable: boolean;
  onDirty: (dirty: boolean) => void;
}) {
  const [data, setData] = useState<BookingOptions | null>(null),
    [dirty, setDirty] = useState(false),
    [busy, setBusy] = useState(false),
    [message, setMessage] = useState("");
  const form = useRef<HTMLFormElement>(null);
  useEffect(() => {
    const c = new AbortController();
    api<BookingOptions>(path, { signal: c.signal })
      .then((r) => setData(r.data))
      .catch((e) => {
        if (!c.signal.aborted) setMessage(readable(e));
      });
    return () => c.abort();
  }, [path]);
  async function save() {
    if (!data) throw new Error("Booking options are not loaded.");
    if (data.minimum_minutes > data.maximum_minutes)
      throw new Error("Maximum length must be at least the minimum length.");
    setBusy(true);
    try {
      const r = await api<BookingOptions>(path, {
        method: "PUT",
        body: JSON.stringify(data),
      });
      setData(r.data);
      setDirty(false);
      onDirty(false);
      setMessage(
        "Booking mode saved. Publish the reviewed update to make it available to customers.",
      );
    } finally {
      setBusy(false);
    }
  }
  usePendingSave(
    path,
    "Service booking mode",
    dirty && writable,
    busy,
    form,
    save,
  );
  return (
    <section
      className="mt-5 rounded-lg border p-4"
      aria-label="Service booking mode"
    >
      <h3 className="font-semibold">How customers choose their booking time</h3>
      <p className="my-2 text-sm">
        Use a fixed length for appointments, or let customers choose a start and
        end date/time for longer bookings.
      </p>
      {message && (
        <p role="status" className="my-3 text-sm">
          {message}
        </p>
      )}
      {data && (
        <form
          ref={form}
          onSubmit={(e) => {
            e.preventDefault();
            void save().catch((e) =>
              setMessage(e instanceof Error ? e.message : readable(e)),
            );
          }}
          onChange={() => {
            setDirty(true);
            onDirty(true);
          }}
        >
          <fieldset disabled={!writable || busy} className="space-y-3">
            <fieldset><legend className="font-medium">Booking length</legend><div className="mt-2 grid gap-2 sm:grid-cols-2">{[{value:false,label:"Use the service duration"},{value:true,label:"Let customers choose start and end"}].map(option=><label key={String(option.value)} className="workspace-choice workspace-mode"><input type="radio" name={path+"-mode"} checked={data.enabled===option.value} onChange={()=>setData({...data,enabled:option.value})}/>{option.label}</label>)}</div></fieldset>
            {data.enabled && (
              <>
                <div className="grid gap-3 sm:grid-cols-2">
                  <label>
                    Minimum booking length (minutes)
                    <input
                      className={input}
                      type="number"
                      required
                      min={1}
                      max={10080}
                      value={data.minimum_minutes}
                      onChange={(e) =>
                        setData({
                          ...data,
                          minimum_minutes: Number(e.target.value),
                        })
                      }
                    />
                  </label>
                  <label>
                    Maximum booking length (minutes)
                    <input
                      className={input}
                      type="number"
                      required
                      min={data.minimum_minutes || 1}
                      max={10080}
                      value={data.maximum_minutes}
                      onChange={(e) =>
                        setData({
                          ...data,
                          maximum_minutes: Number(e.target.value),
                        })
                      }
                    />
                  </label>
                </div>
                <p className="text-sm">
                  60 minutes = 1 hour; 1,440 = 1 day. Maximum supported length
                  is 7 days. The entire booking, including overnight time, must
                  fit location opening hours. Existing bookings and resource capacity
                  still limit available times.
                </p>
              </>
            )}
            {writable && dirty && (
              <button className={button}>
                {busy ? "Saving..." : "Save booking mode"}
              </button>
            )}
          </fieldset>
        </form>
      )}
    </section>
  );
}
