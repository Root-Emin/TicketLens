"use client";

import { useState } from "react";
import { ArrowRight, Building2 } from "lucide-react";

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
import { Field, SelectInput } from "@/components/admin/primitives";
import { ApiError } from "@/lib/api/client";
import type { DepartmentInfo } from "@/lib/api/types";
import { useSetStaffDepartment } from "@/lib/admin/hooks";
import type { StaffMember } from "@/lib/admin/types";

/*
  Moving somebody between teams.

  One field, so a dialog rather than a sheet: this is a decision, taken and
  closed, not a record to read alongside the table.

  The two things it shows beyond the select are both about consequences. The
  from → to line makes the change legible before it is made, which matters
  because the control is a dropdown whose current value is easy to misread. And
  the note about tickets says what does *not* happen: their open tickets stay
  with them, because assignment is per ticket and moving a person does not
  re-route their work. Somebody expecting a handover needs to know they still
  have to do it.

  "No department" is a real option, not an empty state. Taking somebody off a
  team is how you park an account whose owner has left the support org without
  deactivating it — which is just as well, since there is no endpoint to
  deactivate one.
*/

export function AssignDepartmentDialog({
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

  // Keyed remount by the caller resets this when a different person is opened;
  // without that the previous member's choice would be preselected.
  const [choice, setChoice] = useState(member?.department?.id ?? "");

  if (!member) return null;

  const target = departments.find((department) => department.id === choice);
  const unchanged = (member.department?.id ?? "") === choice;

  async function submit() {
    if (!member) return;
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
      <DialogContent className="gap-0 p-0 font-ui sm:max-w-[460px]">
        <DialogHeader className="px-6 pt-6">
          <DialogTitle className="text-ui-xl font-bold tracking-[-0.02em] text-tl-ink">
            Change department
          </DialogTitle>
          <DialogDescription className="text-ui-md text-tl-muted">
            Choose the team {member.name} works on.
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
              <div className="flex min-w-0 items-center gap-1.5 text-ui-sm text-tl-muted">
                <span className="truncate">
                  {member.department?.name ?? "No department"}
                </span>
                {!unchanged && (
                  <>
                    <ArrowRight className="size-3.5 shrink-0" aria-hidden />
                    <span className="truncate font-medium text-tl-ink">
                      {target?.name ?? "No department"}
                    </span>
                  </>
                )}
              </div>
            </div>
          </div>

          <Field label="Department" htmlFor="assign-department">
            <SelectInput
              id="assign-department"
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

          <div className="flex gap-3 rounded-card border border-tl-line bg-tl-blue-soft/60 px-4 py-3">
            <Building2
              className="mt-px size-[18px] shrink-0 text-tl-blue"
              strokeWidth={1.9}
              aria-hidden
            />
            <p className="text-ui-sm leading-relaxed text-tl-ink-soft">
              Their open tickets stay assigned to them. A department decides
              where new work is routed, not who holds the work already out.
            </p>
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
            onClick={submit}
            disabled={assign.isPending || unchanged}
            aria-busy={assign.isPending}
          >
            {assign.isPending && <Spinner className="size-4" />}
            {unchanged ? "No change" : "Move"}
          </ActionButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function describe(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 403) {
      return "Your account cannot move staff between teams. This needs user:write.";
    }
    if (error.status === 404) {
      return "That person is no longer on this organization's roster.";
    }
    if (error.status >= 500) {
      return "The server could not save this. Please try again shortly.";
    }
    return error.message;
  }
  return "Something went wrong moving this person.";
}
