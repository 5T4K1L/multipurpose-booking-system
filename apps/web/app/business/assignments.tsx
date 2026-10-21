"use client";

import { useEffect, useRef, useState } from "react";
import { api, Business, CatalogItem, readable } from "../bookings/api";
import { usePendingSave } from "./drafts";
import { button, input, panel } from "./styles";
const people = [
  {
    label: "People who provide a service",
    parent: "services",
    target: "staff",
    noun: "person",
  },
  {
    label: "Where a person works",
    parent: "staff",
    target: "locations",
    noun: "location",
  },
];
const serviceSetup = [
  {
    label: "Service locations",
    parent: "services",
    target: "locations",
    noun: "location",
  },
  {
    label: "Service resources",
    parent: "services",
    target: "resources",
    noun: "resource",
  },
];
type RelationType = (typeof people)[number];
async function all<T>(path: string, signal?: AbortSignal): Promise<T[]> {
  const items: T[] = [];
  let cursor = "";
  do {
    const result = await api<T[]>(
      `${path}?limit=100${cursor ? `&cursor=${encodeURIComponent(cursor)}` : ""}`,
      { signal },
    );
    items.push(...result.data);
    cursor = result.cursor;
  } while (cursor);
  return items;
}
export default function Assignments({
  business,
  writable,
  revision,
}: {
  business: Business;
  writable: boolean;
  revision: number;
}) {
  return (
    <>
      <Group
        business={business}
        writable={writable}
        revision={revision}
        peopleOnly
      />
      <Group
        business={business}
        writable={writable}
        revision={revision}
        peopleOnly={false}
      />
    </>
  );
}
function Group({
  business,
  writable,
  revision,
  peopleOnly,
}: {
  business: Business;
  writable: boolean;
  revision: number;
  peopleOnly: boolean;
}) {
  const [relation, setRelation] = useState(0);
  const [pending, setPending] = useState(false);
  const choices = peopleOnly ? people : serviceSetup;
  function discard() {
    return !pending || window.confirm("Discard this unsaved assignment?");
  }
  return (
    <section
      id={peopleOnly ? "people-assignments" : "service-setup"}
      className={panel}
      aria-label={peopleOnly ? "Assignments" : "Service setup"}
    >
      <h2 className="text-xl font-semibold">
        {peopleOnly ? "Assignments" : "Service setup"}
      </h2>
      <p className="mt-2 text-sm text-slate-600">
        {peopleOnly
          ? "Choose who provides each service and where they work. Add your own role names under Staff to group people, such as barbers or tutors."
          : "Choose where each service is offered and which rooms or equipment it needs. Every service needs at least one location before customers can book."}
      </p>
      <label className="mt-4 block">
        {peopleOnly ? "Assign people to" : "Set up"}
        <select
          aria-label={peopleOnly ? "Assignment type" : "Service setup type"}
          value={relation}
          onChange={(e) => {
            if (discard()) {
              setPending(false);
              setRelation(Number(e.target.value));
            }
          }}
          className={input}
        >
          {choices.map((item, index) => (
            <option key={item.label} value={index}>
              {item.label}
            </option>
          ))}
        </select>
      </label>
      <Relation
        key={relation}
        business={business}
        writable={writable}
        relation={choices[relation]}
        revision={revision}
        discard={discard}
        onPending={setPending}
      />
    </section>
  );
}
function Relation({
  business,
  writable,
  relation,
  revision,
  discard,
  onPending,
}: {
  business: Business;
  writable: boolean;
  relation: RelationType;
  revision: number;
  discard: () => boolean;
  onPending: (value: boolean) => void;
}) {
  const root = `/api/v1/businesses/${business.id}/catalog`;
  const [parents, setParents] = useState<CatalogItem[]>([]);
  const [targets, setTargets] = useState<CatalogItem[]>([]);
  const [parent, setParent] = useState("");
  const [role, setRole] = useState("");
  const [message, setMessage] = useState("Loading choices...");
  const [version, setVersion] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    Promise.all([
      all<CatalogItem>(`${root}/${relation.parent}`, controller.signal),
      all<CatalogItem>(`${root}/${relation.target}`, controller.signal),
    ])
      .then(([p, t]) => {
        setParents(p);
        setTargets(t);
        setMessage(
          p.length && t.length
            ? ""
            : `Add ${relation.parent === "staff" ? "a person under Staff" : "a service"} and ${relation.target === "staff" ? "a person under Staff" : `a ${relation.noun}`} above first.`,
        );
      })
      .catch((error) => {
        if (!controller.signal.aborted) setMessage(readable(error));
      });
    return () => controller.abort();
  }, [root, relation, revision, version]);
  const peopleChoices =
    relation.parent === "staff"
      ? parents
      : relation.target === "staff"
        ? targets
        : [];
  const roles = [
    ...new Set(
      peopleChoices
        .map((person) => person.role_name)
        .filter((value): value is string => !!value),
    ),
  ].sort();
  return (
    <div className="mt-4">
      {message && (
        <p role="status" className="my-3 text-sm">
          {message}
        </p>
      )}
      <button
        type="button"
        onClick={() => setVersion((v) => v + 1)}
        className="rounded-lg border px-3 py-2 text-sm"
      >
        Refresh choices
      </button>
      {!!roles.length && (
        <label className="mt-4 block">
          Staff role
          <select
            aria-label="Staff role"
            className={input}
            value={role}
            onChange={(e) => {
              if (discard()) {
                setRole(e.target.value);
                setParent("");
                onPending(false);
              }
            }}
          >
            <option value="">All people</option>
            {roles.map((value) => (
              <option key={value}>{value}</option>
            ))}
          </select>
        </label>
      )}
      <label className="mt-4 block">
        {relation.parent === "services" ? "Service" : "Person"}
        <select
          aria-label="Assignment parent"
          value={parent}
          onChange={(e) => {
            if (discard()) {
              setParent(e.target.value);
              onPending(false);
            }
          }}
          className={input}
        >
          <option value="">
            {relation.parent === "services"
              ? "Choose a service"
              : "Choose a person"}
          </option>
          {parents
            .filter(
              (person) =>
                relation.parent !== "staff" ||
                !role ||
                person.role_name === role,
            )
            .map((item) => (
              <option key={item.id} value={item.id}>
                {item.name}
                {item.role_name ? ` (${item.role_name})` : ""}
              </option>
            ))}
        </select>
      </label>
      {parent && (
        <Links
          key={parent}
          path={`${root}/${relation.parent}/${parent}/${relation.target}`}
          targets={targets}
          role={relation.target === "staff" ? role : ""}
          noun={relation.noun}
          writable={writable}
          onPending={onPending}
        />
      )}
    </div>
  );
}
function Links({
  path,
  targets,
  role,
  noun,
  writable,
  onPending,
}: {
  path: string;
  targets: CatalogItem[];
  role: string;
  noun: string;
  writable: boolean;
  onPending: (value: boolean) => void;
}) {
  const formRef = useRef<HTMLFormElement>(null);
  const [linked, setLinked] = useState<string[]>([]);
  const [target, setTarget] = useState("");
  const [loaded, setLoaded] = useState(false);
  const [busy, setBusy] = useState(false);
  const [message, setMessage] = useState("Loading assignments...");
  const [version, setVersion] = useState(0);
  useEffect(() => {
    const controller = new AbortController();
    all<string>(path, controller.signal)
      .then((ids) => {
        setLinked(ids);
        setLoaded(true);
        setMessage("");
      })
      .catch((error) => {
        if (!controller.signal.aborted) setMessage(readable(error));
      });
    return () => controller.abort();
  }, [path, version]);
  async function change(id: string, remove: boolean) {
    setBusy(true);
    try {
      await api(`${path}/${id}`, { method: remove ? "DELETE" : "PUT" });
      setLinked((previous) =>
        remove
          ? previous.filter((value) => value !== id)
          : [...new Set([...previous, id])],
      );
      setTarget("");
      onPending(false);
      setMessage(remove ? "Assignment removed." : "Assignment added.");
    } catch (error) {
      throw new Error(readable(error));
    } finally {
      setBusy(false);
    }
  }
  usePendingSave(
    path,
    `Assign ${noun}`,
    !!target && writable,
    busy || !loaded,
    formRef,
    () => change(target, false),
  );
  return (
    <div className="mt-4">
      {message && (
        <p role="status" className="my-3 text-sm text-teal-900">
          {message}
        </p>
      )}
      <button
        type="button"
        disabled={busy}
        onClick={() => setVersion((v) => v + 1)}
        className="rounded-lg border px-3 py-2 text-sm"
      >
        Refresh assignments
      </button>
      {loaded && !linked.length && (
        <p className="my-3 text-sm text-slate-500">
          No {noun === "person" ? "people" : `${noun}s`} assigned yet.
        </p>
      )}
      <ul className="my-3 divide-y divide-slate-100">
        {linked.map((id) => (
          <li
            key={id}
            className="flex flex-wrap items-center justify-between gap-3 py-3"
          >
            <span className="min-w-0 break-words">
              {targets.find((item) => item.id === id)?.name ??
                `Unavailable ${noun}`}
            </span>
            {writable && (
              <button
                disabled={busy}
                onClick={() =>
                  void change(id, true).catch((error) =>
                    setMessage(error.message),
                  )
                }
                aria-label={`Remove ${targets.find((item) => item.id === id)?.name ?? id}`}
                className="rounded-lg border px-3 py-2"
              >
                Remove
              </button>
            )}
          </li>
        ))}
      </ul>
      {writable && loaded && (
        <form
          ref={formRef}
          onSubmit={(event) => {
            event.preventDefault();
            if (target)
              void change(target, false).catch((error) =>
                setMessage(error.message),
              );
          }}
        >
          <fieldset
            disabled={busy}
            className="flex min-w-0 flex-wrap items-end gap-3"
          >
            <label className="min-w-0 flex-1">
              {noun === "person"
                ? "Person"
                : noun === "location"
                  ? "Location"
                  : "Resource"}
              <select
                aria-label={`Assign ${noun}`}
                required
                value={target}
                onChange={(e) => {
                  setTarget(e.target.value);
                  onPending(!!e.target.value);
                }}
                className={input}
              >
                <option value="">Choose a {noun}</option>
                {targets
                  .filter(
                    (item) =>
                      !linked.includes(item.id) &&
                      (!role || item.role_name === role),
                  )
                  .map((item) => (
                    <option key={item.id} value={item.id}>
                      {item.name}
                      {item.role_name ? ` (${item.role_name})` : ""}
                    </option>
                  ))}
              </select>
            </label>
            <button className={button} disabled={busy || !target}>
              Add assignment
            </button>
          </fieldset>
        </form>
      )}
    </div>
  );
}
