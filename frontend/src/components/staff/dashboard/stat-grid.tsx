import {
  ArrowDownRight,
  ArrowUpRight,
  BellRing,
  CheckCircle2,
  Clock3,
  MessageCircle,
  ShieldAlert,
  UsersRound,
  type LucideIcon,
} from "lucide-react";
import Link from "next/link";

import { cn } from "@/lib/utils";
import { duration } from "@/lib/staff/format";
import type { StatSummary } from "@/lib/staff/queries";
import type { Accent } from "@/lib/staff/types";
import { INK, Sparkline, STROKE, TINT } from "../primitives";

/*
  The six headline metrics.

  Each card is a link into the view it summarises — a number an agent cannot act
  on is decoration. The grid uses auto-fit rather than fixed column counts so
  cards reflow at any width instead of stepping 1 → 2 → 3 → 6 and leaving a
  ragged row at laptop sizes.
*/

interface StatDef {
  key: keyof StatSummary["trends"];
  label: string;
  icon: LucideIcon;
  accent: Accent;
  href: string;
  /** Whether a rise in this number is good news. */
  riseIsGood: boolean;
  format?: (value: number) => string;
}

const STATS: StatDef[] = [
  {
    key: "assigned",
    label: "Assigned to Me",
    icon: UsersRound,
    accent: "blue",
    href: "/staff/tickets?view=assigned",
    riseIsGood: true,
  },
  {
    key: "attention",
    label: "Needs Attention",
    icon: BellRing,
    accent: "red",
    href: "/staff/tickets?view=attention",
    riseIsGood: false,
  },
  {
    key: "waiting",
    label: "Waiting for Customer",
    icon: MessageCircle,
    accent: "orange",
    href: "/staff/tickets?view=waiting",
    riseIsGood: true,
  },
  {
    key: "resolvedToday",
    label: "Resolved Today",
    icon: CheckCircle2,
    accent: "green",
    href: "/staff/tickets?view=resolved",
    riseIsGood: true,
  },
  {
    key: "avgResponseMins",
    label: "Avg. Response Time",
    icon: Clock3,
    accent: "blue",
    href: "/staff/tickets?view=open&sort=sla",
    riseIsGood: false,
    format: duration,
  },
  {
    key: "slaBreached",
    label: "SLA Breached",
    icon: ShieldAlert,
    accent: "red",
    href: "/staff/tickets?view=open&sort=sla",
    riseIsGood: false,
  },
];

function StatCard({ def, stats }: { def: StatDef; stats: StatSummary }) {
  const Icon = def.icon;
  const raw = stats[def.key as keyof StatSummary] as number;
  const value = def.format ? def.format(raw) : String(raw);
  const delta = stats.deltas[def.key] ?? 0;
  const rising = delta >= 0;
  const Arrow = rising ? ArrowUpRight : ArrowDownRight;

  // Good news is green, bad news is red, regardless of which way the arrow
  // points: fewer tickets waiting is good, a faster response time is good.
  const tone: Accent = rising === def.riseIsGood ? "green" : "red";

  return (
    <Link
      href={def.href}
      className="group flex flex-col gap-4 rounded-card border border-tl-line bg-tl-card p-5 shadow-panel transition-colors duration-150 hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
    >
      <div className="flex items-center gap-2.5">
        <span
          className={cn(
            "flex size-9 shrink-0 items-center justify-center rounded-[10px]",
            TINT[def.accent],
          )}
        >
          <Icon className="size-[18px]" strokeWidth={2} aria-hidden />
        </span>
        <span className="min-w-0 flex-1 truncate text-ui-sm font-medium text-tl-muted">
          {def.label}
        </span>
      </div>

      <div className="flex items-end justify-between gap-3">
        <div className="min-w-0">
          <div className="text-[26px] font-bold leading-none tracking-[-0.02em] text-tl-ink">
            {value}
          </div>
          <div className="mt-2.5 flex items-center gap-1 whitespace-nowrap text-ui-xs leading-none">
            <Arrow className={cn("size-3.5 shrink-0", INK[tone])} strokeWidth={2.4} aria-hidden />
            <span className={cn("font-semibold", INK[tone])}>
              {Math.abs(delta)}%
            </span>
            <span className="text-tl-faint">from yesterday</span>
          </div>
        </div>

        <Sparkline
          points={stats.trends[def.key] ?? []}
          stroke={STROKE[def.accent]}
          className="mb-0.5 shrink-0"
        />
      </div>
    </Link>
  );
}

export function StatGrid({ stats }: { stats: StatSummary }) {
  return (
    <div className="grid grid-cols-[repeat(auto-fit,minmax(210px,1fr))] gap-4">
      {STATS.map((def) => (
        <StatCard key={def.key} def={def} stats={stats} />
      ))}
    </div>
  );
}
