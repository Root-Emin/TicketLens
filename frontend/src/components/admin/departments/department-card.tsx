"use client";

import Link from "next/link";
import { ChevronRight, MoreHorizontal, Pencil, Trash2, Inbox } from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/shadcn/dropdown-menu";
import { Chip, InitialsAvatar } from "@/components/staff/primitives";
import type { DepartmentRow, StaffMember } from "@/lib/admin/types";

/*
  One department as a navigable card.

  The whole surface is a link into /departments/[id] — that is the management
  surface for the team's staff. Row actions sit beside the chevron and stop
  propagation so Edit / Delete do not navigate away.
*/

const AVATAR_PREVIEW = 3;

export function DepartmentCard({
  department,
  members,
  canManage,
  onEdit,
  onDelete,
}: {
  department: DepartmentRow;
  members: StaffMember[];
  canManage: boolean;
  onEdit: (department: DepartmentRow) => void;
  onDelete: (department: DepartmentRow) => void;
}) {
  const preview = members.slice(0, AVATAR_PREVIEW);
  const overflow = Math.max(0, department.staffCount - preview.length);

  return (
    <div className="group relative flex items-stretch gap-3 border-b border-tl-line-soft px-4 py-4 transition-colors duration-150 last:border-0 hover:bg-tl-line-soft/40 sm:px-5">
      <Link
        href={`/departments/${department.id}`}
        className="absolute inset-0 rounded-sm focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-tl-blue/30"
        aria-label={`View ${department.name}`}
      />

      <div className="relative z-[1] flex min-w-0 flex-1 flex-col gap-3 pointer-events-none sm:flex-row sm:items-center sm:gap-6">
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-center gap-2">
            <h3 className="truncate text-ui-md font-semibold text-tl-ink group-hover:text-tl-blue">
              {department.name}
            </h3>
            {department.isDefault && (
              <Chip accent="neutral" className="shrink-0">
                Default
              </Chip>
            )}
            {department.staffCount === 0 && (
              <Chip accent="orange" className="shrink-0">
                Unstaffed
              </Chip>
            )}
          </div>
          {department.description ? (
            <p className="mt-0.5 line-clamp-2 text-ui-sm text-tl-faint">
              {department.description}
            </p>
          ) : (
            <p className="mt-0.5 text-ui-sm text-tl-faint-soft">
              No description
            </p>
          )}
        </div>

        <div className="flex flex-wrap items-center gap-x-5 gap-y-2 sm:shrink-0">
          <Metric
            label={department.staffCount === 1 ? "member" : "members"}
            value={department.staffCount}
            warn={department.staffCount === 0}
          />
          <Metric
            label={department.openTickets === 1 ? "open ticket" : "open tickets"}
            value={department.openTickets}
          />

          <AvatarStack preview={preview} overflow={overflow} />
        </div>
      </div>

      <div className="relative z-[1] flex shrink-0 items-center gap-1 self-center pointer-events-none">
        <div className="pointer-events-auto">
          <RowActions
            department={department}
            canManage={canManage}
            onEdit={onEdit}
            onDelete={onDelete}
          />
        </div>
        <ChevronRight
          className="size-4 text-tl-faint-soft transition-colors group-hover:text-tl-blue"
          aria-hidden
        />
      </div>
    </div>
  );
}

function Metric({
  label,
  value,
  warn,
}: {
  label: string;
  value: number;
  warn?: boolean;
}) {
  return (
    <div className="min-w-[4.5rem]">
      <div
        className={
          warn
            ? "text-ui-md font-semibold tabular-nums text-tl-orange-ink"
            : "text-ui-md font-semibold tabular-nums text-tl-ink"
        }
      >
        {value}
      </div>
      <div className="text-ui-xs text-tl-faint">{label}</div>
    </div>
  );
}

function AvatarStack({
  preview,
  overflow,
}: {
  preview: StaffMember[];
  overflow: number;
}) {
  if (preview.length === 0) {
    return (
      <div className="flex h-8 min-w-[5.5rem] items-center text-ui-xs text-tl-faint-soft">
        No members yet
      </div>
    );
  }

  return (
    <div className="flex min-w-[5.5rem] items-center" aria-hidden>
      {preview.map((member, index) => (
        <div
          key={member.id}
          className="rounded-full ring-2 ring-tl-card"
          style={{ marginLeft: index === 0 ? 0 : -8 }}
        >
          <InitialsAvatar
            name={member.name}
            initials={member.initials}
            size={28}
          />
        </div>
      ))}
      {overflow > 0 && (
        <span className="ml-1.5 text-ui-xs font-semibold tabular-nums text-tl-muted">
          +{overflow}
        </span>
      )}
    </div>
  );
}

function RowActions({
  department,
  canManage,
  onEdit,
  onDelete,
}: {
  department: DepartmentRow;
  canManage: boolean;
  onEdit: (department: DepartmentRow) => void;
  onDelete: (department: DepartmentRow) => void;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`Actions for ${department.name}`}
          className="tap-target inline-flex size-8 items-center justify-center rounded-btn text-tl-faint transition-colors duration-150 hover:bg-tl-line-soft hover:text-tl-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
        >
          <MoreHorizontal className="size-4" aria-hidden />
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" sideOffset={6} className="w-56">
        <DropdownMenuItem asChild>
          <Link href={`/departments/${department.id}`}>View department</Link>
        </DropdownMenuItem>
        <DropdownMenuItem asChild>
          <Link href={`/tickets?department_id=${department.id}`}>
            <Inbox className="size-4" aria-hidden />
            View its tickets
          </Link>
        </DropdownMenuItem>
        <DropdownMenuItem
          disabled={!canManage}
          onSelect={() => onEdit(department)}
        >
          <Pencil className="size-4" aria-hidden />
          Edit department
        </DropdownMenuItem>
        <DropdownMenuSeparator />
        <DropdownMenuItem
          variant="destructive"
          disabled={!canManage || department.isDefault}
          onSelect={() => onDelete(department)}
        >
          <Trash2 className="size-4" aria-hidden />
          Delete department
        </DropdownMenuItem>
        {department.isDefault && (
          <p className="px-2 pb-1.5 pt-2 text-ui-xs leading-relaxed text-tl-faint">
            The default department cannot be deleted.
          </p>
        )}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
