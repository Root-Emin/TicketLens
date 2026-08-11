import {
  CircleUser,
  LayoutDashboard,
  LifeBuoy,
  Ticket,
  type LucideIcon,
} from "lucide-react";

/*
  The portal's navigation model, mirroring components/staff/shell/nav-config.ts:
  icons live with the routes rather than in the data layer, and every entry is a
  real URL.

  Four destinations and no saved views. A customer has one queue — their own —
  so the filtering that justifies the staff panel's collapsible groups would
  only add depth to a list of four.
*/

export interface PortalNavLink {
  label: string;
  icon: LucideIcon;
  href: string;
  /** Sub-routes that should keep this item highlighted. */
  match?: (pathname: string) => boolean;
}

export const PORTAL_NAV: PortalNavLink[] = [
  { label: "Dashboard", icon: LayoutDashboard, href: "/portal" },
  {
    label: "My Tickets",
    icon: Ticket,
    href: "/portal/tickets",
    match: (pathname) => pathname.startsWith("/portal/tickets"),
  },
  { label: "Help Center", icon: LifeBuoy, href: "/portal/help" },
  { label: "Profile", icon: CircleUser, href: "/portal/profile" },
];

/** isActive decides which rail row is highlighted for the current route. */
export function isActive(link: PortalNavLink, pathname: string): boolean {
  return link.match ? link.match(pathname) : pathname === link.href;
}

/** Titles the topbar shows next to the menu button on small screens. */
export const PORTAL_TITLES: Record<string, string> = {
  "/portal": "Dashboard",
  "/portal/tickets": "My Tickets",
  "/portal/help": "Help Center",
  "/portal/profile": "Profile",
};
