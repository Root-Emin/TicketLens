"use client";

import { AlertCircle, Loader2, RotateCw, type LucideIcon } from "lucide-react";

import { cn } from "@/lib/utils";
import { ActionButton } from "./button";

/*
  Empty, error and loading, once.

  Every list in the portal can be in one of these three states, and writing them
  per screen is how they drift — one says "No tickets", the next renders a bare
  red string. Each takes an action so the state is a way forward rather than a
  dead end.
*/

export function EmptyState({
  icon: Icon,
  title,
  description,
  action,
  className,
}: {
  icon: LucideIcon;
  title: string;
  description: string;
  action?: React.ReactNode;
  className?: string;
}) {
  return (
    <div
      className={cn(
        "flex flex-col items-center justify-center px-6 py-16 text-center",
        className,
      )}
    >
      <span className="flex size-12 items-center justify-center rounded-full bg-tl-blue-soft text-tl-blue">
        <Icon className="size-6" strokeWidth={1.8} aria-hidden />
      </span>
      <h3 className="mt-4 text-ui-lg font-semibold tracking-[-0.01em] text-tl-ink">
        {title}
      </h3>
      <p className="mt-1.5 max-w-sm text-ui-md leading-relaxed text-tl-muted">
        {description}
      </p>
      {action && <div className="mt-5">{action}</div>}
    </div>
  );
}

/**
 * A failed request.
 *
 * `role="alert"` because this replaces content the reader was waiting for — a
 * screen reader user otherwise sits on a silent page.
 */
export function ErrorState({
  title = "Something went wrong",
  description = "We could not load this right now. Please try again.",
  onRetry,
  retrying,
  className,
}: {
  title?: string;
  description?: string;
  onRetry?: () => void;
  retrying?: boolean;
  className?: string;
}) {
  return (
    <div
      role="alert"
      className={cn(
        "flex flex-col items-center justify-center px-6 py-16 text-center",
        className,
      )}
    >
      <span className="flex size-12 items-center justify-center rounded-full bg-tl-red-soft text-tl-red-ink">
        <AlertCircle className="size-6" strokeWidth={1.8} aria-hidden />
      </span>
      <h3 className="mt-4 text-ui-lg font-semibold tracking-[-0.01em] text-tl-ink">
        {title}
      </h3>
      <p className="mt-1.5 max-w-sm text-ui-md leading-relaxed text-tl-muted">
        {description}
      </p>
      {onRetry && (
        <ActionButton
          variant="secondary"
          size="sm"
          onClick={onRetry}
          disabled={retrying}
          className="mt-5"
        >
          {retrying ? (
            <Loader2 className="size-3.5 animate-spin" aria-hidden />
          ) : (
            <RotateCw className="size-3.5" aria-hidden />
          )}
          Try again
        </ActionButton>
      )}
    </div>
  );
}

/** An inline field- or form-level failure message. */
export function FormError({ children }: { children: React.ReactNode }) {
  return (
    <p
      role="alert"
      className="flex items-start gap-2 rounded-btn bg-tl-red-soft px-3 py-2.5 text-ui-sm leading-relaxed text-tl-red-ink"
    >
      <AlertCircle className="mt-px size-4 shrink-0" aria-hidden />
      <span>{children}</span>
    </p>
  );
}
