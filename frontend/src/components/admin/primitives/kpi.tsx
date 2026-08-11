import Link from "next/link";
import type { LucideIcon } from "lucide-react";

import { Skeleton } from "@/components/shadcn/skeleton";
import { INK, TINT } from "@/components/staff/primitives";
import type { Accent } from "@/lib/staff/types";
import { cn } from "@/lib/utils";

/*
  Compact metric tiles.

  Deliberately smaller than the staff dashboard's stat cards, which are 5 units
  of padding, a 26px figure, a sparkline and a day-over-day delta each. Those
  belong on a screen whose only job is the numbers. Here the numbers are a
  summary above a table that is the actual work, so they get one line of label,
  one figure and nothing else — six of them should read as a strip, not as a
  dashboard the table happens to sit under.

  Tinting is the shared TINT/INK maps from the staff primitives, so "green means
  healthy, amber means look at this" carries the same meaning it does everywhere
  else in the product. Most tiles are neutral: colouring all six would mean
  colouring none.
*/

export interface KpiDef {
  label: string;
  value: number | string;
  icon: LucideIcon;
  accent: Accent;
  /** Where clicking goes, when the number leads somewhere real. */
  href?: string;
  /** One short clause under the figure. */
  hint?: string;
}

export function KpiRow({ items }: { items: KpiDef[] }) {
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-6">
      {items.map((item) => (
        <KpiCard key={item.label} {...item} />
      ))}
    </div>
  );
}

function KpiCard({ label, value, icon: Icon, accent, href, hint }: KpiDef) {
  const inner = (
    <>
      <div className="flex items-center gap-2">
        <span
          className={cn(
            "flex size-6 shrink-0 items-center justify-center rounded-md",
            TINT[accent],
          )}
        >
          <Icon className="size-3.5" strokeWidth={2.1} aria-hidden />
        </span>
        <span className="min-w-0 flex-1 truncate text-ui-xs font-medium text-tl-muted">
          {label}
        </span>
      </div>

      <div className="mt-2.5">
        <span
          className={cn(
            "text-[22px] font-bold leading-none tracking-[-0.02em] tabular-nums",
            accent === "neutral" ? "text-tl-ink" : INK[accent],
          )}
        >
          {value}
        </span>
        {hint && (
          <span className="mt-1.5 block truncate text-ui-xs text-tl-faint">
            {hint}
          </span>
        )}
      </div>
    </>
  );

  const base =
    "rounded-card border border-tl-line bg-tl-card px-3.5 py-3 shadow-panel";

  if (!href) {
    return <div className={base}>{inner}</div>;
  }

  return (
    <Link
      href={href}
      className={cn(
        base,
        "block transition-colors duration-150 hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30",
      )}
    >
      {inner}
    </Link>
  );
}

export function KpiRowSkeleton({ count = 6 }: { count?: number }) {
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 xl:grid-cols-6" aria-hidden>
      {Array.from({ length: count }).map((_, index) => (
        <div
          key={index}
          className="rounded-card border border-tl-line bg-tl-card px-3.5 py-3 shadow-panel"
        >
          <div className="flex items-center gap-2">
            <Skeleton className="size-6 shrink-0 rounded-md" />
            <Skeleton className="h-3 w-full max-w-[70px]" />
          </div>
          <Skeleton className="mt-3 h-5 w-10" />
        </div>
      ))}
    </div>
  );
}
