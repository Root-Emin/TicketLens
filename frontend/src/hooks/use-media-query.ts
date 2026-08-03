"use client";

import { useCallback, useSyncExternalStore } from "react";

/**
 * Tracks a media query.
 *
 * Built on useSyncExternalStore rather than useEffect + setState: matchMedia is
 * an external store, and this is the API React provides for reading one without
 * the cascading render an effect would cause. It also gives a defined server
 * snapshot, so hydration matches instead of flipping on the first commit.
 *
 * Returns false on the server. Anything that would look wrong for that first
 * frame belongs in CSS — use this for behaviour, not layout.
 */
export function useMediaQuery(query: string): boolean {
  const subscribe = useCallback(
    (onChange: () => void) => {
      const list = window.matchMedia(query);
      list.addEventListener("change", onChange);
      return () => list.removeEventListener("change", onChange);
    },
    [query],
  );

  const getSnapshot = useCallback(
    () => window.matchMedia(query).matches,
    [query],
  );

  return useSyncExternalStore(subscribe, getSnapshot, () => false);
}

/** Tailwind's breakpoints, so JS and CSS never disagree about where they sit. */
export const BREAKPOINT = {
  md: "(min-width: 768px)",
  lg: "(min-width: 1024px)",
  xl: "(min-width: 1280px)",
  "2xl": "(min-width: 1536px)",
} as const;
