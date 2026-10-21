"use client";

import { useEffect, useRef, useState } from "react";
import { api, ApiError, Business, CatalogItem, readable } from "../bookings/api";
import ServiceDuration from "./service-duration";
import { usePendingSave } from "./drafts";
import ServiceBranches from "./service-branches";
import Scheduling from "./scheduling";
import Edit from "./edit";
import Settings, {Team} from "./settings";
import Publication from "./publication";
import BusinessImage from "./business-image";

import { button, input, panel } from "./styles";

type Item = CatalogItem & { duration_minutes?: number; all_day?:boolean };
const kinds = ["locations", "services", "staff", "resources"] as const;
type Kind = (typeof kinds)[number];
const labels: Record<Kind, string> = {
  locations: "Locations",
  services: "Services",
  staff: "Staff",
  resources: "Resources",
};

export default function Catalog({
  business,
  onBusinessUpdated,
}: {
  business: Business;
  onBusinessUpdated: (business: Business) => void;
}) {

  const [current, setCurrent] = useState<Business | null>(null);
  const [message, setMessage] = useState("Loading access...");
  const [revision, setRevision] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    api<Business>(`/api/v1/businesses/${business.id}`, {
      signal: controller.signal,
    })
      .then((result) => {
        setCurrent(result.data);
        setMessage("");
      })
      .catch((error) => {
        if (!controller.signal.aborted) setMessage(readable(error));
      });
    return () => controller.abort();
  }, [business.id]);
  if (!current) return <p role="status">{message}</p>;
  async function refreshProfile() {
    try {
      const result = await api<Business>(`/api/v1/businesses/${business.id}`);
      setCurrent(result.data);
      onBusinessUpdated(result.data);
    } catch (error) {
      setMessage(readable(error));
    }
  }
  const writable = ["BUSINESS_OWNER", "BUSINESS_ADMIN"].includes(current.role);
  return (
    <>
      {message && <p role="status">{message}</p>}
      <section className={panel}>
        <h2 className="break-words text-2xl font-semibold">{current.name}</h2>
        <p className="mt-2 text-slate-600">
          {current.timezone} | {current.role.toLowerCase().replaceAll("_", " ")}
        </p>
        <p className="mt-3 text-sm">
          {writable
            ? "Manage your services, branches, people and opening hours."
            : "You have read-only catalog access. Ask an owner or admin to make changes."}
        </p>
        <p className="mt-2 text-sm text-slate-600">
          Connect each service to a branch, set opening hours, then publish your profile.
        </p>
        {writable && (
          <Edit
            path={`/api/v1/businesses/${current.id}`}
            label="business timezone"
            kind="business"
            onSaved={() => void refreshProfile()}
          />
        )}
      </section>
      <nav className="settings-shortcuts" aria-label="Settings sections">{["Services & branches","People & access","Hours & availability",...(writable?["Profile & publishing"]:[])].map((label,index)=><button key={label} onClick={()=>{const section=document.getElementById(`settings-${index}`) as HTMLDetailsElement|null;if(section){section.open=true;section.querySelector("summary")?.focus();section.scrollIntoView({block:"start"});}}}>{label}<span aria-hidden="true"> &#8599;</span></button>)}</nav>
      <div className="settings-sections">
      <details id="settings-0" className="settings-group" open><summary><span className="section-number">01</span><div><strong>Services &amp; branches</strong><p>What customers can book, and where.</p></div><span className="disclosure-mark" aria-hidden="true">+</span></summary><div className="settings-body grid items-start gap-x-6 xl:grid-cols-2">
        {kinds.filter(kind=>kind==="locations"||kind==="services").map((kind) => (
          <CatalogSection
            key={kind}
            business={current}
            kind={kind}
            revision={revision}
            writable={writable}
            onCreated={() => setRevision((v) => v + 1)}
          />
        ))}
      </div>
      </details>
      <details id="settings-1" className="settings-group"><summary><span className="section-number">02</span><div><strong>People &amp; access</strong><p>Your staff details and who can manage this business.</p></div><span className="disclosure-mark" aria-hidden="true">+</span></summary><div className="settings-body"><CatalogSection business={current} kind="staff" writable={writable} onCreated={()=>setRevision(v=>v+1)}/>{writable && <Team business={current}/>}</div></details>
      <details id="settings-2" className="settings-group"><summary><span className="section-number">03</span><div><strong>Hours &amp; availability</strong><p>Opening hours, booking lengths, and preparation time.</p></div><span className="disclosure-mark" aria-hidden="true">+</span></summary><div className="settings-body"><Scheduling business={current} writable={writable}/></div></details>
      {writable && <details id="settings-3" className="settings-group"><summary><span className="section-number">04</span><div><strong>Profile &amp; publishing</strong><p>Review what customers see before it goes live.</p></div><span className="disclosure-mark" aria-hidden="true">+</span></summary><div className="settings-body"><BusinessImage id={current.id}/><Settings business={current}/><Publication business={current}/></div></details>}
      </div>
    </>
  );
}

function CatalogSection({
  business,
  kind,
  writable,
  onCreated,
  revision=0,
}: {
  business: Business;
  kind: Kind;
  writable: boolean;
  onCreated: () => void;
  revision?: number;
}) {
  const root = `/api/v1/businesses/${business.id}/catalog`;
  const formRef = useRef<HTMLFormElement>(null);
  const [dirty, setDirty] = useState(false);
  const [items, setItems] = useState<Item[]>([]);
  const [deleting, setDeleting] = useState<string | null>(null);
  const [cursor, setCursor] = useState("");
  const [locations, setLocations] = useState<Item[]>([]);
  const [locationCursor, setLocationCursor] = useState("");
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("Loading...");
  const [loaded, setLoaded] = useState(false);
  useEffect(() => {
    const controller = new AbortController();
    api<Item[]>(`${root}/${kind}?limit=25`, { signal: controller.signal })
      .then((result) => {
        setItems(result.data);
        setCursor(result.cursor);
        setLoaded(true);
        setMessage("");
      })
      .catch((error) => {
        if (!controller.signal.aborted) setMessage(readable(error));
      });
    return () => controller.abort();
  }, [root, kind]);
  async function refresh(more = false) {
    setBusy(true);
    try {
      const result = await api<Item[]>(
        `${root}/${kind}?limit=25${more ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
      );
      setItems((previous) =>
        more
          ? [
              ...previous,
              ...result.data.filter(
                (item) => !previous.some((old) => old.id === item.id),
              ),
            ]
          : result.data,
      );
      setCursor(result.cursor);
      setLoaded(true);
      setMessage("");
    } catch (error) {
      setMessage(readable(error));
    } finally {
      setBusy(false);
    }
  }
  async function remove(item: Item) {
    setBusy(true);setMessage("");
    try { await api(`${root}/${kind}/${item.id}`,{method:"DELETE"});setItems(previous=>previous.filter(v=>v.id!==item.id));setDeleting(null);setMessage(`${item.name} deleted. Booking history is kept. Publish your reviewed profile to update the public listing.`);onCreated(); }
    catch(error){setMessage(error instanceof ApiError && error.status===409 ? "This item has upcoming bookings. Cancel or move those bookings before deleting it." : readable(error));}
    finally{setBusy(false);}
  }
  async function loadLocations(more = false) {
    setBusy(true);
    try {
      const result = await api<Item[]>(
        `${root}/locations?limit=100${more ? `&cursor=${encodeURIComponent(locationCursor)}` : ""}`,
      );
      setLocations((previous) =>
        more ? [...previous, ...result.data] : result.data,
      );
      setLocationCursor(result.cursor);
      setMessage(
        result.data.length
          ? "Choose the resource location below."
          : "Create a location first, then load locations again.",
      );
    } catch (error) {
      setMessage(readable(error));
    } finally {
      setBusy(false);
    }
  }
  usePendingSave(
    `${root}/${kind}/new`,
    `New ${kind}`,
    dirty,
    busy,
    formRef,
    create,
  );
  async function create() {
    const form = formRef.current;
    if (!form) throw new Error("Form unavailable");
    const values = new FormData(form);
    const body: Record<string, string | number | boolean> = {
      name: String(values.get("name")).trim(),
    };
    if (kind === "locations") {
      body.timezone = String(values.get("timezone")).trim();
      try {
        new Intl.DateTimeFormat("en", { timeZone: body.timezone }).format();
      } catch {
        throw new Error("Choose a valid timezone.");
      }
    }
    if (kind === "staff")
      body.role_name = String(values.get("role_name") ?? "").trim();
    if (kind === "services") {body.duration_minutes = Number(values.get("duration"));body.all_day=values.get("all_day")==="true";}
    if (kind === "resources") {
      body.location_id = String(values.get("location"));
      body.capacity = Number(values.get("capacity"));
    }
    setBusy(true);
    setMessage("");
    try {
      const result = await api<Item>(`${root}/${kind}`, {
        method: "POST",
        body: JSON.stringify(body),
      });
      setItems((previous) => [result.data, ...previous]);
      form.reset();
      setDirty(false);
      setMessage(`${result.data.name} created.`);
      onCreated();
    } catch (error) {
      throw new Error(
        `${readable(error)} If the response was interrupted, refresh this list before submitting again to avoid duplicates.`,
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <section className={panel} aria-label={labels[kind]}>
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="text-xl font-semibold">{labels[kind]}</h2>
        <button
          type="button"
          disabled={busy}
          onClick={() => refresh()}
          className="rounded-lg border px-3 py-2 text-sm"
        >
          Refresh {kind}
        </button>
      </div>
      <p className="mt-3 text-sm text-slate-600">
        {
          {
            locations:
              "Add the places where customers receive your services, such as a branch, studio or online location.",
            services:
              "Add what customers can book, such as a haircut, lesson or consultation, and how long it takes.",
            staff:
              "Add the people who provide your services. Opening hours control customer booking times; assign staff to the services they provide.",
            resources:
              "Add rooms, chairs, vehicles or equipment needed for a booking. Capacity is how many bookings can use one at the same time.",
          }[kind]
        }
      </p>
      {message && (
        <p role="status" className="my-3 break-words text-sm text-teal-900">
          {message}
        </p>
      )}
      {loaded && !items.length && (
        <p className="my-4 text-sm text-slate-500">No {kind} yet.</p>
      )}
      <ul className="workspace-catalog-list">
        {items.map((item) => (
          <li key={item.id} className="workspace-catalog-card">
            <p className="font-medium">{item.name}</p>
            <p className="text-sm text-slate-500">
              {(item.all_day?"Whole day":"") || item.role_name ||
                item.timezone ||
                (item.duration_minutes
                  ? `${item.duration_minutes} minutes`
                  : item.capacity
                    ? `Capacity: ${item.capacity}`
                    : "Bookable staff profile")}
            </p>
            {kind==="services"&&<ServiceBranches businessID={business.id} serviceID={item.id} writable={writable} revision={revision}/>}
            {writable && (
              <Edit
                path={`${root}/${kind}/${item.id}`}
                label={item.name}
                kind={kind}
                onSaved={() => {
                  void refresh();
                  onCreated();
                }}
              />
            )}
            {writable && <div className="workspace-delete">
              {deleting===item.id ? <div role="group" aria-label={`Delete ${item.name}`} className="workspace-delete-confirm"><p>Delete {item.name}? Past bookings stay in your history. Upcoming bookings must be moved or cancelled first.</p><div className="workspace-choices"><button type="button" className="workspace-danger" disabled={busy} onClick={()=>void remove(item)}>{busy?"Deleting...":"Confirm delete"}</button><button type="button" className="workspace-choice" disabled={busy} onClick={()=>setDeleting(null)}>Keep item</button></div></div> : <button type="button" className="workspace-danger" disabled={busy} onClick={()=>setDeleting(item.id)}>Delete {kind==="staff"?"person":kind==="locations"?"branch":kind.slice(0,-1)}</button>}
            </div>}
          </li>
        ))}
      </ul>
      {cursor && (
        <button
          type="button"
          disabled={busy}
          onClick={() => refresh(true)}
          className={`${button} mb-4`}
        >
          Load more {kind}
        </button>
      )}
      {writable && (
        <details>
          <summary className="cursor-pointer font-semibold text-teal-800">
            Add {kind === "staff" ? "staff profile" : kind.slice(0, -1)}
          </summary>
          {kind === "resources" && (
            <div className="mt-4 flex flex-wrap gap-3">
              <button
                type="button"
                disabled={busy}
                onClick={() => loadLocations()}
                className={button}
              >
                Load locations
              </button>
              {locationCursor && (
                <button
                  type="button"
                  disabled={busy}
                  onClick={() => loadLocations(true)}
                  className={button}
                >
                  Load more resource locations
                </button>
              )}
            </div>
          )}
          <form
            ref={formRef}
            onChange={() => setDirty(true)}
            onSubmit={(event) => {
              event.preventDefault();
              void create().catch((error) => setMessage(error.message));
            }}
            className="mt-4"
          >
            <fieldset disabled={busy} className="grid min-w-0 gap-4">
              <label>
                Name
                <input name="name" required maxLength={100} className={input} />
              </label>
              {kind === "staff" && (
                <label>
                  Staff role (optional)
                  <input
                    name="role_name"
                    maxLength={60}
                    placeholder="e.g. Barber, Tutor or Therapist"
                    className={input}
                  />
                  <span className="mt-1 block text-sm">
                    Use your own role names to group people in Assignments. This
                    does not grant account access.
                  </span>
                </label>
              )}
              {kind === "locations" && (
                <label>
                  Timezone
                  <input
                    name="timezone"
                    defaultValue={business.timezone}
                    required
                    maxLength={100}
                    className={input}
                  />
                </label>
              )}
              {kind === "services" && (
                <ServiceDuration/>

              )}
              {kind === "resources" && (
                <>
                  <label>
                    Resource location
                    <select
                      name="location"
                      required
                      defaultValue=""
                      className={input}
                    >
                      <option value="">Choose a location</option>
                      {locations.map((location) => (
                        <option key={location.id} value={location.id}>
                          {location.name}
                        </option>
                      ))}
                    </select>
                  </label>
                  <label>
                    Capacity
                    <input
                      name="capacity"
                      type="number"
                      required
                      defaultValue={1}
                      min={1}
                      max={2147483647}
                      step={1}
                      className={input}
                    />
                  </label>
                </>
              )}
              <button
                className={button}
                disabled={busy || (kind === "resources" && !locations.length)}
              >
                {busy
                  ? "Saving..."
                  : `Create ${kind === "staff" ? "staff profile" : kind.slice(0, -1)}`}
              </button>
              {dirty && (
                <button
                  type="button"
                  className="underline"
                  onClick={() => {
                    formRef.current?.reset();
                    setDirty(false);
                    setMessage("");
                  }}
                >
                  Clear new {kind === "staff" ? "person" : kind.slice(0, -1)}
                </button>
              )}
            </fieldset>
          </form>
        </details>
      )}
    </section>
  );
}
