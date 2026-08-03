import { parseSort, parseView } from "./filters";
import type { Priority, TicketQuery, TicketStatus } from "./types";

/*
  The inbox state lives entirely in the URL.

  Note what is absent: there is no team parameter. Visibility is decided by the
  agent's own department in lib/staff/queries.ts, and accepting a team from the
  query string would invite exactly the cross-department access that scoping
  exists to prevent.

  That is what makes a filtered queue shareable, reload-safe and navigable with
  the browser's back button — none of which is true when filters are component
  state. Every control on the list builds a new href from the current query
  rather than mutating anything.
*/

type RawParams = Record<string, string | string[] | undefined>;

const first = (value: string | string[] | undefined): string | undefined =>
  Array.isArray(value) ? value[0] : value;

const PRIORITIES: Priority[] = ["low", "medium", "high"];
const STATUSES: TicketStatus[] = [
  "open",
  "pending",
  "on_hold",
  "resolved",
  "closed",
];

/** Coerces untrusted query params into a valid TicketQuery. */
export function parseTicketQuery(params: RawParams): TicketQuery {
  const priority = first(params.priority) as Priority | undefined;
  const status = first(params.status) as TicketStatus | undefined;
  const page = Number.parseInt(first(params.page) ?? "1", 10);

  return {
    view: parseView(first(params.view)),
    q: first(params.q)?.trim() || undefined,
    priority: priority && PRIORITIES.includes(priority) ? priority : undefined,
    status: status && STATUSES.includes(status) ? status : undefined,
    sort: parseSort(first(params.sort)),
    page: Number.isFinite(page) && page > 0 ? page : 1,
  };
}

/**
 * Serialises a query back to a query string, omitting defaults so the common
 * case stays a short, readable URL.
 */
export function ticketQueryString(query: TicketQuery): string {
  const params = new URLSearchParams();

  if (query.view !== "assigned") params.set("view", query.view);
  if (query.q) params.set("q", query.q);
  if (query.priority) params.set("priority", query.priority);
  if (query.status) params.set("status", query.status);
  if (query.sort !== "newest") params.set("sort", query.sort);
  if (query.page > 1) params.set("page", String(query.page));

  const s = params.toString();
  return s ? `?${s}` : "";
}

/**
 * Builds an href with some fields replaced.
 *
 * Any change other than paging resets to page 1 — landing on page 4 of a
 * freshly filtered list that only has two pages is a small bug that feels like
 * a broken filter.
 */
export function withQuery(
  base: string,
  query: TicketQuery,
  patch: Partial<TicketQuery>,
): string {
  const next: TicketQuery = { ...query, ...patch };
  if (patch.page === undefined) next.page = 1;
  return `${base}${ticketQueryString(next)}`;
}

/** Preserves the current filters when opening a ticket. */
export function ticketHref(id: string, query: TicketQuery): string {
  return `/staff/tickets/${id}${ticketQueryString(query)}`;
}

export const inboxHref = (query: TicketQuery) =>
  `/staff/tickets${ticketQueryString(query)}`;
