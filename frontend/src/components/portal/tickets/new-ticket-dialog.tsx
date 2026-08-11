"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm, useWatch } from "react-hook-form";
import { Plus, Sparkles } from "lucide-react";
import { z } from "zod";

import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/shadcn/dialog";
import { ActionButton, FormError, useToast } from "@/components/portal/primitives";
import { Spinner } from "@/components/ui/spinner";
import { cn } from "@/lib/utils";
import { ApiError } from "@/lib/api/client";
import { useCreateTicket } from "@/lib/portal/hooks";

/*
  Raising a ticket.

  The customer supplies a subject and a description and nothing else. Priority,
  category and department are absent by design, not by omission: the classifier
  decides them (see docs/api-contract.md §4.3, where POST /tickets rejects both
  fields), and a form that let a customer set their own priority would make the
  routing meaningless — everything would arrive urgent.

  The note under the form says so plainly, because a field that silently isn't
  there reads as a missing feature rather than a deliberate one.
*/

const MAX_SUBJECT = 120;
const MAX_BODY = 4000;

const schema = z.object({
  subject: z
    .string()
    .trim()
    .min(3, "Give your request a short title (at least 3 characters)")
    .max(MAX_SUBJECT, `Keep the subject under ${MAX_SUBJECT} characters`),
  body: z
    .string()
    .trim()
    .min(10, "Tell us a little more — at least 10 characters")
    .max(MAX_BODY, `Keep the description under ${MAX_BODY} characters`),
});

type FormValues = z.infer<typeof schema>;

export function NewTicketDialog({
  trigger,
  className,
}: {
  /** Defaults to the standard "Create Ticket" button. */
  trigger?: React.ReactNode;
  className?: string;
}) {
  const [open, setOpen] = useState(false);
  const router = useRouter();
  const toast = useToast();
  const createTicket = useCreateTicket();
  const [serverError, setServerError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    control,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { subject: "", body: "" },
  });

  // useWatch rather than watch(): the subscription form re-renders only this
  // component, and it returns a value the compiler can memoize.
  const bodyLength = useWatch({ control, name: "body" })?.length ?? 0;

  // A dialog that reopens holding the last attempt's text and error is a
  // surprise. Clearing on close rather than in an effect means nothing flashes
  // and the reset is tied to the event that caused it.
  function onOpenChange(next: boolean) {
    setOpen(next);
    if (!next) {
      reset();
      setServerError(null);
    }
  }

  async function onSubmit(values: FormValues) {
    setServerError(null);
    try {
      const ticket = await createTicket.mutateAsync(values);
      onOpenChange(false);
      toast.success("Ticket created. We're routing it to the right team.");
      router.push(`/portal/tickets/${ticket.id}`);
    } catch (error) {
      setServerError(describe(error));
    }
  }

  const submitting = createTicket.isPending;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogTrigger asChild>
        {trigger ?? (
          <ActionButton className={className}>
            <Plus className="size-4" strokeWidth={2.2} aria-hidden />
            Create Ticket
          </ActionButton>
        )}
      </DialogTrigger>

      <DialogContent className="max-h-[calc(100dvh-2rem)] gap-0 overflow-y-auto p-0 font-ui sm:max-w-[540px]">
        <DialogHeader className="px-6 pt-6">
          <DialogTitle className="text-ui-xl font-bold tracking-[-0.02em] text-tl-ink">
            New request
          </DialogTitle>
          <DialogDescription className="text-ui-md text-tl-muted">
            Describe what you need. We&apos;ll take it from there.
          </DialogDescription>
        </DialogHeader>

        <form onSubmit={handleSubmit(onSubmit)} noValidate className="px-6 pb-6">
          <div className="mt-5 space-y-4">
            <Field
              label="Subject"
              htmlFor="ticket-subject"
              error={errors.subject?.message}
            >
              <input
                id="ticket-subject"
                maxLength={MAX_SUBJECT}
                placeholder="Payments are failing at checkout"
                aria-invalid={errors.subject ? true : undefined}
                aria-describedby={errors.subject ? "ticket-subject-error" : undefined}
                className={controlClass(Boolean(errors.subject), "h-11 px-3.5")}
                {...register("subject")}
              />
            </Field>

            <Field
              label="Description"
              htmlFor="ticket-body"
              error={errors.body?.message}
              hint={`${bodyLength}/${MAX_BODY}`}
            >
              <textarea
                id="ticket-body"
                rows={7}
                maxLength={MAX_BODY}
                placeholder="What happened, what you expected, and anything you have already tried."
                aria-invalid={errors.body ? true : undefined}
                aria-describedby={errors.body ? "ticket-body-error" : undefined}
                className={controlClass(
                  Boolean(errors.body),
                  "resize-y px-3.5 py-2.5 leading-relaxed",
                )}
                {...register("body")}
              />
            </Field>
          </div>

          <div className="mt-4 flex gap-3 rounded-card border border-tl-line bg-tl-blue-soft/60 px-4 py-3">
            <Sparkles
              className="mt-px size-[18px] shrink-0 text-tl-blue"
              strokeWidth={1.9}
              aria-hidden
            />
            <p className="text-ui-sm leading-relaxed text-tl-ink-soft">
              Your request will automatically be analyzed and routed by AI. You
              don&apos;t need to pick a category, priority or team — we work that
              out from what you write.
            </p>
          </div>

          {serverError && (
            <div className="mt-4">
              <FormError>{serverError}</FormError>
            </div>
          )}

          <DialogFooter className="mt-6 flex-col-reverse gap-2 sm:flex-row sm:justify-end">
            <ActionButton
              variant="secondary"
              onClick={() => onOpenChange(false)}
              disabled={submitting}
            >
              Cancel
            </ActionButton>
            <ActionButton type="submit" disabled={submitting} aria-busy={submitting}>
              {submitting && <Spinner className="size-4" />}
              {submitting ? "Sending…" : "Submit request"}
            </ActionButton>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

function controlClass(invalid: boolean, extra: string) {
  return cn(
    "w-full rounded-btn border bg-tl-card text-ui-md text-tl-ink outline-none transition-colors duration-150 placeholder:text-tl-faint focus:ring-2 disabled:opacity-60",
    invalid
      ? "border-red-300 focus:border-tl-red focus:ring-tl-red/15"
      : "border-tl-line focus:border-tl-blue focus:ring-tl-blue/15",
    extra,
  );
}

function Field({
  label,
  htmlFor,
  error,
  hint,
  children,
}: {
  label: string;
  htmlFor: string;
  error?: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className="mb-1.5 flex items-baseline justify-between gap-2">
        <label
          htmlFor={htmlFor}
          className="text-ui-sm font-medium text-tl-ink-soft"
        >
          {label}
        </label>
        {hint && <span className="text-ui-xs tabular-nums text-tl-faint">{hint}</span>}
      </div>
      {children}
      {error && (
        <p id={`${htmlFor}-error`} className="mt-1.5 text-ui-sm text-tl-red-ink">
          {error}
        </p>
      )}
    </div>
  );
}

/** Turns a failed create into something the customer can act on. */
function describe(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 400 || error.status === 422) {
      return error.message || "Please check the subject and description.";
    }
    if (error.status === 403) {
      return "Your account is not allowed to open tickets. Contact support.";
    }
    if (error.status >= 500) {
      return "We couldn't reach support just now. Please try again shortly.";
    }
    return error.message;
  }
  return "Something went wrong. Please try again.";
}
