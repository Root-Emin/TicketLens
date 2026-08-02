import { cn } from "@/lib/utils";
import { pct } from "@/lib/utils";

/*
  A confidence meter.

  The amber tint means "the backend flagged this analysis for human review", and
  it comes from the analysis itself (needs_human_review) rather than from a
  threshold compared here. The cutoff is a backend setting
  (CLASSIFIER_REVIEW_THRESHOLD) that the API does not publish, so re-deriving it
  in the browser would mean the bar quietly disagrees with the routing decision
  the moment that setting changes.

  `threshold` stays available for the marker line, but has no default: we draw
  the cutoff only when a caller can supply the real one.
*/
export function ConfidenceBar({
  label,
  value,
  flagged = false,
  threshold,
}: {
  label: string;
  value: number;
  flagged?: boolean;
  threshold?: number;
}) {
  return (
    <div>
      <div className="mb-1 flex items-center justify-between text-xs">
        <span className="text-muted-foreground">{label}</span>
        <span
          className={cn(
            "font-mono font-medium",
            flagged ? "text-amber-600 dark:text-amber-400" : "text-foreground",
          )}
        >
          {pct(value)}
        </span>
      </div>
      <div className="relative h-2 overflow-hidden rounded-full bg-surface-muted">
        <div
          className={cn(
            "h-full rounded-full transition-all",
            flagged ? "bg-amber-500" : "bg-accent",
          )}
          style={{ width: `${Math.max(2, Math.min(100, value * 100))}%` }}
        />
        {threshold !== undefined && threshold > 0 && threshold < 1 && (
          /* Threshold marker — only drawn when the real cutoff is known. The
             range check keeps a missing or zeroed API field from pinning the
             line to the left edge as if the cutoff were actually zero. */
          <div
            className="absolute top-0 h-full w-px bg-foreground/40"
            style={{ left: `${threshold * 100}%` }}
            title={`Review threshold ${pct(threshold)}`}
          />
        )}
      </div>
    </div>
  );
}
