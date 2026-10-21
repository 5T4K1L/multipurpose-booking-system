"use client";

import { useEffect, useRef, useState } from "react";
import { api, Business, readable } from "../bookings/api";
import { usePendingSave } from "./drafts";
import { button, input, panel } from "./styles";

type Details = {
  tagline: string;
  description: string;
  contact_email: string;
  contact_phone: string;
  website: string;
};
const fields = [
  ["tagline", "Short introduction", 120],
  ["description", "Business description", 2000],
  ["contact_email", "Business contact email", 254],
  ["contact_phone", "Business phone", 40],
  ["website", "Website (https://)", 500],
] as const;
export default function Settings({ business }: { business: Business }) {
  const formRef = useRef<HTMLFormElement>(null);
  const [saved, setSaved] = useState<Details | null>(null);
  const [draft, setDraft] = useState<Details | null>(null);
  const [busy, setBusy] = useState(true);
  const [message, setMessage] = useState("Loading customization...");
  const [version, setVersion] = useState(0);
  const writable = ["BUSINESS_OWNER", "BUSINESS_ADMIN"].includes(business.role);
  const path = `/api/v1/businesses/${business.id}/settings`;
  const dirty = JSON.stringify(draft) !== JSON.stringify(saved);
  useEffect(() => {
    const controller = new AbortController();
    api<Details>(path, { signal: controller.signal })
      .then((result) => {
        setSaved(result.data);
        setDraft(result.data);
        setMessage("");
      })
      .catch((error) => {
        if (!controller.signal.aborted) {
          setSaved(null);
          setDraft(null);
          setMessage(readable(error));
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy(false);
      });
    return () => controller.abort();
  }, [path, version]);
  usePendingSave(
    path,
    "Business details",
    writable && dirty,
    busy,
    formRef,
    save,
  );
  async function save() {
    if (!draft) throw new Error("Business details have not loaded.");
    if (new TextEncoder().encode(JSON.stringify(draft)).length > 4096) {
      throw new Error(
        "Shorten the description or contact details to fit the save limit.",
      );
    }
    setBusy(true);
    try {
      const result = await api<Details>(path, {
        method: "PUT",
        body: JSON.stringify(draft),
      });
      setSaved(result.data);
      setDraft(result.data);
      setMessage("Business details saved.");
    } catch (error) {
      throw new Error(
        `${readable(error)} Reload saved details if the result is uncertain.`,
      );
    } finally {
      setBusy(false);
    }
  }
  return (
    <>
      <section aria-label="Business customization" className={panel}>
        <h2 className="text-2xl font-semibold">Business customization</h2>
        <p className="my-3 text-sm">
          Tell customers about your business and how to contact you. Publish
          saves your pending edits and updates your public profile.
        </p>
        {message && (
          <p role="status" className="my-3 text-sm text-teal-900">
            {message}
          </p>
        )}
        <button
          className={button}
          disabled={busy}
          onClick={() => {
            if (
              !dirty ||
              window.confirm("Discard unsaved business details and reload?")
            ) {
              setBusy(true);
              setVersion((v) => v + 1);
            }
          }}
        >
          Reload saved details
        </button>
        {draft && (
          <form
            ref={formRef}
            onSubmit={(event) => {
              event.preventDefault();
              void save().catch((error) => setMessage(error.message));
            }}
            className="mt-4"
          >
            <fieldset
              disabled={busy || !writable}
              className="grid min-w-0 gap-4 sm:grid-cols-2"
            >
              {fields.map(([key, label, max]) => (
                <label
                  key={key}
                  className={
                    key === "description" ? "min-w-0 sm:col-span-2" : "min-w-0"
                  }
                >
                  {label}
                  {key === "description" ? (
                    <textarea
                      className={input}
                      rows={5}
                      maxLength={max}
                      value={draft[key]}
                      onChange={(e) =>
                        setDraft({ ...draft, [key]: e.target.value })
                      }
                    />
                  ) : (
                    <input
                      className={input}
                      type={
                        key === "contact_email"
                          ? "email"
                          : key === "website"
                            ? "url"
                            : "text"
                      }
                      maxLength={max}
                      value={draft[key]}
                      onChange={(e) =>
                        setDraft({ ...draft, [key]: e.target.value })
                      }
                    />
                  )}
                </label>
              ))}
              {writable && (
                <button className={button}>Save business details</button>
              )}
            </fieldset>
            <p className="mt-3 text-sm text-slate-500">
              {writable
                ? "Saving replaces all details above. Blank fields clear saved values. Save or publish before leaving this business."
                : "Read-only access. An owner or admin can edit these details."}{" "}
              {dirty && "Unsaved changes."}
            </p>
          </form>
        )}
        {saved && (
          <article
            aria-label="Saved business preview"
            className="mt-6 break-words rounded-xl border bg-slate-50 p-4"
          >
            <h3 className="text-lg font-semibold">{business.name}</h3>
            <p className="mt-2 font-medium">
              {saved.tagline || "No introduction saved."}
            </p>
            <p className="mt-3 whitespace-pre-wrap">{saved.description}</p>
            <p className="mt-3">{saved.contact_email}</p>
            <p>{saved.contact_phone}</p>
            <p>{saved.website}</p>
            <p className="mt-3 text-xs text-slate-500">
              Preview of saved details. Review and publish below to update your
              public page.
            </p>
          </article>
        )}
      </section>

    </>
  );
}

type Member = { user_id: number; role: string };
const roles = [
  ["BUSINESS_ADMIN", "Business admin"],
  ["MANAGER", "Manager (read only)"],
  ["STAFF", "Staff (read only)"],
];
export function Team({ business }: { business: Business }) {
  const formRef = useRef<HTMLFormElement>(null);
  const [dirty, setDirty] = useState(false);
  const root = `/api/v1/businesses/${business.id}/memberships`;
  const owner = business.role === "BUSINESS_OWNER";
  const [members, setMembers] = useState<Member[]>([]);
  const [cursor, setCursor] = useState("");
  const [busy, setBusy] = useState(true);
  const [message, setMessage] = useState("Loading team access...");
  const [version, setVersion] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    api<Member[]>(`${root}?limit=50`, { signal: controller.signal })
      .then((result) => {
        setMembers(result.data);
        setCursor(result.cursor);
        setMessage("");
      })
      .catch((error) => {
        if (!controller.signal.aborted) {
          setMembers([]);
          setMessage(readable(error));
        }
      })
      .finally(() => {
        if (!controller.signal.aborted) setBusy(false);
      });
    return () => controller.abort();
  }, [root, version]);
  async function more() {
    setBusy(true);
    try {
      const result = await api<Member[]>(
        `${root}?limit=50&cursor=${encodeURIComponent(cursor)}`,
      );
      setMembers((previous) => [
        ...previous,
        ...result.data.filter(
          (item) => !previous.some((old) => old.user_id === item.user_id),
        ),
      ]);
      setCursor(result.cursor);
    } catch (error) {
      setMessage(readable(error));
    } finally {
      setBusy(false);
    }
  }
  async function change(user: number, role: string, remove = false) {
    if (!Number.isSafeInteger(user) || user < 1)
      throw new Error("Enter a valid account ID.");
    if (
      !window.confirm(
        remove
          ? `Remove account ${user} from this business?`
          : `Give account ${user} the ${role} role in this business?`,
      )
    )
      throw new Error(
        "Team access was not confirmed. No access change was saved.",
      );
    setBusy(true);
    try {
      await api(`${root}/${user}`, {
        method: remove ? "DELETE" : "PUT",
        ...(remove ? {} : { body: JSON.stringify({ role }) }),
      });
      const result = await api<Member[]>(`${root}?limit=50`);
      setMembers(result.data);
      setCursor(result.cursor);
      setMessage(remove ? "Team access removed." : "Team access saved.");
    } catch (error) {
      throw new Error(
        `${readable(error)} Refresh team access to verify the current state.`,
      );
    } finally {
      setBusy(false);
    }
  }
  async function saveTeam() {
    if (!formRef.current) throw new Error("Team form unavailable");
    const values = new FormData(formRef.current);
    await change(Number(values.get("user")), String(values.get("role")));
    formRef.current.reset();
    setDirty(false);
  }
  usePendingSave(root, "Team access", owner && dirty, busy, formRef, saveTeam);
  return (
    <section aria-label="Team access" className={panel}>
      <h2 className="text-2xl font-semibold">Team access</h2>
      <p className="my-3 text-sm">
        Choose who can sign in to this workspace and what they can change.
        Adding someone under Staff does not give them account access.
      </p>
      <p className="mb-3 text-sm">
        Admins can manage services, schedules and bookings. Managers and staff
        can view them. Only the business owner can change access.
      </p>
      {message && (
        <p role="status" className="my-3 text-sm text-teal-900">
          {message}
        </p>
      )}
      <button
        disabled={busy}
        className={button}
        onClick={() => {
          setBusy(true);
          setVersion((v) => v + 1);
        }}
      >
        Refresh team access
      </button>
      <ul className="my-4 divide-y">
        {members.map((member) => (
          <li
            key={member.user_id}
            className="flex flex-wrap items-center justify-between gap-3 py-3"
          >
            <p>
              Account {member.user_id} /{" "}
              {member.role === "BUSINESS_OWNER"
                ? "Owner"
                : (roles.find(([value]) => value === member.role)?.[1] ??
                  member.role)}
            </p>
            {owner && member.role !== "BUSINESS_OWNER" && (
              <button
                className={button}
                disabled={busy}
                onClick={() =>
                  void change(member.user_id, member.role, true).catch(
                    (error) => setMessage(error.message),
                  )
                }
              >
                Remove account {member.user_id}
              </button>
            )}
          </li>
        ))}
      </ul>
      {cursor && (
        <button disabled={busy} className={`${button} mb-4`} onClick={more}>
          Load more members
        </button>
      )}
      {owner && (
        <form
          ref={formRef}
          onChange={() => setDirty(true)}
          onSubmit={(e) => {
            e.preventDefault();
            void saveTeam().catch((error) => setMessage(error.message));
          }}
        >
          <fieldset
            disabled={busy}
            className="grid min-w-0 items-end gap-4 sm:grid-cols-3"
          >
            <label>
              Existing account ID
              <input
                name="user"
                className={input}
                type="number"
                min={1}
                max={Number.MAX_SAFE_INTEGER}
                step={1}
                required
              />
            </label>
            <label>
              Access level
              <select
                aria-label="Business role"
                className={input}
                name="role"
                defaultValue="STAFF"
              >
                {roles.map(([value, label]) => (
                  <option key={value} value={value}>
                    {label}
                  </option>
                ))}
              </select>
            </label>
            <button className={button}>Add or update access</button>
          </fieldset>
          <p className="mt-3 text-sm">
            Ask the account holder for their ID from Your account. This grants
            access immediately to an existing active account; no invitation or
            email is sent.
          </p>
        </form>
      )}
    </section>
  );
}
