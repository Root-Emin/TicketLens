import { cn } from "@/lib/utils";
import type { Accent, Priority, TicketStatus } from "@/lib/staff/types";

/*
  One place decides what a colour means. Everything that tints a surface —
  priority chips, status chips, stat icons, AI predictions — resolves through
  these maps, which is what keeps the six stat cards and the details panel from
  drifting apart as either is edited.
*/

export const TINT: Record<Accent, string> = {
  blue: "bg-blue-50 text-blue-700",
  green: "bg-green-50 text-green-700",
  orange: "bg-amber-50 text-amber-700",
  red: "bg-red-50 text-red-700",
  neutral: "bg-tl-line-soft text-tl-ink-soft",
};

export const DOT: Record<Accent, string> = {
  blue: "bg-tl-blue",
  green: "bg-tl-green",
  orange: "bg-tl-orange",
  red: "bg-tl-red",
  neutral: "bg-slate-400",
};

export const INK: Record<Accent, string> = {
  blue: "text-tl-blue",
  green: "text-tl-green-ink",
  orange: "text-tl-orange-ink",
  red: "text-tl-red-ink",
  neutral: "text-tl-muted",
};

export const STROKE: Record<Accent, string> = {
  blue: "#2563eb",
  green: "#22c55e",
  orange: "#f59e0b",
  red: "#ef4444",
  neutral: "#94a3b8",
};

export const PRIORITY_ACCENT: Record<Priority, Accent> = {
  low: "green",
  medium: "orange",
  high: "red",
};

export const PRIORITY_LABEL: Record<Priority, string> = {
  low: "Low",
  medium: "Medium",
  high: "High",
};

export const STATUS_ACCENT: Record<TicketStatus, Accent> = {
  open: "blue",
  pending: "orange",
  on_hold: "neutral",
  resolved: "green",
  closed: "neutral",
};

export const STATUS_LABEL: Record<TicketStatus, string> = {
  open: "Open",
  pending: "Pending",
  on_hold: "On Hold",
  resolved: "Resolved",
  closed: "Closed",
};

export function Chip({
  accent,
  dot,
  className,
  children,
}: {
  accent: Accent;
  /** Leading status dot. Used by status chips, not by priority chips. */
  dot?: boolean;
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1.5 rounded-md px-2 py-[3px] text-ui-xs font-semibold leading-none whitespace-nowrap",
        TINT[accent],
        className,
      )}
    >
      {dot && (
        <span className={cn("size-1.5 rounded-full", DOT[accent])} aria-hidden />
      )}
      {children}
    </span>
  );
}

export function PriorityBadge({
  priority,
  className,
}: {
  priority: Priority;
  className?: string;
}) {
  return (
    <Chip accent={PRIORITY_ACCENT[priority]} className={className}>
      {PRIORITY_LABEL[priority]}
    </Chip>
  );
}

export function StatusBadge({
  status,
  className,
}: {
  status: TicketStatus;
  className?: string;
}) {
  return (
    <Chip accent={STATUS_ACCENT[status]} dot className={className}>
      {STATUS_LABEL[status]}
    </Chip>
  );
}

/** Neutral round counter, e.g. the total beside a list heading. */
export function CountPill({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <span
      className={cn(
        "inline-flex h-5 min-w-5 items-center justify-center rounded-full bg-tl-line-soft px-1.5 text-ui-xs font-semibold text-tl-muted tabular-nums",
        className,
      )}
    >
      {children}
    </span>
  );
}
