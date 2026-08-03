import "server-only";

import { currentAgent, notifications, tickets } from "./fixtures";
import {
  awaitingReply,
  countsByView,
  isLive,
  needsAttention,
  selectTickets,
  slaBreached,
} from "./filters";
import { minutesSince } from "./format";
import type {
  Agent,
  Notification,
  Paged,
  Ticket,
  TicketQuery,
  ViewId,
} from "./types";

/*
  The staff panel's data access layer.

  Screens never import fixtures.ts — they call these functions. Every read is
  async even though the data is in memory, so replacing them with fetches to
  the Go backend later touches this file and nothing else.

  `server-only` is deliberate: these run during render on the server, which is
  what lets the route segments stream with Suspense instead of shipping the
  whole queue to the browser.
*/

/**
 * Simulated latency. Small, but non-zero — without it the Suspense fallbacks
 * never paint and the skeleton states would be dead code we could not verify.
 */
const LATENCY_MS = 120;

/**
 * Every ticket the signed-in agent is allowed to see.
 *
 * Team membership is the visibility boundary: an agent works their own
 * department's queue and nothing else. This is the single choke point — every
 * read below goes through it, so there is no view, filter, URL or id lookup
 * that can reach another department's ticket.
 *
 * When the Go backend lands this becomes a server-side scope on the query. It
 * must never move into the components: a filter the client applies is a filter
 * the client can skip.
 */
function visibleTickets(): Ticket[] {
  return tickets.filter((ticket) => ticket.team === currentAgent.team);
}

/**
 * Notifications about tickets this agent cannot open are dropped.
 *
 * Otherwise the bell advertises another department's work and every click ends
 * in a 404 — the subject line alone is already more than they should see.
 */
function visibleNotifications(): Notification[] {
  const ids = new Set(visibleTickets().map((t) => t.id));
  return notifications.filter((n) => !n.ticketId || ids.has(n.ticketId));
}

function settle<T>(value: T, ms = LATENCY_MS): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), ms));
}

export async function getCurrentAgent(): Promise<Agent> {
  return settle(currentAgent, 0);
}

export async function listTickets(query: TicketQuery): Promise<Paged<Ticket>> {
  return settle(selectTickets(visibleTickets(), query));
}

/**
 * A ticket by id, or null when it belongs to another department.
 *
 * Returning null rather than the ticket is what turns a guessed URL into a 404
 * instead of a leak — the id is not a capability.
 */
export async function getTicket(id: string): Promise<Ticket | null> {
  return settle(visibleTickets().find((t) => t.id === id) ?? null);
}

/**
 * The unpaged queue, for the command palette. Deliberately not exposed to the
 * list UI — that one pages — but the palette searches everything, and a few
 * dozen rows is far cheaper to ship once than to query per keystroke.
 */
export async function listAllTickets(): Promise<Ticket[]> {
  return settle(
    [...visibleTickets()].sort(
      (a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt),
    ),
    0,
  );
}

/** The first ticket of a view — what /staff/tickets redirects into on desktop. */
export async function getFirstTicketId(
  query: TicketQuery,
): Promise<string | null> {
  const page = selectTickets(visibleTickets(), query);
  return settle(page.items[0]?.id ?? null, 0);
}

export interface SidebarCounts {
  views: Record<ViewId, number>;
  unreadNotifications: number;
}

export async function getSidebarCounts(): Promise<SidebarCounts> {
  return settle(
    {
      views: countsByView(visibleTickets()),
      unreadNotifications: visibleNotifications().filter((n) => !n.read).length,
    },
    0,
  );
}

export async function listNotifications(): Promise<Notification[]> {
  return settle(
    [...visibleNotifications()].sort(
      (a, b) => Date.parse(b.at) - Date.parse(a.at),
    ),
    0,
  );
}

/* ------------------------------------------------------------- dashboard */

export interface StatSummary {
  assigned: number;
  attention: number;
  waiting: number;
  resolvedToday: number;
  avgResponseMins: number;
  slaBreached: number;
  /** Normalised 0..1 sparkline samples, oldest first, per metric. */
  trends: Record<string, number[]>;
  /** Percentage change against the previous day, per metric. */
  deltas: Record<string, number>;
}

/**
 * Sparkline samples. Derived from the queue by bucketing tickets into the last
 * eight days, so the lines move when the fixtures do rather than being drawn
 * from invented numbers.
 */
function trendFor(predicate: (t: Ticket) => boolean): number[] {
  const buckets = new Array(8).fill(0);
  for (const ticket of visibleTickets()) {
    if (!predicate(ticket)) continue;
    const days = Math.floor(minutesSince(ticket.createdAt) / (24 * 60));
    const idx = 7 - Math.min(7, days);
    buckets[idx] += 1;
  }
  const max = Math.max(1, ...buckets);
  return buckets.map((v) => v / max);
}

export async function getStatSummary(): Promise<StatSummary> {
  // Scoped like every other read: the headline numbers describe this agent's
  // department, not the whole company.
  const visible = visibleTickets();
  const mine = visible.filter((t) => t.assignee?.id === currentAgent.id);
  const resolvedToday = visible.filter(
    (t) => t.status === "resolved" && minutesSince(t.updatedAt) < 24 * 60,
  );

  // Mean minutes between a customer message and the agent's next reply.
  const gaps: number[] = [];
  for (const ticket of visible) {
    for (let i = 1; i < ticket.messages.length; i += 1) {
      const prev = ticket.messages[i - 1];
      const cur = ticket.messages[i];
      if (prev.kind === "customer" && cur.kind === "agent") {
        gaps.push((Date.parse(cur.sentAt) - Date.parse(prev.sentAt)) / 60_000);
      }
    }
  }
  const avgResponseMins = gaps.length
    ? Math.round(gaps.reduce((a, b) => a + b, 0) / gaps.length)
    : 0;

  return settle({
    assigned: mine.filter(isLive).length,
    attention: visible.filter(needsAttention).length,
    waiting: visible.filter((t) => t.status === "pending").length,
    resolvedToday: resolvedToday.length,
    avgResponseMins,
    slaBreached: visible.filter((t) => isLive(t) && slaBreached(t)).length,
    trends: {
      assigned: trendFor((t) => t.assignee?.id === currentAgent.id),
      attention: trendFor(needsAttention),
      waiting: trendFor((t) => t.status === "pending"),
      resolvedToday: trendFor((t) => t.status === "resolved"),
      avgResponseMins: trendFor(() => true),
      slaBreached: trendFor(slaBreached),
    },
    // Fixed period-over-period figures: the fixture set is a single snapshot,
    // so there is no honest previous day to diff against.
    deltas: {
      assigned: 20,
      attention: 50,
      waiting: -12,
      resolvedToday: 38,
      avgResponseMins: -8,
      slaBreached: 100,
    },
  });
}

/** The queue the dashboard surfaces under "Needs attention". */
export async function listAttention(limit = 5): Promise<Ticket[]> {
  return settle(
    visibleTickets()
      .filter(needsAttention)
      .sort((a, b) => Date.parse(a.slaDueAt) - Date.parse(b.slaDueAt))
      .slice(0, limit),
  );
}

/** Most recently touched live tickets, for the dashboard activity feed. */
export async function listRecent(limit = 6): Promise<Ticket[]> {
  return settle(
    [...visibleTickets()]
      .filter(isLive)
      .sort((a, b) => Date.parse(b.updatedAt) - Date.parse(a.updatedAt))
      .slice(0, limit),
  );
}

export { awaitingReply, needsAttention, slaBreached };
