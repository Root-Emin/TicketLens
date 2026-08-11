"use client";

import { useState } from "react";
import Link from "next/link";
import { ChevronDown, CircleUser, LifeBuoy, LogOut } from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/shadcn/dropdown-menu";
import { InitialsAvatar } from "@/components/staff/primitives";
import { Spinner } from "@/components/ui/spinner";

/**
 * Account menu.
 *
 * Sign-out posts to the same /api/auth/logout route the staff panel uses, which
 * expires both the token and the role cookie, then hard-navigates so nothing of
 * the previous session survives in the React Query cache.
 */
export function PortalUserMenu({
  name,
  email,
  initials,
}: {
  name: string;
  email: string;
  initials: string;
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
          aria-label={`Account menu for ${name}`}
          className="tap-target flex shrink-0 items-center gap-1.5 rounded-btn p-0.5 transition-colors duration-150 hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
        >
          <InitialsAvatar name={name} initials={initials} size={34} />
          <ChevronDown className="size-4 text-tl-faint" aria-hidden />
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" sideOffset={8} className="w-56">
        <DropdownMenuLabel className="flex flex-col gap-0.5">
          <span className="text-ui-base font-semibold text-tl-ink">{name}</span>
          <span className="text-ui-xs font-normal text-tl-muted">{email}</span>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem asChild>
          <Link href="/portal/profile">
            <CircleUser className="size-4" aria-hidden />
            Profile
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <Link href="/portal/help">
            <LifeBuoy className="size-4" aria-hidden />
            Help Center
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
