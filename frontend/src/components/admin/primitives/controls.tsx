"use client";

import { forwardRef } from "react";
import { ChevronDown, Search } from "lucide-react";

import { cn } from "@/lib/utils";

/*
  Form controls for the filter row and the department form.

  Sized to sit beside the portal's `sm` ActionButton (h-8) in a toolbar and its
  `md` (h-10) in a form, so a row of mixed controls lines up without any call
  site nudging a margin.

  Native <select> rather than a Radix listbox. Three reasons, in order: it is the
  only control that becomes a platform picker on a phone, which is where a
  filter row is hardest to use; it needs no portal, so it cannot escape the
  toolbar's stacking context; and a filter with six options does not need
  typeahead, sections or checkboxes. The panel's one genuinely rich menu — the
  row actions — is Radix, because that one does.
*/

const FIELD_BASE =
  "w-full rounded-btn border bg-tl-card text-tl-ink outline-none transition-colors duration-150 placeholder:text-tl-faint focus:ring-2 disabled:cursor-not-allowed disabled:opacity-60";

const FIELD_TONE = {
  normal: "border-tl-line focus:border-tl-blue focus:ring-tl-blue/15",
  invalid: "border-red-300 focus:border-tl-red focus:ring-tl-red/15",
} as const;

export function fieldClass(invalid?: boolean, extra?: string) {
  return cn(FIELD_BASE, FIELD_TONE[invalid ? "invalid" : "normal"], extra);
}

/** A search box with its magnifier. `type="search"` gives the native clear affordance. */
export const SearchInput = forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement> & { compact?: boolean }
>(({ className, compact = true, ...props }, ref) => (
  <div className={cn("relative", className)}>
    <Search
      className="pointer-events-none absolute left-3 top-1/2 size-4 -translate-y-1/2 text-tl-faint"
      aria-hidden
    />
    <input
      ref={ref}
      type="search"
      className={fieldClass(
        false,
        cn("pl-9 pr-3", compact ? "h-9 text-ui-sm" : "h-10 text-ui-md"),
      )}
      {...props}
    />
  </div>
));
SearchInput.displayName = "SearchInput";

export const TextInput = forwardRef<
  HTMLInputElement,
  React.InputHTMLAttributes<HTMLInputElement> & { invalid?: boolean }
>(({ className, invalid, ...props }, ref) => (
  <input
    ref={ref}
    className={fieldClass(invalid, cn("h-10 px-3.5 text-ui-md", className))}
    {...props}
  />
));
TextInput.displayName = "TextInput";

export const TextArea = forwardRef<
  HTMLTextAreaElement,
  React.TextareaHTMLAttributes<HTMLTextAreaElement> & { invalid?: boolean }
>(({ className, invalid, ...props }, ref) => (
  <textarea
    ref={ref}
    className={fieldClass(
      invalid,
      cn("resize-y px-3.5 py-2.5 text-ui-md leading-relaxed", className),
    )}
    {...props}
  />
));
TextArea.displayName = "TextArea";

/**
 * A select with its own chevron.
 *
 * `appearance-none` plus a drawn chevron, because the platform arrow is a
 * different colour and weight in every browser and this control sits in a row
 * of four. The chevron is `pointer-events-none` so clicks still reach the
 * select underneath.
 */
export const SelectInput = forwardRef<
  HTMLSelectElement,
  React.SelectHTMLAttributes<HTMLSelectElement> & {
    invalid?: boolean;
    compact?: boolean;
  }
>(({ className, invalid, compact = true, children, ...props }, ref) => (
  <div className={cn("relative", className)}>
    <select
      ref={ref}
      className={fieldClass(
        invalid,
        cn(
          "cursor-pointer appearance-none pl-3 pr-8",
          compact ? "h-9 text-ui-sm" : "h-10 text-ui-md",
        ),
      )}
      {...props}
    >
      {children}
    </select>
    <ChevronDown
      className="pointer-events-none absolute right-2.5 top-1/2 size-4 -translate-y-1/2 text-tl-faint"
      aria-hidden
    />
  </div>
));
SelectInput.displayName = "SelectInput";

/**
 * A labelled field for the forms.
 *
 * The label is always rendered and always tied to the control by id — a
 * placeholder is not a label, and the department form's fields are the kind
 * somebody fills in once a quarter and needs told what they are.
 */
export function Field({
  label,
  htmlFor,
  error,
  hint,
  optional,
  children,
}: {
  label: string;
  htmlFor: string;
  error?: string;
  hint?: string;
  optional?: boolean;
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
          {optional && (
            <span className="ml-1.5 font-normal text-tl-faint">Optional</span>
          )}
        </label>
      </div>
      {children}
      {hint && !error && (
        <p className="mt-1.5 text-ui-xs leading-relaxed text-tl-faint">{hint}</p>
      )}
      {error && (
        <p id={`${htmlFor}-error`} className="mt-1.5 text-ui-sm text-tl-red-ink">
          {error}
        </p>
      )}
    </div>
  );
}

/**
 * A titled group of fields inside a form.
 *
 * The forms in this panel are short but not flat: a staff record is an identity,
 * an access level, a team and an availability, and those are four different
 * questions with four different consequences. Grouping them is what stops the
 * dialog reading as a settings dump.
 */
export function FieldGroup({
  title,
  description,
  children,
}: {
  title: string;
  description?: string;
  children: React.ReactNode;
}) {
  return (
    <fieldset className="min-w-0">
      <legend className="text-ui-xs font-semibold uppercase tracking-[0.06em] text-tl-faint">
        {title}
      </legend>
      {description && (
        <p className="mt-1 text-ui-xs leading-relaxed text-tl-muted">
          {description}
        </p>
      )}
      <div className="mt-3 space-y-3.5">{children}</div>
    </fieldset>
  );
}
