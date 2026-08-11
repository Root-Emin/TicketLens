import type { TicketStatus } from "@/lib/api/types";
import type {
  PortalSort,
  PortalStatusFilter,
  PortalTicketQuery,
} from "./types";

/*
  The portal's URL state.

  The query string is the single source of truth, exactly as it is on the staff
  queue and the legacy screens: filters survive a reload, a shared link opens
  the same list, and the back button steps through filter changes.
*/

export const PAGE_SIZE = 10;

export const STATUS_FILTERS: {
  id: PortalStatusFilter;
  label: string;
}[] = [
  { id: "all", label: "All" },
  { id: "open", label: "Open" },
  { id: "in_progress", label: "In Progress" },
  { id: "pending_customer", label: "Waiting Customer" },
  { id: "resolved", label: "Resolved" },
];

export const SORTS: { id: PortalSort; label: string }[] = [
  { id: "newest", label: "Newest" },
  { id: "oldest", label: "Oldest" },
  { id: "updated", label: "Recently Updated" },
];

const STATUS_IDS = STATUS_FILTERS.map((f) => f.id);
const SORT_IDS = SORTS.map((s) => s.id);

/**
 * The API statuses behind each facet.
 *
 * "Resolved" folds in `closed`: a customer reads both as "this is done", and
 * splitting them would leave closed tickets reachable only from "All".
 */
const STATUS_MAP: Record<PortalStatusFilter, TicketStatus[] | undefined> = {
  all: undefined,
  open: ["open"],
  in_progress: ["in_progress"],
  pending_customer: ["pending_customer"],
  resolved: ["resolved", "closed"],
};

/** The backend's `sort` value for each portal sort option. */
const SORT_MAP: Record<PortalSort, string> = {
  newest: "-created_at",
  oldest: "created_at",
  // Not in the documented enum yet; the backend falls back to its default
  // ordering if it does not recognise the value. See the report.
  updated: "-updated_at",
};

export function statusesFor(filter: PortalStatusFilter) {
  return STATUS_MAP[filter];
}

export function sortParamFor(sort: PortalSort): string {
  return SORT_MAP[sort];
}

export function sortLabel(sort: PortalSort): string {
  return SORTS.find((s) => s.id === sort)?.label ?? SORTS[0].label;
}

/** parseQuery reads the ticket list state out of the URL, defaults included. */
export function parseQuery(params: URLSearchParams): PortalTicketQuery {
  const status = params.get("status");
  const sort = params.get("sort");
  const page = Number(params.get("page"));
  const q = params.get("q")?.trim();

  return {
    q: q || undefined,
    status: isStatusFilter(status) ? status : "all",
    sort: isSort(sort) ? sort : "newest",
    page: Number.isFinite(page) && page > 0 ? Math.floor(page) : 1,
  };
}

/**
 * buildSearch turns a patched query back into a query string. Anything at its
 * default is left out, so the common case stays a clean `/portal/tickets`.
 * Every change but paging resets to page 1 — page 4 of a different filter is
 * almost always empty.
 */
export function buildSearch(
  query: PortalTicketQuery,
  patch: Partial<PortalTicketQuery>,
): string {
  const next: PortalTicketQuery = {
    ...query,
    ...patch,
    page: patch.page ?? (hasFacetChange(patch) ? 1 : query.page),
  };

  const params = new URLSearchParams();
  if (next.q) params.set("q", next.q);
  if (next.status !== "all") params.set("status", next.status);
  if (next.sort !== "newest") params.set("sort", next.sort);
  if (next.page > 1) params.set("page", String(next.page));

  const search = params.toString();
  return search ? `?${search}` : "";
}

function hasFacetChange(patch: Partial<PortalTicketQuery>): boolean {
  return "q" in patch || "status" in patch || "sort" in patch;
}

function isStatusFilter(value: string | null): value is PortalStatusFilter {
  return value !== null && (STATUS_IDS as string[]).includes(value);
}

function isSort(value: string | null): value is PortalSort {
  return value !== null && (SORT_IDS as string[]).includes(value);
}
