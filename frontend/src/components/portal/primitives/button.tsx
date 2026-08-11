import { forwardRef } from "react";
import Link from "next/link";

import { cn } from "@/lib/utils";

/*
  The portal's action button.

  The staff panel repeats its button classes at every call site; the portal has
  enough of them to be worth naming once. Styling comes from the same tl-*
  tokens, so the two panels stay visually identical without sharing a component
  that neither owns.

  Not the shadcn Button: that one resolves against the neutral shadcn palette
  (`bg-primary` is near-black here), which would put an off-brand button next to
  every branded one.
*/

type Variant = "primary" | "secondary" | "ghost" | "danger";
type Size = "sm" | "md";

const VARIANTS: Record<Variant, string> = {
  primary: "bg-tl-blue text-white hover:bg-blue-700 focus-visible:ring-tl-blue/40",
  secondary:
    "border border-tl-line bg-tl-card text-tl-ink-soft hover:bg-tl-line-soft focus-visible:ring-tl-blue/30",
  ghost:
    "text-tl-muted hover:bg-tl-line-soft hover:text-tl-ink focus-visible:ring-tl-blue/30",
  danger:
    "border border-red-200 bg-white text-tl-red-ink hover:bg-tl-red-soft focus-visible:ring-tl-red/30",
};

const SIZES: Record<Size, string> = {
  sm: "h-8 gap-1.5 px-3 text-ui-sm",
  md: "h-10 gap-2 px-4 text-ui-base",
};

const BASE =
  "inline-flex shrink-0 items-center justify-center rounded-btn font-semibold transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 disabled:pointer-events-none disabled:opacity-50 [&_svg]:shrink-0";

export function actionClasses(
  variant: Variant = "primary",
  size: Size = "md",
  className?: string,
) {
  return cn(BASE, VARIANTS[variant], SIZES[size], className);
}

export interface ActionButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: Variant;
  size?: Size;
}

export const ActionButton = forwardRef<HTMLButtonElement, ActionButtonProps>(
  ({ className, variant = "primary", size = "md", type = "button", ...props }, ref) => (
    <button
      ref={ref}
      type={type}
      className={actionClasses(variant, size, className)}
      {...props}
    />
  ),
);
ActionButton.displayName = "ActionButton";

/** The same surface as a navigation target. */
export function ActionLink({
  href,
  className,
  variant = "primary",
  size = "md",
  ...props
}: React.ComponentProps<typeof Link> & { variant?: Variant; size?: Size }) {
  return (
    <Link
      href={href}
      className={actionClasses(variant, size, className)}
      {...props}
    />
  );
}
