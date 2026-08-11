"use client";

import { useState } from "react";
import { UserRoundMinus } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/shadcn/dialog";
import { ActionButton, FormError, useToast } from "@/components/portal/primitives";
import { InitialsAvatar } from "@/components/staff/primitives";
import { Spinner } from "@/components/ui/spinner";
import { ApiError } from "@/lib/api/client";
import { useSetStaffDepartment } from "@/lib/admin/hooks";
import type { StaffMember } from "@/lib/admin/types";

/*
  Take somebody off this team.

  Same PUT as a move-to-null in AssignDepartmentDialog, but framed as a
  confirmation against the department they are leaving — managers open this
  from a department's staff table, not from a person-centric move dialog.
*/

export function RemoveStaffDialog({
  member,
  departmentName,
  onOpenChange,
}: {
  /** null closes the dialog. */
  member: StaffMember | null;
  departmentName: string;
  onOpenChange: (open: boolean) => void;
}) {
  const toast = useToast();
  const assign = useSetStaffDepartment();
  const [error, setError] = useState<string | null>(null);

  if (!member) return null;

  async function submit() {
    if (!member) return;
    setError(null);
    try {
      await assign.mutateAsync({
        userId: member.id,
        departmentId: null,
      });
      toast.success(`${member.name} removed from ${departmentName}.`);
      onOpenChange(false);
    } catch (caught) {
      setError(describe(caught));
    }
  }

  function close(next: boolean) {
    onOpenChange(next);
    if (!next) setError(null);
  }

  return (
    <Dialog open onOpenChange={close}>
      <DialogContent className="gap-0 p-0 font-ui sm:max-w-[440px]">
        <DialogHeader className="px-6 pt-6">
          <DialogTitle className="text-ui-xl font-bold tracking-[-0.02em] text-tl-ink">
            Remove from department
          </DialogTitle>
          <DialogDescription className="text-ui-md text-tl-muted">
            {member.name} will leave {departmentName} and return to the
            unassigned pool. Their open tickets stay with them.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 px-6 pt-5">
          <div className="flex items-center gap-3 rounded-card border border-tl-line bg-tl-line-soft/50 px-3.5 py-3">
            <InitialsAvatar
              name={member.name}
              initials={member.initials}
              size={36}
            />
            <div className="min-w-0">
              <div className="truncate text-ui-md font-semibold text-tl-ink">
                {member.name}
              </div>
              <div className="truncate text-ui-sm text-tl-muted">
                {member.email}
              </div>
            </div>
          </div>

          {error && <FormError>{error}</FormError>}
        </div>

        <DialogFooter className="flex-col-reverse gap-2 px-6 pb-6 pt-6 sm:flex-row sm:justify-end">
          <ActionButton
            variant="secondary"
            onClick={() => close(false)}
            disabled={assign.isPending}
          >
            Cancel
          </ActionButton>
          <ActionButton
            variant="danger"
            onClick={() => void submit()}
            disabled={assign.isPending}
            aria-busy={assign.isPending}
          >
            {assign.isPending ? (
              <Spinner className="size-4" />
            ) : (
              <UserRoundMinus className="size-4" aria-hidden />
            )}
            Remove from department
          </ActionButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function describe(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 403) {
      return "Your account cannot move staff. This needs user:write.";
    }
    if (error.status === 404) {
      return "That person is no longer on this organization's roster.";
    }
    if (error.status >= 500) {
      return "The server could not save this. Please try again shortly.";
    }
    return error.message;
  }
  return "Something went wrong removing this person.";
}
