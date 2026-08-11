"use client";

import { useState } from "react";
import { ArrowRight, Lock } from "lucide-react";

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
import {
  Field,
  SelectInput,
  TextInput,
} from "@/components/admin/primitives";
import { ApiError } from "@/lib/api/client";
import type { DepartmentInfo } from "@/lib/api/types";
import { useSetStaffDepartment } from "@/lib/admin/hooks";
import type { StaffMember } from "@/lib/admin/types";

/*
  Edit what this panel can actually write about a person.

  Name, email, role and status have no update route today, so they are shown
  read-only with a lock note. Department is the one field PUT /staff/{id}/department
  accepts — changing it moves them off the previous team in the same write,
  which the from → to line makes explicit before submit.
*/

export function EditStaffDialog({
  member,
  departments,
  onOpenChange,
}: {
  /** null closes the dialog. */
  member: StaffMember | null;
  departments: DepartmentInfo[];
  onOpenChange: (open: boolean) => void;
}) {
  const toast = useToast();
  const assign = useSetStaffDepartment();
  const [error, setError] = useState<string | null>(null);
  const [choice, setChoice] = useState(member?.department?.id ?? "");

  if (!member) return null;

  const target = departments.find((department) => department.id === choice);
  const unchanged = (member.department?.id ?? "") === choice;
  const leaving =
    member.department !== null &&
    choice !== "" &&
    member.department.id !== choice;
  const removing = member.department !== null && choice === "";

  async function submit() {
    if (!member || unchanged) return;
    setError(null);
    try {
      await assign.mutateAsync({
        userId: member.id,
        departmentId: choice === "" ? null : choice,
      });
      toast.success(
        choice === ""
          ? `${member.name} is no longer on a team.`
          : `${member.name} moved to ${target?.name}.`,
      );
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
      <DialogContent className="gap-0 p-0 font-ui sm:max-w-[480px]">
        <DialogHeader className="px-6 pt-6">
          <DialogTitle className="text-ui-xl font-bold tracking-[-0.02em] text-tl-ink">
            Edit staff
          </DialogTitle>
          <DialogDescription className="text-ui-md text-tl-muted">
            Update where {member.name} works. Profile fields without an API stay
            read-only.
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

          <div className="grid gap-3 sm:grid-cols-2">
            <Field label="Name" htmlFor="edit-staff-name">
              <TextInput
                id="edit-staff-name"
                value={member.name}
                disabled
                readOnly
              />
            </Field>
            <Field label="Email" htmlFor="edit-staff-email">
              <TextInput
                id="edit-staff-email"
                value={member.email}
                disabled
                readOnly
              />
            </Field>
            <Field label="Role" htmlFor="edit-staff-role">
              <TextInput
                id="edit-staff-role"
                value={member.role ?? "Not available from API"}
                disabled
                readOnly
              />
            </Field>
            <Field label="Status" htmlFor="edit-staff-status">
              <TextInput
                id="edit-staff-status"
                value={member.status}
                disabled
                readOnly
              />
            </Field>
          </div>

          <p className="flex items-start gap-1.5 text-ui-xs leading-relaxed text-tl-faint">
            <Lock className="mt-px size-3.5 shrink-0" aria-hidden />
            Name, email, role and status need update endpoints that are not
            registered yet. Only department can be changed here.
          </p>

          <Field label="Department" htmlFor="edit-staff-department">
            <SelectInput
              id="edit-staff-department"
              compact={false}
              value={choice}
              onChange={(event) => setChoice(event.target.value)}
            >
              <option value="">No department</option>
              {departments.map((department) => (
                <option key={department.id} value={department.id}>
                  {department.name}
                  {department.is_default ? " (default)" : ""}
                </option>
              ))}
            </SelectInput>
          </Field>

          {(leaving || removing) && (
            <div className="flex items-center gap-2 rounded-card border border-tl-orange/30 bg-tl-orange-soft/40 px-3.5 py-3 text-ui-sm text-tl-ink-soft">
              <span className="font-medium text-tl-ink">
                {member.department?.name}
              </span>
              <ArrowRight className="size-3.5 shrink-0 text-tl-faint" aria-hidden />
              <span className="font-medium text-tl-ink">
                {removing ? "Unassigned pool" : target?.name}
              </span>
              <span className="text-tl-faint">
                — they leave their current team when you save.
              </span>
            </div>
          )}

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
            onClick={() => void submit()}
            disabled={assign.isPending || unchanged}
            aria-busy={assign.isPending}
          >
            {assign.isPending && <Spinner className="size-4" />}
            {unchanged ? "No changes" : "Save"}
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
  return "Something went wrong saving this person.";
}
