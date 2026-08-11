"use client";

import {
  PortalStatGrid,
  PortalStatGridSkeleton,
} from "@/components/portal/dashboard/stat-grid";
import {
  RecentTickets,
  RecentTicketsSkeleton,
} from "@/components/portal/dashboard/recent-tickets";
import { ErrorState, PageHeader } from "@/components/portal/primitives";
import { NewTicketDialog } from "@/components/portal/tickets/new-ticket-dialog";
import { usePortalMe, usePortalOverview } from "@/lib/portal/hooks";

/*
  The portal's front door.

  One request feeds the whole screen: the four numbers and the recent list are
  both derived from the customer's newest page of tickets, so the dashboard
  costs a single round trip rather than a stats call the customer's token could
  not make anyway (see lib/portal/stats.ts).
*/

export default function PortalDashboardPage() {
  const { data: me } = usePortalMe();
  const overview = usePortalOverview();

  const firstName = me?.first_name?.trim();

  return (
    <>
      <PageHeader
        title={firstName ? `Welcome back, ${firstName}` : "Welcome back"}
        description="Here's where your support requests stand."
        action={<NewTicketDialog />}
      />

      {overview.isError ? (
        <ErrorState
          title="We couldn't load your dashboard"
          description="Your tickets are safe — this is only the summary. Try again in a moment."
          onRetry={() => overview.refetch()}
          retrying={overview.isFetching}
          className="rounded-card border border-tl-line bg-tl-card shadow-panel"
        />
      ) : overview.isPending ? (
        <>
          <PortalStatGridSkeleton />
          <RecentTicketsSkeleton />
        </>
      ) : (
        <>
          <PortalStatGrid stats={overview.data.stats} />
          <RecentTickets tickets={overview.data.recent} />
        </>
      )}
    </>
  );
}
