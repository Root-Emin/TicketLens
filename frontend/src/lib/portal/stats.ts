import type { PortalStats, PortalTicketListItem } from "./types";

/*
  The dashboard's four numbers, computed from the customer's own tickets.

  /stats/overview is gated on `stats:read`, which the contract grants to Owner
  and Super Admin only, so the portal cannot read it. Counting a page of the
  customer's tickets is both permitted and exact for anyone with fewer than a
  hundred of them; past that the average is a sample, which is why the caller
  pulls the newest page rather than a random one.
*/

/** How the "average" card is labelled, given what data was available. */
export const AVERAGE_LABEL: Record<PortalStats["averageBasis"], string> = {
  first_response: "Avg. Response Time",
  resolution: "Avg. Resolution Time",
  none: "Avg. Response Time",
};

export function computeStats(tickets: PortalTicketListItem[]): PortalStats {
  let open = 0;
  let waitingReply = 0;
  let resolved = 0;

  for (const ticket of tickets) {
    switch (ticket.status) {
      case "open":
      case "in_progress":
        open += 1;
        break;
      case "pending_customer":
        waitingReply += 1;
        break;
      case "resolved":
      case "closed":
        resolved += 1;
        break;
    }
  }

  return { open, waitingReply, resolved, ...average(tickets) };
}

/**
 * Prefers time-to-first-reply and falls back to time-to-resolution, because
 * only the second can be derived from what `GET /tickets` returns today. The
 * basis travels with the number so the card never claims to measure a clock it
 * did not read.
 */
function average(
  tickets: PortalTicketListItem[],
): Pick<PortalStats, "averageMinutes" | "averageBasis"> {
  const responses = spans(tickets, (t) => t.first_response_at ?? undefined);
  if (responses.length > 0) {
    return {
      averageMinutes: mean(responses),
      averageBasis: "first_response",
    };
  }

  const resolutions = spans(tickets, (t) => t.resolved_at);
  if (resolutions.length > 0) {
    return { averageMinutes: mean(resolutions), averageBasis: "resolution" };
  }

  return { averageMinutes: null, averageBasis: "none" };
}

/** Minutes from creation to `end`, for every ticket that has one. */
function spans(
  tickets: PortalTicketListItem[],
  end: (ticket: PortalTicketListItem) => string | undefined,
): number[] {
  const out: number[] = [];
  for (const ticket of tickets) {
    const to = end(ticket);
    if (!to) continue;
    const minutes = (Date.parse(to) - Date.parse(ticket.created_at)) / 60_000;
    // A negative or unparseable span is bad data, not a fast reply.
    if (Number.isFinite(minutes) && minutes >= 0) out.push(minutes);
  }
  return out;
}

function mean(values: number[]): number {
  const total = values.reduce((sum, value) => sum + value, 0);
  return Math.round(total / values.length);
}
