"use client";

import { useCallback, useMemo, useSyncExternalStore } from "react";

/*
  localStorage-backed state.

  localStorage is an external store, so it is read through useSyncExternalStore
  rather than an effect that calls setState — the effect version causes a
  cascading render and React's lint rules rightly reject it.

  The browser only fires `storage` for *other* tabs, so writes in this tab
  notify a local listener set as well. That also keeps two components reading
  the same key in step.
*/

const listeners = new Set<() => void>();

function emit() {
  for (const listener of listeners) listener();
}

function subscribe(onChange: () => void) {
  listeners.add(onChange);
  window.addEventListener("storage", onChange);
  return () => {
    listeners.delete(onChange);
    window.removeEventListener("storage", onChange);
  };
}

/**
 * Returns `[value, setValue]`, persisted under `key`.
 *
 * `initial` is used on the server and whenever the stored value is missing or
 * unparseable. Pass a stable reference — a fresh object literal each render
 * would defeat the memo below.
 */
export function usePersistentState<T>(
  key: string,
  initial: T,
): [T, (value: T) => void] {
  const getSnapshot = useCallback(
    () => window.localStorage.getItem(key),
    [key],
  );

  // The raw string is the snapshot: useSyncExternalStore compares snapshots by
  // identity, and a freshly parsed object would differ on every call.
  const raw = useSyncExternalStore(subscribe, getSnapshot, () => null);

  const value = useMemo(() => {
    if (raw === null) return initial;
    try {
      return JSON.parse(raw) as T;
    } catch {
      return initial;
    }
  }, [raw, initial]);

  const update = useCallback(
    (next: T) => {
      try {
        window.localStorage.setItem(key, JSON.stringify(next));
      } catch {
        // Private mode or quota: the session still works, it just will not
        // remember this preference next time.
      }
      emit();
    },
    [key],
  );

  return [value, update];
}
