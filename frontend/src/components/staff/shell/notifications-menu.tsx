"use client";

import { useState } from "react";
import Link from "next/link";
import { Bell, Check } from "lucide-react";

import { cn } from "@/lib/utils";
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/shadcn/popover";
import { relativeAge } from "@/lib/staff/format";
import type { Notification } from "@/lib/staff/types";

/**
 * Notifications popover.
 *
 * Read state is local: there is no mutation endpoint behind the fixtures, so
 * "mark all read" clears the badge for the session rather than pretending to
 * persist. When the API lands this becomes a mutation and the local state goes.
 */
export function NotificationsMenu({
  notifications,
}: {
  notifications: Notification[];
}) {
  const [readIds, setReadIds] = useState<string[]>([]);

  const isRead = (n: Notification) => n.read || readIds.includes(n.id);
  const unread = notifications.filter((n) => !isRead(n));

  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label={
            unread.length
              ? `Notifications, ${unread.length} unread`
              : "Notifications"
          }
          className="tap-target relative inline-flex size-10 items-center justify-center rounded-btn text-tl-ink-soft transition-colors duration-150 hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
        >
          <Bell className="size-5" strokeWidth={1.8} aria-hidden />
          {unread.length > 0 && (
            <span
              className="absolute right-1 top-1 inline-flex h-[17px] min-w-[17px] items-center justify-center rounded-full bg-tl-red px-1 text-[10px] font-bold text-white tabular-nums"
              aria-hidden
            >
              {unread.length}
            </span>
          )}
        </button>
      </PopoverTrigger>

      <PopoverContent align="end" sideOffset={8} className="w-[360px] p-0">
        <div className="flex items-center gap-2 border-b border-tl-line px-4 py-3">
          <h2 className="text-ui-md font-semibold text-tl-ink">Notifications</h2>
          {unread.length > 0 && (
            <button
              type="button"
              onClick={() => setReadIds(notifications.map((n) => n.id))}
              className="ml-auto inline-flex items-center gap-1 rounded-md px-2 py-1 text-ui-xs font-medium text-tl-blue transition-colors duration-150 hover:bg-tl-blue-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
            >
              <Check className="size-3" aria-hidden />
              Mark all read
            </button>
          )}
        </div>

        <ul className="max-h-[380px] overflow-y-auto py-1">
          {notifications.length === 0 && (
            <li className="px-4 py-8 text-center text-ui-sm text-tl-muted">
              You are all caught up.
            </li>
          )}

          {notifications.map((notification) => (
            <li key={notification.id}>
              <Link
                href={
                  notification.ticketId
                    ? `/staff/tickets/${notification.ticketId}`
                    : "/staff/notifications"
                }
                onClick={() =>
                  setReadIds((prev) =>
                    prev.includes(notification.id)
                      ? prev
                      : [...prev, notification.id],
                  )
                }
                className="flex gap-3 px-4 py-3 transition-colors duration-150 hover:bg-tl-line-soft/60 focus-visible:outline-none focus-visible:bg-tl-line-soft"
              >
                <span
                  className={cn(
                    "mt-1.5 size-2 shrink-0 rounded-full",
                    isRead(notification) ? "bg-transparent" : "bg-tl-blue",
                  )}
                  aria-hidden
                />
                <span className="min-w-0 flex-1">
                  <span className="flex items-baseline gap-2">
                    <span
                      className={cn(
                        "truncate text-ui-base",
                        isRead(notification)
                          ? "font-medium text-tl-ink-soft"
                          : "font-semibold text-tl-ink",
                      )}
                    >
                      {notification.title}
                    </span>
                    <span className="ml-auto shrink-0 text-ui-xs text-tl-faint">
                      {relativeAge(notification.at)}
                    </span>
                  </span>
                  <span className="mt-0.5 block truncate text-ui-sm text-tl-muted">
                    {notification.body}
                  </span>
                </span>
              </Link>
            </li>
          ))}
        </ul>

        <div className="border-t border-tl-line px-4 py-2.5">
          <Link
            href="/staff/notifications"
            className="text-ui-sm font-medium text-tl-blue hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
          >
            View all notifications
          </Link>
        </div>
      </PopoverContent>
    </Popover>
  );
}
