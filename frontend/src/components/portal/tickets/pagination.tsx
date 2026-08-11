"use client";

import { useRouter } from "next/navigation";
import { ChevronLeft, ChevronRight } from "lucide-react";

import { ActionButton } from "@/components/portal/primitives";
import { buildSearch, PAGE_SIZE } from "@/lib/portal/filters";
import type { PortalTicketQuery } from "@/lib/portal/types";

/*
  Paging, not infinite scroll.

  A support history is something people come back to and search, so a page
  number in the URL is worth more than an endless list: it is shareable, the
  back button behaves, and the footer stays reachable.
*/

export function TicketPagination({
  query,
  total,
}: {
  query: PortalTicketQuery;
  total: number;
}) {
  const router = useRouter();
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));
  if (totalPages <= 1) return null;

  const page = Math.min(query.page, totalPages);
  const first = (page - 1) * PAGE_SIZE + 1;
  const last = Math.min(page * PAGE_SIZE, total);

  const go = (next: number) =>
    router.push(`/portal/tickets${buildSearch(query, { page: next })}`);

  return (
    <nav
      aria-label="Ticket list pages"
      className="flex flex-wrap items-center justify-between gap-3 pt-1"
    >
      <p className="text-ui-sm text-tl-muted" aria-live="polite">
        Showing <span className="font-medium text-tl-ink">{first}–{last}</span> of{" "}
        <span className="font-medium text-tl-ink">{total}</span>
      </p>

      <div className="flex items-center gap-2">
        <ActionButton
          variant="secondary"
          size="sm"
          onClick={() => go(page - 1)}
          disabled={page <= 1}
        >
          <ChevronLeft className="size-3.5" aria-hidden />
          Previous
        </ActionButton>
        <span className="px-1 text-ui-sm tabular-nums text-tl-muted">
          Page {page} of {totalPages}
        </span>
        <ActionButton
          variant="secondary"
          size="sm"
          onClick={() => go(page + 1)}
          disabled={page >= totalPages}
        >
          Next
          <ChevronRight className="size-3.5" aria-hidden />
        </ActionButton>
      </div>
    </nav>
  );
}
