import { initialsOf } from "@/lib/portal/format";
import type {
  DepartmentInfo,
  StaffInfo,
  TicketListItem,
} from "@/lib/api/types";
import type {
  DepartmentRow,
  LoadBand,
  StaffMember,
  StaffQuery,
  WorkforceSummary,
} from "./types";

/*
  Joining the roster to the queue.

  This module used to be much larger. It reconstructed a support team out of
  GET /users and GET /tickets, because the backend had no record of one: a
  person's department was inferred from the departments of their assigned
  tickets, and portal customers were filtered out by matching email addresses
  against the customer list.

  Both of those are gone. staff_departments (migration 00021) made the
  assignment a fact, and GET /staff excludes customers in SQL through
  customers.user_id. What is left here is the one join that is genuinely a
  presentation concern — roster plus open queue equals workload — and the
  filtering and sorting the table does client-side.

  The remaining derivation is honest about being one: how many tickets somebody
  is holding is a property of the queue, and it changes when a ticket moves, not
  when somebody joins a team.
*/

/** Priorities that make a load heavier than its count suggests. */
const PRESSING = new Set(["high", "urgent"]);

/** Account statuses that count as somebody who can be given work. */
const ACTIVE_STATUS = "active";

interface AssignedWork {
  open: number;
  pressing: number;
  lastActiveAt: string | null;
}

function emptyWork(): AssignedWork {
  return { open: 0, pressing: 0, lastActiveAt: null };
}

/** Groups the open queue by who is holding each ticket. */
function indexByAssignee(tickets: TicketListItem[]): Map<string, AssignedWork> {
  const index = new Map<string, AssignedWork>();

  for (const ticket of tickets) {
    if (!ticket.assignee) continue;

    let work = index.get(ticket.assignee.id);
    if (!work) {
      work = emptyWork();
      index.set(ticket.assignee.id, work);
    }

    work.open += 1;
    if (PRESSING.has(ticket.priority)) work.pressing += 1;

    // String comparison is correct for RFC3339 in a fixed offset, which is what
    // the Go handlers emit (time.Time marshals as UTC with a Z suffix).
    if (!work.lastActiveAt || ticket.updated_at > work.lastActiveAt) {
      work.lastActiveAt = ticket.updated_at;
    }
  }

  return index;
}

/**
 * deriveStaff attaches each person's current load to their roster entry.
 *
 * Somebody holding nothing comes back with zero and a null last-active, which
 * is a real and interesting state — a newly placed agent, or a team that is
 * quiet. Their department is unaffected either way, because it is now read
 * rather than inferred.
 */
export function deriveStaff(
  roster: StaffInfo[],
  openTickets: TicketListItem[],
): StaffMember[] {
  const work = indexByAssignee(openTickets);

  return roster.map((person) => {
    const theirs = work.get(person.id) ?? emptyWork();

    return {
      id: person.id,
      email: person.email,
      firstName: person.first_name,
      lastName: person.last_name,
      name: person.full_name || person.email,
      initials: initialsOf(person.full_name || person.email),
      status: person.status,
      joinedAt: person.created_at,
      // Still null: no endpoint returns a user's role, and none lists the roles
      // to look one up through. See types.ts.
      role: null,
      department: person.department,
      openTickets: theirs.open,
      pressingTickets: theirs.pressing,
      lastActiveAt: theirs.lastActiveAt,
    };
  });
}

export function isActive(member: StaffMember): boolean {
  return member.status === ACTIVE_STATUS;
}

/** summarize builds the KPI row. Every number is counted, none estimated. */
export function summarize({
  staff,
  rosterTotal,
  departments,
  openTickets,
  openTicketTotal,
}: {
  staff: StaffMember[];
  /** meta.total from the roster, for detecting a second page. */
  rosterTotal: number;
  departments: DepartmentInfo[];
  openTickets: TicketListItem[];
  /** meta.total from the ticket query, for detecting a truncated sample. */
  openTicketTotal: number;
}): WorkforceSummary {
  const active = staff.filter(isActive);

  return {
    total: staff.length,
    active: active.length,
    inactive: staff.length - active.length,
    unassigned: staff.filter((member) => member.department === null).length,
    idle: active.filter((member) => member.openTickets === 0).length,
    departments: departments.length,
    unassignedTickets: openTickets.filter((ticket) => !ticket.assignee).length,
    sampleTruncated: openTicketTotal > openTickets.length,
    rosterTruncated: rosterTotal > staff.length,
  };
}

/**
 * deriveDepartments adds live load to what GET /departments returns.
 *
 * ticket_count and staff_count are both the backend's own figures now and are
 * passed through untouched — staff_count in particular is counted over the same
 * roster query GET /staff serves, so the number beside a department always
 * matches the list behind it. openTickets is the only derived column.
 */
export function deriveDepartments(
  departments: DepartmentInfo[],
  openTickets: TicketListItem[],
): DepartmentRow[] {
  const open = new Map<string, number>();
  for (const ticket of openTickets) {
    open.set(ticket.department.id, (open.get(ticket.department.id) ?? 0) + 1);
  }

  return departments.map((department) => ({
    id: department.id,
    name: department.name,
    description: department.description,
    category: department.category,
    isDefault: department.is_default,
    ticketCount: department.ticket_count,
    staffCount: department.staff_count,
    openTickets: open.get(department.id) ?? 0,
  }));
}

/*
  Load is expressed relative to the busiest person on the team, not against a
  fixed number of tickets.

  There is no capacity anywhere in the schema, so "heavy" cannot mean "over
  their limit" — it can only mean "carrying much more than everyone else",
  which is the question an administrator is actually asking when they scan this
  column. The consequence is honest: on a team where nobody holds more than two
  tickets, nobody is heavy.
*/
export function loadBand(open: number, busiest: number): LoadBand {
  if (open === 0) return "none";
  if (busiest <= 0) return "light";
  const share = open / busiest;
  if (share <= 0.34) return "light";
  if (share <= 0.67) return "steady";
  return "heavy";
}

/** The largest open-ticket count on the team, the bar's full width. */
export function busiestLoad(staff: StaffMember[]): number {
  return staff.reduce((max, member) => Math.max(max, member.openTickets), 0);
}

/** The empty query — what the page shows before anybody touches a control. */
export const DEFAULT_QUERY: StaffQuery = {
  q: "",
  department: "all",
  status: "all",
  sort: "name",
};

/** True when anything is narrowing the list, which is what shows "Clear". */
export function isFiltered(query: StaffQuery): boolean {
  return (
    query.q.trim() !== "" ||
    query.department !== DEFAULT_QUERY.department ||
    query.status !== DEFAULT_QUERY.status ||
    query.sort !== DEFAULT_QUERY.sort
  );
}

export function applyQuery(
  staff: StaffMember[],
  query: StaffQuery,
): StaffMember[] {
  const term = query.q.trim().toLowerCase();

  const filtered = staff.filter((member) => {
    if (term) {
      const haystack = `${member.name} ${member.email}`.toLowerCase();
      if (!haystack.includes(term)) return false;
    }

    if (query.status === "active" && !isActive(member)) return false;
    if (query.status === "inactive" && isActive(member)) return false;

    if (query.department === "none") {
      if (member.department !== null) return false;
    } else if (query.department !== "all") {
      if (member.department?.id !== query.department) return false;
    }

    return true;
  });

  return sortStaff(filtered, query.sort);
}

function sortStaff(staff: StaffMember[], sort: StaffQuery["sort"]): StaffMember[] {
  const byName = (a: StaffMember, b: StaffMember) => a.name.localeCompare(b.name);

  // Copied before sorting: the input is React Query's cached array, and sorting
  // it in place mutates data other components are rendering from.
  return [...staff].sort((a, b) => {
    switch (sort) {
      case "workload":
        return b.openTickets - a.openTickets || byName(a, b);
      case "recent":
        // Nobody-has-touched-anything sorts last rather than first, which is
        // where a null would land in a plain string comparison.
        if (!a.lastActiveAt && !b.lastActiveAt) return byName(a, b);
        if (!a.lastActiveAt) return 1;
        if (!b.lastActiveAt) return -1;
        return b.lastActiveAt.localeCompare(a.lastActiveAt) || byName(a, b);
      case "department": {
        // Unassigned last: they are the exception, and burying them at the top
        // of an alphabetical list is the opposite of useful.
        const left = a.department?.name ?? "";
        const right = b.department?.name ?? "";
        if (left && !right) return -1;
        if (!left && right) return 1;
        return left.localeCompare(right) || byName(a, b);
      }
      default:
        return byName(a, b);
    }
  });
}
