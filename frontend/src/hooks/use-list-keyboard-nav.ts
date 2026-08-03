"use client";

import { useCallback, useEffect, useRef } from "react";

/**
 * Arrow-key navigation for a list of links.
 *
 * Implements a roving tabindex over elements carrying `data-list-item`: one
 * stop for the whole list in the Tab order, then ↑/↓ (and j/k, for anyone who
 * lives in Linear) to move within it. Enter follows the focused link, which
 * anchors already do — the handler only manages focus.
 *
 * Returns a ref to spread onto the list container.
 */
export function useListKeyboardNav<T extends HTMLElement>() {
  const ref = useRef<T>(null);

  const items = useCallback(
    () =>
      Array.from(
        ref.current?.querySelectorAll<HTMLElement>("[data-list-item]") ?? [],
      ),
    [],
  );

  // Keep exactly one item tabbable: the selected one, or the first.
  useEffect(() => {
    const all = items();
    if (all.length === 0) return;

    const selected = all.findIndex(
      (el) => el.getAttribute("aria-current") === "true",
    );
    all.forEach((el, i) => {
      el.tabIndex = i === (selected === -1 ? 0 : selected) ? 0 : -1;
    });
  }, [items]);

  const onKeyDown = useCallback(
    (event: React.KeyboardEvent<T>) => {
      const isDown = event.key === "ArrowDown" || event.key === "j";
      const isUp = event.key === "ArrowUp" || event.key === "k";
      const isHome = event.key === "Home";
      const isEnd = event.key === "End";
      if (!isDown && !isUp && !isHome && !isEnd) return;

      const all = items();
      if (all.length === 0) return;

      const active = document.activeElement as HTMLElement | null;
      const current = all.findIndex((el) => el === active || el.contains(active));

      let next: number;
      if (isHome) next = 0;
      else if (isEnd) next = all.length - 1;
      else if (current === -1) next = 0;
      else next = current + (isDown ? 1 : -1);

      if (next < 0 || next >= all.length) return;

      event.preventDefault();
      all.forEach((el, i) => {
        el.tabIndex = i === next ? 0 : -1;
      });
      all[next].focus();
    },
    [items],
  );

  return { ref, onKeyDown };
}
