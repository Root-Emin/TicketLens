import { cn } from "@/lib/utils";

/*
  The title block every portal screen opens with.

  One h1 per page lives here, which is also what keeps the heading order sane:
  the topbar carries no heading of its own, so the page's own title is the
  document's first.
*/
export function PageHeader({
  title,
  description,
  action,
  className,
}: {
  title: React.ReactNode;
  description?: React.ReactNode;
  action?: React.ReactNode;
  className?: string;
}) {
  return (
    <header
      className={cn(
        "flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between",
        className,
      )}
    >
      <div className="min-w-0">
        <h1 className="text-ui-2xl font-bold tracking-[-0.02em] text-tl-ink">
          {title}
        </h1>
        {description && (
          <p className="mt-1 text-ui-md text-tl-muted">{description}</p>
        )}
      </div>
      {action && <div className="shrink-0">{action}</div>}
    </header>
  );
}
