import Link from "next/link";
import { memo } from "react";
import { MessageSquare, Paperclip } from "lucide-react";

import { cn } from "@/lib/utils";
import { relativeAge, slaRemaining } from "@/lib/staff/format";
import type { Ticket, TicketQuery } from "@/lib/staff/types";
import { ticketHref } from "@/lib/staff/url";
import { DOT, InitialsAvatar, PriorityBadge, PRIORITY_ACCENT } from "../primitives";

/*
  One row of the queue.

  Memoised because moving the selection re-renders the list, and only two rows
  actually change. `data-list-item` is the hook the keyboard navigation uses to
  find its stops.
*/

function TicketRowImpl({
  ticket,
  query,
  selected,
}: {
  ticket: Ticket;
  query: TicketQuery;
  selected: boolean;
}) {
  const sla = slaRemaining(ticket.createdAt, ticket.slaDueAt);
  const replies = ticket.messages.filter((m) => m.kind !== "note").length - 1;
  const hasAttachments = ticket.messages.some((m) => m.attachments?.length);

  return (
    <li>
      <Link
        data-list-item
        href={ticketHref(ticket.id, query)}
        aria-current={selected ? "true" : undefined}
        className={cn(
          "relative flex gap-3 border-b border-tl-line-soft px-4 py-3 transition-colors duration-150",
          "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-tl-blue/40",
          selected ? "bg-blue-50/70" : "hover:bg-tl-line-soft/50",
        )}
      >
        {selected && (
          <span className="absolute inset-y-0 left-0 w-[3px] bg-tl-blue" aria-hidden />
        )}

        <div className="relative shrink-0 pt-0.5">
          <InitialsAvatar
            name={ticket.customer.name}
            initials={ticket.customer.initials}
            size={32}
          />
          {ticket.unread && (
            <span
              className="absolute -left-0.5 -top-0.5 size-2.5 rounded-full border-2 border-white bg-tl-blue"
              aria-hidden
            />
          )}
        </div>

        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            <span
              className={cn(
                "size-1.5 shrink-0 rounded-full",
                DOT[PRIORITY_ACCENT[ticket.priority]],
              )}
              aria-hidden
            />
            <span className="text-ui-sm font-semibold text-tl-ink">
              {ticket.id}
            </span>
            <PriorityBadge priority={ticket.priority} className="ml-auto" />
          </div>

          <p
            className={cn(
              "mt-1 truncate text-ui-base",
              ticket.unread
                ? "font-semibold text-tl-ink"
                : "font-medium text-tl-ink-soft",
            )}
          >
            {ticket.subject}
          </p>

          <div className="mt-1 flex items-center gap-1.5 text-ui-xs text-tl-muted">
            <span className="truncate">{ticket.customer.name}</span>
            <span className="text-tl-faint-soft" aria-hidden>
              ·
            </span>
            <span className="shrink-0">{relativeAge(ticket.updatedAt)}</span>

            <span className="ml-auto flex shrink-0 items-center gap-2">
              {hasAttachments && (
                <Paperclip className="size-3 text-tl-faint" aria-label="Has attachments" />
              )}
              {replies > 0 && (
                <span className="inline-flex items-center gap-1">
                  <MessageSquare className="size-3" aria-hidden />
                  <span className="tabular-nums">{replies}</span>
                  <span className="sr-only">replies</span>
                </span>
              )}
            </span>
          </div>

          {sla.breached && (
            <p className="mt-1 text-ui-xs font-semibold text-tl-red-ink">
              SLA {sla.label.toLowerCase()}
            </p>
          )}
        </div>
      </Link>
    </li>
  );
}

export const TicketRow = memo(TicketRowImpl);
