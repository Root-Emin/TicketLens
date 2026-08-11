import type { Metadata } from "next";
import { cookies } from "next/headers";

import { DepartmentsWorkspace } from "@/components/admin/departments/departments-workspace";
import { TOKEN_COOKIE } from "@/lib/server/backend";
import { decodeClaims } from "@/lib/server/claims";

export const metadata: Metadata = {
  title: "Departments — TicketLens",
  description: "The teams tickets are routed to, and what each one is carrying.",
};

/**
 * Department management.
 *
 * The roles are read here rather than in the client component for the same
 * reason as the layout: the token is httpOnly. They decide whether the write
 * controls are drawn — the backend decides whether the writes succeed, and the
 * dialogs render its 403 message when they do not.
 */
export default async function DepartmentsPage() {
  const token = (await cookies()).get(TOKEN_COOKIE)?.value;
  const roles = token ? (decodeClaims(token)?.roles ?? []) : [];

  return <DepartmentsWorkspace roles={roles} />;
}
