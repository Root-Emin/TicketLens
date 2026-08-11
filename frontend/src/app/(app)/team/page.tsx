import type { Metadata } from "next";
import { Suspense } from "react";
import { cookies } from "next/headers";

import { StaffWorkspace } from "@/components/admin/staff/staff-workspace";
import { AdminPage, KpiRowSkeleton, TableFrame } from "@/components/admin/primitives";
import { StaffTableSkeleton } from "@/components/admin/staff/staff-table";
import { TOKEN_COOKIE } from "@/lib/server/backend";
import { decodeClaims } from "@/lib/server/claims";

export const metadata: Metadata = {
  title: "Staff — TicketLens",
  description: "Everyone on the support team, across every department.",
};

/**
 * The flat roster: everybody, regardless of team.
 *
 * The department-first view lives at /departments — that is where an
 * administrator goes to work with one team. This screen exists for the
 * questions that cut across teams: who has nobody managing them, who is
 * carrying the most, and where is this person I only know the name of. Both are
 * the same table component reading the same roster.
 *
 * /team rather than /staff: /staff/* is the agent's own panel and one path
 * cannot mean both "the queue I work" and "the people I manage".
 *
 * The Suspense boundary is required, not decorative: the filter state lives in
 * the URL, and useSearchParams opts everything under it into client rendering.
 */
export default async function TeamPage() {
  const token = (await cookies()).get(TOKEN_COOKIE)?.value;
  const roles = token ? (decodeClaims(token)?.roles ?? []) : [];

  return (
    <Suspense fallback={<Loading />}>
      <StaffWorkspace roles={roles} />
    </Suspense>
  );
}

function Loading() {
  return (
    <AdminPage>
      <div className="h-[52px]" />
      <KpiRowSkeleton />
      <TableFrame>
        <StaffTableSkeleton />
      </TableFrame>
    </AdminPage>
  );
}
