"use client";

import { usePathname } from "next/navigation";
import { Menu, Search } from "lucide-react";

import type { Notification } from "@/lib/staff/types";
import { CommandTrigger } from "./command-palette";
import { NotificationsMenu } from "./notifications-menu";
import { ProfileMenu } from "./profile-menu";

/*
  The workspace header.

  There is deliberately no "New Ticket" action here: customers open tickets,
  agents answer them. The space it occupied now holds the three things an agent
  actually reaches for — search, notifications and their own account.

  The header stays put while content scrolls. That is the one piece of fixed
  chrome besides the rail, and it earns it by holding the global actions.
*/

/** Titles for routes that are not the dashboard; the dashboard greets instead. */
const TITLES: Record<string, string> = {
  "/staff/tickets": "Tickets",
  "/staff/knowledge-base": "Knowledge Base",
  "/staff/reports": "Reports",
  "/staff/insights": "AI Insights",
  "/staff/notifications": "Notifications",
};

function titleFor(pathname: string): string | null {
  if (pathname === "/staff") return null;
  if (pathname.startsWith("/staff/tickets")) return TITLES["/staff/tickets"];
  return TITLES[pathname] ?? null;
}

export function Header({
  agent,
  notifications,
  onOpenMenu,
  onOpenSearch,
}: {
  agent: { name: string; initials: string; email: string };
  notifications: Notification[];
  onOpenMenu: () => void;
  onOpenSearch: () => void;
}) {
  const pathname = usePathname();
  const title = titleFor(pathname);

  return (
    <header className="flex shrink-0 items-center gap-3 border-b border-tl-line bg-tl-canvas/80 px-4 py-3 backdrop-blur-sm lg:px-6">
      <button
        type="button"
        onClick={onOpenMenu}
        aria-label="Open navigation"
        className="-ml-1 inline-flex size-10 items-center justify-center rounded-btn text-tl-ink-soft transition-colors duration-150 hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30 lg:hidden"
      >
        <Menu className="size-5" aria-hidden />
      </button>

      {title ? (
        <h1 className="truncate text-ui-lg font-semibold tracking-[-0.01em] text-tl-ink">
          {title}
        </h1>
      ) : (
        <span className="sr-only">Dashboard</span>
      )}

      <div className="ml-auto flex items-center gap-2 md:gap-3">
        <div className="hidden md:block">
          <CommandTrigger onOpen={onOpenSearch} />
        </div>

        {/* Below md the trigger collapses to an icon so the row still fits. */}
        <button
          type="button"
          onClick={onOpenSearch}
          aria-label="Search tickets"
          className="tap-target inline-flex size-10 items-center justify-center rounded-btn text-tl-ink-soft transition-colors duration-150 hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30 md:hidden"
        >
          <Search className="size-5" strokeWidth={1.8} aria-hidden />
        </button>

        <NotificationsMenu notifications={notifications} />
        <ProfileMenu
          name={agent.name}
          initials={agent.initials}
          email={agent.email}
        />
      </div>
    </header>
  );
}
