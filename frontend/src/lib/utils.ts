import { clsx, type ClassValue } from "clsx";
import { extendTailwindMerge } from "tailwind-merge";

/*
  tailwind-merge has to be told about the staff panel's `text-ui-*` font-size
  scale. Left to its own devices it reads `text-ui-sm` as a *colour* — the same
  conflict group as `text-tl-rail-text` — and silently drops whichever came
  first. That turned every indented sidebar label transparent against the navy
  rail: the size class won, the colour class vanished, and the text inherited
  near-black from the shell.

  Registering the scale under font-size keeps size and colour in separate
  groups, so `cn("text-tl-rail-text", "text-ui-sm")` keeps both.
*/
const UI_TEXT_SIZES = [
  "ui-xs",
  "ui-sm",
  "ui-base",
  "ui-md",
  "ui-lg",
  "ui-xl",
  "ui-2xl",
];

const twMerge = extendTailwindMerge({
  extend: {
    classGroups: {
      "font-size": [{ text: UI_TEXT_SIZES }],
    },
  },
});

/** cn merges conditional class names and resolves Tailwind conflicts. */
export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/** formatDate renders an ISO timestamp as a short, locale-stable string. */
export function formatDate(iso: string): string {
  const d = new Date(iso);
  return d.toLocaleString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

/** relativeTime renders "3h ago" style deltas for recent activity. */
export function relativeTime(iso: string): string {
  const then = new Date(iso).getTime();
  const diff = Date.now() - then;
  const mins = Math.round(diff / 60000);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.round(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.round(hours / 24);
  if (days < 30) return `${days}d ago`;
  return formatDate(iso);
}

/** pct renders a 0..1 confidence as a whole percentage. */
export function pct(value: number): string {
  return `${Math.round(value * 100)}%`;
}
