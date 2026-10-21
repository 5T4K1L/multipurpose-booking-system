"use client";

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ReactNode,
  type RefObject,
} from "react";
type Entry = {
  dirty: boolean;
  busy: boolean;
  label: string;
  form: HTMLFormElement | null;
  save: () => Promise<void>;
};
type Registry = {
  register: (id: string, get: () => Entry) => () => void;
  saveAll: (after?: () => Promise<void>) => Promise<void>;
  saving: boolean;
};
const Drafts = createContext<Registry | null>(null);
export function DraftProvider({ children }: { children: ReactNode }) {
  const entries = useRef(new Map<string, () => Entry>());
  const [saving, setSaving] = useState(false);
  const running = useRef(false);
  const register = useCallback((id: string, get: () => Entry) => {
    entries.current.set(id, get);
    return () => {
      if (entries.current.get(id) === get) entries.current.delete(id);
    };
  }, []);
  const saveAll = useCallback(async (after?: () => Promise<void>) => {
    if (running.current)
      throw new Error("Please wait for the current save to finish.");
    const pending = [...entries.current.values()]
      .map((get) => get())
      .filter((entry) => entry.dirty);
    for (const entry of pending) {
      if (entry.busy)
        throw new Error(`Wait for ${entry.label} to finish saving.`);
      if (!entry.form) throw new Error(`Reopen ${entry.label} and try again.`);
      if (!entry.form.checkValidity()) {
        for (
          let parent: HTMLElement | null = entry.form;
          parent;
          parent = parent.parentElement
        )
          if (parent instanceof HTMLDetailsElement) parent.open = true;
        entry.form.reportValidity();
        throw new Error(
          `Check the highlighted fields in ${entry.label}. Nothing was published.`,
        );
      }
    }
    running.current = true;
    setSaving(true);
    try {
      for (const entry of pending) {
        try {
          await entry.save();
        } catch (error) {
          throw new Error(
            `${entry.label}: ${error instanceof Error ? error.message : "could not save"}. Publishing stopped; earlier saves may have completed.`,
          );
        }
      }
      if (after) await after();
    } finally {
      running.current = false;
      setSaving(false);
    }
  }, []);
  return (
    <Drafts.Provider value={{ register, saveAll, saving }}>
      <div inert={saving || undefined}>{children}</div>
    </Drafts.Provider>
  );
}
export function useDrafts() {
  const value = useContext(Drafts);
  if (!value) throw new Error("Missing draft provider");
  return value;
}
export function usePendingSave(
  id: string,
  label: string,
  dirty: boolean,
  busy: boolean,
  form: RefObject<HTMLFormElement | null>,
  save: () => Promise<void>,
) {
  const { register } = useDrafts();
  const latest = useRef<Entry>({ label, dirty, busy, form: null, save });
  useLayoutEffect(() => {
    latest.current = { label, dirty, busy, form: form.current, save };
  });
  useEffect(() => register(id, () => latest.current), [id, register]);
}
