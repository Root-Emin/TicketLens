"use client";

import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import { Inbox, Plus, SearchX } from "lucide-react";

import {
  ActionButton,
  ActionLink,
  EmptyState,
  ErrorState,
  PageHeader,
} from "@/components/portal/primitives";
import { TicketListControls } from "@/components/portal/tickets/list-controls";
import { TicketPagination } from "@/components/portal/tickets/pagination";
import {
  TicketCard,
  TicketListSkeleton,
} from "@/components/portal/tickets/ticket-card";
import { NewTicketDialog } from "@/components/portal/tickets/new-ticket-dialog";
import { parseQuery } from "@/lib/portal/filters";
import { usePortalTickets } from "@/lib/portal/hooks";

/*
  Every ticket the customer has raised.

  The URL is the state: `?q=`, `?status=`, `?sort=` and `?page=` are read here
  and written by the controls, which is what makes a filtered list shareable and
  the back button behave.
*/

function TicketsInner() {
  const params = useSearchParams();
  const query = parseQuery(new URLSearchParams(params.toString()));
  const { data, isPending, isError, isFetching, refetch } =
    usePortalTickets(query);

  const tickets = data?.data ?? [];
  const total = data?.meta.total ?? 0;
  const filtered = Boolean(query.q) || query.status !== "all";

  return (
    <>
      <PageHeader
        title="My Tickets"
        description={
          data
            ? `${total} ${total === 1 ? "request" : "requests"} in total`
            : "Everything you've asked us, in one place."
        }
        action={<NewTicketDialog />}
      />

      <TicketListControls query={query} />

      {isError ? (
        <ErrorState
          onRetry={() => refetch()}
          retrying={isFetching}
          className="rounded-card border border-tl-line bg-tl-card shadow-panel"
        />
      ) : isPending ? (
        <TicketListSkeleton />
      ) : tickets.length === 0 ? (
        // Two different empties: a filter that matched nothing is a dead end
        // you back out of, an account with no tickets is one you start from.
        filtered ? (
          <EmptyState
            icon={SearchX}
            title="No tickets match this view"
            description="Try a different status, or clear the search to see everything you've sent us."
            action={
              <ActionLink href="/portal/tickets" variant="secondary">
                Clear filters
              </ActionLink>
            }
            className="rounded-card border border-tl-line bg-tl-card shadow-panel"
          />
        ) : (
          <EmptyState
            icon={Inbox}
            title="You haven't opened a ticket yet"
            description="Tell us what you need and our AI will route it to the team that can help fastest."
            action={
              <NewTicketDialog
                trigger={
                  <ActionButton>
                    <Plus className="size-4" strokeWidth={2.2} aria-hidden />
                    Create your first ticket
                  </ActionButton>
                }
              />
            }
            className="rounded-card border border-tl-line bg-tl-card shadow-panel"
          />
        )
      ) : (
        <>
          {/* aria-busy marks the list as stale while the next page loads —
              placeholderData keeps the old rows on screen, which is otherwise
              indistinguishable from a page that ignored the click. */}
          <div className="space-y-3" aria-busy={isFetching}>
            {tickets.map((ticket) => (
              <TicketCard key={ticket.id} ticket={ticket} />
            ))}
          </div>
          <TicketPagination query={query} total={total} />
        </>
      )}
    </>
  );
}

export default function PortalTicketsPage() {
  // useSearchParams opts its subtree into client rendering, so the boundary
  // keeps the skeleton meaningful instead of blanking the page.
  return (
    <Suspense fallback={<TicketListSkeleton />}>
      <TicketsInner />
    </Suspense>
  );
}
