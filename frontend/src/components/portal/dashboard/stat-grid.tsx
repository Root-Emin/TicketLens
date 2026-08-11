"use client";

import Link from "next/link";
import {
  CheckCircle2,
  Clock3,
  CircleDot,
  MessageCircleReply,
  type LucideIcon,
} from "lucide-react";

import { Skeleton } from "@/components/shadcn/skeleton";
import { TINT } from "@/components/staff/primitives";
import type { Accent } from "@/lib/staff/types";
import { duration } from "@/lib/portal/format";
import { AVERAGE_LABEL } from "@/lib/portal/stats";
import type { PortalStats } from "@/lib/portal/types";
import { cn } from "@/lib/utils";

/*
  The four headline numbers.

  Three of them are counts and link into the list filtered to exactly what they
  count — a number a customer cannot open is decoration. The fourth is a
  duration and links nowhere, because there is no "slow tickets" view to send
  anyone to.

  No sparklines and no deltas here, unlike the staff dashboard: a customer with
  six tickets has no trend, and drawing one would be inventing a signal.
*/

interface StatDef {
  label: string;
  icon: LucideIcon;
  accent: Accent;
  href?: string;
  value: (stats: PortalStats) => string;
  caption: string;
}

const STATS: StatDef[] = [
  {
    label: "Open Tickets",
    icon: CircleDot,
    accent: "blue",
    href: "/portal/tickets?status=open",
    value: (stats) => String(stats.open),
    caption: "Being worked on",
  },
  {
    label: "Waiting Reply",
    icon: MessageCircleReply,
    accent: "orange",
    href: "/portal/tickets?status=pending_customer",
    value: (stats) => String(stats.waitingReply),
    caption: "Needs your answer",
  },
  {
    label: "Resolved",
    icon: CheckCircle2,
    accent: "green",
    href: "/portal/tickets?status=resolved",
    value: (stats) => String(stats.resolved),
    caption: "Closed out",
  },
  {
    label: AVERAGE_LABEL.first_response,
    icon: Clock3,
    accent: "blue",
    value: (stats) =>
      stats.averageMinutes === null ? "—" : duration(stats.averageMinutes),
    caption: "Across your tickets",
  },
];

export function PortalStatGrid({ stats }: { stats: PortalStats }) {
  return (
    <div className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-4">
      {STATS.map((def) => {
        // The last card names the clock it actually read: without a
        // first_response_at from the API it is measuring time to resolution,
        // and it says so rather than mislabelling the number.
        const label = def.href ? def.label : AVERAGE_LABEL[stats.averageBasis];
        const caption =
          def.href || stats.averageBasis !== "none"
            ? def.caption
            : "No resolved tickets yet";

        const body = (
          <>
            <div className="flex items-center gap-2.5">
              <span
                className={cn(
                  "flex size-9 shrink-0 items-center justify-center rounded-[10px]",
                  TINT[def.accent],
                )}
              >
                <def.icon className="size-[18px]" strokeWidth={2} aria-hidden />
              </span>
              <span className="min-w-0 flex-1 truncate text-ui-sm font-medium text-tl-muted">
                {label}
              </span>
            </div>
            <div className="mt-4">
              <div className="text-[26px] font-bold leading-none tracking-[-0.02em] text-tl-ink tabular-nums">
                {def.value(stats)}
              </div>
              <p className="mt-2 text-ui-xs text-tl-faint">{caption}</p>
            </div>
          </>
        );

        return def.href ? (
          <Link
            key={def.label}
            href={def.href}
            className="rounded-card border border-tl-line bg-tl-card p-4 shadow-panel transition-colors duration-150 hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30 sm:p-5"
          >
            {body}
          </Link>
        ) : (
          <div
            key={def.label}
            className="rounded-card border border-tl-line bg-tl-card p-4 shadow-panel sm:p-5"
          >
            {body}
          </div>
        );
      })}
    </div>
  );
}

export function PortalStatGridSkeleton() {
  return (
    <div className="grid grid-cols-2 gap-3 sm:gap-4 lg:grid-cols-4" aria-hidden>
      {Array.from({ length: 4 }).map((_, index) => (
        <div
          key={index}
          className="rounded-card border border-tl-line bg-tl-card p-4 shadow-panel sm:p-5"
        >
          <div className="flex items-center gap-2.5">
            <Skeleton className="size-9 rounded-[10px]" />
            <Skeleton className="h-3.5 w-20" />
          </div>
          <div className="mt-4 space-y-2.5">
            <Skeleton className="h-6 w-12" />
            <Skeleton className="h-3 w-24" />
          </div>
        </div>
      ))}
    </div>
  );
}
