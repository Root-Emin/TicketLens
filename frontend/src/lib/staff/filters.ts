import { currentAgent } from "./fixtures";
import { minutesSince } from "./format";
import type {
  Paged,
  Priority,
  SortId,
  Ticket,
  TicketQuery,
  TicketStatus,
  ViewId,
} from "./types";

/*
  Every sidebar destination and every filter chip resolves to one call into
  selectTickets. There is one queue and many lenses onto it — the same model
  Linear and Zendesk use for saved views — so a new view is a predicate here
  rather than a new screen.

  The ranking helpers below are ported from the previous panel's data layer.
  They encode real triage rules and were worth keeping.
*/

/*
  Six rows a page. Sized for the queue a single team actually carries — with
  visibility scoped to one department, a 12-row page never filled.
*/
export const PAGE_SIZE = 6;

/** Sort weight for priority. Higher is more urgent. */
const PRIORITY_RANK: Record<Priority, number> = { low: 0, medium: 1, high: 2 };

/** Statuses that still represent live work. */
const LIVE: TicketStatus[] = ["open", "pending", "on_hold"];

export function isLive(ticket: Ticket): boolean {
  return LIVE.includes(ticket.status);
}

/** awaitingReply is true when the customer wrote last and nobody answered. */
export function awaitingReply(ticket: Ticket): boolean {
  const visible = ticket.messages.filter((m) => m.kind !== "note");
  return visible.at(-1)?.kind === "customer";
}

/**
 * waitingSince is the moment the clock started running for staff: the last
 * time the customer wrote, falling back to when the ticket was opened.
 */
export function waitingSince(ticket: Ticket): string {
  const lastCustomer = [...ticket.messages]
    .reverse()
    .find((m) => m.kind === "customer");
  return lastCustomer?.sentAt ?? ticket.createdAt;
}

/** Past its response target. Measured against the frozen anchor, like everything else. */
export function slaBreached(ticket: Ticket): boolean {
  return minutesSince(ticket.slaDueAt) > 0;
}

/** Under an hour of SLA left, or already past it. */
export function slaAtRisk(ticket: Ticket): boolean {
  return minutesSince(ticket.slaDueAt) > -60;
}

/**
 * needsAttention is what the "Needs Attention" view and the dashboard card
 * both mean: live work where the clock is against us.
 */
export function needsAttention(ticket: Ticket): boolean {
  return isLive(ticket) && slaAtRisk(ticket) && awaitingReply(ticket);
}

/**
 * byPriority is the panel's default ordering: most urgent first, then whoever
 * is actually waiting on us, then whoever has been waiting longest.
 *
 * The middle key matters. Without it a ticket we already answered outranks one
 * the customer just wrote in about, purely because it is older — the opposite
 * of what the agent should pick up next.
 */
export function byPriority(a: Ticket, b: Ticket): number {
  const rank = PRIORITY_RANK[b.priority] - PRIORITY_RANK[a.priority];
  if (rank !== 0) return rank;

  const waiting = Number(awaitingReply(b)) - Number(awaitingReply(a));
  if (waiting !== 0) return waiting;

  return minutesSince(waitingSince(b)) - minutesSince(waitingSince(a));
}

/* ------------------------------------------------------------------ views */

const isMine = (t: Ticket) => t.assignee?.id === currentAgent.id;

const VIEW_PREDICATE: Record<ViewId, (t: Ticket) => boolean> = {
  assigned: (t) => isMine(t) && isLive(t),
  attention: needsAttention,
  waiting: (t) => t.status === "pending",
  resolved: (t) => isMine(t) && t.status === "resolved",
  all: () => true,
  unassigned: (t) => t.assignee === null && isLive(t),
  open: isLive,
  hold: (t) => t.status === "on_hold",
  closed: (t) => t.status === "closed" || t.status === "resolved",
};

export const VIEW_LABEL: Record<ViewId, string> = {
  assigned: "Assigned to Me",
  attention: "Needs Attention",
  waiting: "Waiting for Customer",
  resolved: "Resolved by Me",
  all: "All Tickets",
  unassigned: "Unassigned",
  open: "All Open",
  hold: "On Hold",
  closed: "Closed",
};

const VIEW_IDS = Object.keys(VIEW_PREDICATE) as ViewId[];

/** parseView coerces an untrusted URL param to a known view. */
export function parseView(value: string | undefined): ViewId {
  return VIEW_IDS.includes(value as ViewId) ? (value as ViewId) : "assigned";
}

const SORT_IDS: SortId[] = ["newest", "oldest", "priority", "sla"];

export function parseSort(value: string | undefined): SortId {
  return SORT_IDS.includes(value as SortId) ? (value as SortId) : "newest";
}

export const SORT_LABEL: Record<SortId, string> = {
  newest: "Newest",
  oldest: "Oldest",
  priority: "Priority",
  sla: "SLA",
};

/* --------------------------------------------------------------- sorting */

const SORTERS: Record<SortId, (a: Ticket, b: Ticket) => number> = {
  newest: (a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt),
  oldest: (a, b) => Date.parse(a.updatedAt) - Date.parse(b.updatedAt),
  priority: byPriority,
  sla: (a, b) => Date.parse(a.slaDueAt) - Date.parse(b.slaDueAt),
};

/* ------------------------------------------------------------- selection */

function matchesSearch(ticket: Ticket, q: string): boolean {
  const needle = q.trim().toLowerCase();
  if (!needle) return true;
  return (
    ticket.id.toLowerCase().includes(needle) ||
    ticket.subject.toLowerCase().includes(needle) ||
    ticket.customer.name.toLowerCase().includes(needle) ||
    ticket.customer.company.toLowerCase().includes(needle) ||
    ticket.tags.some((tag) => tag.includes(needle))
  );
}

/**
 * selectTickets applies view, search, facet filters and sorting, then pages the
 * result. Pure and synchronous — queries.ts wraps it for the async boundary so
 * the call sites do not change when a real backend lands.
 *
 * `all` is expected to be pre-scoped to the agent's team by queries.ts. This
 * function deliberately has no team filter of its own: a caller must not be
 * able to widen visibility by passing a different team.
 */
export function selectTickets(all: Ticket[], query: TicketQuery): Paged<Ticket> {
  const filtered = all
    .filter(VIEW_PREDICATE[query.view])
    .filter((t) => (query.priority ? t.priority === query.priority : true))
    .filter((t) => (query.status ? t.status === query.status : true))
    .filter((t) => matchesSearch(t, query.q ?? ""))
    .sort(SORTERS[query.sort]);

  const pageCount = Math.max(1, Math.ceil(filtered.length / PAGE_SIZE));
  const page = Math.min(Math.max(1, query.page), pageCount);
  const start = (page - 1) * PAGE_SIZE;

  return {
    items: filtered.slice(start, start + PAGE_SIZE),
    total: filtered.length,
    page,
    pageSize: PAGE_SIZE,
    pageCount,
  };
}

/** countsByView powers the sidebar badges. Derived, never hardcoded. */
export function countsByView(all: Ticket[]): Record<ViewId, number> {
  return VIEW_IDS.reduce(
    (acc, view) => {
      acc[view] = all.filter(VIEW_PREDICATE[view]).length;
      return acc;
    },
    {} as Record<ViewId, number>,
  );
}
