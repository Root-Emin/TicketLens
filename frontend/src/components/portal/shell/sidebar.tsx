"use client";

import Link from "next/link";

import { cn } from "@/lib/utils";
import {
  activeClass,
  idleClass,
  rowClass,
} from "@/components/staff/shell/sidebar-item";
import { isActive, PORTAL_NAV } from "./nav-config";

/*
  The portal rail.

  Same navy surface, same row metrics and the same active marker as the staff
  rail — the row classes are imported from it rather than re-typed, so the two
  panels cannot drift apart on hover colour or focus ring. What is deliberately
  absent is the collapse toggle and the counts: four links do not need to fold,
  and a number beside "My Tickets" would mean something different to a customer
  than a queue depth means to an agent.
*/

export function PortalSidebar({
  pathname,
  customer,
  onNavigate,
}: {
  pathname: string;
  customer: { name: string; email: string } | null;
  /** The mobile drawer passes this so following a link closes the drawer. */
  onNavigate?: () => void;
}) {
  return (
    <div className="flex h-full w-[250px] flex-col bg-tl-navy">
      <div className="flex shrink-0 items-center gap-2.5 px-5 py-[18px]">
        <Link
          href="/portal"
          onClick={onNavigate}
          aria-label="TicketLens portal home"
          className="flex items-center gap-2.5 rounded-btn focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue focus-visible:ring-offset-2 focus-visible:ring-offset-tl-navy"
        >
          <img
            src="/assets/TicketLens_Logo/favicon.svg"
            alt="TicketLens Logo"
            className="size-8 shrink-0 rounded-[9px] object-contain shadow-sm"
          />
          <span className="text-[17px] font-bold tracking-[-0.015em] text-white">
            TicketLens
          </span>
        </Link>
      </div>

      <nav
        aria-label="Portal"
        className="min-h-0 flex-1 space-y-0.5 overflow-y-auto overscroll-contain px-3 pb-2"
      >
        {PORTAL_NAV.map((link) => {
          const Icon = link.icon;
          const active = isActive(link, pathname);

          return (
            <Link
              key={link.href}
              href={link.href}
              onClick={onNavigate}
              aria-current={active ? "page" : undefined}
              className={cn(rowClass, active ? activeClass : idleClass)}
            >
              {/* A rail-edge bar rather than a border, so selection moving does
                  not shift the label by a pixel. */}
              {active && (
                <span
                  className="absolute inset-y-1 left-0 w-[3px] rounded-r bg-tl-blue"
                  aria-hidden
                />
              )}
              <Icon className="size-[18px] shrink-0" strokeWidth={1.8} aria-hidden />
              <span className="truncate">{link.label}</span>
            </Link>
          );
        })}
      </nav>

      {customer && (
        <div className="shrink-0 border-t border-white/[0.07] p-3">
          <div className="rounded-card bg-white/[0.06] px-3 py-2.5">
            <div className="truncate text-ui-base font-semibold text-white">
              {customer.name}
            </div>
            <div className="truncate text-ui-xs text-tl-rail-text">
              {customer.email}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}
