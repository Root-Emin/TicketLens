"use client";

import Link from "next/link";
import { ArrowRight, Inbox } from "lucide-react";

import { Panel, PanelHeader } from "@/components/staff/primitives";
import { EmptyState, TicketStatusBadge } from "@/components/portal/primitives";
import { NewTicketDialog } from "@/components/portal/tickets/new-ticket-dialog";
import { Skeleton } from "@/components/shadcn/skeleton";
import { relativeTime, ticketReference } from "@/lib/portal/format";
import type { PortalTicketListItem } from "@/lib/portal/types";

/*
  The last five tickets, as a list rather than five more cards.

  The dashboard already spends a row on cards; repeating the shape below them
  flattens the page into one texture. A divided list reads as "recent activity"
  and lets five rows fit in the height two cards would take.
*/

export function RecentTickets({
  tickets,
}: {
  tickets: PortalTicketListItem[];
}) {
  return (
    <Panel>
      <PanelHeader
        title="Recent tickets"
        action={
          tickets.length > 0 ? (
            <Link
              href="/portal/tickets"
              className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-ui-sm font-medium text-tl-blue transition-colors duration-150 hover:bg-tl-blue-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
            >
              View all
              <ArrowRight className="size-3.5" aria-hidden />
            </Link>
          ) : undefined
        }
      />

      {tickets.length === 0 ? (
        <div className="border-t border-tl-line-soft">
          <EmptyState
            icon={Inbox}
            title="No tickets yet"
            description="When you open a request it will appear here, along with everything the support team sends back."
            action={<NewTicketDialog />}
            className="py-12"
          />
        </div>
      ) : (
        <ul className="border-t border-tl-line-soft">
          {tickets.map((ticket) => (
            <li key={ticket.id}>
              <Link
                href={`/portal/tickets/${ticket.id}`}
                className="flex flex-col gap-1.5 border-b border-tl-line-soft px-5 py-3.5 transition-colors duration-150 last:border-b-0 hover:bg-tl-line-soft/60 focus-visible:bg-tl-line-soft focus-visible:outline-none sm:flex-row sm:items-center sm:gap-3"
              >
                <div className="min-w-0 sm:flex-1">
                  <div className="flex items-baseline gap-2">
                    <span className="shrink-0 text-ui-xs font-semibold tabular-nums text-tl-faint">
                      {ticketReference(ticket.id)}
                    </span>
                    <span className="truncate text-ui-base font-semibold text-tl-ink">
                      {ticket.subject}
                    </span>
                  </div>
                  <p className="mt-0.5 truncate text-ui-sm text-tl-muted">
                    {ticket.department.name} · updated{" "}
                    <time dateTime={ticket.updated_at}>
                      {relativeTime(ticket.updated_at)}
                    </time>
                  </p>
                </div>
                {/* Below sm the badge takes its own line rather than eating the
                    width the subject needs. */}
                <TicketStatusBadge
                  status={ticket.status}
                  className="shrink-0 self-start sm:self-auto"
                />
              </Link>
            </li>
          ))}
        </ul>
      )}
    </Panel>
  );
}

export function RecentTicketsSkeleton() {
  return (
    <Panel aria-hidden>
      <div className="px-5 py-4">
        <Skeleton className="h-5 w-32" />
      </div>
      <div className="border-t border-tl-line-soft">
        {Array.from({ length: 5 }).map((_, index) => (
          <div
            key={index}
            className="flex items-center gap-3 border-b border-tl-line-soft px-5 py-3.5 last:border-b-0"
          >
            <div className="min-w-0 flex-1 space-y-2">
              <Skeleton className="h-3.5 w-2/3" />
              <Skeleton className="h-3 w-1/3" />
            </div>
            <Skeleton className="h-5 w-16 shrink-0 rounded-md" />
          </div>
        ))}
      </div>
    </Panel>
  );
}
