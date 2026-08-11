import { useMemo } from "react";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";

import { ApiError } from "@/lib/api/client";
import { keys as triageKeys } from "@/lib/api/hooks";
import { adminApi, type DepartmentInput } from "./api";
import { deriveDepartments, deriveStaff, summarize } from "./workforce";

/*
  React Query bindings for the management panel.

  Keys are namespaced under "admin" so they never collide with the staff-facing
  keys in lib/api/hooks.ts or the portal's in lib/portal/hooks.ts — three panels
  ask the same endpoints different questions and must not share a cache entry.

  The workforce is one query rather than three, because it is one answer: the
  roster and the department list are read together on every screen in this
  panel, and a page that rendered half of it while the rest landed would move
  rows under the cursor.
*/

export const adminKeys = {
  all: ["admin"] as const,
  workforce: ["admin", "workforce"] as const,
  departments: ["admin", "departments"] as const,
  org: ["admin", "org"] as const,
};

/**
 * A settled request: whatever came back, or whatever went wrong.
 *
 * The workforce query fans out to three endpoints behind three different
 * permissions, and collapsing them into one thrown error loses the only thing
 * the UI needs to say something true. A session holding department:manage but
 * not user:read can read departments perfectly well; failing the whole query on
 * its roster 403 made /departments announce "you cannot view departments" and
 * name a permission the caller actually holds.
 *
 * So each call is settled independently and its failure travels with it. The
 * screens decide which parts they cannot do without.
 */
async function settle<T>(promise: Promise<T>): Promise<[T, null] | [null, unknown]> {
  try {
    return [await promise, null];
  } catch (error) {
    return [null, error];
  }
}

/** The 403 out of an error, or null if it was something else. */
export function forbidden(error: unknown): ApiError | null {
  return error instanceof ApiError && error.status === 403 ? error : null;
}

export interface WorkforceErrors {
  roster: unknown;
  departments: unknown;
  tickets: unknown;
}

/**
 * True when every part came back 403.
 *
 * That is not a permissions problem to reason about — it is the signature of a
 * token whose organization no longer resolves, which is what a re-seeded
 * database leaves in an open browser tab. The permissions are looked up per
 * (user, organization), so a stale organization id finds no grants at all and
 * every gated route refuses in the same way. Told apart from a genuine
 * permission gap because a real role always holds *something*.
 */
export function sessionLikelyStale(errors: WorkforceErrors): boolean {
  return Boolean(
    forbidden(errors.roster) &&
      forbidden(errors.departments) &&
      forbidden(errors.tickets),
  );
}

/**
 * Everything the staff and department screens are built from.
 *
 * Three requests in parallel, joined here. The join is pure (see
 * workforce.ts), so it is re-run on cache reads rather than memoised into the
 * cache — cheap for an organization that fits in one page of staff.
 */
export function useWorkforce() {
  const query = useQuery({
    queryKey: adminKeys.workforce,
    queryFn: async () => {
      const [[roster, rosterError], [departments, departmentsError], [tickets, ticketsError]] =
        await Promise.all([
          settle(adminApi.listStaff()),
          settle(adminApi.listDepartments()),
          settle(adminApi.listOpenTickets()),
        ]);

      /*
        Only a total failure throws. Anything less is data one screen can work
        without: the department list needs no roster (its headcounts are
        counted server-side), and the roster renders without the queue — it just
        cannot draw a workload bar. Throwing here would take a mostly-usable
        page down to an error state.
      */
      if (rosterError && departmentsError && ticketsError) {
        throw departmentsError;
      }

      return {
        roster: roster?.data ?? [],
        rosterTotal: roster?.meta.total ?? 0,
        departments: departments?.data ?? [],
        openTickets: tickets?.data ?? [],
        openTicketTotal: tickets?.meta.total ?? 0,
        errors: {
          roster: rosterError,
          departments: departmentsError,
          tickets: ticketsError,
        } satisfies WorkforceErrors,
      };
    },
    staleTime: 60 * 1000,
  });

  const data = query.data;

  const derived = useMemo(() => {
    if (!data) return null;

    const staff = deriveStaff(data.roster, data.openTickets);

    return {
      staff,
      departments: data.departments,
      departmentRows: deriveDepartments(data.departments, data.openTickets),
      openTickets: data.openTickets,
      errors: data.errors,
      /** True when the workload columns have no queue behind them. */
      workloadUnavailable: Boolean(data.errors.tickets),
      summary: summarize({
        staff,
        rosterTotal: data.rosterTotal,
        departments: data.departments,
        openTickets: data.openTickets,
        openTicketTotal: data.openTicketTotal,
      }),
    };
  }, [data]);

  return { ...query, derived };
}

/** Departments on their own, for the filter row and the department form. */
export function useDepartments() {
  return useQuery({
    queryKey: adminKeys.departments,
    queryFn: () => adminApi.listDepartments(),
    staleTime: 5 * 60 * 1000,
  });
}

export function useOrganization() {
  return useQuery({
    queryKey: adminKeys.org,
    queryFn: async () => {
      const page = await adminApi.listOrgs();
      // The token carries one organization, so the list is the caller's own.
      return page.data?.[0] ?? null;
    },
    staleTime: 5 * 60 * 1000,
  });
}

/**
 * Moving somebody between teams, or off one entirely.
 *
 * Invalidates the whole admin cache rather than patching the roster in place:
 * a move changes the headcount on two departments as well as the person's own
 * row, and the backend counts both. Re-reading is cheaper than keeping three
 * derived numbers in step by hand.
 */
export function useSetStaffDepartment() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({
      userId,
      departmentId,
    }: {
      userId: string;
      /** null takes them off every team. */
      departmentId: string | null;
    }) => adminApi.setStaffDepartment(userId, departmentId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: adminKeys.all });
      qc.invalidateQueries({ queryKey: triageKeys.departments });
    },
  });
}

/*
  Department writes.

  Each invalidates the triage-side department cache as well as the admin one:
  /tickets reads departments through lib/api/hooks and would otherwise keep
  offering a filter for a department that was just renamed or removed. Deleting
  also invalidates tickets, because the backend moves the orphaned ones to the
  default department as part of the same call (DeleteDepartmentUseCase).
*/

function useDepartmentMutation<TArgs>(
  mutationFn: (args: TArgs) => Promise<unknown>,
  { touchesTickets = false }: { touchesTickets?: boolean } = {},
) {
  const qc = useQueryClient();
  return useMutation({
    mutationFn,
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: adminKeys.all });
      qc.invalidateQueries({ queryKey: triageKeys.departments });
      if (touchesTickets) {
        qc.invalidateQueries({ queryKey: ["tickets"] });
        qc.invalidateQueries({ queryKey: ["stats"] });
      }
    },
  });
}

export function useCreateDepartment() {
  return useDepartmentMutation((input: DepartmentInput) =>
    adminApi.createDepartment(input),
  );
}

export function useUpdateDepartment() {
  return useDepartmentMutation(
    ({ id, input }: { id: string; input: Partial<DepartmentInput> }) =>
      adminApi.updateDepartment(id, input),
  );
}

export function useDeleteDepartment() {
  // touchesTickets, because the delete reassigns them to the default department.
  // It also unassigns the department's staff — that is covered by the admin
  // invalidation above, since the roster is part of the workforce query.
  return useDepartmentMutation((id: string) => adminApi.deleteDepartment(id), {
    touchesTickets: true,
  });
}

/*
  There is deliberately no useChangePassword here. The settings screen imports
  the portal's ChangePasswordCard, which brings its own — one endpoint, one form,
  one place to fix when the minimum length changes.
*/
