import { cn } from "@/lib/utils";

/*
  The white surface every section sits on.

  One hairline and a soft shadow, never a border inside a border — the previous
  build nested three of them around the composer. Panels that contain their own
  sections use PanelSection, which separates with a single divider rather than
  wrapping each child in another box.
*/

export function Panel({
  className,
  children,
  ...props
}: React.ComponentProps<"section">) {
  return (
    <section
      className={cn(
        "rounded-card border border-tl-line bg-tl-card shadow-panel",
        className,
      )}
      {...props}
    >
      {children}
    </section>
  );
}

export function PanelHeader({
  title,
  count,
  action,
  className,
}: {
  title: React.ReactNode;
  count?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("flex items-center gap-2 px-5 py-4", className)}>
      <h2 className="text-ui-lg font-semibold tracking-[-0.01em] text-tl-ink">
        {title}
      </h2>
      {count}
      {action && <div className="ml-auto flex items-center gap-1">{action}</div>}
    </div>
  );
}

/** A divided region inside a Panel. */
export function PanelSection({
  className,
  children,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      className={cn("border-t border-tl-line-soft px-5 py-4", className)}
      {...props}
    >
      {children}
    </div>
  );
}

/** Small uppercase caption used above grouped fields in the details rail. */
export function FieldGroupLabel({ children }: { children: React.ReactNode }) {
  return (
    <h3 className="mb-3 text-ui-xs font-semibold uppercase tracking-[0.06em] text-tl-faint">
      {children}
    </h3>
  );
}
