import { Chip, DOT } from "@/components/staff/primitives";
import type { Accent } from "@/lib/staff/types";
import type { LoadBand } from "@/lib/admin/types";
import { cn } from "@/lib/utils";

/*
  The two status vocabularies this panel adds.

  Both go through the staff panel's Chip and its TINT/DOT maps rather than
  inventing colours, so an amber pill means the same thing on /team as it does
  on an agent's queue.
*/

/**
 * An account's status, straight from `user.status`.
 *
 * Rendered from the raw string rather than a closed union: the column is
 * VARCHAR(50) with no CHECK constraint (migration 00002), so a value this
 * frontend has not seen is a real possibility and showing it verbatim beats
 * showing "Unknown" for something the database is perfectly clear about.
 *
 * Note what this is not. It is an account status — whether the person can sign
 * in — not a presence or an availability. There is no last_seen_at anywhere in
 * the schema and no away/busy flag, so an "Away" pill here would be decoration.
 * What the table can honestly say about availability is how recently somebody
 * touched a ticket, and that is its own column.
 */
const STATUS_ACCENT: Record<string, Accent> = {
  active: "green",
  inactive: "neutral",
  suspended: "red",
  pending: "orange",
  invited: "orange",
};

export function StaffStatusBadge({
  status,
  className,
}: {
  status: string;
  className?: string;
}) {
  const accent = STATUS_ACCENT[status.toLowerCase()] ?? "neutral";
  return (
    <Chip accent={accent} dot className={className}>
      {titleCase(status)}
    </Chip>
  );
}

function titleCase(value: string): string {
  if (!value) return "Unknown";
  return value
    .split(/[_\s]+/)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}

const BAND_ACCENT: Record<LoadBand, Accent> = {
  none: "neutral",
  light: "green",
  steady: "blue",
  heavy: "orange",
};

const BAND_LABEL: Record<LoadBand, string> = {
  none: "No open tickets",
  light: "Light load",
  steady: "Steady load",
  heavy: "Heaviest on the team",
};

/**
 * Open tickets as a bar, measured against the busiest person on the team.
 *
 * Not against a capacity — there is no capacity in the schema, and drawing one
 * would be the panel's most quietly wrong number. The bar answers "who is
 * carrying the most", which is the question a manager scanning this column is
 * actually asking, and the tooltip text says so.
 *
 * A count of zero still draws the track, so an idle row reads as an empty bar
 * rather than as a missing cell.
 */
export function LoadBar({
  open,
  busiest,
  band,
  className,
  /**
   * The trailing count. Off where the surrounding layout already states the
   * number — a card puts "3 open" in its own label row, and repeating it beside
   * the bar reads as two different figures at a glance.
   */
  showCount = true,
  /**
   * Lets the bar fill its container. Capped by default because in a table cell
   * an unbounded bar stretches with the column and stops being comparable
   * between rows; in a card the column is the card, so the cap is wrong.
   */
  fullWidth = false,
}: {
  open: number;
  busiest: number;
  band: LoadBand;
  className?: string;
  showCount?: boolean;
  fullWidth?: boolean;
}) {
  const share = busiest > 0 ? Math.min(1, open / busiest) : 0;

  return (
    <div className={cn("flex items-center gap-2", className)}>
      <div
        className={cn(
          "h-1.5 w-full overflow-hidden rounded-full bg-tl-line-soft",
          !fullWidth && "min-w-[52px] max-w-[96px]",
        )}
        role="img"
        aria-label={`${open} open ${open === 1 ? "ticket" : "tickets"}. ${BAND_LABEL[band]}.`}
      >
        <div
          className={cn("h-full rounded-full transition-[width] duration-200", DOT[BAND_ACCENT[band]])}
          style={{ width: `${Math.round(share * 100)}%` }}
        />
      </div>
      {showCount && (
        <span className="shrink-0 text-ui-xs tabular-nums text-tl-faint">
          {open}
        </span>
      )}
    </div>
  );
}

export { BAND_LABEL };
