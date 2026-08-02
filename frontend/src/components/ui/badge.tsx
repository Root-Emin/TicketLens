import { HTMLAttributes } from "react";
import { cn } from "@/lib/utils";

interface BadgeProps extends HTMLAttributes<HTMLSpanElement> {
  tone?: string;
}

/**
 * Badge is a small pill. Pass tone classes (bg/text/ring) for semantic coloring;
 * the ring-1 gives the subtle outline used throughout the triage UI.
 */
export function Badge({ className, tone, ...props }: BadgeProps) {
  return (
    <span
      className={cn(
        "inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-xs font-medium ring-1 ring-inset",
        tone ?? "bg-surface-muted text-muted-foreground ring-border",
        className,
      )}
      {...props}
    />
  );
}
