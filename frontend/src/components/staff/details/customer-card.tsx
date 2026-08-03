import { Building2, Mail } from "lucide-react";

import { relativeAge } from "@/lib/staff/format";
import type { Ticket } from "@/lib/staff/types";
import { FieldGroupLabel, InitialsAvatar } from "../primitives";

/** Who is on the other end, and how much history they have with us. */
export function CustomerCard({ customer }: { customer: Ticket["customer"] }) {
  return (
    <section className="rounded-card border border-tl-line bg-tl-card p-5 shadow-panel">
      <FieldGroupLabel>Customer</FieldGroupLabel>

      <div className="flex items-center gap-3">
        <InitialsAvatar
          name={customer.name}
          initials={customer.initials}
          size={40}
        />
        <div className="min-w-0">
          <p className="truncate text-ui-md font-semibold text-tl-ink">
            {customer.name}
          </p>
          <p className="truncate text-ui-xs text-tl-muted">{customer.company}</p>
        </div>
      </div>

      <dl className="mt-4 space-y-2">
        <div className="flex items-center gap-2">
          <dt className="sr-only">Email</dt>
          <Mail className="size-3.5 shrink-0 text-tl-faint" aria-hidden />
          <dd className="truncate text-ui-sm text-tl-ink-soft">
            {customer.email}
          </dd>
        </div>
        <div className="flex items-center gap-2">
          <dt className="sr-only">History</dt>
          <Building2 className="size-3.5 shrink-0 text-tl-faint" aria-hidden />
          <dd className="truncate text-ui-sm text-tl-ink-soft">
            {customer.ticketCount} tickets · customer for{" "}
            {relativeAge(customer.since).replace(" ago", "")}
          </dd>
        </div>
      </dl>
    </section>
  );
}

/** The last few things that happened, newest first. */
export function HistoryCard({ timeline }: { timeline: Ticket["timeline"] }) {
  return (
    <section className="rounded-card border border-tl-line bg-tl-card p-5 shadow-panel">
      <FieldGroupLabel>History</FieldGroupLabel>
      <ol className="space-y-3">
        {[...timeline]
          .reverse()
          .slice(0, 4)
          .map((event) => (
            <li key={event.id} className="flex gap-2.5">
              <span
                className="mt-1.5 size-1.5 shrink-0 rounded-full bg-tl-faint-soft"
                aria-hidden
              />
              <div className="min-w-0">
                <p className="text-ui-sm text-tl-ink">{event.summary}</p>
                <p className="text-ui-xs text-tl-faint">
                  {event.actor} · {relativeAge(event.at)}
                </p>
              </div>
            </li>
          ))}
      </ol>
    </section>
  );
}
