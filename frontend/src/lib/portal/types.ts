import type { TicketListItem } from "@/lib/api/types";

/*
  View-level types for the customer portal.

  The wire types stay in lib/api/types.ts and are not re-declared here — these
  only describe the portal's own URL state, its derived numbers, and the two
  fields the list endpoint would need to grow for the customer screens to be
  complete.
*/

/** The status facet above the ticket list. `all` applies no status filter. */
export type PortalStatusFilter =
  | "all"
  | "open"
  | "in_progress"
  | "pending_customer"
  | "resolved";

export type PortalSort = "newest" | "oldest" | "updated";

/** Everything the tickets screen reads out of the URL. */
export interface PortalTicketQuery {
  q?: string;
  status: PortalStatusFilter;
  sort: PortalSort;
  page: number;
}

/**
 * A list item, plus two fields the portal can use if the backend supplies them.
 *
 * Both are optional on purpose: `GET /tickets` returns neither today, and every
 * consumer here degrades rather than inventing a value.
 *
 * - `snippet`: the opening message, truncated. There is no `description` column
 *   — the first ticket_message is the description — so only the API can produce
 *   this without an N+1 from the browser.
 * - `first_response_at`: when an agent first replied. Turns the dashboard's
 *   average from "time to resolution" into the "average response time" the
 *   customer actually cares about.
 */
export interface PortalTicketListItem extends TicketListItem {
  snippet?: string;
  first_response_at?: string | null;
}

/**
 * The four dashboard numbers, derived on the client from the customer's own
 * tickets — /stats/overview needs `stats:read`, which a customer never holds.
 */
export interface PortalStats {
  open: number;
  waitingReply: number;
  resolved: number;
  /** Average turnaround in minutes, or null when there is nothing to average. */
  averageMinutes: number | null;
  /** Which clock the average measures, so the card can label itself honestly. */
  averageBasis: "first_response" | "resolution" | "none";
}
