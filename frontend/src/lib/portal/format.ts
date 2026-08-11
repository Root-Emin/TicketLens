/*
  Portal formatting.

  Deliberately separate from lib/staff/format.ts: that module measures
  everything against a frozen fixture timestamp so the server and client agree,
  which is right for a mock panel and wrong here — the portal reads live data
  and a customer expects "2 minutes ago" to mean two minutes ago.

  The trade is that these functions are only safe inside client components. Every
  portal screen is one, because the data arrives through React Query.
*/

export { formatDate, relativeTime } from "@/lib/utils";

const stamp = new Intl.DateTimeFormat("en-GB", {
  day: "numeric",
  month: "short",
  year: "numeric",
});

const clock = new Intl.DateTimeFormat("en-GB", {
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

/** longDate renders "4 Aug 2026" — used for a ticket's created date. */
export function longDate(iso: string): string {
  return stamp.format(new Date(iso));
}

/** timeOnly renders "14:05" — the timestamp under a message bubble. */
export function timeOnly(iso: string): string {
  return clock.format(new Date(iso));
}

/** dayHeading separates one day of a conversation from the next. */
export function dayHeading(iso: string): string {
  const date = new Date(iso);
  const today = new Date();
  const yesterday = new Date(today);
  yesterday.setDate(today.getDate() - 1);

  if (sameDay(date, today)) return "Today";
  if (sameDay(date, yesterday)) return "Yesterday";
  return stamp.format(date);
}

function sameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

/** duration renders a minute count as "1h 32m". */
export function duration(minutes: number): string {
  if (minutes < 60) return `${Math.round(minutes)}m`;
  const hours = Math.floor(minutes / 60);
  const rest = Math.round(minutes % 60);
  if (hours < 24) return rest === 0 ? `${hours}h` : `${hours}h ${rest}m`;
  const days = Math.floor(hours / 24);
  const restHours = hours % 24;
  return restHours === 0 ? `${days}d` : `${days}d ${restHours}h`;
}

/**
 * ticketReference renders the short id a customer quotes on the phone. The
 * backend has no human-facing ticket number, so the first block of the UUID
 * stands in — stable, unique enough to search on, and short enough to read out.
 */
export function ticketReference(id: string): string {
  return `#${id.split("-")[0].toUpperCase()}`;
}

/** initialsOf builds the two-letter avatar label for a person's name. */
export function initialsOf(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0].slice(0, 2).toUpperCase();
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}
