"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import {
  Building2,
  Inbox,
  Plus,
  UserRoundX,
  UsersRound,
} from "lucide-react";

import {
  ActionButton,
  ActionLink,
  EmptyState,
  ErrorState,
  PageHeader,
} from "@/components/portal/primitives";
import {
  AdminPage,
  Footnote,
  ForbiddenState,
  KpiRow,
  KpiRowSkeleton,
  TableFrame,
  TableToolbar,
  StaleSessionState,
  type KpiDef,
} from "@/components/admin/primitives";
import { StaffTableSkeleton } from "@/components/admin/staff/staff-table";
import { ApiError } from "@/lib/api/client";
import { forbidden, sessionLikelyStale, useWorkforce } from "@/lib/admin/hooks";
import type { DepartmentRow, StaffMember } from "@/lib/admin/types";
import { can, PERMISSION } from "@/lib/auth/permissions";
import { DeleteDepartmentDialog } from "./delete-department-dialog";
import { DepartmentCard } from "./department-card";
import { DepartmentDialog } from "./department-dialog";

/*
  Departments — the entry point for workforce management by team.

  Hierarchy: Departments → Department detail → Staff on that team.
  This screen lists every department so a manager can open one and manage its
  people. Writes (create/edit/delete) stay behind department:manage; the list
  itself is readable with department:manage or ticket:read.
*/

export function DepartmentsWorkspace({ roles }: { roles: string[] }) {
  const { derived, isLoading, isError, error, refetch, isRefetching } =
    useWorkforce();

  const [editing, setEditing] = useState<DepartmentRow | null>(null);
  const [creating, setCreating] = useState(false);
  const [deleting, setDeleting] = useState<DepartmentRow | null>(null);

  const canManage = can(roles, PERMISSION.manageDepartments);

  const rows = useMemo(() => derived?.departmentRows ?? [], [derived]);
  const staff = useMemo(() => derived?.staff ?? [], [derived]);
  const rosterForbidden = Boolean(forbidden(derived?.errors.roster));

  const sorted = useMemo(
    () =>
      [...rows].sort(
        (a, b) =>
          Number(b.isDefault) - Number(a.isDefault) ||
          b.openTickets - a.openTickets ||
          a.name.localeCompare(b.name),
      ),
    [rows],
  );

  const membersByDepartment = useMemo(() => {
    const map = new Map<string, StaffMember[]>();
    for (const member of staff) {
      const id = member.department?.id;
      if (!id) continue;
      const list = map.get(id);
      if (list) list.push(member);
      else map.set(id, [member]);
    }
    for (const list of map.values()) {
      list.sort(
        (a, b) =>
          b.openTickets - a.openTickets || a.name.localeCompare(b.name),
      );
    }
    return map;
  }, [staff]);

  const takenCategories = useMemo(() => {
    const taken = new Map<string, string>();
    for (const row of rows) {
      if (row.category) taken.set(row.category, row.name);
    }
    return taken;
  }, [rows]);

  const fallbackName =
    rows.find((row) => row.isDefault)?.name ?? "the default department";

  const errors = derived?.errors;
  const staleSession = errors ? sessionLikelyStale(errors) : false;
  const departmentsForbidden = Boolean(
    forbidden(errors?.departments) ??
      (error instanceof ApiError && error.status === 403),
  );

  const newButton = canManage ? (
    <ActionButton onClick={() => setCreating(true)}>
      <Plus className="size-4" strokeWidth={2.2} aria-hidden />
      New department
    </ActionButton>
  ) : null;

  return (
    <AdminPage>
      <PageHeader
        title="Departments"
        description="Manage your support teams, members, and workload."
        action={
          <div className="flex flex-wrap items-center gap-2">
            <ActionLink href="/team" variant="secondary">
              <UsersRound className="size-4" aria-hidden />
              All staff
            </ActionLink>
            {newButton}
          </div>
        }
      />

      {staleSession ? (
        <TableFrame>
          <StaleSessionState />
        </TableFrame>
      ) : departmentsForbidden ? (
        <TableFrame>
          <ForbiddenState
            title="You can't view this organization's departments"
            description="Reading departments needs either department:manage or ticket:read. Ask an organization admin to grant one of those permissions to your role."
            permission={PERMISSION.manageDepartments}
          />
        </TableFrame>
      ) : isError ? (
        <TableFrame>
          <ErrorState
            title="Couldn't load departments"
            description="Departments are read together with the staff list and the open queue, and one of them failed."
            onRetry={() => void refetch()}
            retrying={isRefetching}
          />
        </TableFrame>
      ) : (
        <>
          {isLoading || !derived ? (
            <KpiRowSkeleton count={5} />
          ) : (
            <KpiRow items={kpis(sorted, derived.summary.unassigned)} />
          )}

          <TableFrame>
            <TableToolbar>
              <h2 className="text-ui-md font-semibold text-tl-ink">
                All departments
              </h2>
              <span className="text-ui-sm tabular-nums text-tl-muted">
                {sorted.length}
              </span>
              {!canManage && (
                <span className="ml-auto text-ui-xs text-tl-faint">
                  Read-only — editing needs department:manage
                </span>
              )}
            </TableToolbar>

            {isLoading || !derived ? (
              <StaffTableSkeleton rows={4} />
            ) : sorted.length === 0 ? (
              <EmptyState
                icon={Building2}
                title="No departments yet"
                description="Every organization is created with a default department. If this list is empty, something went wrong provisioning it."
                action={newButton}
              />
            ) : (
              <>
                {sorted.map((department) => (
                  <DepartmentCard
                    key={department.id}
                    department={department}
                    members={
                      rosterForbidden
                        ? []
                        : (membersByDepartment.get(department.id) ?? [])
                    }
                    canManage={canManage}
                    onEdit={setEditing}
                    onDelete={setDeleting}
                  />
                ))}
                <UnassignedBanner count={derived.summary.unassigned} />
              </>
            )}
          </TableFrame>

          {rosterForbidden && (
            <Footnote>
              Member avatars are hidden because reading the roster needs{" "}
              <code className="rounded bg-tl-line-soft px-1 py-0.5 font-mono text-ui-xs">
                user:read
              </code>
              . Department headcounts still come from the departments API.
            </Footnote>
          )}

          <Footnote>
            Open a department to manage its staff — add people, move them, or
            remove them from the team.{" "}
            <strong className="font-semibold text-tl-muted">Open tickets</strong>{" "}
            are counted from the current queue sample.
          </Footnote>
        </>
      )}

      <DepartmentDialog
        key={editing?.id ?? (creating ? "new" : "idle")}
        open={creating || editing !== null}
        department={editing}
        takenCategories={takenCategories}
        onOpenChange={(open) => {
          if (!open) {
            setCreating(false);
            setEditing(null);
          }
        }}
      />

      <DeleteDepartmentDialog
        department={deleting}
        fallbackName={fallbackName}
        onOpenChange={(open) => {
          if (!open) setDeleting(null);
        }}
      />
    </AdminPage>
  );
}

function kpis(departments: DepartmentRow[], unassignedStaff: number): KpiDef[] {
  const open = departments.reduce((sum, d) => sum + d.openTickets, 0);
  const staffed = departments.reduce((sum, d) => sum + d.staffCount, 0);
  const empty = departments.filter((d) => d.staffCount === 0).length;

  return [
    {
      label: "Departments",
      value: departments.length,
      icon: Building2,
      accent: "neutral",
      hint: "Including the default",
    },
    {
      label: "Staff placed",
      value: staffed,
      icon: UsersRound,
      accent: "neutral",
      href: "/team",
      hint: "Across every department",
    },
    {
      label: "Empty teams",
      value: empty,
      icon: UserRoundX,
      accent: empty > 0 ? "orange" : "neutral",
      hint: empty > 0 ? "Receiving work with nobody on them" : "All staffed",
    },
    {
      label: "Unplaced staff",
      value: unassignedStaff,
      icon: UserRoundX,
      accent: unassignedStaff > 0 ? "orange" : "neutral",
      href: "/team?department=none",
      hint: "On the roster, on no team",
    },
    {
      label: "Open tickets",
      value: open,
      icon: Inbox,
      accent: "blue",
      hint: "Across every department",
    },
  ];
}

function UnassignedBanner({ count }: { count: number }) {
  if (count === 0) return null;

  return (
    <div className="border-t border-dashed border-tl-line bg-tl-orange-soft/25 px-4 py-3.5 sm:px-5">
      <Link
        href="/team?department=none"
        className="flex items-center gap-2 rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
      >
        <UserRoundX className="size-4 shrink-0 text-tl-orange-ink" aria-hidden />
        <div className="min-w-0 flex-1">
          <div className="text-ui-md font-semibold text-tl-ink">
            Unassigned staff
          </div>
          <p className="text-ui-sm text-tl-faint">
            On the roster, not on a team — place them from a department or All
            staff.
          </p>
        </div>
        <span className="tabular-nums text-ui-md font-semibold text-tl-orange-ink">
          {count}
        </span>
      </Link>
    </div>
  );
}
