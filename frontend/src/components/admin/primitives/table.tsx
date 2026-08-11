import { cn } from "@/lib/utils";

/*
  The panel's data table.

  Three rules, and they are the whole design:

    1. One hairline between rows, none between columns. Column rules turn a list
       of people into a spreadsheet, and the eye then reads down instead of
       across — the wrong direction for a table whose unit is a person.
    2. The header is a caption, not a band. 11px, uppercase, muted, on the same
       white as the rows; the divider under it does the separating. A filled grey
       header bar costs 40px of visual weight at the top of every screen.
    3. Numeric columns are tabular and right-aligned, so a column of workloads
       can be compared by its silhouette without reading a digit.

  The horizontal scroll lives on the wrapper rather than the page, which is what
  keeps `body` from ever scrolling sideways on a phone. Below the table's
  breakpoint each screen renders its own card list instead — a table squeezed to
  360px is not a mobile design, it is a desktop design with a scrollbar.
*/

export function TableFrame({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <section
      className={cn(
        "overflow-hidden rounded-card border border-tl-line bg-tl-card shadow-panel",
        className,
      )}
    >
      {children}
    </section>
  );
}

/** The scrolling region a wide table sits in. */
export function TableScroll({ children }: { children: React.ReactNode }) {
  return <div className="w-full overflow-x-auto">{children}</div>;
}

export function Table({
  className,
  children,
  ...props
}: React.ComponentProps<"table">) {
  return (
    <table
      className={cn("w-full border-collapse text-left", className)}
      {...props}
    >
      {children}
    </table>
  );
}

export function Th({
  className,
  numeric,
  children,
  ...props
}: React.ComponentProps<"th"> & { numeric?: boolean }) {
  return (
    <th
      scope="col"
      className={cn(
        "whitespace-nowrap border-b border-tl-line px-4 py-2.5 text-ui-xs font-semibold uppercase tracking-[0.06em] text-tl-faint",
        numeric && "text-right",
        className,
      )}
      {...props}
    >
      {children}
    </th>
  );
}

export function Tr({
  className,
  children,
  ...props
}: React.ComponentProps<"tr">) {
  return (
    <tr
      className={cn(
        "border-b border-tl-line-soft transition-colors duration-150 last:border-0 hover:bg-tl-line-soft/60",
        className,
      )}
      {...props}
    >
      {children}
    </tr>
  );
}

export function Td({
  className,
  numeric,
  children,
  ...props
}: React.ComponentProps<"td"> & { numeric?: boolean }) {
  return (
    <td
      className={cn(
        "px-4 py-3 align-middle text-ui-base text-tl-ink-soft",
        numeric && "text-right tabular-nums",
        className,
      )}
      {...props}
    >
      {children}
    </td>
  );
}

/**
 * The bar above a table: a count on the left, controls on the right.
 *
 * Separate from PanelHeader because that one renders an h2, and these tables sit
 * under a page h1 with nothing between — a second heading level for "24 people"
 * would be a heading that is really a status line.
 */
export function TableToolbar({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "flex flex-wrap items-center gap-x-3 gap-y-2 border-b border-tl-line px-4 py-3",
        className,
      )}
    >
      {children}
    </div>
  );
}

/** The mobile stand-in for a row: one card per record. */
export function RecordCard({
  className,
  children,
  ...props
}: React.ComponentProps<"div">) {
  return (
    <div
      className={cn(
        "border-b border-tl-line-soft px-4 py-3.5 last:border-0",
        className,
      )}
      {...props}
    >
      {children}
    </div>
  );
}

/** A label/value pair inside a RecordCard. */
export function RecordField({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="min-w-0">
      <dt className="text-ui-xs font-medium text-tl-faint">{label}</dt>
      <dd className="mt-0.5 truncate text-ui-sm text-tl-ink-soft">{children}</dd>
    </div>
  );
}
