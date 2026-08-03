import { Inbox } from "lucide-react";

import { InboxShell } from "@/components/staff/inbox/inbox-shell";
import { TicketList } from "@/components/staff/inbox/ticket-list";
import { listTickets } from "@/lib/staff/queries";
import { parseTicketQuery } from "@/lib/staff/url";

/**
 * The queue with nothing open.
 *
 * On mobile this is the whole screen; from md up the empty right pane invites
 * a selection rather than auto-opening the first ticket, which would mark
 * someone else's ticket as read just because you visited a view.
 */
export default async function TicketsPage({
  searchParams,
}: {
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const query = parseTicketQuery(await searchParams);
  const page = await listTickets(query);

  return (
    <InboxShell
      hasDetail={false}
      list={<TicketList page={page} query={query} />}
      detail={
        <div className="flex h-full flex-col items-center justify-center gap-3 bg-tl-canvas px-6 text-center">
          <span className="flex size-12 items-center justify-center rounded-full bg-white shadow-panel">
            <Inbox className="size-5 text-tl-faint" aria-hidden />
          </span>
          <div>
            <p className="text-ui-md font-semibold text-tl-ink">
              No ticket selected
            </p>
            <p className="mt-1 text-ui-sm text-tl-muted">
              Pick one from the queue, or press{" "}
              <kbd className="rounded border border-tl-line bg-white px-1.5 py-0.5 font-sans text-ui-xs font-medium">
                ⌘K
              </kbd>{" "}
              to search.
            </p>
          </div>
        </div>
      }
    />
  );
}
