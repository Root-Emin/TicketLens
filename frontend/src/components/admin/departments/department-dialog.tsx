"use client";

import { useState } from "react";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

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
import {
  Field,
  FieldGroup,
  SelectInput,
  TextArea,
  TextInput,
} from "@/components/admin/primitives";
import { ApiError } from "@/lib/api/client";
import { ALL_CATEGORIES, CATEGORY_LABELS } from "@/lib/api/labels";
import type { Category } from "@/lib/api/types";
import { useCreateDepartment, useUpdateDepartment } from "@/lib/admin/hooks";
import type { DepartmentRow } from "@/lib/admin/types";

/*
  Creating and editing a department.

  A dialog rather than the sheet the staff detail uses, because this one is a
  form that is submitted and closed rather than a record that is read alongside
  the table. Same distinction the portal already draws: NewTicketDialog is a
  dialog, the staff details rail is not.

  The three fields are grouped into two, which is not padding — identity and
  routing are different decisions. Renaming a department is cosmetic; changing
  its category re-points the classifier, and every ticket predicted into that
  category afterwards lands somewhere new. The group description says so, at the
  moment somebody is about to do it.

  Category uniqueness (one department per category, per organization) is a
  backend constraint, so a clash comes back as a 409 and is rendered as the
  server's own message rather than pre-empted with a client-side check that
  could disagree with it.
*/

const MAX_NAME = 255;
const MAX_DESCRIPTION = 500;

const schema = z.object({
  name: z
    .string()
    .trim()
    .min(2, "Give the department a name of at least 2 characters")
    .max(MAX_NAME, `Keep the name under ${MAX_NAME} characters`),
  description: z
    .string()
    .trim()
    .max(MAX_DESCRIPTION, `Keep the description under ${MAX_DESCRIPTION} characters`),
  // "" is the documented way to clear a category on PATCH.
  category: z.string(),
});

type FormValues = z.infer<typeof schema>;

export function DepartmentDialog({
  open,
  department,
  takenCategories,
  onOpenChange,
}: {
  open: boolean;
  /** null creates; a row edits it. */
  department: DepartmentRow | null;
  /** Categories already claimed by another department, which cannot be reused. */
  takenCategories: Map<string, string>;
  onOpenChange: (open: boolean) => void;
}) {
  const toast = useToast();
  const create = useCreateDepartment();
  const update = useUpdateDepartment();
  const [serverError, setServerError] = useState<string | null>(null);

  const editing = department !== null;

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    // Keyed remount (see the caller) is what refills these when the row
    // changes; a defaultValues object alone would keep the first row's text.
    defaultValues: {
      name: department?.name ?? "",
      description: department?.description ?? "",
      category: department?.category ?? "",
    },
  });

  function close(next: boolean) {
    onOpenChange(next);
    if (!next) {
      reset();
      setServerError(null);
    }
  }

  async function onSubmit(values: FormValues) {
    setServerError(null);
    const input = {
      name: values.name,
      description: values.description,
      category: values.category === "" ? null : values.category,
    };

    try {
      if (editing) {
        await update.mutateAsync({ id: department.id, input });
        toast.success(`${values.name} updated.`);
      } else {
        await create.mutateAsync(input);
        toast.success(`${values.name} created.`);
      }
      close(false);
    } catch (error) {
      setServerError(describe(error, editing));
    }
  }

  const submitting = create.isPending || update.isPending;

  return (
    <Dialog open={open} onOpenChange={close}>
      <DialogContent className="max-h-[calc(100dvh-2rem)] gap-0 overflow-y-auto p-0 font-ui sm:max-w-[520px]">
        <DialogHeader className="px-6 pt-6">
          <DialogTitle className="text-ui-xl font-bold tracking-[-0.02em] text-tl-ink">
            {editing ? `Edit ${department.name}` : "New department"}
          </DialogTitle>
          <DialogDescription className="text-ui-md text-tl-muted">
            {editing
              ? "Renaming is cosmetic. Changing the category changes where the classifier sends new tickets."
              : "A department is a routing target: the classifier sends tickets to whichever one claims the predicted category."}
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} noValidate className="px-6 pb-6">
          <div className="mt-5 space-y-6">
            <FieldGroup title="Identity">
              <Field label="Name" htmlFor="department-name" error={errors.name?.message}>
                <TextInput
                  id="department-name"
                  maxLength={MAX_NAME}
                  placeholder="Payment Operations"
                  autoComplete="off"
                  invalid={Boolean(errors.name)}
                  aria-invalid={errors.name ? true : undefined}
                  aria-describedby={errors.name ? "department-name-error" : undefined}
                  {...register("name")}
                />
              </Field>

              <Field
                label="Description"
                htmlFor="department-description"
                optional
                error={errors.description?.message}
                hint="Shown to administrators only. Customers never see it."
              >
                <TextArea
                  id="department-description"
                  rows={3}
                  maxLength={MAX_DESCRIPTION}
                  placeholder="Refunds, chargebacks and billing disputes."
                  invalid={Boolean(errors.description)}
                  {...register("description")}
                />
              </Field>
            </FieldGroup>

            <FieldGroup
              title="Routing"
              description="At most one department per organization may claim a category. Leaving it unset means the classifier never routes here on its own — tickets arrive only when somebody moves them."
            >
              <Field label="Classifier category" htmlFor="department-category" optional>
                <SelectInput
                  id="department-category"
                  compact={false}
                  {...register("category")}
                >
                  <option value="">No category — manual routing only</option>
                  {ALL_CATEGORIES.map((category) => {
                    const takenBy = takenCategories.get(category);
                    const unavailable =
                      takenBy !== undefined && category !== department?.category;

                    return (
                      <option
                        key={category}
                        value={category}
                        disabled={unavailable}
                      >
                        {CATEGORY_LABELS[category as Category]}
                        {unavailable ? ` — taken by ${takenBy}` : ""}
                      </option>
                    );
                  })}
                </SelectInput>
              </Field>
            </FieldGroup>
          </div>

          {serverError && (
            <div className="mt-5">
              <FormError>{serverError}</FormError>
            </div>
          )}

          <DialogFooter className="mt-6 flex-col-reverse gap-2 sm:flex-row sm:justify-end">
            <ActionButton
              variant="secondary"
              onClick={() => close(false)}
              disabled={submitting}
            >
              Cancel
            </ActionButton>
            <ActionButton type="submit" disabled={submitting} aria-busy={submitting}>
              {submitting && <Spinner className="size-4" />}
              {editing ? "Save changes" : "Create department"}
            </ActionButton>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

/** Turns a failed write into something an administrator can act on. */
function describe(error: unknown, editing: boolean): string {
  if (error instanceof ApiError) {
    if (error.status === 409) {
      return (
        error.message ||
        "Another department already uses that name or category in this organization."
      );
    }
    if (error.status === 400 || error.status === 422) {
      return error.message || "Please check the name and category.";
    }
    if (error.status === 403) {
      return "Your account cannot manage departments. This needs department:manage.";
    }
    if (error.status >= 500) {
      return "The server could not save this. Please try again shortly.";
    }
    return error.message;
  }
  return editing
    ? "Something went wrong saving this department."
    : "Something went wrong creating this department.";
}
