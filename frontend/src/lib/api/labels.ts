import type { Category, TicketPriority, TicketStatus } from "./types";

/*
  Human-readable labels and Tailwind class hints for the taxonomy enums. Kept in
  one place so the queue, detail and dashboard render a category or priority the
  same way everywhere. The identifiers themselves stay the frozen snake_case
  values from backend/docs/taxonomy.md.
*/

export const CATEGORY_LABELS: Record<Category, string> = {
  technical_issue: "Technical Issue",
  integration: "Integration",
  payment_ops: "Payment Ops",
  billing: "Billing",
  onboarding: "Onboarding",
  how_to: "How-To",
  account_access: "Account Access",
  feature_request: "Feature Request",
  sales: "Sales",
  compliance: "Compliance",
};

export const ALL_CATEGORIES: Category[] = [
  "technical_issue",
  "integration",
  "payment_ops",
  "billing",
  "onboarding",
  "how_to",
  "account_access",
  "feature_request",
  "sales",
  "compliance",
];

export const STATUS_LABELS: Record<TicketStatus, string> = {
  open: "Open",
  in_progress: "In Progress",
  pending_customer: "Pending Customer",
  resolved: "Resolved",
  closed: "Closed",
};

export const ALL_STATUSES: TicketStatus[] = [
  "open",
  "in_progress",
  "pending_customer",
  "resolved",
  "closed",
];

export const PRIORITY_LABELS: Record<TicketPriority, string> = {
  low: "Low",
  normal: "Normal",
  high: "High",
  urgent: "Urgent",
};

export const ALL_PRIORITIES: TicketPriority[] = ["low", "normal", "high", "urgent"];

/** priorityClasses returns badge styling keyed to business impact. */
export function priorityClasses(p: TicketPriority): string {
  switch (p) {
    case "urgent":
      return "bg-red-500/15 text-red-600 dark:text-red-400 ring-red-500/30";
    case "high":
      return "bg-amber-500/15 text-amber-600 dark:text-amber-400 ring-amber-500/30";
    case "normal":
      return "bg-sky-500/15 text-sky-600 dark:text-sky-400 ring-sky-500/30";
    case "low":
      return "bg-slate-500/15 text-slate-500 dark:text-slate-400 ring-slate-500/30";
  }
}

/** statusClasses returns badge styling for the workflow state. */
export function statusClasses(s: TicketStatus): string {
  switch (s) {
    case "open":
      return "bg-emerald-500/15 text-emerald-600 dark:text-emerald-400 ring-emerald-500/30";
    case "in_progress":
      return "bg-indigo-500/15 text-indigo-600 dark:text-indigo-400 ring-indigo-500/30";
    case "pending_customer":
      return "bg-amber-500/15 text-amber-600 dark:text-amber-400 ring-amber-500/30";
    case "resolved":
      return "bg-slate-500/15 text-slate-500 dark:text-slate-400 ring-slate-500/30";
    case "closed":
      return "bg-slate-500/10 text-slate-400 ring-slate-500/20";
  }
}
