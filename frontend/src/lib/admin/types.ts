import type { DepartmentRef, TicketPriority } from "@/lib/api/types";

/*
  The management panel's view of the organization.

  A staff member as an administrator thinks of them — someone with a team, a
  workload and a last-seen time — is assembled in workforce.ts from GET /staff
  and GET /tickets. The identity and the team come from the roster, which is a
  real record; only the workload is derived, because how much somebody is
  carrying is a property of the queue rather than of their employment.

  Every field below is marked with where it comes from. A field with no source
  is not invented here; it is typed, left null, and the UI says so.
*/

/** Which fields the staff table can be ordered by. */
export type StaffSort = "name" | "workload" | "recent" | "department";

/** The status facets the staff list offers, over `user.status`. */
export type StaffStatusFilter = "all" | "active" | "inactive";

export interface StaffQuery {
  q: string;
  /** Department id, or "all", or "none" for people holding no tickets. */
  department: string;
  status: StaffStatusFilter;
  sort: StaffSort;
}

export interface StaffMember {
  /* --- from GET /users (org-scoped) --------------------------------------- */
  id: string;
  email: string;
  firstName: string;
  lastName: string;
  /** Display name, falling back to the email when a profile has no names. */
  name: string;
  initials: string;
  /** Raw account status: "active", "inactive", "suspended", … */
  status: string;
  joinedAt: string;

  /*
    Not returned by any endpoint.

    There is no role on dto.UserInfo or dto.StaffInfo, and no GET /roles to look
    one up through — POST /roles/assign takes a role id the frontend has no way
    to obtain. This stays null until the backend exposes it; the table renders a
    dash and says why rather than guessing.
  */
  role: string | null;

  /**
   * The team this person is on, or null when they are on none.
   *
   * A record now, not a derivation: staff_departments (migration 00021) holds
   * it and GET /staff returns it joined. Being unassigned is a real state with
   * its own meaning — somebody newly granted an account, or returned to the
   * pool when their department was deleted — rather than the side effect of
   * holding no tickets that it used to be.
   */
  department: DepartmentRef | null;

  /* --- derived from the open queue ---------------------------------------- */
  openTickets: number;
  /** Open tickets at high or urgent priority — the part of a load that hurts. */
  pressingTickets: number;
  /** Most recent `updated_at` across their tickets, or null if they hold none. */
  lastActiveAt: string | null;
}

export interface WorkforceSummary {
  /** Everybody on the roster. Portal customers are excluded server-side. */
  total: number;
  active: number;
  inactive: number;
  /** Staff on no team — the people a manager has to place. */
  unassigned: number;
  /** Active staff currently holding no open tickets. */
  idle: number;
  departments: number;
  /** Open tickets with nobody on them. An operations number, not a headcount. */
  unassignedTickets: number;
  /**
   * True when the workload sample hit the API's 100-row ceiling, so the derived
   * columns describe the most recent 100 open tickets rather than all of them.
   * Surfaced in the UI; a number whose basis is invisible is worse than none.
   */
  sampleTruncated: boolean;
  /** True when the organization has more staff than one page of GET /staff. */
  rosterTruncated: boolean;
}

export interface DepartmentRow {
  /* --- from GET /departments ---------------------------------------------- */
  id: string;
  name: string;
  description: string;
  category: string | null;
  isDefault: boolean;
  /** Lifetime ticket count, counted by the backend. */
  ticketCount: number;
  /** People assigned to this department, counted by the backend. */
  staffCount: number;

  /* --- derived from the open-ticket sample --------------------------------- */
  openTickets: number;
}

/** Bucketed load, so the table can colour a bar without inventing a capacity. */
export type LoadBand = "none" | "light" | "steady" | "heavy";

export interface PriorityCount {
  priority: TicketPriority;
  count: number;
}
