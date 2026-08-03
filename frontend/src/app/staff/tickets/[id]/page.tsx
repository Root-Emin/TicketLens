import { notFound } from "next/navigation";

import { InboxShell } from "@/components/staff/inbox/inbox-shell";
import { TicketList } from "@/components/staff/inbox/ticket-list";
import { TicketWorkspace } from "@/components/staff/inbox/ticket-workspace";
import { getCurrentAgent, getTicket, listTickets } from "@/lib/staff/queries";
import { parseTicketQuery } from "@/lib/staff/url";

/**
 * A ticket, in the context of the queue it came from.
 *
 * The route is the single source of truth for selection: the highlighted row
 * and the thread below cannot disagree, because both are derived from `id`.
 */
export default async function TicketPage({
  params,
  searchParams,
}: {
  params: Promise<{ id: string }>;
  searchParams: Promise<Record<string, string | string[] | undefined>>;
}) {
  const [{ id }, raw] = await Promise.all([params, searchParams]);
  const query = parseTicketQuery(raw);

  const [ticket, page, agent] = await Promise.all([
    getTicket(id),
    listTickets(query),
    getCurrentAgent(),
  ]);

  if (!ticket) notFound();

  return (
    <InboxShell
      hasDetail
      list={<TicketList page={page} query={query} selectedId={ticket.id} />}
      detail={
        <TicketWorkspace
          ticket={ticket}
          query={query}
          agent={{
            id: agent.id,
            name: agent.name,
            initials: agent.initials,
          }}
        />
      }
    />
  );
}
