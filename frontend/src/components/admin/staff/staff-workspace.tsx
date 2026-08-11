"use client";

import { useMemo, useState } from "react";
import { useSearchParams } from "next/navigation";
import {
  Building2,
  CircleSlash,
  Inbox,
  PauseCircle,
  UserCheck,
  UserRoundX,
  UsersRound,
} from "lucide-react";

import {
  ActionLink,
  ErrorState,
  PageHeader,
} from "@/components/portal/primitives";
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
import { UNASSIGNED_HREF } from "@/components/admin/shell/nav-config";
import { ApiError } from "@/lib/api/client";
import { useMe } from "@/lib/api/hooks";
import { forbidden, sessionLikelyStale, useWorkforce } from "@/lib/admin/hooks";
import { can, PERMISSION } from "@/lib/auth/permissions";
import type { StaffMember, WorkforceSummary } from "@/lib/admin/types";
import { parseStaffQuery } from "@/lib/admin/url";
import { applyQuery, busiestLoad, isFiltered } from "@/lib/admin/workforce";
import { AssignDepartmentDialog } from "./assign-department-dialog";
import { StaffDetailSheet } from "./staff-detail-sheet";
import { StaffFilters } from "./staff-filters";
import { StaffTable, StaffTableSkeleton } from "./staff-table";
import { EditStaffDialog } from "@/components/admin/departments/edit-staff-dialog";

/*
  The staff screen.

  Structure, top to bottom: who this page is for and the one action it offers,
  six numbers, then the table with its controls in its own toolbar. No hero, no
  full-width chart — an operations screen earns its space with rows, and the
  summary above them is a strip rather than a dashboard.

  Filtering happens client-side against a single fetch, which is right at this
  size: an organization is one page of users (100), the join that produces a
  department is local, and a round trip per keystroke would be slower and no
  more correct. It is also the reason the filters can offer "no open work",
  which no query parameter on GET /users could express.
*/

export function StaffWorkspace({ roles }: { roles: string[] }) {
  const params = useSearchParams();
  const query = useMemo(
    () => parseStaffQuery(new URLSearchParams(params.toString())),
    [params],
  );

  const { derived, isLoading, isError, error, refetch, isRefetching } =
    useWorkforce();
  const { data: me } = useMe();

  const [selected, setSelected] = useState<StaffMember | null>(null);
  const [assigning, setAssigning] = useState<StaffMember | null>(null);
  const [editingStaff, setEditingStaff] = useState<StaffMember | null>(null);

  const canAssign = can(roles, PERMISSION.writeUsers);

  // Memoised rather than written inline: `derived?.staff ?? []` is a fresh array
  // on every render while the data is loading, which would re-run both memos
  // below on every keystroke in the search box.
  const staff = useMemo(() => derived?.staff ?? [], [derived]);
  const visible = useMemo(() => applyQuery(staff, query), [staff, query]);
  // Measured over everyone, not the filtered subset: a bar that rescaled when
  // you picked a department would make the same person look busier.
  const busiest = useMemo(() => busiestLoad(staff), [staff]);

  // The roster is what this screen is; a 403 on the queue only costs the
  // workload columns, and one on departments only costs the filter's options.
  const errors = derived?.errors;
  const staleSession = errors ? sessionLikelyStale(errors) : false;
  const rosterForbidden = Boolean(
    forbidden(errors?.roster) ??
      (error instanceof ApiError && error.status === 403),
  );

  return (
    <AdminPage>
      <PageHeader
        title="Staff"
        description="Manage your support team, their departments and how work is spread across them."
        action={
          <div className="flex flex-wrap items-center gap-2">
            <ActionLink href="/departments" variant="secondary">
              <Building2 className="size-4" aria-hidden />
              Manage departments
            </ActionLink>
            <ActionLink href={UNASSIGNED_HREF}>
              <UserRoundX className="size-4" aria-hidden />
              Assign open tickets
            </ActionLink>
          </div>
        }
      />

      {staleSession ? (
        <TableFrame>
          <StaleSessionState />
        </TableFrame>
      ) : rosterForbidden ? (
        <TableFrame>
          <ForbiddenState
            title="You can't view this organization's staff"
            description="Listing people needs an account with user:read. The support agent role deliberately does not carry it — agents work their own queue and do not see the roster."
            permission={PERMISSION.readUsers}
          />
        </TableFrame>
      ) : isError ? (
        <TableFrame>
          <ErrorState
            title="Couldn't load the workforce"
            description="The staff list, departments and open queue are read together, and one of them failed."
            onRetry={() => void refetch()}
            retrying={isRefetching}
          />
        </TableFrame>
      ) : (
        <>
          {isLoading || !derived ? (
            <KpiRowSkeleton />
          ) : (
            <KpiRow items={kpis(derived.summary)} />
          )}

          <TableFrame>
            <TableToolbar>
              <StaffFilters
                query={query}
                departments={derived?.departments ?? []}
                resultCount={visible.length}
                totalCount={staff.length}
              />
            </TableToolbar>

            {isLoading || !derived ? (
              <StaffTableSkeleton />
            ) : (
              <StaffTable
                staff={visible}
                busiest={busiest}
                currentUserId={me?.id}
                canAssign={canAssign}
                filtered={isFiltered(query)}
                onOpenProfile={setSelected}
                onChangeDepartment={setAssigning}
                onEditStaff={setEditingStaff}
              />
            )}
          </TableFrame>

          {derived && <Basis summary={derived.summary} />}
        </>
      )}

      <StaffDetailSheet
        member={selected}
        busiest={busiest}
        isCurrentUser={Boolean(selected && me && selected.id === me.id)}
        canAssign={canAssign}
        onChangeDepartment={() => {
          // Hand the sheet's subject to the dialog and close the sheet, rather
          // than stacking a dialog on top of it: two layers of scrim over one
          // person is a lot of chrome for a one-field change.
          setAssigning(selected);
          setSelected(null);
        }}
        onOpenChange={(open) => {
          if (!open) setSelected(null);
        }}
      />

      {/* Keyed so opening a different person resets the select. */}
      <AssignDepartmentDialog
        key={assigning?.id ?? "idle"}
        member={assigning}
        departments={derived?.departments ?? []}
        onOpenChange={(open) => {
          if (!open) setAssigning(null);
        }}
      />

      <EditStaffDialog
        key={editingStaff?.id ?? "idle-edit"}
        member={editingStaff}
        departments={derived?.departments ?? []}
        onOpenChange={(open) => {
          if (!open) setEditingStaff(null);
        }}
      />
    </AdminPage>
  );
}

/** Six numbers, and what each is counted over. */
function kpis(summary: WorkforceSummary): KpiDef[] {
  return [
    {
      label: "Total staff",
      value: summary.total,
      icon: UsersRound,
      accent: "neutral",
      hint: "Portal customers excluded",
    },
    {
      label: "Active",
      value: summary.active,
      icon: UserCheck,
      accent: "green",
      hint: "Accounts able to sign in",
    },
    {
      label: "Not active",
      value: summary.inactive,
      icon: PauseCircle,
      accent: summary.inactive > 0 ? "orange" : "neutral",
      hint: "Suspended or disabled",
    },
    {
      // The headline number of this whole panel: people nobody has placed.
      label: "No department",
      value: summary.unassigned,
      icon: UserRoundX,
      accent: summary.unassigned > 0 ? "orange" : "neutral",
      href: "/team?department=none",
      hint: "Waiting to be placed on a team",
    },
    {
      label: "No open work",
      value: summary.idle,
      icon: CircleSlash,
      accent: "neutral",
      hint: "Active, holding nothing",
    },
    {
      label: "Unassigned tickets",
      value: summary.unassignedTickets,
      icon: Inbox,
      accent: summary.unassignedTickets > 0 ? "red" : "neutral",
      href: UNASSIGNED_HREF,
      hint: "Nobody has picked these up",
    },
  ];
}

/** Says where the derived columns come from, once, under the table. */
function Basis({ summary }: { summary: WorkforceSummary }) {
  return (
    <div className="space-y-2">
      <Footnote>
        <strong className="font-semibold text-tl-muted">Role</strong> is empty
        because no endpoint returns a user&apos;s role, and none lists the roles
        to look one up through.{" "}
        <strong className="font-semibold text-tl-muted">Workload</strong> and{" "}
        <strong className="font-semibold text-tl-muted">last active</strong> are
        counted over the organization&apos;s open queue — a person carries what
        is assigned to them, which is not the same as the team they are on.
      </Footnote>

      {summary.sampleTruncated && (
        <Footnote>
          The open queue is larger than the 100 tickets the API returns in one
          page, so the workload columns describe the 100 most recently updated.
          Headcount and department numbers are unaffected.
        </Footnote>
      )}

      {summary.rosterTruncated && (
        <Footnote>
          This organization has more staff than one page of{" "}
          <code className="font-mono">GET /staff</code> returns, so the roster
          shows the first 100.
        </Footnote>
      )}
    </div>
  );
}
