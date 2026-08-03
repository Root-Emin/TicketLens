"use client";

import { Suspense, useState } from "react";
import { usePathname, useSearchParams } from "next/navigation";

import { Sheet, SheetContent, SheetTitle } from "@/components/shadcn/sheet";
import { TooltipProvider } from "@/components/shadcn/tooltip";
import type { SidebarCounts } from "@/lib/staff/queries";
import { parseView } from "@/lib/staff/filters";
import type { Notification } from "@/lib/staff/types";
import type { PaletteTicket } from "@/lib/staff/palette";
import { CommandPalette, useCommandPalette } from "./command-palette";
import { Header } from "./header";
import { Sidebar, type SidebarState } from "./sidebar";

/*
  Chrome for every staff route.

  Scroll model: this component owns the viewport (h-dvh) and never scrolls
  itself. `main` has no overflow of its own — each route decides. The dashboard
  renders one tall scrolling column; the inbox renders a fixed multi-pane shell.
  That is what keeps "one page scroll" true where it matters without forcing the
  inbox into a layout no comparable product uses.

  h-dvh rather than h-screen so mobile browser chrome does not clip the footer.
*/

export interface ShellData {
  agent: { name: string; initials: string; email: string; role: string };
  counts: SidebarCounts;
  notifications: Notification[];
  paletteTickets: PaletteTicket[];
}

/**
 * Reads the active view from the URL.
 *
 * Split out because useSearchParams forces client-side rendering up to the
 * nearest Suspense boundary. Isolating it here means only the nav's active
 * highlighting waits for hydration — the rail's markup still ships in the
 * initial HTML via the fallback below.
 */
function ActiveSidebar(props: Omit<SidebarProps, keyof SidebarState>) {
  const pathname = usePathname();
  const params = useSearchParams();
  const onTickets = pathname.startsWith("/staff/tickets");

  return (
    <Sidebar
      {...props}
      pathname={pathname}
      onTickets={onTickets}
      activeView={onTickets ? parseView(params.get("view") ?? undefined) : null}
    />
  );
}

type SidebarProps = React.ComponentProps<typeof Sidebar>;

/** Same rail, no active highlighting — what the server renders. */
function StaticSidebar(props: Omit<SidebarProps, keyof SidebarState>) {
  return (
    <Sidebar
      {...props}
      pathname=""
      onTickets={false}
      activeView={null}
    />
  );
}

export function AppShell({
  data,
  children,
}: {
  data: ShellData;
  children: React.ReactNode;
}) {
  const [menuOpen, setMenuOpen] = useState(false);
  const { open: searchOpen, setOpen: setSearchOpen } = useCommandPalette();

  const railProps = {
    counts: data.counts,
    agent: {
      name: data.agent.name,
      initials: data.agent.initials,
      role: data.agent.role,
    },
  };

  return (
    <TooltipProvider delayDuration={200}>
      <div className="flex h-dvh overflow-hidden bg-tl-canvas font-ui text-tl-ink antialiased">
        <a
          href="#main"
          className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-btn focus:bg-tl-blue focus:px-4 focus:py-2 focus:text-ui-base focus:font-semibold focus:text-white"
        >
          Skip to content
        </a>

        <aside className="hidden shrink-0 lg:block">
          <Suspense fallback={<StaticSidebar {...railProps} />}>
            <ActiveSidebar {...railProps} />
          </Suspense>
        </aside>

        {/* Mobile navigation drawer */}
        <Sheet open={menuOpen} onOpenChange={setMenuOpen}>
          <SheetContent
            side="left"
            className="w-[250px] border-0 bg-tl-navy p-0 [&>button]:text-white"
          >
            <SheetTitle className="sr-only">Navigation</SheetTitle>
            <Suspense fallback={<StaticSidebar {...railProps} forceExpanded />}>
              <ActiveSidebar
                {...railProps}
                forceExpanded
                onNavigate={() => setMenuOpen(false)}
              />
            </Suspense>
          </SheetContent>
        </Sheet>

        <div className="flex min-w-0 flex-1 flex-col">
          <Header
            agent={data.agent}
            notifications={data.notifications}
            onOpenMenu={() => setMenuOpen(true)}
            onOpenSearch={() => setSearchOpen(true)}
          />

          <main id="main" className="min-h-0 flex-1">
            {children}
          </main>
        </div>

        <CommandPalette
          open={searchOpen}
          onOpenChange={setSearchOpen}
          tickets={data.paletteTickets}
        />
      </div>
    </TooltipProvider>
  );
}
