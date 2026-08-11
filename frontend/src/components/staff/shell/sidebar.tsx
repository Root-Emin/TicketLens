"use client";

import Link from "next/link";
import { ChevronDown, PanelLeftClose, PanelLeftOpen } from "lucide-react";

import { cn } from "@/lib/utils";
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from "@/components/shadcn/collapsible";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/shadcn/tooltip";
import { usePersistentState } from "@/hooks/use-persistent-state";
import type { SidebarCounts } from "@/lib/staff/queries";
import type { ViewId } from "@/lib/staff/types";
import { PresenceAvatar } from "../primitives";
import {
  ALL_TICKETS,
  ALL_TICKETS_LINKS,
  DASHBOARD,
  NOTIFICATIONS_LINK,
  TOOL_LINKS,
  WORKSPACE_LINKS,
  type NavLink,
} from "./nav-config";
import {
  activeClass,
  idleClass,
  rowClass,
  SidebarItem,
} from "./sidebar-item";

export interface SidebarState {
  activeView: ViewId | null;
  onTickets: boolean;
  pathname: string;
}

interface SidebarProps extends SidebarState {
  counts: SidebarCounts;
  agent: { name: string; initials: string; role: string };
  /** Mobile drawer passes this so following a link closes the drawer. */
  onNavigate?: () => void;
  /** The drawer renders the rail permanently expanded. */
  forceExpanded?: boolean;
}

function GroupCaption({ children }: { children: React.ReactNode }) {
  return (
    <div className="px-3 pb-1.5 pt-4 text-ui-xs font-medium text-tl-rail-caption">
      {children}
    </div>
  );
}

/** A collapsible nav section with a chevron that rotates on open. */
function NavGroup({
  label,
  icon: Icon,
  defaultOpen,
  collapsed,
  active,
  children,
}: {
  label: string;
  icon: NavLink["icon"];
  defaultOpen?: boolean;
  collapsed: boolean;
  active?: boolean;
  children: React.ReactNode;
}) {
  // Collapsed rail has no room for a disclosure; the group header becomes a
  // plain icon link and the children are reachable from the expanded rail.
  if (collapsed) return null;

  return (
    <Collapsible defaultOpen={defaultOpen}>
      <CollapsibleTrigger
        className={cn(rowClass, "w-full", active ? activeClass : idleClass)}
      >
        <Icon className="size-[18px] shrink-0" strokeWidth={1.8} aria-hidden />
        <span className="truncate">{label}</span>
        <ChevronDown
          className="ml-auto size-4 shrink-0 transition-transform duration-200 group-data-[state=open]:rotate-180"
          aria-hidden
        />
      </CollapsibleTrigger>
      <CollapsibleContent className="overflow-hidden data-[state=closed]:animate-collapsible-up data-[state=open]:animate-collapsible-down">
        <div className="mt-0.5 space-y-0.5 pl-[26px]">{children}</div>
      </CollapsibleContent>
    </Collapsible>
  );
}

export function Sidebar({
  counts,
  agent,
  activeView,
  onTickets,
  pathname,
  onNavigate,
  forceExpanded = false,
}: SidebarProps) {
  const [storedCollapsed, setCollapsed] = usePersistentState(
    "tl.sidebar.collapsed",
    false,
  );
  const collapsed = forceExpanded ? false : storedCollapsed;

  const isView = (link: NavLink) => onTickets && link.view === activeView;

  const countFor = (link: NavLink) =>
    link.countKey ? counts.views[link.countKey] : undefined;

  return (
    <div
      className={cn(
        "flex h-full flex-col bg-tl-navy transition-[width] duration-200",
        collapsed ? "w-[68px]" : "w-[250px]",
      )}
    >
      {/* Brand */}
      <div
        className={cn(
          "flex shrink-0 items-center gap-2.5 px-5 py-[18px]",
          collapsed && "justify-center px-0",
        )}
      >
        <Link
          href={DASHBOARD.href}
          onClick={onNavigate}
          aria-label="TicketLens dashboard"
          className="flex items-center gap-2.5 rounded-btn focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue focus-visible:ring-offset-2 focus-visible:ring-offset-tl-navy"
        >
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
      </div>

      <nav
        aria-label="Main"
        className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-3 pb-2"
      >
        <SidebarItem
          link={DASHBOARD}
          active={pathname === DASHBOARD.href}
          collapsed={collapsed}
          onNavigate={onNavigate}
        />

        {!collapsed && <GroupCaption>My Workspace</GroupCaption>}
        {collapsed && <div className="h-3" />}

        <div className="space-y-0.5">
          {WORKSPACE_LINKS.map((link) => (
            <SidebarItem
              key={link.label}
              link={link}
              active={isView(link)}
              collapsed={collapsed}
              count={countFor(link)}
              onNavigate={onNavigate}
            />
          ))}
        </div>

        <div className="mt-0.5">
          {collapsed ? (
            <SidebarItem
              link={ALL_TICKETS}
              active={isView(ALL_TICKETS)}
              collapsed
              onNavigate={onNavigate}
            />
          ) : (
            <NavGroup
              label={ALL_TICKETS.label}
              icon={ALL_TICKETS.icon}
              defaultOpen
              collapsed={collapsed}
              active={isView(ALL_TICKETS)}
            >
              {ALL_TICKETS_LINKS.map((link) => (
                <SidebarItem
                  key={link.label}
                  link={link}
                  active={isView(link)}
                  collapsed={false}
                  count={countFor(link)}
                  indent
                  onNavigate={onNavigate}
                />
              ))}
            </NavGroup>
          )}
        </div>


        <Divider />

        <div className="space-y-0.5">
          {TOOL_LINKS.map((link) => (
            <SidebarItem
              key={link.label}
              link={link}
              active={pathname === link.href}
              collapsed={collapsed}
              onNavigate={onNavigate}
            />
          ))}
        </div>

        <Divider />

        <SidebarItem
          link={NOTIFICATIONS_LINK}
          active={pathname === NOTIFICATIONS_LINK.href}
          collapsed={collapsed}
          count={counts.unreadNotifications}
          onNavigate={onNavigate}
        />
      </nav>

      {/* Collapse control + profile */}
      <div className="shrink-0 border-t border-white/[0.07] p-3">
        {!forceExpanded && (
          <CollapseToggle
            collapsed={collapsed}
            onToggle={() => setCollapsed(!collapsed)}
          />
        )}

        <div
          className={cn(
            "mt-2 rounded-card bg-white/[0.06] p-3",
            collapsed && "flex justify-center bg-transparent p-0",
          )}
        >
          {collapsed ? (
            <Tooltip>
              <TooltipTrigger asChild>
                <button
                  type="button"
                  className="rounded-full focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue focus-visible:ring-offset-2 focus-visible:ring-offset-tl-navy"
                >
                  <PresenceAvatar
                    name={agent.name}
                    initials={agent.initials}
                    size={34}
                    ringClass="border-tl-navy"
                  />
                  <span className="sr-only">{agent.name}, online</span>
                </button>
              </TooltipTrigger>
              <TooltipContent side="right" sideOffset={8}>
                {agent.name} · {agent.role}
              </TooltipContent>
            </Tooltip>
          ) : (
            <>
              <div className="flex items-center gap-2.5">
                <PresenceAvatar
                  name={agent.name}
                  initials={agent.initials}
                  size={36}
                  ringClass="border-[#1a2436]"
                />
                <div className="min-w-0 flex-1">
                  <div className="truncate text-ui-base font-semibold text-white">
                    {agent.name}
                  </div>
                  <div className="truncate text-ui-xs text-tl-rail-text">
                    {agent.role}
                  </div>
                </div>
              </div>
              <div className="mt-2.5 flex items-center gap-1.5">
                <span className="size-[7px] rounded-full bg-tl-green" aria-hidden />
                <span className="text-ui-xs text-tl-rail-text">Online</span>
              </div>
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function Divider() {
  return <div className="my-3 h-px bg-white/[0.07]" />;
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
