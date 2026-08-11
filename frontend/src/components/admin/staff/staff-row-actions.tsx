"use client";

import Link from "next/link";
import {
  Building2,
  Copy,
  Inbox,
  MoreHorizontal,
  Pencil,
  ShieldCheck,
  UserRound,
  UserRoundMinus,
} from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/shadcn/dropdown-menu";
import { useToast } from "@/components/portal/primitives";
import type { StaffMember } from "@/lib/admin/types";

/*
  Per-row actions.

  Split into what the backend can do and what it cannot, with a line saying so
  rather than a menu that looks complete and fails on click.

  Available, because a route exists:
    View profile
    Edit staff              — department is writable; identity fields are not
    Change / move department — PUT /staff/{id}/department
    Remove from department  — same PUT with null, confirmation elsewhere
    View assigned tickets
    Copy email address

  Unavailable:
    Change role   — no role list endpoint
    Deactivate    — no update-user route registered
*/

export function StaffRowActions({
  member,
  canAssign,
  onViewProfile,
  onChangeDepartment,
  onEditStaff,
  onRemoveFromDepartment,
}: {
  member: StaffMember;
  /** Whether this session probably holds user:write. */
  canAssign: boolean;
  onViewProfile: () => void;
  onChangeDepartment: () => void;
  /** Opens the edit dialog when provided. */
  onEditStaff?: () => void;
  /**
   * When set, the menu offers "Remove from department" instead of only a
   * generic move — used on a department detail page.
   */
  onRemoveFromDepartment?: () => void;
}) {
  const toast = useToast();

  async function copyEmail() {
    try {
      await navigator.clipboard.writeText(member.email);
      toast.success(`Copied ${member.email}`);
    } catch {
      toast.error("Could not copy — your browser blocked clipboard access.");
    }
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`Actions for ${member.name}`}
          className="tap-target inline-flex size-8 items-center justify-center rounded-btn text-tl-faint transition-colors duration-150 hover:bg-tl-line-soft hover:text-tl-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
        >
          <MoreHorizontal className="size-4" aria-hidden />
        </button>
      </DropdownMenuTrigger>

      <DropdownMenuContent align="end" sideOffset={6} className="w-[268px]">
        <DropdownMenuLabel className="truncate text-ui-sm font-semibold text-tl-ink">
          {member.name}
        </DropdownMenuLabel>
        <DropdownMenuSeparator />

        <DropdownMenuItem onSelect={onViewProfile}>
          <UserRound className="size-4" aria-hidden />
          View profile
        </DropdownMenuItem>

        {onEditStaff && (
          <DropdownMenuItem disabled={!canAssign} onSelect={onEditStaff}>
            <Pencil className="size-4" aria-hidden />
            Edit staff
          </DropdownMenuItem>
        )}

        <DropdownMenuItem disabled>
          <ShieldCheck className="size-4" aria-hidden />
          Change role
        </DropdownMenuItem>

        <DropdownMenuItem disabled={!canAssign} onSelect={onChangeDepartment}>
          <Building2 className="size-4" aria-hidden />
          {onRemoveFromDepartment
            ? "Move to another department"
            : member.department
              ? "Change department"
              : "Assign to department"}
        </DropdownMenuItem>

        <DropdownMenuItem asChild>
          <Link href={`/tickets?assignee_id=${member.id}`}>
            <Inbox className="size-4" aria-hidden />
            View assigned tickets
          </Link>
        </DropdownMenuItem>

        <DropdownMenuItem onSelect={copyEmail}>
          <Copy className="size-4" aria-hidden />
          Copy email address
        </DropdownMenuItem>

        <DropdownMenuSeparator />

        {onRemoveFromDepartment && member.department && (
          <DropdownMenuItem
            variant="destructive"
            disabled={!canAssign}
            onSelect={onRemoveFromDepartment}
          >
            <UserRoundMinus className="size-4" aria-hidden />
            Remove from department
          </DropdownMenuItem>
        )}

        <DropdownMenuItem disabled>
          <UserRoundMinus className="size-4" aria-hidden />
          Deactivate account
        </DropdownMenuItem>

        <p className="px-2 pb-1.5 pt-2 text-ui-xs leading-relaxed text-tl-faint">
          {canAssign
            ? "Roles and account status need API endpoints that don't exist yet."
            : "Moving somebody between teams needs user:write, which this account doesn't hold."}
        </p>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
