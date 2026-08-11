"use client";

import { forwardRef, useId, useState } from "react";
import { Eye, EyeOff } from "lucide-react";

import { cn } from "@/lib/utils";

/*
  Form fields for the auth screens.

  Wired for react-hook-form: each one forwards its ref and spreads the rest of
  the props, so `{...register("email")}` works unchanged. The error is passed
  in rather than read from a context — these are used on two forms with
  different schemas.

  Validation state is announced, not just coloured: `aria-invalid` plus
  `aria-describedby` pointing at the message is what makes a failed submit
  audible to a screen reader.
*/

const CONTROL =
  "h-11 w-full rounded-btn border bg-tl-card px-3.5 text-ui-md text-tl-ink outline-none transition-colors duration-150 placeholder:text-tl-faint focus:ring-2 disabled:opacity-60";

const CONTROL_TONE = {
  normal: "border-tl-line focus:border-tl-blue focus:ring-tl-blue/15",
  invalid: "border-red-300 focus:border-tl-red focus:ring-tl-red/15",
};

export interface FieldProps
  extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string;
  error?: string;
}

export const TextField = forwardRef<HTMLInputElement, FieldProps>(
  ({ label, error, className, id, ...props }, ref) => {
    const generatedId = useId();
    const fieldId = id ?? generatedId;
    const errorId = `${fieldId}-error`;

    return (
      <div className={className}>
        <label
          htmlFor={fieldId}
          className="mb-1.5 block text-ui-sm font-medium text-tl-ink-soft"
        >
          {label}
        </label>
        <input
          ref={ref}
          id={fieldId}
          aria-invalid={error ? true : undefined}
          aria-describedby={error ? errorId : undefined}
          className={cn(CONTROL, error ? CONTROL_TONE.invalid : CONTROL_TONE.normal)}
          {...props}
        />
        <FieldError id={errorId} message={error} />
      </div>
    );
  },
);
TextField.displayName = "TextField";

/**
 * A password field with a reveal toggle.
 *
 * The toggle is a button, not a checkbox, and it swaps its own accessible name
 * with its state — "Show password" / "Hide password" — so the control tells you
 * what it will do rather than what it currently is.
 */
export const PasswordField = forwardRef<HTMLInputElement, FieldProps>(
  ({ label, error, className, id, ...props }, ref) => {
    const generatedId = useId();
    const fieldId = id ?? generatedId;
    const errorId = `${fieldId}-error`;
    const [visible, setVisible] = useState(false);
    const Icon = visible ? EyeOff : Eye;

    return (
      <div className={className}>
        <label
          htmlFor={fieldId}
          className="mb-1.5 block text-ui-sm font-medium text-tl-ink-soft"
        >
          {label}
        </label>
        <div className="relative">
          <input
            ref={ref}
            id={fieldId}
            type={visible ? "text" : "password"}
            aria-invalid={error ? true : undefined}
            aria-describedby={error ? errorId : undefined}
            className={cn(
              CONTROL,
              "pr-11",
              error ? CONTROL_TONE.invalid : CONTROL_TONE.normal,
            )}
            {...props}
          />
          <button
            type="button"
            onClick={() => setVisible((current) => !current)}
            aria-label={visible ? "Hide password" : "Show password"}
            className="absolute right-1 top-1/2 inline-flex size-9 -translate-y-1/2 items-center justify-center rounded-md text-tl-faint transition-colors duration-150 hover:bg-tl-line-soft hover:text-tl-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
          >
            <Icon className="size-[18px]" strokeWidth={1.8} aria-hidden />
          </button>
        </div>
        <FieldError id={errorId} message={error} />
      </div>
    );
  },
);
PasswordField.displayName = "PasswordField";

function FieldError({ id, message }: { id: string; message?: string }) {
  if (!message) return null;
  return (
    <p id={id} className="mt-1.5 text-ui-sm text-tl-red-ink">
      {message}
    </p>
  );
}

/** A labelled checkbox sized for the auth forms. */
export function CheckboxField({
  label,
  id,
  ...props
}: React.InputHTMLAttributes<HTMLInputElement> & { label: string }) {
  const generatedId = useId();
  const fieldId = id ?? generatedId;

  return (
    <label
      htmlFor={fieldId}
      className="inline-flex cursor-pointer items-center gap-2 text-ui-sm text-tl-ink-soft select-none"
    >
      <input
        id={fieldId}
        type="checkbox"
        className="size-4 rounded border-tl-line text-tl-blue accent-tl-blue focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
        {...props}
      />
      {label}
    </label>
  );
}
