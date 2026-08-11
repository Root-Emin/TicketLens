"use client";

import { useState } from "react";
import { ArrowRightLeft, UsersRound } from "lucide-react";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/shadcn/dialog";
import { ActionButton, FormError, useToast } from "@/components/portal/primitives";
import { Spinner } from "@/components/ui/spinner";
import { ApiError } from "@/lib/api/client";
import { useDeleteDepartment } from "@/lib/admin/hooks";
import type { DepartmentRow } from "@/lib/admin/types";

/*
  Deleting a department.

  The important thing this dialog does is say what actually happens, which is
  not "the tickets are deleted": DeleteDepartmentUseCase moves every ticket to
  the organization's default department first, because the foreign key has no ON
  DELETE action. An administrator who thinks they are archiving an empty shelf
  and is in fact re-routing 40 live tickets into General has been badly served by
  a confirmation dialog that said "are you sure?".

  So the count is named, the destination is named, and the button says what it
  does. The default department cannot be deleted at all and is refused before
  the dialog opens.
*/

export function DeleteDepartmentDialog({
  department,
  fallbackName,
  onOpenChange,
}: {
  /** null closes the dialog. */
  department: DepartmentRow | null;
  /** The organization's default department — where the tickets land. */
  fallbackName: string;
  onOpenChange: (open: boolean) => void;
}) {
  const toast = useToast();
  const remove = useDeleteDepartment();
  const [error, setError] = useState<string | null>(null);

  async function confirm() {
    if (!department) return;
    setError(null);
    try {
      await remove.mutateAsync(department.id);
      toast.success(`${department.name} deleted.`);
      onOpenChange(false);
    } catch (caught) {
      setError(describe(caught));
    }
  }

  function close(next: boolean) {
    onOpenChange(next);
    if (!next) setError(null);
  }

  const moving = department?.ticketCount ?? 0;
  const staff = department?.staffCount ?? 0;

  return (
    <Dialog open={Boolean(department)} onOpenChange={close}>
      <DialogContent className="gap-0 p-0 font-ui sm:max-w-[480px]">
        <DialogHeader className="px-6 pt-6">
          <DialogTitle className="text-ui-xl font-bold tracking-[-0.02em] text-tl-ink">
            Delete {department?.name}?
          </DialogTitle>
          <DialogDescription className="text-ui-md leading-relaxed text-tl-muted">
            The department is removed from the organization and stops receiving
            routed tickets. This cannot be undone.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3 px-6 pt-5">
          {moving > 0 ? (
            <div className="flex gap-3 rounded-card border border-tl-line bg-tl-orange-soft/70 px-4 py-3">
              <ArrowRightLeft
                className="mt-px size-[18px] shrink-0 text-tl-orange-ink"
                strokeWidth={1.9}
                aria-hidden
              />
              <p className="text-ui-sm leading-relaxed text-tl-ink-soft">
                <strong className="font-semibold">
                  {moving} {moving === 1 ? "ticket" : "tickets"}
                </strong>{" "}
                will be moved to <strong className="font-semibold">{fallbackName}</strong>{" "}
                first. They stay open and keep their assignees — only the
                department changes.
              </p>
            </div>
          ) : (
            <p className="text-ui-sm leading-relaxed text-tl-muted">
              No tickets have ever been routed here, so nothing moves.
            </p>
          )}

          {/*
            The other half of the consequence, and it is deliberately not the
            same as the tickets'. Staff are unassigned rather than moved — see
            DeleteDepartmentUseCase — so this says where they go, because
            "returned to the pool" and "silently added to General" are very
            different things to do to six people.
          */}
          {staff > 0 && (
            <div className="flex gap-3 rounded-card border border-tl-line bg-tl-line-soft/60 px-4 py-3">
              <UsersRound
                className="mt-px size-[18px] shrink-0 text-tl-muted"
                strokeWidth={1.9}
                aria-hidden
              />
              <p className="text-ui-sm leading-relaxed text-tl-ink-soft">
                <strong className="font-semibold">
                  {staff} {staff === 1 ? "person" : "people"}
                </strong>{" "}
                will be left without a department. Their accounts and their open
                tickets are untouched — you will need to place them on another
                team.
              </p>
            </div>
          )}

          {error && (
            <div className="mt-4">
              <FormError>{error}</FormError>
            </div>
          )}
        </div>

        <DialogFooter className="flex-col-reverse gap-2 px-6 pb-6 pt-6 sm:flex-row sm:justify-end">
          <ActionButton
            variant="secondary"
            onClick={() => close(false)}
            disabled={remove.isPending}
          >
            Cancel
          </ActionButton>
          <ActionButton
            variant="danger"
            onClick={confirm}
            disabled={remove.isPending}
            aria-busy={remove.isPending}
          >
            {remove.isPending && <Spinner className="size-4" />}
            {moving > 0 ? `Move ${moving} and delete` : "Delete department"}
          </ActionButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function describe(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 409) {
      return error.message || "The default department cannot be deleted.";
    }
    if (error.status === 403) {
      return "Your account cannot manage departments. This needs department:manage.";
    }
    if (error.status >= 500) {
      return "The server could not delete this. Please try again shortly.";
    }
    return error.message;
  }
  return "Something went wrong deleting this department.";
}
