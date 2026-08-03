import {
  BellRing,
  BookOpen,
  CheckCircle2,
  CircleDot,
  Clock3,
  FileText,
  Inbox,
  LayoutDashboard,
  MinusCircle,
  Sparkles,
  UserRound,
  UsersRound,
  type LucideIcon,
} from "lucide-react";

import type { ViewId } from "@/lib/staff/types";

/*
  The navigation model.

  Icons live here rather than in the data layer: which glyph represents
  "Needs Attention" is a rendering decision, and keeping lucide out of
  lib/staff means the fixtures can be swapped for API responses without
  dragging an icon library along.

  Every destination is a real URL. Saved views are query parameters over one
  queue rather than separate screens, so adding a view is a line here plus a
  predicate in lib/staff/filters.ts.

  There is no Teams section. An agent sees their own department's queue and
  nothing else, so cross-team navigation would only ever lead to an empty list
  — the scoping lives in lib/staff/queries.ts.
*/

export interface NavLink {
  label: string;
  icon: LucideIcon;
  href: string;
  /** Present when this link is a saved view over the ticket queue. */
  view?: ViewId;
  /** Which sidebar count to show, if any. */
  countKey?: ViewId;
  /** Filled pill instead of plain text. Reserved for the workspace group. */
  tone?: "blue" | "red" | "orange";
}

export const viewHref = (view: ViewId) => `/staff/tickets?view=${view}`;

export const DASHBOARD: NavLink = {
  label: "Dashboard",
  icon: LayoutDashboard,
  href: "/staff",
};

export const WORKSPACE_LINKS: NavLink[] = [
  {
    label: "Assigned to Me",
    icon: UsersRound,
    href: viewHref("assigned"),
    view: "assigned",
    countKey: "assigned",
    tone: "blue",
  },
  {
    label: "Needs Attention",
    icon: BellRing,
    href: viewHref("attention"),
    view: "attention",
    countKey: "attention",
    tone: "red",
  },
  {
    label: "Waiting for Customer",
    icon: Clock3,
    href: viewHref("waiting"),
    view: "waiting",
    countKey: "waiting",
    tone: "orange",
  },
  {
    label: "Resolved by Me",
    icon: CheckCircle2,
    href: viewHref("resolved"),
    view: "resolved",
  },
];

export const ALL_TICKETS: NavLink = {
  label: "All Tickets",
  icon: Inbox,
  href: viewHref("all"),
  view: "all",
};

export const ALL_TICKETS_LINKS: NavLink[] = [
  {
    label: "Unassigned",
    icon: UserRound,
    href: viewHref("unassigned"),
    view: "unassigned",
    countKey: "unassigned",
  },
  {
    label: "All Open",
    icon: CircleDot,
    href: viewHref("open"),
    view: "open",
    countKey: "open",
  },
  {
    label: "On Hold",
    icon: MinusCircle,
    href: viewHref("hold"),
    view: "hold",
    countKey: "hold",
  },
  {
    label: "Closed",
    icon: CheckCircle2,
    href: viewHref("closed"),
    view: "closed",
    countKey: "closed",
  },
];

export const TOOL_LINKS: NavLink[] = [
  { label: "Knowledge Base", icon: BookOpen, href: "/staff/knowledge-base" },
  { label: "Reports", icon: FileText, href: "/staff/reports" },
  { label: "AI Insights", icon: Sparkles, href: "/staff/insights" },
];

export const NOTIFICATIONS_LINK: NavLink = {
  label: "Notifications",
  icon: BellRing,
  href: "/staff/notifications",
};

/** Flat list of every destination, for the command palette. */
export const ALL_DESTINATIONS: NavLink[] = [
  DASHBOARD,
  ...WORKSPACE_LINKS,
  ALL_TICKETS,
  ...ALL_TICKETS_LINKS,
  ...TOOL_LINKS,
  NOTIFICATIONS_LINK,
];
