import { AlertCircle, LogIn, ShieldOff } from "lucide-react";

import { cn } from "@/lib/utils";

/*
  The management panel's page frame.

  Wider than the portal's 1100px column: these screens are tables of eight
  columns, and a reading measure tuned for a customer's ticket list would put a
  horizontal scrollbar under every one of them on a laptop. 1440 is the point
  where the staff table stops needing to scroll at all on a 16" display, and the
  container stays centred beyond that rather than letting rows stretch to a
  27" width where the eye loses the row it is on.
*/

export function AdminPage({
  className,
  children,
}: {
  className?: string;
  children: React.ReactNode;
}) {
  return (
    <div
      className={cn(
        "mx-auto flex w-full max-w-[1440px] flex-col gap-5 px-4 py-6 lg:px-8",
        className,
      )}
    >
      {children}
    </div>
  );
}

/**
 * A section caption inside a page — smaller than the page's h1, and used where
 * a Panel header would be too heavy for what follows.
 */
export function SectionLabel({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <h2
      className={cn(
        "text-ui-xs font-semibold uppercase tracking-[0.06em] text-tl-faint",
        className,
      )}
    >
      {children}
    </h2>
  );
}

/**
 * What the panel shows when the API says no.
 *
 * Every screen gated in lib/auth/permissions.ts renders this on a 403 rather
 * than an empty table, because the two look identical and mean opposite things:
 * "there is nobody here" is data, "you may not see who is here" is a
 * permissions problem somebody has to go and fix.
 */
export function ForbiddenState({
  title = "You don't have access to this",
  description,
  permission,
  className,
}: {
  title?: string;
  description: string;
  /** The exact permission string the route is registered with. */
  permission?: string;
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
      <span className="flex size-12 items-center justify-center rounded-full bg-tl-orange-soft text-tl-orange-ink">
        <ShieldOff className="size-6" strokeWidth={1.8} aria-hidden />
      </span>
      <h3 className="mt-4 text-ui-lg font-semibold tracking-[-0.01em] text-tl-ink">
        {title}
      </h3>
      <p className="mt-1.5 max-w-md text-ui-md leading-relaxed text-tl-muted">
        {description}
      </p>
      {permission && (
        <p className="mt-3 text-ui-xs text-tl-faint">
          Required permission:{" "}
          <code className="rounded bg-tl-line-soft px-1.5 py-0.5 font-mono text-tl-ink-soft">
            {permission}
          </code>
        </p>
      )}
    </div>
  );
}

/**
 * Every permission refused at once.
 *
 * Almost never a permissions problem, and telling somebody to go and ask for
 * `department:manage` when they already hold `*` sends them somewhere useless.
 * Permissions are granted per (user, organization), so a token naming an
 * organization that no longer exists finds no grants at all and every gated
 * route refuses identically — which is exactly what a re-seeded database leaves
 * behind in an open tab.
 *
 * Signing in again mints a token against the organization that exists now, so
 * that is the action offered. A plain link rather than a fetch: /login clears
 * and rewrites both cookies, and a half-cleared session is what caused this.
 */
export function StaleSessionState({ className }: { className?: string }) {
  return (
    <div
      role="alert"
      className={cn(
        "flex flex-col items-center justify-center px-6 py-16 text-center",
        className,
      )}
    >
      <span className="flex size-12 items-center justify-center rounded-full bg-tl-blue-soft text-tl-blue">
        <LogIn className="size-6" strokeWidth={1.8} aria-hidden />
      </span>
      <h3 className="mt-4 text-ui-lg font-semibold tracking-[-0.01em] text-tl-ink">
        Your session is out of date
      </h3>
      <p className="mt-1.5 max-w-md text-ui-md leading-relaxed text-tl-muted">
        Every request is coming back refused, which means the organization in
        your sign-in token no longer exists — usually because the database was
        re-seeded. Signing in again fixes it.
      </p>
      <a
        href="/login"
        className="mt-5 inline-flex h-9 items-center gap-1.5 rounded-btn bg-tl-blue px-3.5 text-ui-sm font-semibold text-white transition-colors duration-150 hover:bg-blue-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/40"
      >
        <LogIn className="size-3.5" aria-hidden />
        Sign in again
      </a>
    </div>
  );
}

/**
 * A quiet line explaining where a number came from, or why a column is empty.
 *
 * Used a lot in this panel, because a great deal of what it shows is derived
 * from the ticket queue rather than recorded anywhere. A derived number with no
 * stated basis is the kind of thing people build reports on.
 */
export function Footnote({
  children,
  className,
}: {
  children: React.ReactNode;
  className?: string;
}) {
  return (
    <p
      className={cn(
        "flex items-start gap-1.5 text-ui-xs leading-relaxed text-tl-faint",
        className,
      )}
    >
      <AlertCircle
        className="mt-[1px] size-3.5 shrink-0 text-tl-faint-soft"
        aria-hidden
      />
      <span>{children}</span>
    </p>
  );
}
