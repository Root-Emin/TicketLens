import Link from "next/link";
import { ArrowRight, Inbox } from "lucide-react";

import { cn } from "@/lib/utils";
import { relativeAge, slaRemaining } from "@/lib/staff/format";
import type { Ticket } from "@/lib/staff/types";
import {
  InitialsAvatar,
  Panel,
  PanelHeader,
  PriorityBadge,
  StatusBadge,
} from "../primitives";

/**
 * A compact ticket list used twice on the dashboard: once for the SLA-critical
 * queue and once for recent activity. Same rows, different source and emphasis,
 * so the two sections cannot drift apart visually.
 */
export function TicketSummaryList({
  title,
  tickets,
  href,
  linkLabel,
  emphasis = "sla",
  emptyMessage,
}: {
  title: string;
  tickets: Ticket[];
  href: string;
  linkLabel: string;
  /** What the right-hand column reports. */
  emphasis?: "sla" | "updated";
  emptyMessage: string;
}) {
  return (
    <Panel className="flex min-w-0 flex-col">
      <PanelHeader
        title={title}
        action={
          <Link
            href={href}
            className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-ui-sm font-medium text-tl-blue transition-colors duration-150 hover:bg-tl-blue-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
          >
            {linkLabel}
            <ArrowRight className="size-3.5" aria-hidden />
          </Link>
        }
      />

      {tickets.length === 0 ? (
        <div className="flex flex-col items-center gap-2 border-t border-tl-line-soft px-5 py-10 text-center">
          <Inbox className="size-6 text-tl-faint-soft" aria-hidden />
          <p className="text-ui-base text-tl-muted">{emptyMessage}</p>
        </div>
      ) : (
        <ul className="border-t border-tl-line-soft">
          {tickets.map((ticket) => {
            const sla = slaRemaining(ticket.createdAt, ticket.slaDueAt);

            return (
              <li key={ticket.id} className="border-b border-tl-line-soft last:border-b-0">
                <Link
                  href={`/staff/tickets/${ticket.id}`}
                  className="flex items-center gap-3 px-5 py-3 transition-colors duration-150 hover:bg-tl-line-soft/50 focus-visible:outline-none focus-visible:bg-tl-line-soft"
                >
                  <InitialsAvatar
                    name={ticket.customer.name}
                    initials={ticket.customer.initials}
                    size={32}
                  />

                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-ui-sm font-semibold text-tl-ink">
                        {ticket.id}
                      </span>
                      <PriorityBadge priority={ticket.priority} />
                    </div>
                    <p className="mt-0.5 truncate text-ui-base font-medium text-tl-ink">
                      {ticket.subject}
                    </p>
                    <p className="mt-0.5 truncate text-ui-xs text-tl-muted">
                      {ticket.customer.name} · {ticket.customer.company}
                    </p>
                  </div>

                  <div className="hidden shrink-0 flex-col items-end gap-1.5 sm:flex">
                    {emphasis === "sla" ? (
                      <span
                        className={cn(
                          "text-ui-xs font-semibold",
                          sla.breached ? "text-tl-red-ink" : "text-tl-orange-ink",
                        )}
                      >
                        {sla.label}
                      </span>
                    ) : (
                      <StatusBadge status={ticket.status} />
                    )}
                    <span className="text-ui-xs text-tl-faint">
                      {relativeAge(ticket.updatedAt)}
                    </span>
                  </div>
                </Link>
              </li>
            );
          })}
        </ul>
      )}
    </Panel>
  );
}
