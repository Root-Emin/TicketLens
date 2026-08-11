import type { StaffQuery, StaffSort, StaffStatusFilter } from "./types";
import { DEFAULT_QUERY } from "./workforce";

/*
  The staff list's query lives in the URL.

  Same contract as the agent queue and the customer portal: every control
  navigates rather than setting state, so a filtered list is bookmarkable, the
  back button steps through filter changes, and a reload keeps the view. It also
  means a link to "everyone in Payment Operations carrying something" is
  something one administrator can paste to another.

  Defaults are omitted from the written URL, so the unfiltered page is /team and
  not /team?q=&department=all&status=all&sort=name.
*/

const SORTS: StaffSort[] = ["name", "workload", "recent", "department"];
const STATUSES: StaffStatusFilter[] = ["all", "active", "inactive"];

export const SORT_LABELS: Record<StaffSort, string> = {
  name: "Name (A–Z)",
  workload: "Workload (highest first)",
  recent: "Last active (most recent)",
  department: "Department",
};

export const STATUS_LABELS: Record<StaffStatusFilter, string> = {
  all: "Any status",
  active: "Active",
  inactive: "Not active",
};

/** parseStaffQuery narrows untrusted search params onto the query type. */
export function parseStaffQuery(params: URLSearchParams): StaffQuery {
  const sort = params.get("sort") as StaffSort | null;
  const status = params.get("status") as StaffStatusFilter | null;

  return {
    q: params.get("q") ?? "",
    department: params.get("department") || DEFAULT_QUERY.department,
    status: status && STATUSES.includes(status) ? status : DEFAULT_QUERY.status,
    sort: sort && SORTS.includes(sort) ? sort : DEFAULT_QUERY.sort,
  };
}

/**
 * buildStaffSearch returns the search string for the current query plus a patch.
 *
 * Returns "" rather than "?" when nothing is set, so the default view's URL is
 * clean.
 */
export function buildStaffSearch(
  current: StaffQuery,
  patch: Partial<StaffQuery>,
): string {
  const next: StaffQuery = { ...current, ...patch };
  const params = new URLSearchParams();

  const q = next.q.trim();
  if (q) params.set("q", q);
  if (next.department !== DEFAULT_QUERY.department) {
    params.set("department", next.department);
  }
  if (next.status !== DEFAULT_QUERY.status) params.set("status", next.status);
  if (next.sort !== DEFAULT_QUERY.sort) params.set("sort", next.sort);

  const search = params.toString();
  return search ? `?${search}` : "";
}
