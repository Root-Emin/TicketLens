"use client";

import Link from "next/link";

import { cn } from "@/lib/utils";
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/shadcn/tooltip";
import type { NavLink } from "./nav-config";

/*
  A single rail row.

  Rows are 36px tall in the expanded rail but present a 44px touch target on
  coarse pointers via the padded hit area, which is what keeps the nav usable
  on a tablet without making the desktop rail feel loose.
*/

const PILL_TONE = {
  blue: "bg-tl-blue text-white",
  red: "bg-tl-red text-white",
  orange: "bg-tl-orange text-white",
} as const;

export const rowClass =
  "group relative flex h-9 items-center gap-3 rounded-btn px-3 text-ui-base font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue focus-visible:ring-offset-2 focus-visible:ring-offset-tl-navy";

export const idleClass =
  "text-tl-rail-text hover:bg-tl-navy-hover/70 hover:text-white";
export const activeClass = "bg-tl-navy-hover text-white";

export function SidebarItem({
  link,
  active,
  collapsed,
  count,
  indent,
  onNavigate,
}: {
  link: NavLink;
  active: boolean;
  collapsed: boolean;
  count?: number;
  /** Sub-items sit under a parent and lose their icon in the expanded rail. */
  indent?: boolean;
  onNavigate?: () => void;
}) {
  const Icon = link.icon;

  const badge =
    count !== undefined && count > 0 ? (
      <span
        className={cn(
          "ml-auto inline-flex h-[19px] min-w-[22px] items-center justify-center rounded-full px-1.5 text-ui-xs font-semibold tabular-nums",
          link.tone ? PILL_TONE[link.tone] : "text-tl-rail-text",
        )}
      >
        {count}
      </span>
    ) : null;

  const body = (
    <Link
      href={link.href}
      onClick={onNavigate}
      aria-current={active ? "page" : undefined}
      className={cn(
        rowClass,
        active ? activeClass : idleClass,
        collapsed && "justify-center px-0",
        indent && !collapsed && "h-[30px] text-ui-sm",
      )}
    >
      {/* The active marker is a rail-edge bar rather than a border, so it does
          not shift the label by a pixel when selection moves. */}
      {active && !collapsed && (
        <span
          className="absolute inset-y-1 left-0 w-[3px] rounded-r bg-tl-blue"
          aria-hidden
        />
      )}
      {(!indent || collapsed) && (
        <Icon className="size-[18px] shrink-0" strokeWidth={1.8} aria-hidden />
      )}
      {!collapsed && <span className="truncate">{link.label}</span>}
      {!collapsed && badge}
      {collapsed && count !== undefined && count > 0 && link.tone && (
        <span
          className={cn(
            "absolute right-1.5 top-1.5 size-2 rounded-full",
            PILL_TONE[link.tone],
          )}
          aria-hidden
        />
      )}
    </Link>
  );

  if (!collapsed) return body;

  // Collapsed: the label only exists in the tooltip, so it carries the
  // accessible name and the count with it.
  return (
    <Tooltip>
      <TooltipTrigger asChild>{body}</TooltipTrigger>
      <TooltipContent side="right" sideOffset={8}>
        {link.label}
        {count !== undefined && count > 0 ? ` · ${count}` : ""}
      </TooltipContent>
    </Tooltip>
  );
}
