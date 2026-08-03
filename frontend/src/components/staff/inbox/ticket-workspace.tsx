"use client";

import { useState } from "react";

import { Sheet, SheetContent, SheetTitle } from "@/components/shadcn/sheet";
import { ConversationPane } from "../conversation/conversation-pane";
import { DetailsPanel } from "../details/details-panel";
import type { Ticket, TicketQuery } from "@/lib/staff/types";

/*
  Conversation plus details.

  The details rail is a permanent third column from xl up, and a drawer below
  that — same component either way, so there is no second implementation to
  keep in step. Which one shows is decided in CSS, not JavaScript, so it is
  correct on the first paint.
*/

export function TicketWorkspace({
  ticket,
  query,
  agent,
}: {
  ticket: Ticket;
  query: TicketQuery;
  agent: { id: string; name: string; initials: string };
}) {
  const [detailsOpen, setDetailsOpen] = useState(false);

  return (
    <div className="flex h-full min-h-0">
      <div className="min-h-0 min-w-0 flex-1">
        <ConversationPane
          ticket={ticket}
          query={query}
          ownId={agent.id}
          ownName={agent.name}
          ownInitials={agent.initials}
          onOpenDetails={() => setDetailsOpen(true)}
        />
      </div>

      <aside className="hidden min-h-0 w-[340px] shrink-0 border-l border-tl-line xl:block">
        <DetailsPanel ticket={ticket} />
      </aside>

      <Sheet open={detailsOpen} onOpenChange={setDetailsOpen}>
        <SheetContent side="right" className="w-full p-0 sm:max-w-[380px]">
          <SheetTitle className="sr-only">Ticket details</SheetTitle>
          <DetailsPanel ticket={ticket} />
        </SheetContent>
      </Sheet>
    </div>
  );
}
