"use client";

import { use } from "react";

import { ReplyBox } from "@/components/portal/conversation/reply-box";
import { Thread } from "@/components/portal/conversation/thread";
import { TicketHeader } from "@/components/portal/conversation/ticket-header";
import {
  ActionLink,
  EmptyState,
  ErrorState,
  isClosedStatus,
} from "@/components/portal/primitives";
import { Panel } from "@/components/staff/primitives";
import { Skeleton } from "@/components/shadcn/skeleton";
import { FileQuestion } from "lucide-react";

import { ApiError } from "@/lib/api/client";
import { usePortalMe, usePortalTicket } from "@/lib/portal/hooks";

/*
  One ticket, as a conversation.

  Client-rendered because the thread has to reflect a reply the moment it is
  accepted, and because the session cookie is httpOnly — a server render here
  would fetch the same data a second time.
*/

export default function PortalTicketDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  // Next 16 hands route params to client components as a promise.
  const { id } = use(params);

  const { data: me } = usePortalMe();
  const { data: ticket, isPending, isError, error, isFetching, refetch } =
    usePortalTicket(id);

  const notFound = error instanceof ApiError && error.status === 404;

  if (isError) {
    return (
      <Panel>
        {notFound ? (
          <EmptyState
            icon={FileQuestion}
            title="Ticket not found"
            description="This request either doesn't exist or isn't on your account."
            action={
              <ActionLink href="/portal/tickets" variant="secondary">
                Back to my tickets
              </ActionLink>
            }
          />
        ) : (
          <ErrorState
            title="We couldn't load this ticket"
            onRetry={() => refetch()}
            retrying={isFetching}
          />
        )}
      </Panel>
    );
  }

  if (isPending) return <TicketDetailSkeleton />;

  const customerName = me
    ? `${me.first_name} ${me.last_name}`.trim() || me.email
    : ticket.customer.full_name;

  return (
    <Panel className="overflow-hidden">
      <TicketHeader ticket={ticket} />
      <Thread messages={ticket.messages} customerName={customerName} />
      <ReplyBox ticketId={ticket.id} resolved={isClosedStatus(ticket.status)} />
    </Panel>
  );
}

function TicketDetailSkeleton() {
  return (
    <Panel className="overflow-hidden" aria-hidden>
      <div className="space-y-3 border-b border-tl-line px-5 py-4">
        <Skeleton className="h-3 w-20" />
        <Skeleton className="h-6 w-2/3" />
        <div className="flex gap-4">
          <Skeleton className="h-3.5 w-40" />
          <Skeleton className="h-3.5 w-36" />
        </div>
      </div>

      <div className="space-y-5 px-5 py-5">
        {[0, 1, 2].map((index) => (
          <div
            key={index}
            className={index % 2 === 1 ? "flex justify-end" : "flex gap-3"}
          >
            {index % 2 === 0 && <Skeleton className="size-8 shrink-0 rounded-full" />}
            <Skeleton
              className={index % 2 === 1 ? "h-16 w-1/2" : "h-20 w-2/3"}
            />
          </div>
        ))}
      </div>

      <div className="border-t border-tl-line px-5 py-4">
        <Skeleton className="h-[76px] w-full" />
      </div>
    </Panel>
  );
}
