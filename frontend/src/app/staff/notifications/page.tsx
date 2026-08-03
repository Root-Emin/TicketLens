import Link from "next/link";
import { BellOff } from "lucide-react";

import { relativeAge } from "@/lib/staff/format";
import { listNotifications } from "@/lib/staff/queries";
import { Panel, PanelHeader, CountPill } from "@/components/staff/primitives";

export default async function NotificationsPage() {
  const notifications = await listNotifications();
  const unread = notifications.filter((n) => !n.read).length;

  return (
    <div className="h-full overflow-y-auto">
      <div className="mx-auto max-w-[820px] px-4 py-6 lg:px-6">
        <Panel>
          <PanelHeader
            title="Notifications"
            count={unread > 0 ? <CountPill>{unread} new</CountPill> : undefined}
          />

          {notifications.length === 0 ? (
            <div className="flex flex-col items-center gap-2 border-t border-tl-line-soft px-5 py-14 text-center">
              <BellOff className="size-6 text-tl-faint-soft" aria-hidden />
              <p className="text-ui-base text-tl-muted">Nothing to catch up on.</p>
            </div>
          ) : (
            <ul className="border-t border-tl-line-soft">
              {notifications.map((notification) => (
                <li
                  key={notification.id}
                  className="border-b border-tl-line-soft last:border-b-0"
                >
                  <Link
                    href={
                      notification.ticketId
                        ? `/staff/tickets/${notification.ticketId}`
                        : "/staff/notifications"
                    }
                    className="flex gap-3 px-5 py-4 transition-colors duration-150 hover:bg-tl-line-soft/50 focus-visible:bg-tl-line-soft focus-visible:outline-none"
                  >
                    <span
                      className={`mt-1.5 size-2 shrink-0 rounded-full ${
                        notification.read ? "bg-transparent" : "bg-tl-blue"
                      }`}
                      aria-hidden
                    />
                    <span className="min-w-0 flex-1">
                      <span className="flex items-baseline gap-3">
                        <span
                          className={`truncate text-ui-md ${
                            notification.read
                              ? "font-medium text-tl-ink-soft"
                              : "font-semibold text-tl-ink"
                          }`}
                        >
                          {notification.title}
                        </span>
                        <span className="ml-auto shrink-0 text-ui-xs text-tl-faint">
                          {relativeAge(notification.at)}
                        </span>
                      </span>
                      <span className="mt-0.5 block text-ui-sm text-tl-muted">
                        {notification.body}
                      </span>
                    </span>
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </Panel>
      </div>
    </div>
  );
}
