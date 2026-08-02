import { Badge } from "@/components/ui/badge";
import {
  CATEGORY_LABELS,
  PRIORITY_LABELS,
  STATUS_LABELS,
  priorityClasses,
  statusClasses,
} from "@/lib/api/labels";
import type { Category, TicketPriority, TicketStatus } from "@/lib/api/types";

export function PriorityBadge({ priority }: { priority: TicketPriority }) {
  return <Badge tone={priorityClasses(priority)}>{PRIORITY_LABELS[priority]}</Badge>;
}

export function StatusBadge({ status }: { status: TicketStatus }) {
  return <Badge tone={statusClasses(status)}>{STATUS_LABELS[status]}</Badge>;
}

export function CategoryBadge({ category }: { category: Category | null }) {
  if (!category) return <Badge>Unclassified</Badge>;
  return <Badge>{CATEGORY_LABELS[category]}</Badge>;
}
