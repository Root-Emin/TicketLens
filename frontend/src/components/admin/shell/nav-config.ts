import {
  Building2,
  Inbox,
  LayoutDashboard,
  SlidersHorizontal,
  UserRoundX,
  UsersRound,
  type LucideIcon,
} from "lucide-react";

import { canAny, PERMISSION } from "@/lib/auth/permissions";

/*
  The management panel's navigation model.

  Grouped, because the panel is two jobs rather than one: an administrator is
  either looking at their people or at the queue those people work, and a flat
  list of five puts "Departments" next to "Ticket Queue" as though they were the
  same kind of thing.

  Every entry is a route that exists and answers. There are no placeholders
  here — the sections an administrator might expect but the backend cannot serve
  (SLA policy, business hours, invitations, audit trail) are absent rather than
  present-and-dead, because a nav item that leads to "coming soon" costs a click
  every time somebody forgets.

  Two of the five are pre-existing screens (/dashboard, /tickets) reached under
  their real URLs, so nothing about the queue's behaviour or its bookmarks
  changed when the rail was rebuilt around it.
*/

export interface AdminNavLink {
  label: string;
  icon: LucideIcon;
  href: string;
  /** Sub-routes and query states that should keep this row highlighted. */
  match?: (pathname: string, search: URLSearchParams) => boolean;
  /** Short line under the label in the mobile drawer, where there is room. */
  hint?: string;
  /**
   * What the destination's own API calls need. Any one of them is enough,
   * mirroring RequireAnyPermission on the matching route. A row whose session
   * cannot satisfy this is not drawn — see lib/auth/permissions.ts for why that
   * is a hint and not the boundary.
   */
  permissions: string[];
}

export interface AdminNavGroup {
  caption: string;
  links: AdminNavLink[];
}

/** The unassigned queue, spelled once — the rail and the KPI card both link here. */
export const UNASSIGNED_HREF = "/tickets?assignee_id=unassigned";

export const ADMIN_NAV: AdminNavGroup[] = [
  {
    caption: "Overview",
    links: [
      {
        label: "Overview",
        icon: LayoutDashboard,
        href: "/dashboard",
        hint: "Volume, routing and AI accuracy",
        permissions: [PERMISSION.readStats],
      },
    ],
  },
  {
    caption: "Workforce",
    links: [
      /*
        Departments first, and that order is the information architecture rather
        than a preference. A support organization is a set of teams; a person is
        a member of one. So the way in is the team — open Payment Operations,
        see who is on it — and the flat roster below is the cross-cutting view
        for the questions a single team cannot answer: who is on no team at all,
        and who is carrying the most across the whole organization.
      */
      {
        label: "Departments",
        icon: Building2,
        href: "/departments",
        match: (pathname) => pathname.startsWith("/departments"),
        hint: "Teams, their staff and routing",
        // Matches the route: GET /departments takes either, and the write
        // actions inside the page are gated separately on department:manage.
        permissions: [PERMISSION.manageDepartments, PERMISSION.readTickets],
      },
      {
        label: "All staff",
        icon: UsersRound,
        href: "/team",
        match: (pathname) => pathname.startsWith("/team"),
        hint: "Everyone, across every team",
        permissions: [PERMISSION.readUsers],
      },
    ],
  },
  {
    caption: "Operations",
    links: [
      {
        label: "Ticket Queue",
        icon: Inbox,
        href: "/tickets",
        // Not active when the unassigned view below is, so the two rows never
        // both light up on the same URL.
        match: (pathname, search) =>
          pathname.startsWith("/tickets") &&
          search.get("assignee_id") !== "unassigned",
        hint: "Every ticket in the organization",
        permissions: [PERMISSION.readTickets],
      },
      {
        label: "Unassigned",
        icon: UserRoundX,
        href: UNASSIGNED_HREF,
        match: (pathname, search) =>
          pathname.startsWith("/tickets") &&
          search.get("assignee_id") === "unassigned",
        hint: "Waiting for somebody to pick up",
        permissions: [PERMISSION.readTickets],
      },
    ],
  },
  {
    caption: "Settings",
    links: [
      {
        label: "Organization",
        icon: SlidersHorizontal,
        href: "/settings",
        match: (pathname) => pathname.startsWith("/settings"),
        hint: "Workspace details and your account",
        // The account half of this screen — changing your own password — needs
        // no permission at all, so the row survives a session that cannot read
        // the organization and the org card renders its own forbidden state.
        permissions: [PERMISSION.readOrg, PERMISSION.readUsers],
      },
    ],
  },
];

/** isActive decides which rail row is highlighted for the current location. */
export function isActive(
  link: AdminNavLink,
  pathname: string,
  search: URLSearchParams,
): boolean {
  return link.match ? link.match(pathname, search) : pathname === link.href;
}

/**
 * visibleNav drops the rows this session has no business seeing, and any group
 * left empty by that.
 */
export function visibleNav(roles: string[] | undefined): AdminNavGroup[] {
  return ADMIN_NAV.map((group) => ({
    ...group,
    links: group.links.filter((link) => canAny(roles, link.permissions)),
  })).filter((group) => group.links.length > 0);
}

/** Small-screen orientation labels for the topbar. */
export const ADMIN_TITLES: Record<string, string> = {
  "/dashboard": "Overview",
  "/team": "All staff",
  "/departments": "Departments",
  "/tickets": "Ticket Queue",
  "/settings": "Organization",
};

export function titleFor(pathname: string): string {
  if (pathname.startsWith("/tickets/")) return "Ticket";
  if (pathname.startsWith("/departments/")) return "Department";
  return ADMIN_TITLES[pathname] ?? "TicketLens";
}
