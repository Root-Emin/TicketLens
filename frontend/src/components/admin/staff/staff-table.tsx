"use client";

import Link from "next/link";
import { UsersRound } from "lucide-react";

import { EmptyState } from "@/components/portal/primitives";
import { InitialsAvatar } from "@/components/staff/primitives";
import { Skeleton } from "@/components/shadcn/skeleton";
import {
  LoadBar,
  RecordCard,
  RecordField,
  StaffStatusBadge,
  Table,
  TableScroll,
  Td,
  Th,
  Tr,
} from "@/components/admin/primitives";
import type { StaffMember } from "@/lib/admin/types";
import { loadBand } from "@/lib/admin/workforce";
import { relativeTime } from "@/lib/utils";
import { StaffRowActions } from "./staff-row-actions";

/*
  The staff table, and its mobile self.

  Below xl the table becomes a list of cards rather than a horizontally
  scrolling grid. Eight columns cannot be scanned through a 360px window at any
  scroll position, and the card keeps the two things a phone is actually used
  for here — finding somebody, and seeing whether they are buried.

  Rows are not links. The row's affordances are its actions menu and the
  department chips, and making the whole row navigate would swallow those
  clicks; the name button opens the detail sheet, which is what "click the
  person" should do.
*/

const COLUMNS = [
  "Staff",
  "Role",
  "Department",
  "Status",
  "Open",
  "Workload",
  "Last active",
] as const;

export function StaffTable({
  staff,
  busiest,
  currentUserId,
  canAssign,
  onOpenProfile,
  onChangeDepartment,
  onEditStaff,
  onRemoveFromDepartment,
  emptyAction,
  emptyTitle,
  emptyDescription,
  filtered,
  /** Hides the Department column where it would repeat the page's own subject. */
  hideDepartment = false,
}: {
  staff: StaffMember[];
  /** Highest open-ticket count on the whole team — the load bar's full width. */
  busiest: number;
  currentUserId?: string;
  /** Whether this session probably holds user:write. */
  canAssign: boolean;
  onOpenProfile: (member: StaffMember) => void;
  onChangeDepartment: (member: StaffMember) => void;
  onEditStaff?: (member: StaffMember) => void;
  onRemoveFromDepartment?: (member: StaffMember) => void;
  emptyAction?: React.ReactNode;
  emptyTitle?: string;
  emptyDescription?: string;
  /** Whether a filter is narrowing the list, which changes the empty copy. */
  filtered: boolean;
  hideDepartment?: boolean;
}) {
  if (staff.length === 0) {
    return (
      <EmptyState
        icon={UsersRound}
        title={
          emptyTitle ??
          (filtered ? "Nobody matches these filters" : "No staff yet")
        }
        description={
          emptyDescription ??
          (filtered
            ? "Try a different department, or clear the filters to see everyone in the organization."
            : "People appear here once they hold a role in this organization. New accounts are created by signing up and being granted a role.")
        }
        action={emptyAction}
      />
    );
  }

  const columns = hideDepartment
    ? COLUMNS.filter((column) => column !== "Department")
    : COLUMNS;

  const rowProps = {
    busiest,
    canAssign,
    onOpenProfile,
    onChangeDepartment,
    onEditStaff,
    onRemoveFromDepartment,
    hideDepartment,
  };

  return (
    <>
      <TableScroll>
        <Table className="hidden xl:table">
          <thead>
            <tr>
              {columns.map((column) => (
                <Th
                  key={column}
                  numeric={column === "Open"}
                  className={column === "Workload" ? "w-[152px]" : undefined}
                >
                  {column}
                </Th>
              ))}
              <Th className="w-12">
                <span className="sr-only">Actions</span>
              </Th>
            </tr>
          </thead>
          <tbody>
            {staff.map((member) => (
              <StaffRow
                key={member.id}
                member={member}
                isCurrentUser={member.id === currentUserId}
                {...rowProps}
              />
            ))}
          </tbody>
        </Table>
      </TableScroll>

      <div className="xl:hidden">
        {staff.map((member) => (
          <StaffCard
            key={member.id}
            member={member}
            isCurrentUser={member.id === currentUserId}
            {...rowProps}
          />
        ))}
      </div>
    </>
  );
}

interface RowProps {
  member: StaffMember;
  busiest: number;
  isCurrentUser: boolean;
  canAssign: boolean;
  hideDepartment: boolean;
  onOpenProfile: (member: StaffMember) => void;
  onChangeDepartment: (member: StaffMember) => void;
  onEditStaff?: (member: StaffMember) => void;
  onRemoveFromDepartment?: (member: StaffMember) => void;
}

function StaffRow({
  member,
  busiest,
  isCurrentUser,
  canAssign,
  hideDepartment,
  onOpenProfile,
  onChangeDepartment,
  onEditStaff,
  onRemoveFromDepartment,
}: RowProps) {
  return (
    <Tr>
      <Td className="min-w-[220px]">
        <Identity
          member={member}
          isCurrentUser={isCurrentUser}
          onOpenProfile={onOpenProfile}
        />
      </Td>

      <Td>
        <RoleCell role={member.role} />
      </Td>

      {!hideDepartment && (
        <Td className="min-w-[180px]">
          <DepartmentCell member={member} />
        </Td>
      )}

      <Td>
        <StaffStatusBadge status={member.status} />
      </Td>

      <Td numeric className="font-semibold text-tl-ink">
        {member.openTickets}
      </Td>

      <Td>
        <LoadBar
          open={member.openTickets}
          busiest={busiest}
          band={loadBand(member.openTickets, busiest)}
        />
      </Td>

      <Td className="whitespace-nowrap text-ui-sm text-tl-faint">
        <LastActive at={member.lastActiveAt} />
      </Td>

      <Td className="text-right">
        <StaffRowActions
          member={member}
          canAssign={canAssign}
          onViewProfile={() => onOpenProfile(member)}
          onChangeDepartment={() => onChangeDepartment(member)}
          onEditStaff={onEditStaff ? () => onEditStaff(member) : undefined}
          onRemoveFromDepartment={
            onRemoveFromDepartment
              ? () => onRemoveFromDepartment(member)
              : undefined
          }
        />
      </Td>
    </Tr>
  );
}

function StaffCard({
  member,
  busiest,
  isCurrentUser,
  canAssign,
  hideDepartment,
  onOpenProfile,
  onChangeDepartment,
  onEditStaff,
  onRemoveFromDepartment,
}: RowProps) {
  return (
    <RecordCard>
      <div className="flex items-start gap-3">
        <div className="min-w-0 flex-1">
          <Identity
            member={member}
            isCurrentUser={isCurrentUser}
            onOpenProfile={onOpenProfile}
          />
        </div>
        <StaffStatusBadge status={member.status} className="mt-1 shrink-0" />
        <StaffRowActions
          member={member}
          canAssign={canAssign}
          onViewProfile={() => onOpenProfile(member)}
          onChangeDepartment={() => onChangeDepartment(member)}
          onEditStaff={onEditStaff ? () => onEditStaff(member) : undefined}
          onRemoveFromDepartment={
            onRemoveFromDepartment
              ? () => onRemoveFromDepartment(member)
              : undefined
          }
        />
      </div>

      <dl className="mt-3 grid grid-cols-2 gap-x-4 gap-y-3">
        {!hideDepartment && (
          <RecordField label="Department">
            <DepartmentCell member={member} />
          </RecordField>
        )}
        <RecordField label="Role">
          <RoleCell role={member.role} />
        </RecordField>
        <RecordField label="Workload">
          <LoadBar
            open={member.openTickets}
            busiest={busiest}
            band={loadBand(member.openTickets, busiest)}
          />
        </RecordField>
        <RecordField label="Last active">
          <LastActive at={member.lastActiveAt} />
        </RecordField>
      </dl>
    </RecordCard>
  );
}

function Identity({
  member,
  isCurrentUser,
  onOpenProfile,
}: {
  member: StaffMember;
  isCurrentUser: boolean;
  onOpenProfile: (member: StaffMember) => void;
}) {
  return (
    <div className="flex min-w-0 items-center gap-3">
      <InitialsAvatar
        name={member.name}
        initials={member.initials}
        size={34}
        className="shrink-0"
      />
      <div className="min-w-0">
        <button
          type="button"
          onClick={() => onOpenProfile(member)}
          className="flex max-w-full items-center gap-1.5 rounded-sm text-left focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
        >
          <span className="truncate text-ui-md font-semibold text-tl-ink hover:text-tl-blue">
            {member.name}
          </span>
          {isCurrentUser && (
            <span className="shrink-0 rounded bg-tl-line-soft px-1.5 py-0.5 text-ui-xs font-semibold text-tl-muted">
              You
            </span>
          )}
        </button>
        <div className="truncate text-ui-sm text-tl-faint">{member.email}</div>
      </div>
    </div>
  );
}

/**
 * The role column.
 *
 * GET /users returns no role and there is no endpoint that lists them, so this
 * is honestly empty rather than guessed at. The table's footnote carries the
 * explanation once, instead of a tooltip on every row.
 */
function RoleCell({ role }: { role: string | null }) {
  if (!role) {
    return (
      <span className="text-ui-sm text-tl-faint-soft" title="Not returned by the API">
        —
      </span>
    );
  }

  return (
    <span className="inline-flex rounded-md bg-tl-line-soft px-2 py-[3px] text-ui-xs font-semibold text-tl-ink-soft">
      {role}
    </span>
  );
}

/**
 * The team this person is on.
 *
 * A record rather than a derivation now (staff_departments, migration 00021),
 * which is why it is one value and not a list. The chip links into the
 * department's own page — the roster is where you look somebody up, and the
 * department is where you go next.
 *
 * Unassigned reads as a call to action rather than as a blank, because it is
 * one: it is the state a manager opens this screen to clear.
 */
function DepartmentCell({ member }: { member: StaffMember }) {
  if (!member.department) {
    return (
      <span className="inline-flex rounded-md bg-tl-orange-soft px-2 py-[3px] text-ui-xs font-semibold text-tl-orange-ink">
        Unassigned
      </span>
    );
  }

  return (
    <Link
      href={`/departments/${member.department.id}`}
      className="inline-flex max-w-[170px] truncate rounded-md bg-tl-blue-soft px-2 py-[3px] text-ui-xs font-semibold text-tl-blue transition-colors duration-150 hover:bg-blue-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
    >
      {member.department.name}
    </Link>
  );
}

/**
 * When this person last touched a ticket.
 *
 * The closest thing to availability the schema supports. There is no
 * last_seen_at on users, so this is the newest `updated_at` across the tickets
 * they hold — which means somebody with no tickets has never "been active",
 * and the cell says exactly that rather than "Never", which would read as an
 * accusation.
 */
function LastActive({ at }: { at: string | null }) {
  if (!at) return <span className="text-tl-faint-soft">No activity</span>;
  return <time dateTime={at}>{relativeTime(at)}</time>;
}

export function StaffTableSkeleton({ rows = 6 }: { rows?: number }) {
  return (
    <div aria-hidden>
      {Array.from({ length: rows }).map((_, index) => (
        <div
          key={index}
          className="flex items-center gap-3 border-b border-tl-line-soft px-4 py-3.5 last:border-0"
        >
          <Skeleton className="size-[34px] shrink-0 rounded-full" />
          <div className="min-w-0 flex-1 space-y-1.5">
            <Skeleton className="h-3.5 w-40 max-w-full" />
            <Skeleton className="h-3 w-56 max-w-full" />
          </div>
          <Skeleton className="hidden h-5 w-28 rounded-md sm:block" />
          <Skeleton className="hidden h-5 w-16 rounded-md md:block" />
          <Skeleton className="hidden h-1.5 w-24 rounded-full xl:block" />
        </div>
      ))}
    </div>
  );
}
