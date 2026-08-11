"use client";

import { useMemo, useState } from "react";
import Link from "next/link";
import {
  ArrowLeft,
  Building2,
  ChevronRight,
  Inbox,
  Pencil,
  UserPlus,
  UserRoundX,
  UsersRound,
} from "lucide-react";

import {
  ActionButton,
  ActionLink,
  ErrorState,
  PageHeader,
} from "@/components/portal/primitives";
import { Chip } from "@/components/staff/primitives";
import {
  AdminPage,
  Footnote,
  ForbiddenState,
  KpiRow,
  KpiRowSkeleton,
  StaleSessionState,
  TableFrame,
  TableToolbar,
  type KpiDef,
} from "@/components/admin/primitives";
import { AssignDepartmentDialog } from "@/components/admin/staff/assign-department-dialog";
import { StaffDetailSheet } from "@/components/admin/staff/staff-detail-sheet";
import {
  StaffCardGrid,
  StaffCardGridSkeleton,
} from "@/components/admin/staff/staff-card-grid";
import { ApiError } from "@/lib/api/client";
import { CATEGORY_LABELS } from "@/lib/api/labels";
import type { Category } from "@/lib/api/types";
import { useMe } from "@/lib/api/hooks";
import { forbidden, sessionLikelyStale, useWorkforce } from "@/lib/admin/hooks";
import type { StaffMember } from "@/lib/admin/types";
import { busiestLoad } from "@/lib/admin/workforce";
import { can, PERMISSION } from "@/lib/auth/permissions";
import { AddStaffDialog } from "./add-staff-dialog";
import { DepartmentDialog } from "./department-dialog";
import { EditStaffDialog } from "./edit-staff-dialog";
import { RemoveStaffDialog } from "./remove-staff-dialog";

/*
  One department — and the people on it.

  This is the staff-management surface for a team: add, edit (department),
  move, and remove. The flat roster at /team remains for cross-team questions.
*/

export function DepartmentDetail({
  departmentId,
  roles,
}: {
  departmentId: string;
  roles: string[];
}) {
  const { derived, isLoading, isError, error, refetch, isRefetching } =
    useWorkforce();
  const { data: me } = useMe();

  const [selected, setSelected] = useState<StaffMember | null>(null);
  const [assigning, setAssigning] = useState<StaffMember | null>(null);
  const [editingStaff, setEditingStaff] = useState<StaffMember | null>(null);
  const [removing, setRemoving] = useState<StaffMember | null>(null);
  const [addingStaff, setAddingStaff] = useState(false);
  const [editingDept, setEditingDept] = useState(false);

  const canAssign = can(roles, PERMISSION.writeUsers);
  const canManage = can(roles, PERMISSION.manageDepartments);
  const canReadRoster = can(roles, PERMISSION.readUsers);

  const department = derived?.departmentRows.find(
    (row) => row.id === departmentId,
  );

  const members = useMemo(
    () =>
      (derived?.staff ?? [])
        .filter((member) => member.department?.id === departmentId)
        .sort(
          (a, b) =>
            b.openTickets - a.openTickets || a.name.localeCompare(b.name),
        ),
    [derived, departmentId],
  );

  const busiest = useMemo(
    () => busiestLoad(derived?.staff ?? []),
    [derived],
  );

  const unplaced = useMemo(
    () => (derived?.staff ?? []).filter((member) => member.department === null),
    [derived],
  );

  const unassignedInDepartment = useMemo(() => {
    if (!derived) return 0;
    return derived.openTickets.filter(
      (ticket) =>
        ticket.department.id === departmentId && ticket.assignee === null,
    ).length;
  }, [derived, departmentId]);

  const errors = derived?.errors;
  const staleSession = errors ? sessionLikelyStale(errors) : false;
  const departmentsForbidden = Boolean(
    forbidden(errors?.departments) ??
      (error instanceof ApiError &&
        error.status === 403 &&
        !forbidden(errors?.roster)),
  );
  const rosterForbidden = Boolean(forbidden(errors?.roster));

  if (staleSession) {
    return (
      <AdminPage>
        <Breadcrumb name="Department" />
        <TableFrame>
          <StaleSessionState />
        </TableFrame>
      </AdminPage>
    );
  }

  if (departmentsForbidden) {
    return (
      <AdminPage>
        <Breadcrumb name="Department" />
        <TableFrame>
          <ForbiddenState
            title="You can't view this department"
            description="Reading departments needs either department:manage or ticket:read. Ask an organization admin to grant one of those permissions."
            permission={PERMISSION.manageDepartments}
          />
        </TableFrame>
      </AdminPage>
    );
  }

  if (isError && !derived) {
    return (
      <AdminPage>
        <Breadcrumb name="Department" />
        <TableFrame>
          <ErrorState
            title="Couldn't load this department"
            onRetry={() => void refetch()}
            retrying={isRefetching}
          />
        </TableFrame>
      </AdminPage>
    );
  }

  if (isLoading || !derived) {
    return (
      <AdminPage>
        <Breadcrumb name="Department" />
        <div className="h-[52px]" />
        <KpiRowSkeleton count={4} />
        <TableFrame>
          <StaffCardGridSkeleton cards={4} />
        </TableFrame>
      </AdminPage>
    );
  }

  if (!department) {
    return (
      <AdminPage>
        <Breadcrumb name="Not found" />
        <TableFrame>
          <ForbiddenState
            title="That department doesn't exist"
            description="It may have been deleted, in which case its tickets moved to the default department and its staff were returned to the unassigned pool."
          />
        </TableFrame>
      </AdminPage>
    );
  }

  const addStaffButton = canAssign ? (
    <ActionButton onClick={() => setAddingStaff(true)}>
      <UserPlus className="size-4" aria-hidden />
      Add staff
    </ActionButton>
  ) : null;

  return (
    <AdminPage>
      <Breadcrumb name={department.name} />

      <PageHeader
        title={department.name}
        description="Manage members, tickets, and department settings."
        action={
          <div className="flex flex-wrap items-center gap-2">
            {canManage && (
              <ActionButton
                variant="secondary"
                onClick={() => setEditingDept(true)}
              >
                <Pencil className="size-4" aria-hidden />
                Edit department
              </ActionButton>
            )}
            {addStaffButton}
          </div>
        }
      />

      {department.description && (
        <p className="-mt-2 max-w-2xl text-ui-md text-tl-muted">
          {department.description}
        </p>
      )}

      <div className="flex flex-wrap items-center gap-2">
        {department.isDefault && (
          <Chip accent="neutral">Default department</Chip>
        )}
        {department.category ? (
          <Chip accent="blue">
            {CATEGORY_LABELS[department.category as Category] ??
              department.category}
          </Chip>
        ) : (
          <Chip accent="neutral">Manual routing only</Chip>
        )}
        <ActionLink
          href={`/tickets?department_id=${department.id}`}
          variant="secondary"
          className="ml-auto"
        >
          <Inbox className="size-4" aria-hidden />
          View queue
        </ActionLink>
      </div>

      <KpiRow
        items={kpis(department, {
          unassignedTickets: unassignedInDepartment,
          members: members.length,
        })}
      />

      <TableFrame>
        <TableToolbar>
          <h2 className="text-ui-md font-semibold text-tl-ink">Staff</h2>
          <span className="text-ui-sm tabular-nums text-tl-muted">
            {rosterForbidden ? department.staffCount : members.length}
          </span>

          <div className="ml-auto flex flex-wrap items-center gap-2">
            {!rosterForbidden && unplaced.length > 0 && (
              <Link
                href="/team?department=none"
                className="text-ui-sm font-medium text-tl-blue hover:underline"
              >
                {unplaced.length} unassigned{" "}
                {unplaced.length === 1 ? "person" : "people"}
              </Link>
            )}
            {!canAssign && canReadRoster && (
              <span className="text-ui-xs text-tl-faint">
                Read-only — moving staff needs user:write
              </span>
            )}
            {addStaffButton && (
              <ActionButton onClick={() => setAddingStaff(true)}>
                <UserPlus className="size-4" aria-hidden />
                Add staff
              </ActionButton>
            )}
          </div>
        </TableToolbar>

        {rosterForbidden ? (
          <ForbiddenState
            title="You can't view this department's staff"
            description="Reading the roster needs user:read. Department details above are still available; ask an admin to grant user:read to manage members."
            permission={PERMISSION.readUsers}
            className="py-12"
          />
        ) : (
          /*
            Cards, not the table the flat roster uses.

            A department is a handful of people you are looking at rather than
            scanning past, and the questions are per-person — who is this, how
            loaded, last seen when. The table's seven columns are built for
            comparing a hundred rows and leave four of them near-empty for a
            team of three. /team keeps the table, where the comparison is the
            point.
          */
          <StaffCardGrid
            staff={members}
            busiest={busiest}
            currentUserId={me?.id}
            canAssign={canAssign}
            onOpenProfile={setSelected}
            onChangeDepartment={setAssigning}
            onEditStaff={setEditingStaff}
            onRemoveFromDepartment={setRemoving}
            emptyTitle="No staff on this team"
            emptyDescription="Add people from the organization roster to place them on this department. Unassigned staff and people on other teams can both be moved here."
            emptyAction={addStaffButton}
          />
        )}
      </TableFrame>

      {members.length === 0 &&
        !rosterForbidden &&
        department.openTickets > 0 && (
          <Footnote>
            This department has {department.openTickets} open{" "}
            {department.openTickets === 1 ? "ticket" : "tickets"} and nobody on
            it. Tickets routed here will sit unassigned until somebody is
            placed on the team.
          </Footnote>
        )}

      <StaffDetailSheet
        member={selected}
        busiest={busiest}
        isCurrentUser={Boolean(selected && me && selected.id === me.id)}
        canAssign={canAssign}
        onChangeDepartment={() => {
          setAssigning(selected);
          setSelected(null);
        }}
        onOpenChange={(open) => {
          if (!open) setSelected(null);
        }}
      />

      <AssignDepartmentDialog
        key={assigning?.id ?? "idle-assign"}
        member={assigning}
        departments={derived.departments}
        onOpenChange={(open) => {
          if (!open) setAssigning(null);
        }}
      />

      <EditStaffDialog
        key={editingStaff?.id ?? "idle-edit"}
        member={editingStaff}
        departments={derived.departments}
        onOpenChange={(open) => {
          if (!open) setEditingStaff(null);
        }}
      />

      <RemoveStaffDialog
        key={removing?.id ?? "idle-remove"}
        member={removing}
        departmentName={department.name}
        onOpenChange={(open) => {
          if (!open) setRemoving(null);
        }}
      />

      <AddStaffDialog
        open={addingStaff}
        departmentId={department.id}
        departmentName={department.name}
        staff={derived.staff}
        onOpenChange={setAddingStaff}
      />

      {canManage && (
        <DepartmentDialog
          key={editingDept ? department.id : "idle-dept"}
          open={editingDept}
          department={department}
          takenCategories={takenCategories(derived.departmentRows)}
          onOpenChange={setEditingDept}
        />
      )}
    </AdminPage>
  );
}

function takenCategories(rows: { category: string | null; name: string }[]) {
  const taken = new Map<string, string>();
  for (const row of rows) {
    if (row.category) taken.set(row.category, row.name);
  }
  return taken;
}

function kpis(
  department: {
    staffCount: number;
    openTickets: number;
    ticketCount: number;
    id: string;
  },
  extras: { unassignedTickets: number; members: number },
): KpiDef[] {
  return [
    {
      label: "Members",
      value: extras.members || department.staffCount,
      icon: UsersRound,
      accent:
        (extras.members || department.staffCount) === 0 ? "orange" : "neutral",
      hint: "Assigned to this team",
    },
    {
      label: "Open tickets",
      value: department.openTickets,
      icon: Inbox,
      accent: "blue",
      href: `/tickets?department_id=${department.id}`,
      hint: "Currently in the queue",
    },
    {
      label: "Unassigned",
      value: extras.unassignedTickets,
      icon: UserRoundX,
      accent: extras.unassignedTickets > 0 ? "orange" : "neutral",
      hint: "Open tickets in this department with no assignee",
    },
    {
      label: "Total tickets",
      value: department.ticketCount,
      icon: Building2,
      accent: "neutral",
      hint: "Ever routed here",
    },
  ];
}

function Breadcrumb({ name }: { name: string }) {
  return (
    <nav aria-label="Breadcrumb" className="-mb-1">
      <ol className="flex items-center gap-1.5 text-ui-sm text-tl-muted">
        <li>
          <Link
            href="/departments"
            className="inline-flex items-center gap-1.5 rounded-sm hover:text-tl-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
          >
            <ArrowLeft className="size-3.5" aria-hidden />
            Departments
          </Link>
        </li>
        <ChevronRight className="size-3.5 text-tl-faint-soft" aria-hidden />
        <li className="truncate font-medium text-tl-ink" aria-current="page">
          {name}
        </li>
      </ol>
    </nav>
  );
}
