"use client";

import { useState } from "react";
import { usePathname } from "next/navigation";

import { Sheet, SheetContent, SheetTitle } from "@/components/shadcn/sheet";
import { TooltipProvider } from "@/components/shadcn/tooltip";
import { ToastProvider } from "@/components/portal/primitives";
import { initialsOf } from "@/lib/portal/format";
import { usePortalMe } from "@/lib/portal/hooks";
import { PortalSidebar } from "./sidebar";
import { PortalTopbar } from "./topbar";

/*
  Chrome for every /portal route.

  Scroll model matches the staff shell: this component owns the viewport (h-dvh)
  and never scrolls, `main` has no overflow of its own, and each route decides.
  Every portal page is one scrolling column.

  The signed-in account is fetched once here and read from the React Query cache
  by anything below, so the rail, the topbar and the profile screen never
  disagree about who is signed in.
*/

export function PortalShell({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [menuOpen, setMenuOpen] = useState(false);
  const { data: me, isLoading } = usePortalMe();

  // Close on navigation, wherever it came from: a card, a redirect after
  // creating a ticket, or the back button — none of which pass through
  // onNavigate. Adjusting state during render is React's documented answer for
  // deriving from a changing value; an effect would paint the open drawer over
  // the new route first.
  const [lastPath, setLastPath] = useState(pathname);
  if (pathname !== lastPath) {
    setLastPath(pathname);
    setMenuOpen(false);
  }

  const customer = me
    ? {
        name: `${me.first_name} ${me.last_name}`.trim() || me.email,
        email: me.email,
        initials: initialsOf(`${me.first_name} ${me.last_name}`.trim() || me.email),
      }
    : null;

  return (
    <TooltipProvider delayDuration={200}>
      <ToastProvider>
        <div className="flex h-dvh overflow-hidden bg-tl-canvas font-ui text-tl-ink antialiased">
          <a
            href="#portal-main"
            className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-btn focus:bg-tl-blue focus:px-4 focus:py-2 focus:text-ui-base focus:font-semibold focus:text-white"
          >
            Skip to content
          </a>

          <aside className="hidden shrink-0 lg:block">
            <PortalSidebar pathname={pathname} customer={customer} />
          </aside>

          <Sheet open={menuOpen} onOpenChange={setMenuOpen}>
            <SheetContent
              side="left"
              className="w-[250px] border-0 bg-tl-navy p-0 [&>button]:text-white"
            >
              <SheetTitle className="sr-only">Navigation</SheetTitle>
              <PortalSidebar
                pathname={pathname}
                customer={customer}
                onNavigate={() => setMenuOpen(false)}
              />
            </SheetContent>
          </Sheet>

          <div className="flex min-w-0 flex-1 flex-col">
            <PortalTopbar
              pathname={pathname}
              customer={customer}
              loading={isLoading}
              onOpenMenu={() => setMenuOpen(true)}
            />

            <main id="portal-main" className="min-h-0 flex-1 overflow-y-auto">
              <div className="mx-auto flex max-w-[1100px] flex-col gap-6 px-4 py-6 lg:px-8">
                {children}
              </div>
            </main>
          </div>
        </div>
      </ToastProvider>
    </TooltipProvider>
  );
}
