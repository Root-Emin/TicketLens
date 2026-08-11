import type { Metadata } from "next";
import { cookies } from "next/headers";

import { AdminShell } from "@/components/admin/shell/admin-shell";
import { TOKEN_COOKIE } from "@/lib/server/backend";
import { decodeClaims } from "@/lib/server/claims";

export const metadata: Metadata = {
  title: "TicketLens — Administration",
  description: "Manage your support team, departments and ticket operations.",
};

/**
 * The management workspace's frame.
 *
 * A server component over a client shell, for one reason: the session token is
 * an httpOnly cookie, so the roles it carries can only be read here. They travel
 * down as a prop and are used for one thing — hiding navigation this session
 * cannot use. Nothing is authorized by them; see lib/auth/permissions.ts.
 *
 * Everything else the shell needs (the account, the organization) is live data
 * behind React Query, which is why it is fetched in the client shell rather than
 * awaited here and passed as props that would go stale.
 */
export default async function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const token = (await cookies()).get(TOKEN_COOKIE)?.value;
  const roles = token ? (decodeClaims(token)?.roles ?? []) : [];

  return <AdminShell roles={roles}>{children}</AdminShell>;
}
