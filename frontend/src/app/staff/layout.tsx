import type { Metadata } from "next";

import { AppShell } from "@/components/staff/shell/app-shell";
import { teamLabel } from "@/lib/staff/fixtures";
import { paletteTicketsFrom } from "@/lib/staff/palette";
import {
  getCurrentAgent,
  getSidebarCounts,
  listAllTickets,
  listNotifications,
} from "@/lib/staff/queries";

export const metadata: Metadata = {
  title: "TicketLens — Support Workspace",
};

/**
 * Loads the chrome's data on the server once per navigation and hands it to
 * the client shell. The palette gets the whole queue so search is instant and
 * offline — it is a few dozen rows, not something worth a round trip per
 * keystroke.
 */
export default async function StaffLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const [agent, counts, notifications, all] = await Promise.all([
    getCurrentAgent(),
    getSidebarCounts(),
    listNotifications(),
    listAllTickets(),
  ]);

  return (
    <AppShell
      data={{
        agent: {
          name: agent.name,
          initials: agent.initials,
          email: agent.email,
          // The rail no longer lists teams, so the agent's own department is
          // surfaced on their profile card instead.
          role: teamLabel[agent.team],
        },
        counts,
        notifications,
        paletteTickets: paletteTicketsFrom(all),
      }}
    >
      {children}
    </AppShell>
  );
}
