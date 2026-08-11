import type { Metadata } from "next";
import { cookies } from "next/headers";

import { SettingsWorkspace } from "@/components/admin/settings/settings-workspace";
import { TOKEN_COOKIE } from "@/lib/server/backend";
import { decodeClaims } from "@/lib/server/claims";

export const metadata: Metadata = {
  title: "Organization — TicketLens",
  description: "Workspace details and your account.",
};

/** Organization settings. Roles come from the httpOnly token, read server-side. */
export default async function SettingsPage() {
  const token = (await cookies()).get(TOKEN_COOKIE)?.value;
  const roles = token ? (decodeClaims(token)?.roles ?? []) : [];

  return <SettingsWorkspace roles={roles} />;
}
