import Link from "next/link";
import { Ticket } from "lucide-react";

import { LensRings } from "./lens-rings";

/*
  The two-column frame behind /login and /register.

  Left is brand and is decorative: it disappears below lg rather than stacking,
  because a phone that has to scroll past a marketing panel to reach a password
  field is a worse sign-in, not a more branded one. Nothing on that side is
  needed to complete the form, so hiding it costs no information.

  Right is the form, and it is the only thing on a small screen.
*/

const POINTS = [
  "Track every request in one place",
  "AI routes your ticket to the right team",
  "Talk to support without leaving the thread",
];

export function AuthShell({
  title,
  subtitle,
  children,
  footer,
}: {
  title: string;
  subtitle: string;
  children: React.ReactNode;
  footer?: React.ReactNode;
}) {
  return (
    <div className="flex min-h-dvh bg-tl-canvas font-ui text-tl-ink antialiased">
      {/* Brand panel */}
      <aside className="relative hidden w-[46%] max-w-[620px] shrink-0 overflow-hidden bg-tl-navy lg:flex lg:flex-col">
        <LensRings className="pointer-events-none absolute -right-[18%] -top-[12%] w-[86%]" />

        <div className="relative flex h-full flex-col justify-between p-12">
          <Link
            href="/"
            className="flex w-fit items-center gap-2.5 rounded-btn focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue focus-visible:ring-offset-2 focus-visible:ring-offset-tl-navy"
          >
            <img
              src="/assets/TicketLens_Logo/favicon.svg"
              alt="TicketLens Logo"
              className="size-9 rounded-[10px] object-contain shadow-sm"
            />
            <span className="text-[19px] font-bold tracking-[-0.015em] text-white">
              TicketLens
            </span>
          </Link>

          <div className="max-w-[420px]">
            <p className="text-[30px] font-bold leading-[1.25] tracking-[-0.02em] text-white">
              Support that reads your request before anyone picks it up.
            </p>
            <ul className="mt-7 space-y-3">
              {POINTS.map((point) => (
                <li
                  key={point}
                  className="flex items-center gap-3 text-ui-md text-tl-rail-text"
                >
                  <span
                    className="size-1.5 shrink-0 rounded-full bg-tl-blue"
                    aria-hidden
                  />
                  {point}
                </li>
              ))}
            </ul>
          </div>

          <p className="text-ui-sm text-tl-rail-caption">
            AI-assisted support triage
          </p>
        </div>
      </aside>

      {/* Form panel */}
      <main className="flex min-w-0 flex-1 items-center justify-center px-4 py-10 sm:px-8">
        <div className="w-full max-w-[400px]">
          {/* The wordmark repeats here for the small screens that never see
              the panel on the left. */}
          <div className="mb-8 flex items-center gap-2.5 lg:hidden">
            <img
              src="/assets/TicketLens_Logo/favicon.svg"
              alt="TicketLens Logo"
              className="size-9 rounded-[10px] object-contain shadow-sm"
            />
            <span className="text-[19px] font-bold tracking-[-0.015em] text-tl-ink">
              TicketLens
            </span>
          </div>

          <h1 className="text-ui-2xl font-bold tracking-[-0.02em] text-tl-ink">
            {title}
          </h1>
          <p className="mt-1.5 text-ui-md text-tl-muted">{subtitle}</p>

          <div className="mt-7">{children}</div>

          {footer && (
            <div className="mt-7 text-center text-ui-md text-tl-muted">
              {footer}
            </div>
          )}
        </div>
      </main>
    </div>
  );
}
