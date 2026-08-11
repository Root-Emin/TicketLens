"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Check, ChevronDown, Search, X } from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/shadcn/dropdown-menu";
import {
  buildSearch,
  SORTS,
  sortLabel,
  STATUS_FILTERS,
} from "@/lib/portal/filters";
import type { PortalTicketQuery } from "@/lib/portal/types";
import { cn } from "@/lib/utils";

/*
  Search, status facets and sort for the ticket list.

  Every control navigates rather than setting state, matching the staff queue:
  the query already lives in the URL, so a filtered list is bookmarkable, the
  back button steps through filter changes, and a reload keeps the view.

  Search debounces so typing does not push one history entry per keystroke.
*/

const DEBOUNCE_MS = 300;

export function TicketListControls({ query }: { query: PortalTicketQuery }) {
  const router = useRouter();
  const [term, setTerm] = useState(query.q ?? "");
  const debounce = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Keep the field in step when the query changes from somewhere else — the
  // topbar search, a nav click, the back button — rather than from typing here.
  // Adjusting state during render is React's documented answer for deriving
  // from props; an effect would render once with the stale value first.
  const [lastQ, setLastQ] = useState(query.q);
  if (query.q !== lastQ) {
    setLastQ(query.q);
    setTerm(query.q ?? "");
  }

  const navigate = (patch: Partial<PortalTicketQuery>) =>
    router.push(`/portal/tickets${buildSearch(query, patch)}`);

  const onSearchChange = (value: string) => {
    setTerm(value);
    if (debounce.current) clearTimeout(debounce.current);
    debounce.current = setTimeout(() => {
      router.push(
        `/portal/tickets${buildSearch(query, { q: value.trim() || undefined })}`,
      );
    }, DEBOUNCE_MS);
  };

  useEffect(
    () => () => {
      if (debounce.current) clearTimeout(debounce.current);
    },
    [],
  );

  return (
    <div className="space-y-3">
      <div className="relative">
        <Search
          className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-tl-faint"
          aria-hidden
        />
        <input
          type="search"
          value={term}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder="Search your tickets…"
          aria-label="Search your tickets"
          className="h-10 w-full rounded-btn border border-tl-line bg-tl-card pl-9 pr-3 text-ui-md text-tl-ink outline-none transition-colors duration-150 placeholder:text-tl-faint focus:border-tl-blue focus:ring-2 focus:ring-tl-blue/15"
        />
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        {/* A tab list would promise panel switching; these are filters over one
            list, so they are buttons that carry their own pressed state. */}
        <div
          role="group"
          aria-label="Filter by status"
          className="flex flex-wrap items-center gap-1.5"
        >
          {STATUS_FILTERS.map((filter) => {
            const active = query.status === filter.id;
            return (
              <button
                key={filter.id}
                type="button"
                aria-pressed={active}
                onClick={() => navigate({ status: filter.id })}
                className={chipClass(active)}
              >
                {filter.label}
              </button>
            );
          })}
        </div>

        <div className="ml-auto flex items-center gap-1.5">
          <DropdownMenu>
            <DropdownMenuTrigger asChild>
              <button type="button" className={chipClass(false)}>
                Sort: {sortLabel(query.sort)}
                <ChevronDown className="size-3 opacity-60" aria-hidden />
              </button>
            </DropdownMenuTrigger>
            <DropdownMenuContent align="end" className="w-48">
              <DropdownMenuLabel>Sort by</DropdownMenuLabel>
              <DropdownMenuSeparator />
              {SORTS.map((option) => (
                <DropdownMenuItem
                  key={option.id}
                  onSelect={() => navigate({ sort: option.id })}
                  className="justify-between"
                >
                  {option.label}
                  {query.sort === option.id && (
                    <Check className="size-4" aria-hidden />
                  )}
                </DropdownMenuItem>
              ))}
            </DropdownMenuContent>
          </DropdownMenu>

          {(query.q || query.status !== "all" || query.sort !== "newest") && (
            <button
              type="button"
              onClick={() =>
                navigate({ q: undefined, status: "all", sort: "newest" })
              }
              className={chipClass(false)}
            >
              <X className="size-3" aria-hidden />
              Clear
            </button>
          )}
        </div>
      </div>
    </div>
  );
}

function chipClass(active: boolean) {
  return cn(
    "inline-flex h-8 shrink-0 items-center gap-1.5 rounded-lg border px-3 text-ui-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30",
    active
      ? "border-tl-blue bg-tl-blue-soft text-tl-blue"
      : "border-tl-line bg-tl-card text-tl-ink-soft hover:bg-tl-line-soft",
  );
}
