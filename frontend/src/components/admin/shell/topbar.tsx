"use client";

import { Menu } from "lucide-react";

import { titleFor } from "./nav-config";

/*
  The compact bar that replaces the rail below lg.

  It is `lg:hidden` on purpose, which is where this panel departs from the other
  two. The staff and portal headers stay on screen at every width because they
  carry global search, notifications and the account menu; this panel has none of
  those — search belongs to the list it filters, there is no notification
  endpoint for an administrator, and the account sits at the foot of the rail
  where a workspace tool puts it. A persistent desktop header here would be
  56px of empty chrome above every table, on the one screen that is short of
  vertical room.

  So on desktop the rail is the chrome and each page owns its own header. Below
  lg the rail becomes a drawer, and this bar is what opens it.
*/

export function AdminTopbar({
  pathname,
  onOpenMenu,
}: {
  pathname: string;
  onOpenMenu: () => void;
}) {
  return (
    <header className="flex shrink-0 items-center gap-3 border-b border-tl-line bg-tl-canvas/80 px-4 py-3 backdrop-blur-sm lg:hidden">
      <button
        type="button"
        onClick={onOpenMenu}
        aria-label="Open navigation"
        className="tap-target -ml-1 inline-flex size-10 items-center justify-center rounded-btn text-tl-ink-soft transition-colors duration-150 hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
      >
        <Menu className="size-5" aria-hidden />
      </button>

      {/* The page renders the real h1; this is an orientation label only, which
          is why it is a span. */}
      <span className="truncate text-ui-lg font-semibold tracking-[-0.01em] text-tl-ink">
        {titleFor(pathname)}
      </span>
    </header>
  );
}
