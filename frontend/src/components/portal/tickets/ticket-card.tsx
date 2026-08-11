"use client";

import Link from "next/link";
import { Building2, MessageSquare } from "lucide-react";

import { Skeleton } from "@/components/shadcn/skeleton";
import {
  TicketPriorityBadge,
  TicketStatusBadge,
} from "@/components/portal/primitives";
import { relativeTime, ticketReference } from "@/lib/portal/format";
import type { PortalTicketListItem } from "@/lib/portal/types";
import { cn } from "@/lib/utils";

/*
  One ticket in the customer's list.

  The whole card is the link rather than the subject alone: a 40px title inside
  a 120px card leaves most of the target dead, and on a phone that is the
  difference between tapping a ticket and tapping nothing.

  The description snippet renders only when the API supplies one. `GET /tickets`
  returns no body today — the first message is the description and it is not
  included in a list item — so rather than fetch every ticket to show two lines
  of each, the card omits the line and keeps its meta row. See lib/portal/types.
*/

export function TicketCard({
  ticket,
  className,
}: {
  ticket: PortalTicketListItem;
  className?: string;
}) {
  return (
    <Link
      href={`/portal/tickets/${ticket.id}`}
      className={cn(
        "group block rounded-card border border-tl-line bg-tl-card p-4 shadow-panel transition-colors duration-150 hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30 sm:p-5",
        className,
      )}
    >
      {/* Badges sit beside the subject on a wide card and drop below it on a
          narrow one. Sharing the row at phone widths left the title barely
          twenty characters before the ellipsis. */}
      <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between sm:gap-3">
        <div className="min-w-0 sm:flex-1">
          <span className="text-ui-xs font-semibold tabular-nums text-tl-faint">
            {ticketReference(ticket.id)}
          </span>
          <h3 className="mt-0.5 line-clamp-2 text-ui-lg font-semibold tracking-[-0.01em] text-tl-ink group-hover:text-tl-blue">
            {ticket.subject}
          </h3>
        </div>

        <div className="flex shrink-0 flex-wrap items-center gap-1.5">
          <TicketStatusBadge status={ticket.status} />
          <TicketPriorityBadge priority={ticket.priority} />
        </div>
      </div>

      {ticket.snippet && (
        <p className="mt-2 line-clamp-2 text-ui-md leading-relaxed text-tl-muted">
          {ticket.snippet}
        </p>
      )}

      <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1.5 text-ui-sm text-tl-faint">
        <span className="inline-flex items-center gap-1.5">
          <Building2 className="size-3.5" strokeWidth={1.9} aria-hidden />
          {ticket.department.name}
        </span>
        <span className="inline-flex items-center gap-1.5">
          <MessageSquare className="size-3.5" strokeWidth={1.9} aria-hidden />
          {ticket.message_count}
          <span className="sr-only">messages</span>
        </span>
        <span className="ml-auto flex flex-wrap items-center gap-x-3 gap-y-1">
          <span>
            Opened{" "}
            <time dateTime={ticket.created_at}>
              {relativeTime(ticket.created_at)}
            </time>
          </span>
          <span aria-hidden className="hidden sm:inline">
            ·
          </span>
          <span>
            Updated{" "}
            <time dateTime={ticket.updated_at}>
              {relativeTime(ticket.updated_at)}
            </time>
          </span>
        </span>
      </div>
    </Link>
  );
}

/** The card's shape while the page loads, so the list does not jump. */
export function TicketCardSkeleton() {
  return (
    <div className="rounded-card border border-tl-line bg-tl-card p-4 shadow-panel sm:p-5">
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0 flex-1 space-y-2">
          <Skeleton className="h-3 w-16" />
          <Skeleton className="h-4 w-2/3" />
        </div>
        <div className="flex shrink-0 gap-1.5">
          <Skeleton className="h-5 w-16 rounded-md" />
          <Skeleton className="h-5 w-14 rounded-md" />
        </div>
      </div>
      <div className="mt-3 flex items-center gap-4">
        <Skeleton className="h-3 w-28" />
        <Skeleton className="h-3 w-10" />
        <Skeleton className="ml-auto h-3 w-40" />
      </div>
    </div>
  );
}

export function TicketListSkeleton({ rows = 5 }: { rows?: number }) {
  return (
    <div className="space-y-3" aria-hidden>
      {Array.from({ length: rows }).map((_, index) => (
        <TicketCardSkeleton key={index} />
      ))}
    </div>
  );
}
