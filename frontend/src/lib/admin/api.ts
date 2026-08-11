import { api, buildTicketQuery } from "@/lib/api/client";
import type {
  CollectionResponse,
  DepartmentInfo,
  ListResponse,
  OrgInfo,
  PageResult,
  StaffInfo,
  TicketListItem,
} from "@/lib/api/types";

/*
  The management panel's calls, and only the ones that exist.

  Every function here maps onto a route in
  internal/infrastructure/http/router/router.go. Nothing is stubbed: where the
  backend has no endpoint — creating a staff member, changing somebody's role,
  deactivating an account — there is no function, and the UI disables the action
  rather than calling something that would 404.

  Permissions, for the record, since the panel's whole point is that it is the
  administrator's:

    GET  /staff                     user:read
    PUT  /staff/{id}/department     user:write
    GET  /departments               department:manage OR ticket:read
    POST/PATCH/DELETE /departments  department:manage
    GET  /tickets                   ticket:read
    GET  /organizations             org:read

  The seeded `agent` role holds department:manage and ticket:read but neither
  user permission, so an agent who reached this panel would get a roster of 403
  and could not move themselves onto a team. Verified against the running
  backend, not assumed.
*/

/** The API's hard ceiling (dto.MaxPageSize / pagination.MaxPerPage). */
export const MAX_PAGE_SIZE = 100;

/**
 * The literal the backend accepts wherever a UUID otherwise would be, for
 * "nobody". Spelled `unassignedLiteral` in handler/triage/handler.go and used
 * by both ?assignee_id on the queue and ?department_id on the roster.
 */
export const UNASSIGNED = "unassigned";

/**
 * The statuses that mean "somebody is still working this".
 *
 * Resolved and closed are excluded deliberately: a workload is what is on a
 * person's desk now, and counting everything they ever touched would rank the
 * longest-serving agent as the most overloaded.
 */
export const OPEN_STATUSES = ["open", "in_progress", "pending_customer"] as const;

export interface DepartmentInput {
  name: string;
  description: string;
  /** null clears the category; the backend accepts "" for the same effect. */
  category: string | null;
}

export const adminApi = {
  /**
   * The support roster.
   *
   * Note this is /staff and not /users. GET /users returns everybody holding a
   * role in the organization, which includes portal customers — three of the
   * seeded ones — and carries no department. /staff excludes them in SQL, using
   * customers.user_id rather than an email comparison, and joins the department
   * each person is assigned to (migration 00021).
   *
   * `department` is the filter: a UUID for one team, UNASSIGNED for the people
   * on none, omitted for everybody.
   */
  listStaff: (department?: string) =>
    api.get<ListResponse<StaffInfo>>(
      `/staff${buildTicketQuery({
        department_id: department,
        page: 1,
        page_size: MAX_PAGE_SIZE,
      })}`,
    ),

  /** Places somebody on a team, or takes them off one when departmentId is null. */
  setStaffDepartment: (userId: string, departmentId: string | null) =>
    api.put<StaffInfo>(`/staff/${userId}/department`, {
      department_id: departmentId,
    }),

  listDepartments: () =>
    api.get<CollectionResponse<DepartmentInfo>>("/departments"),

  createDepartment: (input: DepartmentInput) =>
    api.post<DepartmentInfo>("/departments", {
      name: input.name,
      description: input.description,
      category: input.category,
    }),

  updateDepartment: (id: string, input: Partial<DepartmentInput>) =>
    api.patch<DepartmentInfo>(`/departments/${id}`, {
      ...(input.name !== undefined ? { name: input.name } : {}),
      ...(input.description !== undefined
        ? { description: input.description }
        : {}),
      // "" clears the category server-side; undefined leaves it untouched.
      ...(input.category !== undefined
        ? { category: input.category ?? "" }
        : {}),
    }),

  deleteDepartment: (id: string) => api.delete<null>(`/departments/${id}`),

  /**
   * The open queue, newest first, for the workload columns.
   *
   * Still needed even though the roster now carries a real department: how many
   * tickets somebody is holding is a fact about the queue, not about the
   * assignment, and there is no per-agent count endpoint. One request rather
   * than one per person; the 100-row ceiling is the API's and the caller is told
   * when it was reached.
   */
  listOpenTickets: () =>
    api.get<ListResponse<TicketListItem>>(
      `/tickets${buildTicketQuery({
        status: OPEN_STATUSES,
        sort: "-updated_at",
        page: 1,
        page_size: MAX_PAGE_SIZE,
      })}`,
    ),

  /** The organizations this token can read. In practice, the caller's own. */
  listOrgs: () => api.get<PageResult<OrgInfo>>("/organizations"),
};
