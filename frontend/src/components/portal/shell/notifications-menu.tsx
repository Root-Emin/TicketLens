"use client";

import { Bell } from "lucide-react";

import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/shadcn/popover";

/*
  Notifications, unwired.

  There is no notifications endpoint on the backend and no realtime channel the
  portal subscribes to, so this shows an honest empty state rather than a badge
  over invented rows. When `GET /notifications` lands, the list replaces the
  paragraph and the unread count lights the dot — nothing else here changes.
*/

export function PortalNotifications() {
  return (
    <Popover>
      <PopoverTrigger asChild>
        <button
          type="button"
          aria-label="Notifications"
          className="tap-target relative inline-flex size-10 items-center justify-center rounded-btn text-tl-ink-soft transition-colors duration-150 hover:bg-white focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
        >
          <Bell className="size-5" strokeWidth={1.8} aria-hidden />
        </button>
      </PopoverTrigger>

      <PopoverContent align="end" sideOffset={8} className="w-[320px] p-0">
        <div className="border-b border-tl-line px-4 py-3">
          <h2 className="text-ui-md font-semibold text-tl-ink">Notifications</h2>
        </div>
        <div className="px-4 py-8 text-center">
          <p className="text-ui-sm text-tl-muted">
            You&apos;re all caught up.
          </p>
          <p className="mt-1 text-ui-xs text-tl-faint">
            We&apos;ll let you know here when support replies.
          </p>
        </div>
      </PopoverContent>
    </Popover>
  );
}
