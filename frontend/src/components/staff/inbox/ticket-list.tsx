"use client";

import Link from "next/link";
import { ChevronLeft, ChevronRight, Inbox, SearchX } from "lucide-react";

import { cn } from "@/lib/utils";
import { useListKeyboardNav } from "@/hooks/use-list-keyboard-nav";
import { VIEW_LABEL } from "@/lib/staff/filters";
import type { Paged, Ticket, TicketQuery } from "@/lib/staff/types";
import { withQuery } from "@/lib/staff/url";
import { CountPill } from "../primitives";
import { TicketFilters } from "./ticket-filters";
import { TicketRow } from "./ticket-row";

/*
  The queue pane.

  A client component only so the list can own arrow-key navigation; the rows
  themselves are memoised and the data arrives already filtered from the server.
*/

export function TicketList({
  page,
  query,
  selectedId,
}: {
  page: Paged<Ticket>;
  query: TicketQuery;
  selectedId?: string;
}) {
  const { ref, onKeyDown } = useListKeyboardNav<HTMLDivElement>();

  const from = (page.page - 1) * page.pageSize + 1;
  const to = Math.min(page.page * page.pageSize, page.total);

  return (
    <div className="flex h-full min-h-0 flex-col bg-tl-card">
      <div className="flex shrink-0 items-center gap-2 px-4 pb-3 pt-4">
        <h2 className="text-ui-lg font-semibold tracking-[-0.01em] text-tl-ink">
          {VIEW_LABEL[query.view]}
        </h2>
        <CountPill>{page.total}</CountPill>
      </div>

      <TicketFilters query={query} />

      <div
        ref={ref}
        onKeyDown={onKeyDown}
        className="min-h-0 flex-1 overflow-y-auto overscroll-contain border-t border-tl-line-soft"
      >
        {page.items.length === 0 ? (
          <EmptyState query={query} />
        ) : (
          <ul aria-label={`${VIEW_LABEL[query.view]} tickets`}>
            {page.items.map((ticket) => (
              <TicketRow
                key={ticket.id}
                ticket={ticket}
                query={query}
                selected={ticket.id === selectedId}
              />
            ))}
          </ul>
        )}
      </div>

      {page.pageCount > 1 && (
        <nav
          aria-label="Ticket list pages"
          className="flex shrink-0 items-center justify-between gap-2 border-t border-tl-line px-4 py-3"
        >
          <span className="text-ui-xs text-tl-muted">
            {from}–{to} of {page.total}
          </span>

          <div className="flex items-center gap-1">
            <PageLink
              query={query}
              page={page.page - 1}
              disabled={page.page <= 1}
              label="Previous page"
            >
              <ChevronLeft className="size-3.5" strokeWidth={2.4} aria-hidden />
            </PageLink>

            {Array.from({ length: page.pageCount }).map((_, i) => (
              <PageLink
                key={i}
                query={query}
                page={i + 1}
                current={page.page === i + 1}
              >
                {i + 1}
              </PageLink>
            ))}

            <PageLink
              query={query}
              page={page.page + 1}
              disabled={page.page >= page.pageCount}
              label="Next page"
            >
              <ChevronRight className="size-3.5" strokeWidth={2.4} aria-hidden />
            </PageLink>
          </div>
        </nav>
      )}
    </div>
  );
}

function PageLink({
  query,
  page,
  current,
  disabled,
  label,
  children,
}: {
  query: TicketQuery;
  page: number;
  current?: boolean;
  disabled?: boolean;
  label?: string;
  children: React.ReactNode;
}) {
  const className = cn(
    "tap-target inline-flex size-7 items-center justify-center rounded-lg border text-ui-xs font-semibold transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30",
    current
      ? "border-tl-blue bg-tl-blue text-white"
      : "border-tl-line bg-white text-tl-muted hover:bg-tl-line-soft hover:text-tl-ink",
  );

  // A disabled page is not a destination, so it is a button rather than a link
  // that goes nowhere.
  if (disabled) {
    return (
      <button type="button" disabled aria-label={label} className={cn(className, "opacity-40")}>
        {children}
      </button>
    );
  }

  return (
    <Link
      href={withQuery("/staff/tickets", query, { page })}
      aria-label={label}
      aria-current={current ? "page" : undefined}
      className={className}
    >
      {children}
    </Link>
  );
}

function EmptyState({ query }: { query: TicketQuery }) {
  const filtered = Boolean(query.q || query.priority || query.status);

  return (
    <div className="flex flex-col items-center gap-3 px-6 py-16 text-center">
      {filtered ? (
        <SearchX className="size-7 text-tl-faint-soft" aria-hidden />
      ) : (
        <Inbox className="size-7 text-tl-faint-soft" aria-hidden />
      )}

      <div>
        <p className="text-ui-md font-semibold text-tl-ink">
          {filtered ? "No matching tickets" : "Nothing here"}
        </p>
        <p className="mt-1 text-ui-sm text-tl-muted">
          {filtered
            ? "Try removing a filter or searching for something broader."
            : `${VIEW_LABEL[query.view]} is empty. Good place to be.`}
        </p>
      </div>

      {filtered && (
        <Link
          href={withQuery("/staff/tickets", query, {
            q: undefined,
            priority: undefined,
            status: undefined,
          })}
          className="rounded-btn border border-tl-line bg-white px-3 py-1.5 text-ui-sm font-medium text-tl-ink-soft transition-colors duration-150 hover:bg-tl-line-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
        >
          Clear filters
        </Link>
      )}
    </div>
  );
}
