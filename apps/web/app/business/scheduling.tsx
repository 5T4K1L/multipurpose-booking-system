"use client";

import { useEffect, useRef, useState } from "react";
import {
  api,
  ApiError,
  Business,
  CatalogItem,
  readable,
} from "../bookings/api";
import { usePendingSave } from "./drafts";
import BookingOptionsEditor from "./booking-options";
import { localDateTime, localToInstant } from "../bookings/local-time";
import { button, input, panel } from "./styles";

type Kind = "locations" | "services" | "staff" | "resources";
const days = [
  "Sunday",
  "Monday",
  "Tuesday",
  "Wednesday",
  "Thursday",
  "Friday",
  "Saturday",
];
type Hour = {
  weekday: number;
  start_minute: number;
  end_minute: number;
  start_text?: string;
  end_text?: string;
};
type Schedule = {
  timezone: string;
  weekly_hours: Hour[];
  closed_dates: string[];
};
type Rules = {
  configured: boolean;
  buffer_before_minutes: number;
  buffer_after_minutes: number;
  minimum_lead_minutes: number;
};
type Window = { starts_at: string; ends_at: string };
type Data = Schedule | Rules | Window[];
const time = (minute: number) =>
  `${String(Math.floor(minute / 60)).padStart(2, "0")}:${String(minute % 60).padStart(2, "0")}`;
function minutes(value: string) {
  const [h, m] = value.split(":").map(Number);
  return h * 60 + m;
}

export default function Scheduling({business,writable}:{business:Business;writable:boolean}) {
  return <section aria-label="Scheduling" className="workspace-scheduling space-y-6">
    <div><h2 className="text-2xl font-semibold">When customers can book</h2><p className="mt-2 text-sm text-slate-600">Set your opening hours, then choose how much time each service needs. Existing bookings automatically reduce available times.</p></div>
    <ScheduleSection business={business} writable={writable} kind="locations"/>
    <ScheduleSection business={business} writable={writable} kind="services"/>
  </section>;
}
function ScheduleSection({business,writable,kind}:{business:Business;writable:boolean;kind:"locations"|"services"}){
  const [dirty,setDirty]=useState(false),[optionsDirty,setOptionsDirty]=useState(false),[saving,setSaving]=useState(false);
  function discard(){return !(dirty||optionsDirty)||window.confirm("Discard unsaved scheduling changes?");}
  useEffect(()=>{if(!dirty&&!optionsDirty)return;const warn=(e:BeforeUnloadEvent)=>e.preventDefault();window.addEventListener("beforeunload",warn);return()=>window.removeEventListener("beforeunload",warn);},[dirty,optionsDirty]);
  return <section className={panel} aria-label={kind==="locations"?"Opening hours":"Booking length and notice"}>
    <h3 className="text-xl font-semibold">{kind==="locations"?"Opening hours":"Booking length and notice"}</h3>
    <p className="mt-2 text-sm text-slate-600">{kind==="locations"?"Choose the days and times you are open. Add dates when you will be closed.":"Bookings use the service duration by default. Change the booking length below; preparation and advance notice are optional."}</p>
    <Choices onOptionsDirty={setOptionsDirty} initialSelected="" business={business} kind={kind} writable={writable} dirty={dirty} discard={discard} onDirty={setDirty} saving={saving} onSaving={setSaving}/>
  </section>;
}
function Choices({
  onOptionsDirty,
  initialSelected,
  business,
  kind,
  writable,
  dirty,
  discard,
  onDirty,
  saving,
  onSaving,
}: {
  onOptionsDirty: (dirty: boolean) => void;
  initialSelected: string;
  business: Business;
  kind: Kind;
  writable: boolean;
  dirty: boolean;
  discard: () => boolean;
  onDirty: (value: boolean) => void;
  saving: boolean;
  onSaving: (value: boolean) => void;
}) {
  const root = `/api/v1/businesses/${business.id}/catalog/${kind}`;
  const [items, setItems] = useState<CatalogItem[]>([]);
  const [cursor, setCursor] = useState("");
  const [selected, setSelected] = useState(initialSelected);
  const [version, setVersion] = useState(0);
  const [message, setMessage] = useState("Loading choices...");
  const [loading, setLoading] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    api<CatalogItem[]>(`${root}?limit=100`, { signal: controller.signal })
      .then((result) => {
        setItems(result.data);
        setSelected(old=>result.data.some(item=>item.id===old)?old:result.data[0]?.id??"");
        setCursor(result.cursor);
        setMessage(result.data.length ? "" : kind === "locations" ? "Add a branch in Services & branches to set opening hours." : "Add a service in Services & branches to set booking rules.");
      })
      .catch((error) => {
        if (!controller.signal.aborted) setMessage(readable(error));
      });
    return () => controller.abort();
  }, [root, version, kind]);
  async function more() {
    setLoading(true);
    try {
      const result = await api<CatalogItem[]>(
        `${root}?limit=100&cursor=${encodeURIComponent(cursor)}`,
      );
      setItems((previous) => [
        ...previous,
        ...result.data.filter(
          (item) => !previous.some((old) => old.id === item.id),
        ),
      ]);
      setCursor(result.cursor);
    } catch (error) {
      setMessage(readable(error));
    } finally {
      setLoading(false);
    }
  }
  return (
    <div className="mt-4">
      {message && <p role="status">{message}</p>}
      <div className="schedule-choice-toolbar"><div className="schedule-list-tools">
        <button
          type="button"
          className="workspace-text-action"
          disabled={saving || loading}
          onClick={() => {
            if (discard()) {
              onDirty(false);
              onOptionsDirty(false);
              setSelected("");
              setVersion((v) => v + 1);
            }
          }}
        >
          Refresh list
        </button>
        {cursor && (
          <button
            type="button"
            className="workspace-text-action"
            disabled={saving || loading}
            onClick={more}
          >
            Show more
          </button>
        )}
      </div>
      <div className="workspace-choices schedule-current-choices" aria-label={kind==="locations"?"Branches":"Services"}>
        {items.length===1?<p className="font-semibold">{items[0].name}</p>:items.map(item=><button type="button" key={item.id} className="workspace-choice" disabled={saving} aria-pressed={selected===item.id} onClick={()=>{if(selected!==item.id&&discard()){onDirty(false);onOptionsDirty(false);setSelected(item.id);}}}>{item.name}</button>)}
      </div>
</div>
      {selected && (kind === "staff" || kind === "resources") && (
        <p className="mt-4 text-sm">Follows the location opening hours. No separate availability schedule is needed. Existing bookings and resource capacity still prevent double bookings.</p>
      )}
      {selected && (kind === "locations" || kind === "services") && (
        <Editor
          timezone={business.timezone}
          key={`schedule:${selected}`}
          path={`${root}/${selected}/${kind === "locations" ? "schedule" : kind === "services" ? "scheduling-rules" : "availability-windows"}`}
          kind={kind}
          writable={writable}
          dirty={dirty}
          discard={discard}
          onDirty={onDirty}
          saving={saving}
          onSaving={onSaving}
        />
      )}
      {selected && kind === "services" && items.find(item=>item.id===selected)?.all_day && <p className="schedule-notice">Whole day &middot; Opening to closing. Customers choose a date; branch hours determine the start and end times.</p>}
      {selected && kind === "services" && !items.find(item=>item.id===selected)?.all_day && (
        <BookingOptionsEditor
          onDirty={onOptionsDirty}
          key={`options:${selected}`}
          path={`${root}/${selected}/booking-options`}
          writable={writable}
        />
      )}
    </div>
  );
}
function Editor({
  timezone,
  path,
  kind,
  writable,
  dirty,
  discard,
  onDirty,
  saving,
  onSaving,
}: {
  timezone: string;
  path: string;
  kind: Kind;
  writable: boolean;
  dirty: boolean;
  discard: () => boolean;
  onDirty: (value: boolean) => void;
  saving: boolean;
  onSaving: (value: boolean) => void;
}) {
  const [zone, setZone] = useState(timezone);
  const formRef = useRef<HTMLFormElement>(null);
  const [data, setData] = useState<Data | null>(null);
  const [message, setMessage] = useState("Loading saved configuration...");
  const [version, setVersion] = useState(0);
  const [loading, setLoading] = useState(true);
  const [blocked, setBlocked] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    api<Data>(path, { signal: controller.signal })
      .then((result) => {
        setData(result.data);
        setMessage("");
        setLoading(false);
        setBlocked(false);
      })
      .catch((error) => {
        if (!controller.signal.aborted) {
          setMessage(readable(error));
          setLoading(false);
          setBlocked(true);
        }
      });
    return () => controller.abort();
  }, [path, version]);
  usePendingSave(
    path,
    "Scheduling",
    dirty && writable,
    saving || loading || blocked,
    formRef,
    save,
  );
  async function save() {
    if (!formRef.current) throw new Error("Scheduling has not loaded.");
    const values = new FormData(formRef.current);
    let body: object;
    if (kind === "locations") {
      const schedule = data as Schedule;
      const weekly_hours = schedule.weekly_hours.map((_, i) => ({
        weekday: Number(values.get(`day-${i}`)),
        start_minute: minutes(String(values.get(`start-${i}`))),
        end_minute: minutes(String(values.get(`end-${i}`))),
      }));
      const sorted = [...weekly_hours].sort(
        (a, b) => a.weekday - b.weekday || a.start_minute - b.start_minute,
      );
      if (
        sorted.some(
          (h, i) =>
            h.start_minute >= h.end_minute ||
            (i > 0 &&
              sorted[i - 1].weekday === h.weekday &&
              sorted[i - 1].end_minute > h.start_minute),
        )
      ) {
        throw new Error(
          "Hours must end after they start and must not overlap on the same day.",
        );
      }
      const closed_dates = schedule.closed_dates.map((_, i) =>
        String(values.get(`closed-${i}`)),
      );
      if (new Set(closed_dates).size !== closed_dates.length) {
        throw new Error("Each closed date must be unique.");
      }
      body = { weekly_hours, closed_dates };
    } else if (kind === "services") {
      body = Object.fromEntries(
        [
          "buffer_before_minutes",
          "buffer_after_minutes",
          "minimum_lead_minutes",
        ].map((key) => [key, Number(values.get(key))]),
      );
    } else {
      const windows = (data as Window[]).map((_, i) => ({
        starts_at: localToInstant(String(values.get(`from-${i}`)), zone),
        ends_at: localToInstant(String(values.get(`until-${i}`)), zone),
      }));
      const sorted = [...windows].sort((a, b) =>
        a.starts_at.localeCompare(b.starts_at),
      );
      if (
        sorted.some(
          (w, i) =>
            !Number.isFinite(Date.parse(w.starts_at)) ||
            !Number.isFinite(Date.parse(w.ends_at)) ||
            Date.parse(w.ends_at) <= Date.parse(w.starts_at) ||
            Date.parse(w.ends_at) - Date.parse(w.starts_at) > 7 * 86400000 ||
            (i > 0 && sorted[i - 1].ends_at > w.starts_at),
        )
      ) {
        throw new Error(
          "Availability windows must be ordered start to end, last at most seven days and not overlap.",
        );
      }
      body = { windows };
    }
    if (new TextEncoder().encode(JSON.stringify(body)).length > 4096) {
      throw new Error(
        "Reduce the number of schedule entries to fit the save limit.",
      );
    }
    onSaving(true);
    setMessage("");
    try {
      const result = await api<Data>(path, {
        method: "PUT",
        body: JSON.stringify(body),
      });
      setData(result.data);
      onDirty(false);
      setMessage("Scheduling saved.");
    } catch (error) {
      if (error instanceof ApiError && [401, 403, 404].includes(error.status))
        setBlocked(true);
      throw new Error(
        `${readable(error)} Reload saved configuration if the outcome is uncertain.`,
      );
    } finally {
      onSaving(false);
    }
  }
  function change(next: Data) {
    setData(next);
    onDirty(true);
  }
  const schedule = kind === "locations" ? (data as Schedule | null) : null;
  const rules = kind === "services" ? (data as Rules | null) : null;
  const windows = Array.isArray(data) ? data : null;
  return (
    <div className="mt-5">
      {message && (
        <p role="status" className="my-3 text-sm text-teal-900">
          {message}
        </p>
      )}
      {blocked && <button type="button" className="workspace-text-action" onClick={()=>setVersion(v=>v+1)}>Retry loading</button>}
      {data && (
        <form
          ref={formRef}
          onSubmit={(event) => {
            event.preventDefault();
            void save().catch((error) => setMessage(error.message));
          }}
          className="mt-4"
          onChange={() => onDirty(true)}
        >
          <p className="mb-4 text-sm text-slate-600">
            {writable
              ? "Save your changes here to update when customers can book."
              : "Read-only access. Ask an owner or admin to change scheduling."}{" "}
            {dirty && "You have unsaved changes."}
          </p>
          <fieldset
            disabled={!writable || saving || loading || blocked}
            className="min-w-0 space-y-4"
          >
            {schedule && (
              <>
                <p className="font-medium">
                  Location timezone: {schedule.timezone}
                </p>
                <p className="text-sm">
                  Weekly hours use the location&apos;s local time. Use 24:00 for
                  the end of a day; split overnight hours across two days. Days
                  without hours are closed.
                </p>
                {schedule.weekly_hours.map((h, i) => (
                  <div
                    key={i}
                    className="schedule-hours-row"
                  >
                    <fieldset className="schedule-day"><legend className="sr-only">Day</legend><details><summary>{days[h.weekday]} <span aria-hidden="true">&#9998;</span></summary><input type="hidden" name={`day-${i}`} value={h.weekday}/><div className="workspace-choices workspace-weekdays" aria-label={`Day ${i+1}`}>
{days.map((day,n)=><button type="button" key={day} className="workspace-choice" aria-label={day} aria-pressed={h.weekday===n} onClick={(event)=>{change({...schedule,weekly_hours:schedule.weekly_hours.map((v,j)=>j===i?{...v,weekday:n}:v)});const picker=event.currentTarget.closest("details");if(picker)picker.open=false;}}>{day.slice(0,3)}</button>)}
</div></details></fieldset>
                    <label>
                      Opens
                      <input
                        aria-label={`Opens ${i + 1}`}
                        name={`start-${i}`}
                        required
                        pattern="([01][0-9]|2[0-3]):[0-5][0-9]"
                        placeholder="09:00"
                        className={input}
                        value={h.start_text ?? time(h.start_minute)}
                        onChange={(e) =>
                          change({
                            ...schedule,
                            weekly_hours: schedule.weekly_hours.map((v, j) =>
                              j === i
                                ? { ...v, start_text: e.target.value }
                                : v,
                            ),
                          })
                        }
                      />
                    </label>
                    <label>
                      Closes
                      <input
                        aria-label={`Closes ${i + 1}`}
                        name={`end-${i}`}
                        required
                        pattern="(([01][0-9]|2[0-3]):[0-5][0-9]|24:00)"
                        placeholder="17:00"
                        className={input}
                        value={h.end_text ?? time(h.end_minute)}
                        onChange={(e) =>
                          change({
                            ...schedule,
                            weekly_hours: schedule.weekly_hours.map((v, j) =>
                              j === i ? { ...v, end_text: e.target.value } : v,
                            ),
                          })
                        }
                      />
                    </label>
                    {writable && (
                      <button
                        type="button"
                        className="workspace-text-action"
                        onClick={() =>
                          change({
                            ...schedule,
                            weekly_hours: schedule.weekly_hours.filter(
                              (_, j) => j !== i,
                            ),
                          })
                        }
                      >
                        Remove<span className="sr-only"> hours {i + 1}</span>
                      </button>
                    )}
                  </div>
                ))}
                {writable && (
                  <button
                    type="button"
                    className="workspace-text-action"
                    disabled={schedule.weekly_hours.length >= 28}
                    onClick={() =>
                      change({
                        ...schedule,
                        weekly_hours: [
                          ...schedule.weekly_hours,
                          { weekday: 1, start_minute: 540, end_minute: 1020 },
                        ],
                      })
                    }
                  >
                    Add opening hours
                  </button>
                )}
<details className="schedule-advanced"><summary>Overnight opening hours</summary>                {writable && (
                  <button
                    type="button"
                    className="workspace-text-action"
                    onClick={() =>
                      change({
                        ...schedule,
                        weekly_hours: days.map((_, weekday) => ({
                          weekday,
                          start_minute: 0,
                          end_minute: 1440,
                        })),
                      })
                    }
                  >
                    Set all days to 24 hours
                  </button>
                )}
                <p className="text-sm">
                  Whole-day services use opening-to-closing hours on one date. Custom-length bookings must fit opening hours for their entire duration.
                </p>
</details>
                <h3 className="font-semibold">Closed dates</h3>
                <p className="text-sm">
                  Block holidays or other days when this location is closed.
                </p>
                {schedule.closed_dates.map((date, i) => (
                  <div
                    key={i}
                    className="flex min-w-0 flex-wrap items-end gap-3"
                  >
                    <label className="min-w-0 flex-1">
                      Closed date {i + 1}
                      <input
                        name={`closed-${i}`}
                        type="date"
                        min="0001-01-01"
                        max="9999-12-31"
                        required
                        className={input}
                        value={date}
                        onChange={(e) =>
                          change({
                            ...schedule,
                            closed_dates: schedule.closed_dates.map((v, j) =>
                              j === i ? e.target.value : v,
                            ),
                          })
                        }
                      />
                    </label>
                    {writable && (
                      <button
                        type="button"
                        className="workspace-text-action"
                        onClick={() =>
                          change({
                            ...schedule,
                            closed_dates: schedule.closed_dates.filter(
                              (_, j) => j !== i,
                            ),
                          })
                        }
                      >
                        Remove date {i + 1}
                      </button>
                    )}
                  </div>
                ))}
                {writable && (
                  <button
                    type="button"
                    className="workspace-text-action"
                    disabled={schedule.closed_dates.length >= 128}
                    onClick={() =>
                      change({
                        ...schedule,
                        closed_dates: [...schedule.closed_dates, ""],
                      })
                    }
                  >
                    Add closed date
                  </button>
                )}
                {!schedule.weekly_hours.length && (
                  <p>
                    No opening hours: this location is closed to new bookings.
                  </p>
                )}
              </>
            )}
            {rules && (
              <details className="rounded-lg border p-4"><summary className="cursor-pointer font-medium">Optional: preparation and advance notice</summary>
                <p>
                  {rules.configured
                    ? "Set time for preparation and cleanup, and how far ahead customers must book."
                    : "Optional: defaults are zero preparation, zero cleanup and no advance notice. Your opening hours already allow bookings."}
                </p>
                {(
                  [
                    [
                      "buffer_before_minutes",
                      "Preparation before service (minutes)",
                      1440,
                    ],
                    [
                      "buffer_after_minutes",
                      "Cleanup after service (minutes)",
                      1440,
                    ],
                    [
                      "minimum_lead_minutes",
                      "Minimum advance notice (minutes)",
                      525600,
                    ],
                  ] as const
                ).map(([key, label, max]) => (
                  <label key={key} className="block">
                    {label}
                    <input
                      name={key}
                      type="number"
                      required
                      min={0}
                      max={max}
                      step={1}
                      className={input}
                      value={rules[key]}
                      onChange={(e) =>
                        change({
                          ...rules,
                          [key]:
                            e.target.value === "" ? "" : Number(e.target.value),
                        } as Rules)
                      }
                    />
                  </label>
                ))}
              </details>
            )}
            {windows && (
              <>
                <label className="block">
                  Availability timezone
                  <select
                    className={input}
                    value={zone}
                    disabled={dirty}
                    onChange={(e) => setZone(e.target.value)}
                  >
                    {[...new Set([timezone, "UTC"])].map((z) => (
                      <option key={z}>{z}</option>
                    ))}
                  </select>
                </label>
                <p className="text-sm">
                  Choose the start and end of each period when this person or
                  resource is available. Times use {zone}. Each window can span
                  up to seven days; add more windows for later dates. Empty or
                  expired windows block bookings. Include preparation and
                  cleanup time.
                </p>
                {windows.map((w, i) => (
                  <div
                    key={i}
                    className="schedule-hours-row"
                  >
                    <label>
                      Available from
                      <input
                        aria-label={`Available from ${i + 1}`}
                        name={`from-${i}`}
                        type="datetime-local"
                        min="2000-01-01T00:00"
                        max="2100-12-31T23:59"
                        step={60}
                        required
                        className={input}
                        value={
                          w.starts_at.endsWith("Z") ||
                          /[+-]\d\d:\d\d$/.test(w.starts_at)
                            ? localDateTime(w.starts_at, zone)
                            : w.starts_at
                        }
                        onChange={(e) =>
                          change(
                            windows.map((v, j) =>
                              j === i ? { ...v, starts_at: e.target.value } : v,
                            ),
                          )
                        }
                      />
                    </label>
                    <label>
                      Available until
                      <input
                        aria-label={`Available until ${i + 1}`}
                        name={`until-${i}`}
                        type="datetime-local"
                        min="2000-01-01T00:00"
                        max="2100-12-31T23:59"
                        step={60}
                        required
                        className={input}
                        value={
                          w.ends_at.endsWith("Z") ||
                          /[+-]\d\d:\d\d$/.test(w.ends_at)
                            ? localDateTime(w.ends_at, zone)
                            : w.ends_at
                        }
                        onChange={(e) =>
                          change(
                            windows.map((v, j) =>
                              j === i ? { ...v, ends_at: e.target.value } : v,
                            ),
                          )
                        }
                      />
                    </label>
                    {writable && (
                      <button
                        type="button"
                        className="workspace-text-action"
                        onClick={() =>
                          change(windows.filter((_, j) => j !== i))
                        }
                      >
                        Remove window {i + 1}
                      </button>
                    )}
                  </div>
                ))}
                {!windows.length && (
                  <p>
                    Add availability so this person or resource can be used in
                    new bookings.
                  </p>
                )}
                {writable && (
                  <button
                    type="button"
                    className="workspace-text-action"
                    disabled={windows.length >= 32}
                    onClick={() =>
                      change([...windows, { starts_at: "", ends_at: "" }])
                    }
                  >
                    Add availability window
                  </button>
                )}
              </>
            )}
            {writable && dirty && (
              <div className="schedule-footer">
                <button className={button}>
                  {saving ? "Saving..." : "Save changes"}
                </button>
      <button
        type="button"
        className="workspace-text-action"
        disabled={saving || loading}
        onClick={() => {
          if (discard()) {
            onDirty(false);
            setLoading(true);
            setVersion((v) => v + 1);
          }
        }}
      >
        Reset to saved
      </button>
              </div>
            )}
          </fieldset>
        </form>
      )}
    </div>
  );
}
