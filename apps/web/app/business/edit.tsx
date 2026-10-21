"use client";

import { useRef, useState } from "react";
import { api, ApiError, readable } from "../bookings/api";
import { button, input } from "./styles";
import ServiceDuration from "./service-duration";
import { usePendingSave } from "./drafts";
import timezones from "./timezones.json";

type Editable = {
  name: string;
  role_name?: string;
  timezone?: string;
  duration_minutes?: number;
  all_day?:boolean;
  capacity?: number;
};
export default function Edit({
  path,
  label,
  kind,
  onSaved,
}: {
  path: string;
  label: string;
  kind: string;
  onSaved: () => void;
}) {
  const formRef = useRef<HTMLFormElement>(null);
  const [dirty, setDirty] = useState(false);
  const [item, setItem] = useState<Editable | null>(null);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("");
  async function open() {
    setBusy(true);
    setMessage("");
    try {
      setItem((await api<Editable>(path)).data);
    } catch (error) {
      setMessage(readable(error));
    } finally {
      setBusy(false);
    }
  }
  usePendingSave(path, `Edit ${label}`, dirty && !!item, busy, formRef, save);
  async function save() {
    if (!formRef.current) throw new Error("Form unavailable");
    const form = new FormData(formRef.current);
    const body: Record<string, string | number | boolean> = {
      name: kind === "business" ? item!.name : String(form.get("name")).trim(),
    };
    if (kind === "business" || kind === "locations")
      body.timezone = String(form.get("timezone"));
    if (kind === "staff")
      body.role_name = String(form.get("role_name") ?? "").trim();
    if (kind === "services") {body.duration_minutes = Number(form.get("duration"));body.all_day=form.get("all_day")==="true";}
    if (kind === "resources") body.capacity = Number(form.get("capacity"));
    setBusy(true);
    setMessage("");
    try {
      await api(path, { method: "PUT", body: JSON.stringify(body) });
      setItem(null);
      setDirty(false);
      setMessage("Changes saved.");
      onSaved();
    } catch (error) {
      throw new Error(
        error instanceof ApiError && error.status === 409
          ? "This resource has booking history. Its capacity cannot be changed; you can still rename it."
          : `${readable(error)} Reload the item before retrying if the result is uncertain.`,
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <div className="mt-3">
      {!item && (
        <button
          type="button"
          className="rounded-lg border px-3 py-2 text-sm"
          disabled={busy}
          onClick={open}
        >
          Edit {label}
        </button>
      )}
      {message && (
        <p role="status" className="my-2 text-sm text-teal-900">
          {message}
        </p>
      )}
      {item && (
        <form
          aria-label={`Edit ${label}`}
          ref={formRef}
          onChange={() => setDirty(true)}
          onSubmit={(event) => {
            event.preventDefault();
            void save().catch((error) => setMessage(error.message));
          }}
          className="mt-3 rounded-xl border bg-slate-50 p-4"
        >
          <fieldset disabled={busy} className="grid min-w-0 gap-4">
            {kind === "business" ? <div><p className="text-sm text-slate-500">Business name</p><p className="font-semibold">{item.name}</p><p className="text-sm">Your business name is fixed after setup.</p></div> : <>
            <label>
              Name
              <input
                name="name"
                defaultValue={item.name}
                required
                maxLength={100}
                className={input}
              />
            </label>
            </>}
            {kind === "staff" && (
              <label>
                Staff role (optional)
                <input
                  name="role_name"
                  defaultValue={item.role_name ?? ""}
                  maxLength={60}
                  placeholder="e.g. Barber, Tutor or Therapist"
                  className={input}
                />
              </label>
            )}
            {(kind === "business" || kind === "locations") && (
              <>
                <label>
                  Timezone
                  <select
                    aria-label="Edit timezone"
                    name="timezone"
                    defaultValue={item.timezone}
                    className={input}
                  >
                    {[...new Set([item.timezone ?? "UTC", ...timezones])]
                      .sort()
                      .map((zone) => (
                        <option key={zone} value={zone}>
                          {zone.replaceAll("_", " ")}
                        </option>
                      ))}
                  </select>
                </label>
                <p className="text-sm">
                  {kind === "business"
                    ? "Business timezone is the default for new locations. Existing location timezones stay unchanged."
                    : "Changing timezone reinterprets this location's weekly hours and closed dates. Existing booking instants stay unchanged; review scheduling afterward."}
                </p>
              </>
            )}
            {kind === "services" && (
              <>
                <ServiceDuration allDay={item.all_day} minutes={item.duration_minutes}/>

                <p className="text-sm">
                  New bookings and reschedules use the new duration. Existing
                  bookings retain their saved times.
                </p>
              </>
            )}
            {kind === "resources" && (
              <>
                <label>
                  Capacity
                  <input
                    name="capacity"
                    type="number"
                    defaultValue={item.capacity}
                    required
                    min={1}
                    max={2147483647}
                    step={1}
                    className={input}
                  />
                </label>
                <p className="text-sm">
                  Capacity can change only before this resource has booking
                  history. Its location stays fixed.
                </p>
              </>
            )}
            <p className="text-sm">
              Saving replaces these fields. Another admin may have edited them
              since you opened this form.
            </p>
            <div className="flex flex-wrap gap-3">
              <button className={button}>
                {busy ? "Saving..." : "Save changes"}
              </button>
              <button
                type="button"
                className="rounded-lg border px-4 py-3"
                onClick={() => {
                  setItem(null);
                  setDirty(false);
                  setMessage("");
                }}
              >
                Cancel edit
              </button>
            </div>
          </fieldset>
        </form>
      )}
    </div>
  );
}
