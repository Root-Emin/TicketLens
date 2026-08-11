"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { X } from "lucide-react";

import { ActionButton } from "@/components/portal/primitives";
import { SearchInput, SelectInput } from "@/components/admin/primitives";
import type { DepartmentInfo } from "@/lib/api/types";
import type { StaffQuery } from "@/lib/admin/types";
import {
  buildStaffSearch,
  SORT_LABELS,
  STATUS_LABELS,
} from "@/lib/admin/url";
import { isFiltered } from "@/lib/admin/workforce";

/*
  Search, three facets and a sort, on one line.

  Deliberately not a card. The queue screen wraps its filters in a bordered
  panel, which puts a box inside a box once the table below has its own border,
  and on a laptop that costs about 90px of the table's height before a single
  row is drawn. Here the controls live in the table's own toolbar, above the
  header row, separated by the same hairline that separates everything else.

  Search debounces so typing does not push one history entry per keystroke. The
  facets navigate immediately — a select fires once, and waiting 300ms to act on
  a deliberate choice reads as lag.
*/

const DEBOUNCE_MS = 300;

export function StaffFilters({
  query,
  departments,
  resultCount,
  totalCount,
}: {
  query: StaffQuery;
  departments: DepartmentInfo[];
  resultCount: number;
  totalCount: number;
}) {
  const router = useRouter();
  const [term, setTerm] = useState(query.q);
  const debounce = useRef<ReturnType<typeof setTimeout> | null>(null);

  // Keep the field in step when the query changes from somewhere else — a
  // "Clear" click, the back button — rather than from typing here. Adjusting
  // state during render is React's documented answer for deriving from props;
  // an effect would render once with the stale value first.
  const [lastQ, setLastQ] = useState(query.q);
  if (query.q !== lastQ) {
    setLastQ(query.q);
    setTerm(query.q);
  }

  const navigate = (patch: Partial<StaffQuery>) =>
    router.push(`/team${buildStaffSearch(query, patch)}`, { scroll: false });

  function onSearchChange(value: string) {
    setTerm(value);
    if (debounce.current) clearTimeout(debounce.current);
    debounce.current = setTimeout(() => navigate({ q: value }), DEBOUNCE_MS);
  }

  useEffect(
    () => () => {
      if (debounce.current) clearTimeout(debounce.current);
    },
    [],
  );

  const filtered = isFiltered(query);

  return (
    <div className="flex w-full flex-wrap items-center gap-2">
      <SearchInput
        value={term}
        onChange={(event) => onSearchChange(event.target.value)}
        placeholder="Search staff by name or email…"
        aria-label="Search staff"
        className="min-w-[200px] flex-1 sm:max-w-[280px]"
      />

      <SelectInput
        value={query.department}
        onChange={(event) => navigate({ department: event.target.value })}
        aria-label="Filter by department"
        className="w-[168px]"
      >
        <option value="all">All departments</option>
        {departments.map((department) => (
          <option key={department.id} value={department.id}>
            {department.name}
          </option>
        ))}
        {/*
          Not a department — the absence of one. Unplaced staff on the roster.
        */}
        <option value="none">No department</option>
      </SelectInput>

      <SelectInput
        value={query.status}
        onChange={(event) =>
          navigate({ status: event.target.value as StaffQuery["status"] })
        }
        aria-label="Filter by account status"
        className="w-[136px]"
      >
        {Object.entries(STATUS_LABELS).map(([value, label]) => (
          <option key={value} value={value}>
            {label}
          </option>
        ))}
      </SelectInput>

      <SelectInput
        value={query.sort}
        onChange={(event) =>
          navigate({ sort: event.target.value as StaffQuery["sort"] })
        }
        aria-label="Sort staff"
        className="w-[196px]"
      >
        {Object.entries(SORT_LABELS).map(([value, label]) => (
          <option key={value} value={value}>
            Sort: {label}
          </option>
        ))}
      </SelectInput>

      <div className="ml-auto flex items-center gap-2">
        <span
          className="whitespace-nowrap text-ui-sm tabular-nums text-tl-muted"
          aria-live="polite"
        >
          {filtered
            ? `${resultCount} of ${totalCount}`
            : `${totalCount} ${totalCount === 1 ? "person" : "people"}`}
        </span>

        {filtered && (
          <ActionButton
            variant="ghost"
            size="sm"
            onClick={() =>
              navigate({ q: "", department: "all", status: "all", sort: "name" })
            }
          >
            <X className="size-3.5" aria-hidden />
            Clear
          </ActionButton>
        )}
      </div>
    </div>
  );
}
