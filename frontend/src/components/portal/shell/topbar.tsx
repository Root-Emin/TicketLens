"use client";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { Menu, Search } from "lucide-react";

import { Skeleton } from "@/components/shadcn/skeleton";
import { PORTAL_TITLES } from "./nav-config";
import { PortalNotifications } from "./notifications-menu";
import { PortalUserMenu } from "./user-menu";

/*
  The portal header.

  Search is a real control, not a decoration: submitting navigates to the ticket
  list with `?q=`, which is the same URL state the list's own search box writes.
  One place owns the query, so the two can never disagree.

  Below md the field collapses to an icon that focuses the list's search instead
  of trying to fit an input next to the avatar.
*/

export function PortalTopbar({
  pathname,
  customer,
  loading,
  onOpenMenu,
}: {
  pathname: string;
  customer: { name: string; email: string; initials: string } | null;
  loading: boolean;
  onOpenMenu: () => void;
}) {
  const router = useRouter();
  const [term, setTerm] = useState("");
  const title = PORTAL_TITLES[pathname] ?? "Ticket";

  function onSubmit(event: React.FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const q = term.trim();
    router.push(q ? `/portal/tickets?q=${encodeURIComponent(q)}` : "/portal/tickets");
  }

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

      {/* The page renders the real h1; this is a small-screen orientation label
          only, which is why it is a span and hidden on desktop. */}
      <span className="truncate text-ui-lg font-semibold tracking-[-0.01em] text-tl-ink lg:hidden">
        {title}
      </span>

      <form
        onSubmit={onSubmit}
        role="search"
        className="ml-auto hidden md:block"
      >
        <div className="relative">
          <Search
            className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-tl-faint"
            aria-hidden
          />
          <input
            type="search"
            value={term}
            onChange={(event) => setTerm(event.target.value)}
            placeholder="Search your tickets…"
            aria-label="Search your tickets"
            className="h-9 w-[220px] rounded-btn border border-tl-line bg-white pl-9 pr-3 text-ui-sm text-tl-ink outline-none transition-[width,border-color] duration-150 placeholder:text-tl-faint focus:w-[280px] focus:border-tl-blue focus:ring-2 focus:ring-tl-blue/15"
          />
        </div>
      </form>

      <div className="ml-auto flex items-center gap-2 md:ml-3 md:gap-3">
        <button
          type="button"
          onClick={() => router.push("/portal/tickets")}
          aria-label="Search your tickets"
          className="tap-target inline-flex size-10 items-center justify-center rounded-btn text-tl-ink-soft transition-colors duration-150 hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30 md:hidden"
        >
          <Search className="size-5" strokeWidth={1.8} aria-hidden />
        </button>

        <PortalNotifications />

        {loading || !customer ? (
          <Skeleton className="size-[34px] rounded-full" />
        ) : (
          <PortalUserMenu
            name={customer.name}
            email={customer.email}
            initials={customer.initials}
          />
        )}
      </div>
    </header>
  );
}
