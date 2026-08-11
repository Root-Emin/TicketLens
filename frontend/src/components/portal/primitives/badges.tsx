import { Chip } from "@/components/staff/primitives";
import type { Accent } from "@/lib/staff/types";
import { PRIORITY_LABELS, STATUS_LABELS } from "@/lib/api/labels";
import type { TicketPriority, TicketStatus } from "@/lib/api/types";

/*
  Status and priority chips for the portal.

  The chip itself is the shared primitive the staff panel already uses, so both
  panels tint a status the same way. What differs is the vocabulary: the staff
  fixtures speak a three-value priority and a five-value status of their own,
  while the portal reads the API enums from docs/taxonomy.md. Only the mapping
  lives here; the labels come from lib/api/labels.ts, which is already the one
  place those strings are written down.
*/

const STATUS_ACCENT: Record<TicketStatus, Accent> = {
  open: "blue",
  in_progress: "blue",
  pending_customer: "orange",
  resolved: "green",
  closed: "neutral",
};

const PRIORITY_ACCENT: Record<TicketPriority, Accent> = {
  low: "neutral",
  normal: "blue",
  high: "orange",
  urgent: "red",
};

export function TicketStatusBadge({
  status,
  className,
}: {
  status: TicketStatus;
  className?: string;
}) {
  return (
    <Chip accent={STATUS_ACCENT[status]} dot className={className}>
      {STATUS_LABELS[status]}
    </Chip>
  );
}

export function TicketPriorityBadge({
  priority,
  className,
}: {
  priority: TicketPriority;
  className?: string;
}) {
  return (
    <Chip accent={PRIORITY_ACCENT[priority]} className={className}>
      {PRIORITY_LABELS[priority]}
    </Chip>
  );
}

/** True once a ticket is done, which is what closes the reply box. */
export function isClosedStatus(status: TicketStatus): boolean {
  return status === "resolved" || status === "closed";
}
