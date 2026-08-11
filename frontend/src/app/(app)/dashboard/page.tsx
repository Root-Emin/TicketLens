"use client";

import { AlertTriangle, CheckCircle2, Inbox, ShieldQuestion } from "lucide-react";

import { Card, CardBody, CardHeader, CardTitle } from "@/components/ui/card";
import { CenteredSpinner } from "@/components/ui/spinner";
import { useStats } from "@/lib/api/hooks";
import {
  CATEGORY_LABELS,
  PRIORITY_LABELS,
  STATUS_LABELS,
  priorityClasses,
} from "@/lib/api/labels";
import type { Category, TicketPriority, TicketStatus } from "@/lib/api/types";
import { cn, pct } from "@/lib/utils";

export default function DashboardPage() {
  const { data: stats, isLoading, isError } = useStats();

  if (isLoading) return <CenteredSpinner label="Loading dashboard…" />;
  if (isError || !stats)
    return (
      <div className="p-6">
        <Card className="p-8 text-center text-sm text-red-500">
          Failed to load statistics.
        </Card>
      </div>
    );

  const statusEntries = Object.entries(stats.tickets).filter(
    ([k]) => k !== "total",
  ) as [TicketStatus, number][];
  const maxDept = Math.max(1, ...stats.by_department.map((d) => d.count));
  const maxCat = Math.max(1, ...Object.values(stats.by_category));

  return (
    <div className="p-6">
      <h1 className="mb-5 text-xl font-semibold">Overview</h1>

      {/* Headline stats */}
      <div className="mb-6 grid grid-cols-2 gap-4 lg:grid-cols-4">
        <Stat
          icon={<Inbox className="h-5 w-5" />}
          label="Total tickets"
          value={stats.tickets.total}
        />
        <Stat
          icon={<CheckCircle2 className="h-5 w-5 text-emerald-500" />}
          label="Open"
          value={stats.tickets.open}
        />
        <Stat
          icon={<AlertTriangle className="h-5 w-5 text-amber-500" />}
          label="Needs review"
          value={stats.ai.needs_review}
        />
        <Stat
          icon={<ShieldQuestion className="h-5 w-5 text-muted-foreground" />}
          label="Unrouted"
          value={stats.ai.unmapped_count}
        />
      </div>

      {/* AI acceptance */}
      <Card className="mb-6">
        <CardHeader>
          <CardTitle>AI performance</CardTitle>
        </CardHeader>
        <CardBody className="grid grid-cols-1 gap-6 sm:grid-cols-3">
          <AcceptRate
            label="Priority accept rate"
            value={stats.ai.priority_accept_rate}
            hint="Share the agent left unchanged"
          />
          <AcceptRate
            label="Department accept rate"
            value={stats.ai.department_accept_rate}
            hint="Of routed tickets"
          />
          <AcceptRate
            label="Avg. priority confidence"
            value={stats.ai.avg_priority_confidence}
            hint={`Across ${stats.ai.analyzed} analyzed`}
          />
        </CardBody>
      </Card>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        {/* Status breakdown */}
        <Card>
          <CardHeader>
            <CardTitle>By status</CardTitle>
          </CardHeader>
          <CardBody className="space-y-2">
            {statusEntries.map(([status, count]) => (
              <div key={status} className="flex items-center justify-between text-sm">
                <span className="text-muted-foreground">
                  {STATUS_LABELS[status]}
                </span>
                <span className="font-medium text-foreground">{count}</span>
              </div>
            ))}
          </CardBody>
        </Card>

        {/* Priority breakdown */}
        <Card>
          <CardHeader>
            <CardTitle>By priority</CardTitle>
          </CardHeader>
          <CardBody className="space-y-2">
            {(Object.entries(stats.by_priority) as [TicketPriority, number][]).map(
              ([priority, count]) => (
                <div
                  key={priority}
                  className="flex items-center justify-between text-sm"
                >
                  <span
                    className={cn(
                      "rounded-full px-2 py-0.5 text-xs ring-1 ring-inset",
                      priorityClasses(priority),
                    )}
                  >
                    {PRIORITY_LABELS[priority]}
                  </span>
                  <span className="font-medium text-foreground">{count}</span>
                </div>
              ),
            )}
          </CardBody>
        </Card>

        {/* Category distribution */}
        <Card>
          <CardHeader>
            <CardTitle>By category</CardTitle>
          </CardHeader>
          <CardBody className="space-y-2">
            {Object.entries(stats.by_category)
              .sort((a, b) => b[1] - a[1])
              .map(([category, count]) => (
                <BarRow
                  key={category}
                  label={
                    CATEGORY_LABELS[category as Category] ?? category
                  }
                  count={count}
                  max={maxCat}
                />
              ))}
            {Object.keys(stats.by_category).length === 0 && (
              <p className="text-sm text-muted-foreground">No data yet.</p>
            )}
          </CardBody>
        </Card>

        {/* Department load */}
        <Card>
          <CardHeader>
            <CardTitle>By department</CardTitle>
          </CardHeader>
          <CardBody className="space-y-2">
            {stats.by_department
              .sort((a, b) => b.count - a.count)
              .map((d) => (
                <BarRow
                  key={d.department_id}
                  label={d.name}
                  count={d.count}
                  max={maxDept}
                />
              ))}
            {stats.by_department.length === 0 && (
              <p className="text-sm text-muted-foreground">No data yet.</p>
            )}
          </CardBody>
        </Card>
      </div>
    </div>
  );
}

function Stat({
  icon,
  label,
  value,
}: {
  icon: React.ReactNode;
  label: string;
  value: number;
}) {
  return (
    <Card>
      <CardBody className="flex items-center gap-3">
        <div className="rounded-lg bg-surface-muted p-2">{icon}</div>
        <div>
          <div className="text-2xl font-semibold text-foreground">{value}</div>
          <div className="text-xs text-muted-foreground">{label}</div>
        </div>
      </CardBody>
    </Card>
  );
}

function AcceptRate({
  label,
  value,
  hint,
}: {
  label: string;
  value: number;
  hint: string;
}) {
  return (
    <div>
      <div className="text-3xl font-semibold text-foreground">{pct(value)}</div>
      <div className="text-sm font-medium text-foreground">{label}</div>
      <div className="text-xs text-muted-foreground">{hint}</div>
    </div>
  );
}

function BarRow({
  label,
  count,
  max,
}: {
  label: string;
  count: number;
  max: number;
}) {
  return (
    <div className="flex items-center gap-3 text-sm">
      <span className="w-32 shrink-0 truncate text-muted-foreground">{label}</span>
      <div className="h-2 flex-1 overflow-hidden rounded-full bg-surface-muted">
        <div
          className="h-full rounded-full bg-accent"
          style={{ width: `${(count / max) * 100}%` }}
        />
      </div>
      <span className="w-8 shrink-0 text-right font-medium text-foreground">
        {count}
      </span>
    </div>
  );
}
