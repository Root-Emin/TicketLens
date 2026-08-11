import type { Metadata } from "next";
import { cookies } from "next/headers";

import { DepartmentDetail } from "@/components/admin/departments/department-detail";
import { TOKEN_COOKIE } from "@/lib/server/backend";
import { decodeClaims } from "@/lib/server/claims";

export const metadata: Metadata = {
  title: "Department — TicketLens",
  description: "A support team, its staff and what it is carrying.",
};

/**
 * One department.
 *
 * The title stays generic because the department's name is not known until the
 * roster query resolves on the client — and fetching it again here, server-side,
 * to fill in a <title> would double every request on the page for a browser tab.
 */
export default async function DepartmentPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const [{ id }, cookieStore] = await Promise.all([params, cookies()]);
  const token = cookieStore.get(TOKEN_COOKIE)?.value;
  const roles = token ? (decodeClaims(token)?.roles ?? []) : [];

  return <DepartmentDetail departmentId={id} roles={roles} />;
}
