"use client";

import { ChevronDown, CircleUser, LogOut, Settings, Moon } from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/shadcn/dropdown-menu";
import { PresenceAvatar } from "../primitives";

/** Account menu. Sign-out posts to the existing logout route. */
export function ProfileMenu({
  name,
  initials,
  email,
}: {
  name: string;
  initials: string;
  email: string;
}) {
  async function signOut() {
    await fetch("/api/auth/logout", { method: "POST" });
    window.location.href = "/login";
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`Account menu for ${name}`}
          className="tap-target flex shrink-0 items-center gap-1.5 rounded-btn p-0.5 transition-colors duration-150 hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
        >
          <PresenceAvatar name={name} initials={initials} size={36} />
          <ChevronDown className="size-4 text-tl-faint" aria-hidden />
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" sideOffset={8} className="w-56">
        <DropdownMenuLabel className="flex flex-col gap-0.5">
          <span className="text-ui-base font-semibold text-tl-ink">{name}</span>
          <span className="text-ui-xs font-normal text-tl-muted">{email}</span>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        <DropdownMenuItem>
          <CircleUser className="size-4" aria-hidden />
          Profile
        </DropdownMenuItem>
        <DropdownMenuItem>
          <Settings className="size-4" aria-hidden />
          Settings
        </DropdownMenuItem>
        <DropdownMenuItem>
          <Moon className="size-4" aria-hidden />
          Set yourself away
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem variant="destructive" onSelect={signOut}>
          <LogOut className="size-4" aria-hidden />
          Sign out
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
