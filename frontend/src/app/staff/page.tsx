import { Suspense } from "react";

import { StatGrid } from "@/components/staff/dashboard/stat-grid";
import { TicketSummaryList } from "@/components/staff/dashboard/ticket-summary-list";
import { Skeleton } from "@/components/shadcn/skeleton";
import { greeting } from "@/lib/staff/format";
import {
  getCurrentAgent,
  getStatSummary,
  listAttention,
  listRecent,
} from "@/lib/staff/queries";

/*
  The dashboard.

  One scroll container, top to bottom — the header above it is the only fixed
  chrome. Each data-backed section streams behind its own Suspense boundary so
  the greeting paints immediately and the slower panels fill in rather than
  holding up the whole page.
*/

export default async function StaffDashboardPage() {
  const agent = await getCurrentAgent();
  const firstName = agent.name.split(" ")[0];

  return (
    <div className="h-full overflow-y-auto">
      <div className="mx-auto flex max-w-[1600px] flex-col gap-6 px-4 py-6 lg:px-6">
        <header>
          <h1 className="text-ui-2xl font-bold tracking-[-0.02em] text-tl-ink">
            {greeting()}, {firstName}!{" "}
            <span aria-hidden>👋</span>
          </h1>
          <p className="mt-1 text-ui-md text-tl-muted">
            Here&apos;s what&apos;s happening with your tickets today.
          </p>
        </header>

        <Suspense fallback={<StatGridSkeleton />}>
          <StatsSection />
        </Suspense>

        <div className="grid gap-6 xl:grid-cols-2">
          <Suspense fallback={<ListSkeleton title="Needs attention" />}>
            <AttentionSection />
          </Suspense>
          <Suspense fallback={<ListSkeleton title="Recent activity" />}>
            <RecentSection />
          </Suspense>
        </div>
      </div>
    </div>
  );
}

async function StatsSection() {
  const stats = await getStatSummary();
  return <StatGrid stats={stats} />;
}

async function AttentionSection() {
  const tickets = await listAttention(5);
  return (
    <TicketSummaryList
      title="Needs attention"
      tickets={tickets}
      href="/staff/tickets?view=attention"
      linkLabel="View all"
      emphasis="sla"
      emptyMessage="Nothing is against the clock right now."
    />
  );
}

async function RecentSection() {
  const tickets = await listRecent(5);
  return (
    <TicketSummaryList
      title="Recent activity"
      tickets={tickets}
      href="/staff/tickets?view=open"
      linkLabel="View all"
      emphasis="updated"
      emptyMessage="No open tickets in your queue."
    />
  );
}

function StatGridSkeleton() {
  return (
    <div className="grid grid-cols-[repeat(auto-fit,minmax(210px,1fr))] gap-4">
      {Array.from({ length: 6 }).map((_, i) => (
        <div
          key={i}
          className="flex flex-col gap-4 rounded-card border border-tl-line bg-tl-card p-5 shadow-panel"
        >
          <div className="flex items-center gap-2.5">
            <Skeleton className="size-9 rounded-[10px]" />
            <Skeleton className="h-3.5 w-24" />
          </div>
          <div className="flex items-end justify-between gap-3">
            <div className="space-y-2.5">
              <Skeleton className="h-6 w-12" />
              <Skeleton className="h-3 w-28" />
            </div>
            <Skeleton className="h-7 w-16" />
          </div>
        </div>
      ))}
    </div>
  );
}

function ListSkeleton({ title }: { title: string }) {
  return (
    <section className="rounded-card border border-tl-line bg-tl-card shadow-panel">
      <div className="px-5 py-4">
        <h2 className="text-ui-lg font-semibold tracking-[-0.01em] text-tl-ink">
          {title}
        </h2>
      </div>
      <div className="border-t border-tl-line-soft">
        {Array.from({ length: 5 }).map((_, i) => (
          <div
            key={i}
            className="flex items-center gap-3 border-b border-tl-line-soft px-5 py-3 last:border-b-0"
          >
            <Skeleton className="size-8 rounded-full" />
            <div className="min-w-0 flex-1 space-y-1.5">
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-3.5 w-3/4" />
              <Skeleton className="h-3 w-1/3" />
            </div>
          </div>
        ))}
      </div>
    </section>
  );
}
