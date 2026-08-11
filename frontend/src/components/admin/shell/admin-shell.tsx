"use client";

import { Suspense, useState } from "react";
import { usePathname, useSearchParams } from "next/navigation";

import { Sheet, SheetContent, SheetTitle } from "@/components/shadcn/sheet";
import { TooltipProvider } from "@/components/shadcn/tooltip";
import { ToastProvider } from "@/components/portal/primitives";
import { useMe } from "@/lib/api/hooks";
import { useOrganization } from "@/lib/admin/hooks";
import { initialsOf } from "@/lib/portal/format";
import { roleLabel } from "@/lib/auth/permissions";
import { AdminSidebar } from "./sidebar";
import { AdminTopbar } from "./topbar";
import type { AdminAccount } from "./account-menu";

/*
  Chrome for every management route.

  Scroll model matches the staff and portal shells: this component owns the
  viewport (h-dvh) and never scrolls, `main` has no overflow of its own, and each
  route decides. h-dvh rather than h-screen so mobile browser chrome does not
  clip the footer.

  The signed-in account and the organization are fetched once here and read from
  the React Query cache by anything below, so the rail, the account menu and the
  settings screen never disagree about who is signed in or where.
*/

export function AdminShell({
  roles,
  children,
}: {
  /**
   * Role names decoded from the session token by the layout above, which is a
   * server component — the cookie is httpOnly and cannot be read here. Used only
   * to hide navigation this session cannot use; see lib/auth/permissions.ts.
   */
  roles: string[];
  children: React.ReactNode;
}) {
  const pathname = usePathname();
  const [menuOpen, setMenuOpen] = useState(false);

  const { data: me } = useMe();
  const { data: org } = useOrganization();

  // Close on navigation, wherever it came from — a KPI card, a row action, the
  // back button — none of which pass through onNavigate. Adjusting state during
  // render is React's documented answer for deriving from a changing value; an
  // effect would paint the open drawer over the new route first.
  const [lastPath, setLastPath] = useState(pathname);
  if (pathname !== lastPath) {
    setLastPath(pathname);
    setMenuOpen(false);
  }

  const account: AdminAccount | null = me
    ? {
        name: `${me.first_name} ${me.last_name}`.trim() || me.email,
        email: me.email,
        initials: initialsOf(
          `${me.first_name} ${me.last_name}`.trim() || me.email,
        ),
        panelLabel: roleLabel(roles),
      }
    : null;

  const organization = org ? { name: org.name, slug: org.slug } : null;

  const railProps = { organization, account, roles };

  return (
    <TooltipProvider delayDuration={200}>
      <ToastProvider>
        <div className="flex h-dvh overflow-hidden bg-tl-canvas font-ui text-tl-ink antialiased">
          <a
            href="#admin-main"
            className="sr-only focus:not-sr-only focus:absolute focus:left-4 focus:top-4 focus:z-50 focus:rounded-btn focus:bg-tl-blue focus:px-4 focus:py-2 focus:text-ui-base focus:font-semibold focus:text-white"
          >
            Skip to content
          </a>

          <aside className="hidden shrink-0 lg:block">
            <Suspense fallback={<StaticRail {...railProps} />}>
              <ActiveRail {...railProps} />
            </Suspense>
          </aside>

          <Sheet open={menuOpen} onOpenChange={setMenuOpen}>
            <SheetContent
              side="left"
              className="w-[254px] border-0 bg-tl-navy p-0 [&>button]:text-white"
            >
              <SheetTitle className="sr-only">Navigation</SheetTitle>
              <Suspense fallback={<StaticRail {...railProps} forceExpanded />}>
                <ActiveRail
                  {...railProps}
                  forceExpanded
                  onNavigate={() => setMenuOpen(false)}
                />
              </Suspense>
            </SheetContent>
          </Sheet>

          <div className="flex min-w-0 flex-1 flex-col">
            <AdminTopbar
              pathname={pathname}
              onOpenMenu={() => setMenuOpen(true)}
            />

            <main id="admin-main" className="min-h-0 flex-1 overflow-y-auto">
              {children}
            </main>
          </div>
        </div>
      </ToastProvider>
    </TooltipProvider>
  );
}

type RailProps = React.ComponentProps<typeof AdminSidebar>;

/**
 * Reads the location for the active row.
 *
 * Split out because useSearchParams forces client-side rendering up to the
 * nearest Suspense boundary, and one rail row ("Unassigned") is distinguished
 * from another by a query parameter alone. Isolating it here means only the
 * highlighting waits for hydration — the rail's markup still ships in the
 * initial HTML via the fallback below.
 */
function ActiveRail(props: Omit<RailProps, "pathname" | "search">) {
  const pathname = usePathname();
  const params = useSearchParams();

  return (
    <AdminSidebar
      {...props}
      pathname={pathname}
      search={new URLSearchParams(params.toString())}
    />
  );
}

/** The same rail, nothing highlighted — what the server renders. */
function StaticRail(props: Omit<RailProps, "pathname" | "search">) {
  return (
    <AdminSidebar {...props} pathname="" search={new URLSearchParams()} />
  );
}
