"use client";

import Link from "next/link";
import { Inbox, UsersRound } from "lucide-react";

import { EmptyState } from "@/components/portal/primitives";
import { InitialsAvatar } from "@/components/staff/primitives";
import { Skeleton } from "@/components/shadcn/skeleton";
import { LoadBar, StaffStatusBadge } from "@/components/admin/primitives";
import type { StaffMember } from "@/lib/admin/types";
import { loadBand } from "@/lib/admin/workforce";
import { relativeTime } from "@/lib/utils";
import { cn } from "@/lib/utils";
import { StaffRowActions } from "./staff-row-actions";

/*
  A team, as cards.

  Used on a department page, where the table is the wrong instrument. A table is
  for comparing a hundred people down a column; a department is three to eight
  people you are looking *at* rather than scanning past, and the questions are
  per-person — who is this, how loaded are they, when were they last on
  something. Cards put those four facts together instead of scattering them
  across seven columns of which four are empty for a small team.

  The flat roster at /team keeps its table for the opposite reason: there the
  comparison is the whole point.

  Card metrics are the same primitives the table uses — LoadBar, the status
  chip, the same relative timestamp — so a person reads identically in both
  places. Nothing here is a second vocabulary.
*/

export function StaffCardGrid({
  staff,
  busiest,
  currentUserId,
  canAssign,
  onOpenProfile,
  onChangeDepartment,
  onEditStaff,
  onRemoveFromDepartment,
  emptyTitle,
  emptyDescription,
  emptyAction,
}: {
  staff: StaffMember[];
  /** Highest open-ticket count across the organization — the bar's full width. */
  busiest: number;
  currentUserId?: string;
  canAssign: boolean;
  onOpenProfile: (member: StaffMember) => void;
  onChangeDepartment: (member: StaffMember) => void;
  onEditStaff?: (member: StaffMember) => void;
  onRemoveFromDepartment?: (member: StaffMember) => void;
  emptyTitle: string;
  emptyDescription: string;
  emptyAction?: React.ReactNode;
}) {
  if (staff.length === 0) {
    return (
      <EmptyState
        icon={UsersRound}
        title={emptyTitle}
        description={emptyDescription}
        action={emptyAction}
      />
    );
  }

  return (
    <div className="grid grid-cols-[repeat(auto-fill,minmax(260px,1fr))] gap-3 p-4 sm:p-5">
      {staff.map((member) => (
        <StaffCard
          key={member.id}
          member={member}
          busiest={busiest}
          isCurrentUser={member.id === currentUserId}
          canAssign={canAssign}
          onOpenProfile={onOpenProfile}
          onChangeDepartment={onChangeDepartment}
          onEditStaff={onEditStaff}
          onRemoveFromDepartment={onRemoveFromDepartment}
        />
      ))}
    </div>
  );
}

function StaffCard({
  member,
  busiest,
  isCurrentUser,
  canAssign,
  onOpenProfile,
  onChangeDepartment,
  onEditStaff,
  onRemoveFromDepartment,
}: {
  member: StaffMember;
  busiest: number;
  isCurrentUser: boolean;
  canAssign: boolean;
  onOpenProfile: (member: StaffMember) => void;
  onChangeDepartment: (member: StaffMember) => void;
  onEditStaff?: (member: StaffMember) => void;
  onRemoveFromDepartment?: (member: StaffMember) => void;
}) {
  return (
    <article
      className={cn(
        "flex flex-col rounded-card border border-tl-line bg-tl-card p-4 shadow-panel transition-colors duration-150",
        "hover:border-slate-300",
      )}
    >
      <header className="flex items-start gap-3">
        <InitialsAvatar
          name={member.name}
          initials={member.initials}
          size={40}
          className="mt-0.5"
        />

        <div className="min-w-0 flex-1">
          {/*
            The name opens the detail sheet. A button rather than the whole card
            being clickable: the card also holds an actions menu and a ticket
            link, and a card-wide click target swallows both.
          */}
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
          <p className="truncate text-ui-sm text-tl-faint">{member.email}</p>
        </div>

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
      </header>

      <div className="mt-3 flex items-center gap-2">
        <StaffStatusBadge status={member.status} />
        {member.pressingTickets > 0 && (
          <span className="inline-flex rounded-md bg-tl-orange-soft px-2 py-[3px] text-ui-xs font-semibold text-tl-orange-ink">
            {member.pressingTickets} high/urgent
          </span>
        )}
      </div>

      {/* mt-auto pins the footer, so cards in a row line their metrics up even
          when one name wraps to two lines. */}
      <div className="mt-auto space-y-2.5 pt-3.5">
        <div>
          <div className="mb-1 flex items-baseline justify-between gap-2">
            <span className="text-ui-xs font-medium text-tl-faint">Workload</span>
            <span className="text-ui-xs tabular-nums text-tl-muted">
              {member.openTickets} open
            </span>
          </div>
          <LoadBar
            open={member.openTickets}
            busiest={busiest}
            band={loadBand(member.openTickets, busiest)}
            showCount={false}
            fullWidth
          />
        </div>

        <div className="flex items-center justify-between gap-2 border-t border-tl-line-soft pt-2.5">
          <span className="truncate text-ui-xs text-tl-faint">
            {member.lastActiveAt ? (
              <>
                Active <time dateTime={member.lastActiveAt}>{relativeTime(member.lastActiveAt)}</time>
              </>
            ) : (
              "No ticket activity"
            )}
          </span>

          {member.openTickets > 0 && (
            <Link
              href={`/tickets?assignee_id=${member.id}`}
              className="inline-flex shrink-0 items-center gap-1 rounded-sm text-ui-xs font-semibold text-tl-blue hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
            >
              <Inbox className="size-3" aria-hidden />
              Tickets
            </Link>
          )}
        </div>
      </div>
    </article>
  );
}

export function StaffCardGridSkeleton({ cards = 3 }: { cards?: number }) {
  return (
    <div
      className="grid grid-cols-[repeat(auto-fill,minmax(260px,1fr))] gap-3 p-4 sm:p-5"
      aria-hidden
    >
      {Array.from({ length: cards }).map((_, index) => (
        <div
          key={index}
          className="rounded-card border border-tl-line bg-tl-card p-4 shadow-panel"
        >
          <div className="flex items-start gap-3">
            <Skeleton className="size-10 shrink-0 rounded-full" />
            <div className="min-w-0 flex-1 space-y-1.5">
              <Skeleton className="h-4 w-28 max-w-full" />
              <Skeleton className="h-3 w-40 max-w-full" />
            </div>
          </div>
          <Skeleton className="mt-3 h-5 w-20 rounded-md" />
          <Skeleton className="mt-4 h-1.5 w-full rounded-full" />
          <Skeleton className="mt-3 h-3 w-24" />
        </div>
      ))}
    </div>
  );
}
