"use client";

import Link from "next/link";
import { ArrowLeft, Building2, CalendarDays, Sparkles } from "lucide-react";

import {
  TicketPriorityBadge,
  TicketStatusBadge,
} from "@/components/portal/primitives";
import { longDate, ticketReference } from "@/lib/portal/format";
import type { TicketDetail } from "@/lib/api/types";

/*
  What the ticket is, above the conversation.

  The department line is the one place the AI shows itself to a customer: they
  never chose it, so saying it was routed there — rather than leaving a team
  name to appear from nowhere — is what makes the routing feel deliberate
  instead of arbitrary. No confidence numbers, no model name, no category: those
  are the agent's tools, and exposing them here would invite an argument about
  a prediction the customer cannot change.
*/

export function TicketHeader({ ticket }: { ticket: TicketDetail }) {
  const routedByAi = ticket.latest_analysis !== null && !ticket.department_overridden;

  return (
    <header className="border-b border-tl-line px-4 py-4 sm:px-5">
      <Link
        href="/portal/tickets"
        className="inline-flex items-center gap-1.5 rounded-md text-ui-sm font-medium text-tl-muted transition-colors duration-150 hover:text-tl-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
      >
        <ArrowLeft className="size-3.5" aria-hidden />
        All tickets
      </Link>

      <div className="mt-3 flex flex-wrap items-start justify-between gap-x-4 gap-y-3">
        <div className="min-w-0 flex-1">
          <span className="text-ui-xs font-semibold tabular-nums text-tl-faint">
            {ticketReference(ticket.id)}
          </span>
          <h1 className="mt-0.5 text-ui-xl font-bold leading-snug tracking-[-0.02em] text-tl-ink">
            {ticket.subject}
          </h1>
        </div>

        <div className="flex shrink-0 items-center gap-1.5">
          <TicketStatusBadge status={ticket.status} />
          <TicketPriorityBadge priority={ticket.priority} />
        </div>
      </div>

      <dl className="mt-4 flex flex-wrap items-center gap-x-6 gap-y-2 text-ui-sm">
        <div className="flex items-center gap-1.5">
          <Building2 className="size-3.5 text-tl-faint-soft" strokeWidth={1.9} aria-hidden />
          <dt className="sr-only">Assigned department</dt>
          <dd className="text-tl-muted">
            <span className="font-medium text-tl-ink-soft">
              {ticket.department.name}
            </span>
            {routedByAi && (
              <span className="ml-1.5 inline-flex items-center gap-1 text-tl-blue">
                <Sparkles className="size-3" strokeWidth={2} aria-hidden />
                routed by AI
              </span>
            )}
          </dd>
        </div>

        <div className="flex items-center gap-1.5">
          <CalendarDays className="size-3.5 text-tl-faint-soft" strokeWidth={1.9} aria-hidden />
          <dt className="sr-only">Created</dt>
          <dd className="text-tl-muted">
            Opened{" "}
            <time dateTime={ticket.created_at} className="font-medium text-tl-ink-soft">
              {longDate(ticket.created_at)}
            </time>
          </dd>
        </div>
      </dl>
    </header>
  );
}
