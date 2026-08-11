"use client";

import { useMemo, useState } from "react";
import { UserPlus, UsersRound } from "lucide-react";

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
  SearchInput,
  StaffStatusBadge,
} from "@/components/admin/primitives";
import { ApiError } from "@/lib/api/client";
import { useSetStaffDepartment } from "@/lib/admin/hooks";
import type { StaffMember } from "@/lib/admin/types";
import { isActive } from "@/lib/admin/workforce";
import { cn } from "@/lib/utils";

/*
  Place people onto a department from the organization's roster.

  Candidates are everyone not already on this team. People already here are
  filtered out so the same assignment cannot be submitted twice. The only write
  is PUT /staff/{id}/department — one person at a time, run in sequence for a
  multi-select so a mid-batch failure leaves a clear partial result.
*/

type RosterFilter = "all" | "active" | "unassigned" | "other";

const FILTERS: { id: RosterFilter; label: string }[] = [
  { id: "all", label: "All" },
  { id: "active", label: "Active" },
  { id: "unassigned", label: "Unassigned" },
  { id: "other", label: "Other department" },
];

export function AddStaffDialog({
  open,
  departmentId,
  departmentName,
  staff,
  onOpenChange,
}: {
  open: boolean;
  departmentId: string;
  departmentName: string;
  /** Full org roster (including people already on this team). */
  staff: StaffMember[];
  onOpenChange: (open: boolean) => void;
}) {
  const toast = useToast();
  const assign = useSetStaffDepartment();

  const [query, setQuery] = useState("");
  const [filter, setFilter] = useState<RosterFilter>("all");
  const [selected, setSelected] = useState<Set<string>>(() => new Set());
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const candidates = useMemo(() => {
    const term = query.trim().toLowerCase();

    return staff
      .filter((member) => member.department?.id !== departmentId)
      .filter((member) => {
        if (filter === "active" && !isActive(member)) return false;
        if (filter === "unassigned" && member.department !== null) return false;
        if (filter === "other" && member.department === null) return false;
        if (term) {
          const haystack = `${member.name} ${member.email}`.toLowerCase();
          if (!haystack.includes(term)) return false;
        }
        return true;
      })
      .sort((a, b) => a.name.localeCompare(b.name));
  }, [staff, departmentId, filter, query]);

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  }

  function reset() {
    setQuery("");
    setFilter("all");
    setSelected(new Set());
    setError(null);
    setSubmitting(false);
  }

  function close(next: boolean) {
    onOpenChange(next);
    if (!next) reset();
  }

  async function submit() {
    if (selected.size === 0) return;
    setError(null);
    setSubmitting(true);

    const ids = [...selected];
    let done = 0;

    try {
      for (const userId of ids) {
        await assign.mutateAsync({ userId, departmentId });
        done += 1;
      }
      toast.success(
        done === 1
          ? `Added 1 person to ${departmentName}.`
          : `Added ${done} people to ${departmentName}.`,
      );
      close(false);
    } catch (caught) {
      setError(
        done > 0
          ? `${describe(caught)} ${done} of ${ids.length} were added before it failed.`
          : describe(caught),
      );
    } finally {
      setSubmitting(false);
    }
  }

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent className="gap-0 p-0 font-ui sm:max-w-[520px]">
        <DialogHeader className="px-6 pt-6">
          <DialogTitle className="text-ui-xl font-bold tracking-[-0.02em] text-tl-ink">
            Add staff
          </DialogTitle>
          <DialogDescription className="text-ui-md text-tl-muted">
            Choose people from this organization to place on {departmentName}.
            Anyone already on this team is hidden.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-3 px-6 pt-5">
          <SearchInput
            compact={false}
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Search staff…"
            aria-label="Search staff"
          />

          <div className="flex flex-wrap gap-1.5" role="tablist" aria-label="Filter staff">
            {FILTERS.map((item) => (
              <button
                key={item.id}
                type="button"
                role="tab"
                aria-selected={filter === item.id}
                onClick={() => setFilter(item.id)}
                className={cn(
                  "h-8 rounded-btn px-3 text-ui-sm font-semibold transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30",
                  filter === item.id
                    ? "bg-tl-blue text-white"
                    : "bg-tl-line-soft text-tl-muted hover:text-tl-ink",
                )}
              >
                {item.label}
              </button>
            ))}
          </div>

          <div className="max-h-[320px] overflow-y-auto rounded-card border border-tl-line">
            {candidates.length === 0 ? (
              <div className="flex flex-col items-center px-4 py-10 text-center">
                <UsersRound className="size-5 text-tl-faint-soft" aria-hidden />
                <p className="mt-2 text-ui-sm font-medium text-tl-ink">
                  Nobody to add
                </p>
                <p className="mt-1 max-w-xs text-ui-sm text-tl-faint">
                  {staff.every((m) => m.department?.id === departmentId)
                    ? "Everyone on the roster is already on this team."
                    : "No one matches this search or filter."}
                </p>
              </div>
            ) : (
              <ul className="divide-y divide-tl-line-soft">
                {candidates.map((member) => {
                  const checked = selected.has(member.id);
                  return (
                    <li key={member.id}>
                      <label
                        className={cn(
                          "flex cursor-pointer items-center gap-3 px-3.5 py-3 transition-colors hover:bg-tl-line-soft/50",
                          checked && "bg-tl-blue-soft/40",
                        )}
                      >
                        <input
                          type="checkbox"
                          className="size-4 rounded border-tl-line text-tl-blue focus:ring-tl-blue/30"
                          checked={checked}
                          onChange={() => toggle(member.id)}
                        />
                        <InitialsAvatar
                          name={member.name}
                          initials={member.initials}
                          size={32}
                          className="shrink-0"
                        />
                        <div className="min-w-0 flex-1">
                          <div className="truncate text-ui-md font-semibold text-tl-ink">
                            {member.name}
                          </div>
                          <div className="truncate text-ui-sm text-tl-faint">
                            {member.email}
                          </div>
                        </div>
                        <div className="hidden shrink-0 flex-col items-end gap-1 sm:flex">
                          <StaffStatusBadge status={member.status} />
                          <span className="text-ui-xs text-tl-faint">
                            {member.department?.name ?? "Unassigned"}
                          </span>
                        </div>
                      </label>
                    </li>
                  );
                })}
              </ul>
            )}
          </div>

          {selected.size > 0 && (
            <p className="text-ui-sm text-tl-muted">
              {selected.size} selected
            </p>
          )}

          {error && <FormError>{error}</FormError>}
        </div>

        <DialogFooter className="flex-col-reverse gap-2 px-6 pb-6 pt-5 sm:flex-row sm:justify-end">
          <ActionButton
            variant="secondary"
            onClick={() => close(false)}
            disabled={submitting}
          >
            Cancel
          </ActionButton>
          <ActionButton
            onClick={() => void submit()}
            disabled={submitting || selected.size === 0}
            aria-busy={submitting}
          >
            {submitting ? (
              <Spinner className="size-4" />
            ) : (
              <UserPlus className="size-4" aria-hidden />
            )}
            Add to department
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
  return "Something went wrong adding staff.";
}
