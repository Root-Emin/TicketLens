import type { Metadata } from "next";

import { PortalShell } from "@/components/portal/shell/portal-shell";

export const metadata: Metadata = {
  title: "TicketLens — Support Portal",
  description: "Open a request and follow it from one place.",
};

/**
 * The customer portal's frame.
 *
 * A thin server component over a client shell: the portal reads live,
 * per-account data through React Query, so there is nothing useful to fetch on
 * the server here — the session cookie is httpOnly and the request would only
 * be made twice.
 */
export default function PortalLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return <PortalShell>{children}</PortalShell>;
}
