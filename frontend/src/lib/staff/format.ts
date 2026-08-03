import { NOW_ISO } from "./fixtures";

/*
  Formatting for the staff panel, in English.

  Everything is measured against NOW_ISO rather than Date.now() so the server
  and the client always produce the same string — a live clock here would cause
  hydration mismatches on every relative timestamp on the screen.

  The Intl formatters pin an explicit timeZone for the same reason: without it
  the server renders in UTC and the browser renders in local time.
*/

const NOW_MS = Date.parse(NOW_ISO);
const TZ = "Europe/Istanbul";

const clock = new Intl.DateTimeFormat("en-GB", {
  hour: "2-digit",
  minute: "2-digit",
  hour12: true,
  timeZone: TZ,
});

const dayAndTime = new Intl.DateTimeFormat("en-GB", {
  day: "numeric",
  month: "short",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
  hour12: true,
  timeZone: TZ,
});

const dayOnly = new Intl.DateTimeFormat("en-GB", {
  weekday: "long",
  day: "numeric",
  month: "long",
  timeZone: TZ,
});

/** minutesSince returns whole minutes elapsed since an ISO timestamp. */
export function minutesSince(iso: string): number {
  return Math.round((NOW_MS - Date.parse(iso)) / 60_000);
}

/** timeOnly renders "09:41 am" — used inside message bubbles. */
export function timeOnly(iso: string): string {
  return clock.format(new Date(iso)).toUpperCase();
}

/** fullDate renders "22 May 2024, 03:00 PM" — used for due dates. */
export function fullDate(iso: string): string {
  return dayAndTime.format(new Date(iso)).toUpperCase().replace(",", ",");
}

/** dayHeading renders "Monday, 3 August" — the separator between message days. */
export function dayHeading(iso: string): string {
  const mins = minutesSince(iso);
  if (mins < 24 * 60) return "Today";
  if (mins < 48 * 60) return "Yesterday";
  return dayOnly.format(new Date(iso));
}

/**
 * relativeAge renders a compact elapsed time: "5m ago", "3h ago", "2d ago".
 * Used in list columns where the number matters more than the sentence.
 */
export function relativeAge(iso: string): string {
  const mins = minutesSince(iso);
  if (mins < 1) return "just now";
  if (mins < 60) return `${mins}m ago`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h ago`;
  const days = Math.floor(hours / 24);
  if (days < 7) return `${days}d ago`;
  const weeks = Math.floor(days / 7);
  return `${weeks}w ago`;
}

/**
 * slaRemaining describes the SLA clock. `breached` flips the caller to the
 * danger treatment, and `ratio` (0..1, elapsed share of the window) drives the
 * progress bar.
 */
export function slaRemaining(
  createdAt: string,
  dueAt: string,
): { label: string; breached: boolean; ratio: number } {
  const due = Date.parse(dueAt);
  const start = Date.parse(createdAt);
  const remainingMins = Math.round((due - NOW_MS) / 60_000);
  const window = Math.max(1, due - start);
  const elapsed = Math.min(1, Math.max(0, (NOW_MS - start) / window));

  if (remainingMins < 0) {
    return {
      label: `Breached ${compact(-remainingMins)} ago`,
      breached: true,
      ratio: 1,
    };
  }

  return {
    label: `${compact(remainingMins)} left`,
    breached: false,
    ratio: elapsed,
  };
}

/** compact turns a minute count into "45m", "3h", "2d". */
function compact(mins: number): string {
  if (mins < 60) return `${mins}m`;
  const hours = Math.floor(mins / 60);
  if (hours < 24) return `${hours}h`;
  return `${Math.floor(hours / 24)}d`;
}

/** percent renders a 0..1 confidence as a whole percentage. */
export function percent(value: number): string {
  return `${Math.round(value * 100)}%`;
}

/** greeting picks the salutation for the frozen anchor's hour. */
export function greeting(): string {
  const hour = Number(
    new Intl.DateTimeFormat("en-GB", {
      hour: "numeric",
      hour12: false,
      timeZone: TZ,
    }).format(new Date(NOW_ISO)),
  );
  if (hour < 12) return "Good morning";
  if (hour < 18) return "Good afternoon";
  return "Good evening";
}

/** duration renders an average response time as "1h 32m". */
export function duration(mins: number): string {
  const hours = Math.floor(mins / 60);
  const rest = mins % 60;
  if (hours === 0) return `${rest}m`;
  return `${hours}h ${rest}m`;
}
