"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { Check, ChevronDown, Search, SlidersHorizontal, X } from "lucide-react";

import { cn } from "@/lib/utils";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/shadcn/dropdown-menu";
import { SORT_LABEL } from "@/lib/staff/filters";
import type { Priority, SortId, TicketQuery, TicketStatus } from "@/lib/staff/types";
import { withQuery } from "@/lib/staff/url";
import { PRIORITY_LABEL, STATUS_LABEL } from "../primitives";

/*
  Filter, sort and search controls.

  Every one of them navigates rather than setting state: the query already lives
  in the URL, so the server re-renders the list and the result is bookmarkable.
  The search box debounces so typing does not fire a navigation per keystroke.
*/

const PRIORITIES: Priority[] = ["high", "medium", "low"];
const STATUSES: TicketStatus[] = [
  "open",
  "pending",
  "on_hold",
  "resolved",
  "closed",
];
const SORTS: SortId[] = ["newest", "oldest", "priority", "sla"];

function Chip({
  active,
  children,
  ...props
}: React.ComponentProps<"button"> & { active?: boolean }) {
  return (
    <button
      type="button"
      className={cn(
        "inline-flex h-7 shrink-0 items-center gap-1 rounded-lg border px-2.5 text-ui-xs font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30",
        active
          ? "border-tl-blue bg-tl-blue-soft text-tl-blue"
          : "border-tl-line bg-white text-tl-ink-soft hover:bg-tl-line-soft",
      )}
      {...props}
    >
      {children}
    </button>
  );
}

export function TicketFilters({ query }: { query: TicketQuery }) {
  const router = useRouter();
  const [term, setTerm] = useState(query.q ?? "");
  const debounce = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Keep the field in step when the query changes from elsewhere — a nav click,
  // the back button — rather than from typing here. Adjusting state during
  // render is React's documented answer for deriving from props; an effect
  // would render once with the stale value first.
  const [lastQ, setLastQ] = useState(query.q);
  if (query.q !== lastQ) {
    setLastQ(query.q);
    setTerm(query.q ?? "");
  }

  const navigate = (patch: Partial<TicketQuery>) =>
    router.push(withQuery("/staff/tickets", query, patch));

  const onSearchChange = (value: string) => {
    setTerm(value);
    if (debounce.current) clearTimeout(debounce.current);
    debounce.current = setTimeout(() => {
      router.push(
        withQuery("/staff/tickets", query, { q: value.trim() || undefined }),
      );
    }, 300);
  };

  useEffect(
    () => () => {
      if (debounce.current) clearTimeout(debounce.current);
    },
    [],
  );

  const hasFacets = Boolean(query.priority || query.status || query.q);

  return (
    <div className="shrink-0 space-y-3 px-4 pb-3">
      <div className="relative">
        <Search
          className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-tl-faint"
          aria-hidden
        />
        <input
          type="search"
          value={term}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="Search in this view…"
          aria-label="Search tickets in this view"
          className="h-9 w-full rounded-btn border border-tl-line bg-white pl-9 pr-3 text-ui-sm text-tl-ink outline-none transition-colors duration-150 placeholder:text-tl-faint focus:border-tl-blue focus:ring-2 focus:ring-tl-blue/15"
        />
      </div>

      <div className="flex flex-wrap items-center gap-1.5">
        <FacetMenu
          label="Priority"
          value={query.priority ? PRIORITY_LABEL[query.priority] : null}
          options={PRIORITIES.map((p) => ({
            id: p,
            label: PRIORITY_LABEL[p],
          }))}
          onSelect={(id) =>
            navigate({ priority: (id as Priority) ?? undefined })
          }
          onClear={() => navigate({ priority: undefined })}
        />

        <FacetMenu
          label="Status"
          value={query.status ? STATUS_LABEL[query.status] : null}
          options={STATUSES.map((s) => ({ id: s, label: STATUS_LABEL[s] }))}
          onSelect={(id) => navigate({ status: (id as TicketStatus) ?? undefined })}
          onClear={() => navigate({ status: undefined })}
        />

        <FacetMenu
          label="Sort"
          value={SORT_LABEL[query.sort]}
          alwaysShowValue
          options={SORTS.map((s) => ({ id: s, label: SORT_LABEL[s] }))}
          onSelect={(id) => navigate({ sort: id as SortId })}
        />

        {hasFacets && (
          <Chip
            onClick={() =>
              navigate({ priority: undefined, status: undefined, q: undefined })
            }
          >
            <X className="size-3" aria-hidden />
            Clear
          </Chip>
        )}
      </div>
    </div>
  );
}

function FacetMenu({
  label,
  value,
  options,
  onSelect,
  onClear,
  alwaysShowValue,
}: {
  label: string;
  value: string | null;
  options: { id: string; label: string }[];
  onSelect: (id: string | null) => void;
  onClear?: () => void;
  alwaysShowValue?: boolean;
}) {
  const active = value !== null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Chip active={active && !alwaysShowValue}>
          {alwaysShowValue ? `${label}: ${value}` : (value ?? label)}
          <ChevronDown className="size-3 opacity-60" aria-hidden />
        </Chip>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="start" className="w-44">
        <DropdownMenuLabel>{label}</DropdownMenuLabel>
        <DropdownMenuSeparator />
        {options.map((option) => (
          <DropdownMenuItem
            key={option.id}
            onSelect={() => onSelect(option.id)}
            className="justify-between"
          >
            {option.label}
            {value === option.label && <Check className="size-4" aria-hidden />}
          </DropdownMenuItem>
        ))}
        {onClear && active && (
          <>
            <DropdownMenuSeparator />
            <DropdownMenuItem onSelect={onClear}>
              <SlidersHorizontal className="size-4" aria-hidden />
              Clear {label.toLowerCase()}
            </DropdownMenuItem>
          </>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
