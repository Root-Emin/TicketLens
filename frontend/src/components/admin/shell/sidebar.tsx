"use client";

import Link from "next/link";
import { PanelLeftClose, PanelLeftOpen } from "lucide-react";

import {
  activeClass,
  idleClass,
  rowClass,
} from "@/components/staff/shell/sidebar-item";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/shadcn/tooltip";
import { Skeleton } from "@/components/shadcn/skeleton";
import { usePersistentState } from "@/hooks/use-persistent-state";
import { cn } from "@/lib/utils";
import { isActive, visibleNav, type AdminNavLink } from "./nav-config";
import { AdminAccountMenu, type AdminAccount } from "./account-menu";

/*
  The management rail.

  Same navy surface, same 36px rows, same blue edge marker and the same focus
  ring as the staff and portal rails — the row classes are imported from
  components/staff/shell/sidebar-item rather than re-typed, which is what stops
  the three panels drifting apart on a hover colour. An administrator who opens
  /staff to check what their agents see should not feel they have changed
  product.

  What is different is structure, and only structure. The agent's rail is a set
  of saved views over one queue, so it nests and it counts. This one is five
  destinations across four jobs, so it groups and it does not count: a number
  beside "Departments" would be a headcount pretending to be an inbox.

  The workspace name sits under the wordmark because this panel is the only
  place in TicketLens where "which organization am I administering" is a
  question worth answering on every screen.
*/

interface SidebarProps {
  pathname: string;
  search: URLSearchParams;
  organization: { name: string; slug: string } | null;
  account: AdminAccount | null;
  /** Role names from the session token, for hiding rows this session cannot use. */
  roles: string[];
  /** The mobile drawer passes this so following a link closes the drawer. */
  onNavigate?: () => void;
  /** The drawer renders the rail permanently expanded. */
  forceExpanded?: boolean;
}

export function AdminSidebar({
  pathname,
  search,
  organization,
  account,
  roles,
  onNavigate,
  forceExpanded = false,
}: SidebarProps) {
  const groups = visibleNav(roles);
  const [storedCollapsed, setCollapsed] = usePersistentState(
    "tl.admin.sidebar.collapsed",
    false,
  );
  const collapsed = forceExpanded ? false : storedCollapsed;

  return (
    <div
      className={cn(
        "flex h-full flex-col bg-tl-navy transition-[width] duration-200",
        collapsed ? "w-[68px]" : "w-[254px]",
      )}
    >
      <div
        className={cn(
          "flex shrink-0 flex-col gap-3 px-5 py-[18px]",
          collapsed && "items-center px-0",
        )}
      >
        <Link
          href="/dashboard"
          onClick={onNavigate}
          aria-label="TicketLens overview"
          className="flex items-center gap-2.5 rounded-btn focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue focus-visible:ring-offset-2 focus-visible:ring-offset-tl-navy"
        >
          {/* eslint-disable-next-line @next/next/no-img-element -- an SVG at a
              fixed 32px; next/image would add a layout wrapper and a request
              round-trip for a file already in public/. Matches the staff and
              portal rails. */}
          <img
            src="/assets/TicketLens_Logo/favicon.svg"
            alt="TicketLens Logo"
            className="size-8 shrink-0 rounded-[9px] object-contain shadow-sm"
          />
          {!collapsed && (
            <span className="text-[17px] font-bold tracking-[-0.015em] text-white">
              TicketLens
            </span>
          )}
        </Link>

        {!collapsed && <WorkspaceChip organization={organization} />}
      </div>

      <nav
        aria-label="Administration"
        className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-3 pb-2"
      >
        {groups.map((group, index) => (
          <div key={group.caption} className={index > 0 ? "mt-1" : undefined}>
            {collapsed ? (
              // A caption cannot be read at 68px, and dropping it silently would
              // run four groups together. The hairline keeps the grouping.
              index > 0 && <div className="mx-2 my-2.5 h-px bg-white/[0.07]" />
            ) : (
              <div className="px-3 pb-1.5 pt-4 text-ui-xs font-medium uppercase tracking-[0.06em] text-tl-rail-caption">
                {group.caption}
              </div>
            )}

            <div className="space-y-0.5">
              {group.links.map((link) => (
                <NavRow
                  key={link.href}
                  link={link}
                  active={isActive(link, pathname, search)}
                  collapsed={collapsed}
                  onNavigate={onNavigate}
                />
              ))}
            </div>
          </div>
        ))}
      </nav>

      <div className="shrink-0 border-t border-white/[0.07] p-3">
        {!forceExpanded && (
          <CollapseToggle
            collapsed={collapsed}
            onToggle={() => setCollapsed(!collapsed)}
          />
        )}

        <div className="mt-2">
          {account ? (
            <AdminAccountMenu
              account={account}
              collapsed={collapsed}
              onNavigate={onNavigate}
            />
          ) : (
            <div
              className={cn(
                "flex items-center gap-2.5 rounded-card bg-white/[0.06] p-3",
                collapsed && "justify-center bg-transparent p-0",
              )}
              aria-hidden
            >
              <Skeleton className="size-9 shrink-0 rounded-full bg-white/10" />
              {!collapsed && (
                <div className="min-w-0 flex-1 space-y-1.5">
                  <Skeleton className="h-3 w-24 bg-white/10" />
                  <Skeleton className="h-2.5 w-32 bg-white/10" />
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

/**
 * Which organization is being administered.
 *
 * Read-only: the token carries exactly one organization and there is no
 * switcher endpoint, so this is a label rather than a disabled control that
 * invites a click it cannot honour.
 */
function WorkspaceChip({
  organization,
}: {
  organization: { name: string; slug: string } | null;
}) {
  if (!organization) {
    return <Skeleton className="h-[42px] w-full rounded-btn bg-white/10" aria-hidden />;
  }

  return (
    <div className="rounded-btn bg-white/[0.06] px-2.5 py-2">
      <div className="truncate text-ui-sm font-semibold text-white">
        {organization.name}
      </div>
      <div className="truncate text-ui-xs text-tl-rail-text">
        Admin workspace · {organization.slug}
      </div>
    </div>
  );
}

function NavRow({
  link,
  active,
  collapsed,
  onNavigate,
}: {
  link: AdminNavLink;
  active: boolean;
  collapsed: boolean;
  onNavigate?: () => void;
}) {
  const Icon = link.icon;

  const body = (
    <Link
      href={link.href}
      onClick={onNavigate}
      aria-current={active ? "page" : undefined}
      className={cn(
        rowClass,
        active ? activeClass : idleClass,
        collapsed && "justify-center px-0",
      )}
    >
      {/* A rail-edge bar rather than a border, so selection moving does not
          shift the label by a pixel. */}
      {active && !collapsed && (
        <span
          className="absolute inset-y-1 left-0 w-[3px] rounded-r bg-tl-blue"
          aria-hidden
        />
      )}
      <Icon className="size-[18px] shrink-0" strokeWidth={1.8} aria-hidden />
      {!collapsed && <span className="truncate">{link.label}</span>}
    </Link>
  );

  if (!collapsed) return body;

  // Collapsed: the label only exists in the tooltip, so it carries the
  // accessible name.
  return (
    <Tooltip>
      <TooltipTrigger asChild>{body}</TooltipTrigger>
      <TooltipContent side="right" sideOffset={8}>
        {link.label}
      </TooltipContent>
    </Tooltip>
  );
}

function CollapseToggle({
  collapsed,
  onToggle,
}: {
  collapsed: boolean;
  onToggle: () => void;
}) {
  const Icon = collapsed ? PanelLeftOpen : PanelLeftClose;
  const label = collapsed ? "Expand sidebar" : "Collapse sidebar";

  return (
    <button
      type="button"
      onClick={onToggle}
      aria-label={label}
      aria-pressed={collapsed}
      className={cn(
        rowClass,
        idleClass,
        "hidden w-full lg:flex",
        collapsed && "justify-center px-0",
      )}
    >
      <Icon className="size-[18px] shrink-0" strokeWidth={1.8} aria-hidden />
      {!collapsed && <span className="truncate">Collapse</span>}
    </button>
  );
}
