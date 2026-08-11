"use client";

import { useState } from "react";
import Link from "next/link";
import { ChevronsUpDown, Inbox, LifeBuoy, LogOut, SlidersHorizontal } from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/shadcn/dropdown-menu";
import { InitialsAvatar, PresenceAvatar } from "@/components/staff/primitives";
import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";

/*
  The signed-in administrator, at the foot of the rail.

  Deliberately the only account surface in this panel: the staff panel puts its
  profile menu in the topbar, but that topbar also carries search and
  notifications, neither of which this panel has. One account control, at the
  bottom of the navigation, is where an administrator looks for it.

  Sign-out posts to the same /api/auth/logout route the other two panels use,
  which expires both the token and the role cookie, then hard-navigates so
  nothing of the previous session survives in the React Query cache.
*/

export interface AdminAccount {
  name: string;
  email: string;
  initials: string;
  /**
   * The panel this session resolved to, from the login-time role cookie
   * (lib/auth/roles.ts). Shown rather than a role name because the cookie holds
   * a panel, not a role, and the two are not the same claim.
   */
  panelLabel: string;
}

export function AdminAccountMenu({
  account,
  collapsed,
  onNavigate,
}: {
  account: AdminAccount;
  collapsed: boolean;
  onNavigate?: () => void;
}) {
  const [signingOut, setSigningOut] = useState(false);

  async function signOut() {
    setSigningOut(true);
    try {
      await fetch("/api/auth/logout", { method: "POST" });
    } finally {
      window.location.href = "/login";
    }
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`Account menu for ${account.name}`}
          className={cn(
            "flex w-full items-center gap-2.5 rounded-card bg-white/[0.06] p-3 text-left transition-colors duration-150 hover:bg-white/[0.1] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue focus-visible:ring-offset-2 focus-visible:ring-offset-tl-navy",
            collapsed && "justify-center bg-transparent p-0 hover:bg-transparent",
          )}
        >
          {collapsed ? (
            <PresenceAvatar
              name={account.name}
              initials={account.initials}
              size={34}
              ringClass="border-tl-navy"
            />
          ) : (
            <>
              <InitialsAvatar
                name={account.name}
                initials={account.initials}
                size={36}
              />
              <span className="min-w-0 flex-1">
                <span className="block truncate text-ui-base font-semibold text-white">
                  {account.name}
                </span>
                <span className="block truncate text-ui-xs text-tl-rail-text">
                  {account.email}
                </span>
              </span>
              <ChevronsUpDown
                className="size-4 shrink-0 text-tl-rail-text"
                aria-hidden
              />
            </>
          )}
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent
        side={collapsed ? "right" : "top"}
        align="start"
        sideOffset={10}
        className="w-60"
      >
        <DropdownMenuLabel className="flex flex-col gap-0.5">
          <span className="text-ui-base font-semibold text-tl-ink">
            {account.name}
          </span>
          <span className="text-ui-xs font-normal text-tl-muted">
            {account.email}
          </span>
          <span className="mt-1 inline-flex w-fit rounded-md bg-tl-blue-soft px-1.5 py-0.5 text-ui-xs font-semibold text-tl-blue">
            {account.panelLabel}
          </span>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />

        <DropdownMenuItem asChild>
          <Link href="/settings" onClick={onNavigate}>
            <SlidersHorizontal className="size-4" aria-hidden />
            Organization settings
          </Link>
        </DropdownMenuItem>

        {/*
          The two narrower panels, reachable because an administrator holds `*`
          and canAccess lets the owner into both (lib/auth/roles.ts). Being able
          to open what your own people see is the point of listing them.
        */}
        <DropdownMenuItem asChild>
          <Link href="/staff" onClick={onNavigate}>
            <Inbox className="size-4" aria-hidden />
            Open the agent panel
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <Link href="/portal" onClick={onNavigate}>
            <LifeBuoy className="size-4" aria-hidden />
            Open the customer portal
          </Link>
        </DropdownMenuItem>

        <DropdownMenuSeparator />
        <DropdownMenuItem
          variant="destructive"
          disabled={signingOut}
          // onSelect rather than onClick: Radix closes the menu on select, and
          // preventing that would leave it open over a navigating page.
          onSelect={signOut}
        >
          {signingOut ? (
            <Spinner className="size-4" />
          ) : (
            <LogOut className="size-4" aria-hidden />
          )}
          Sign out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
