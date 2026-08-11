import type { Metadata } from "next";
import Link from "next/link";

import { Bench } from "@/components/marketing/bench";
import { CATEGORIES, DEPARTMENTS, PRIORITIES } from "@/lib/marketing/taxonomy";
import { cn } from "@/lib/utils";

export const metadata: Metadata = {
  title: "TicketLens — ticket triage that admits what it doesn't know",
  description:
    "TicketLens reads every incoming support request, sorts it into one of ten classes, sets a priority, and routes it. When it isn't confident, it flags the ticket for a person instead of guessing.",
};

/*
  The public landing page.

  Everything factual here is restated from the backend: the ten classes come
  from domain/triage/model/category.go, the four departments from cmd/seed, and
  the flags named in the "Uncertainty" section are real columns on
  domain/triage/model/ai_analysis.go. Nothing on this page describes behaviour
  the system does not have — see lib/marketing/taxonomy.ts.
*/
export default function LandingPage() {
  return (
    <div className="bg-tl-canvas font-ui text-tl-ink antialiased">
      <Nav />
      <Hero />
      <Heads />
      <Taxonomy />
      <Uncertainty />
      <Close />
      <Footer />
    </div>
  );
}

/* -------------------------------------------------------------------------- */

function Nav() {
  return (
    <header className="bg-tlm-bench">
      <div className="mx-auto flex max-w-6xl items-center justify-between px-5 py-5 sm:px-8">
        <Link
          href="/"
          className="flex items-center gap-2.5 rounded-btn focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tlm-beam focus-visible:ring-offset-4 focus-visible:ring-offset-tlm-bench"
        >
          {/* eslint-disable-next-line @next/next/no-img-element */}
          <img
            src="/assets/TicketLens_Logo/favicon.svg"
            alt=""
            className="size-8 rounded-[9px] object-contain"
          />
          <span className="text-[18px] font-bold tracking-[-0.015em] text-white">
            TicketLens
          </span>
        </Link>

        <nav className="flex items-center gap-1.5">
          <Link
            href="/login"
            className="rounded-btn px-3.5 py-2 text-[14px] font-medium text-tlm-muted transition-colors hover:text-tlm-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tlm-beam focus-visible:ring-offset-2 focus-visible:ring-offset-tlm-bench"
          >
            Sign in
          </Link>
          {/*
            Dropped below sm: at 390px it and "Sign in" crowd the wordmark hard
            enough to wrap mid-label. Nothing is lost — the hero carries the
            same action a screen-height below, and sign-in is the link a
            returning visitor comes to the nav looking for.
          */}
          <Link
            href="/register"
            className="hidden rounded-btn bg-white px-3.5 py-2 text-[14px] font-semibold text-tlm-bench transition-colors hover:bg-tlm-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tlm-beam focus-visible:ring-offset-2 focus-visible:ring-offset-tlm-bench sm:block"
          >
            Create an account
          </Link>
        </nav>
      </div>
    </header>
  );
}

function Hero() {
  return (
    <section className="bg-tlm-bench pb-16 sm:pb-20 lg:pb-28">
      <div className="mx-auto max-w-6xl px-5 sm:px-8">
        <p className="font-mono text-ui-xs uppercase tracking-[0.22em] text-tlm-beam">
          Triage — the part before anyone answers
        </p>

        {/*
          text-balance rather than a hand-tuned measure: at 1440px a 19ch cap
          gives a clean two-line break, but the same cap on a 390px phone drops
          the final "it." onto a line of its own. Balancing lets the browser
          even them out at whatever width it actually gets.
        */}
        <h1 className="type-display mt-6 max-w-[19ch] text-balance text-hero font-bold leading-[0.94] text-white">
          Every ticket read before anyone opens it.
        </h1>

        <p className="mt-7 max-w-[60ch] text-lede leading-[1.6] text-tlm-muted">
          TicketLens reads each incoming request, sorts it into one of ten
          classes, sets a priority, and sends it to the team that handles it.
          When it is not confident enough, it says so instead of guessing.
        </p>

        <div className="mt-9 mb-14 flex flex-wrap items-center gap-3 sm:mb-16">
          <Link
            href="/login"
            className="rounded-btn bg-tl-blue px-5 py-3 text-[15px] font-semibold text-white transition-colors hover:bg-[#1d4fd8] focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tlm-beam focus-visible:ring-offset-2 focus-visible:ring-offset-tlm-bench"
          >
            Open the workspace
          </Link>
          <Link
            href="/register"
            className="rounded-btn border border-tlm-bench-line px-5 py-3 text-[15px] font-medium text-tlm-ink transition-colors hover:border-tlm-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tlm-beam focus-visible:ring-offset-2 focus-visible:ring-offset-tlm-bench"
          >
            Create an account
          </Link>
        </div>

        <Bench />
      </div>
    </section>
  );
}

/* -------------------------------------------------------------------------- */

function Heads() {
  return (
    <Section>
      <SectionHead
        eyebrow="Two heads"
        title="Two calls, made separately."
        lede="One model, two predictions. They are kept apart because they answer different questions, and because a class does not imply an urgency: a payment that never landed and a question about how to issue a refund are both payment_ops, and only one of them is worth waking someone up for."
      />

      <div className="mt-14 grid gap-px overflow-hidden rounded-card border border-tl-line bg-tl-line md:grid-cols-2">
        <div className="bg-tl-card p-7 sm:p-9">
          <p className="font-mono text-ui-xs uppercase tracking-[0.16em] text-tl-faint">
            Head one
          </p>
          <h3 className="type-display mt-3 text-[26px] font-bold leading-tight">
            Category
          </h3>
          <p className="mt-3 text-[15px] leading-relaxed text-tl-ink-soft">
            Ten fixed classes, the same for every organization on the platform.
            The model learns this list and nothing else, which is what makes one
            organization&rsquo;s corrections worth anything to the next.
          </p>
          <ul className="mt-6 flex flex-wrap gap-1.5">
            {CATEGORIES.map((c) => (
              <li
                key={c.slug}
                className="rounded-md bg-tl-blue-soft px-2 py-1 font-mono text-[11px] text-tl-blue"
              >
                {c.slug}
              </li>
            ))}
          </ul>
        </div>

        <div className="bg-tl-card p-7 sm:p-9">
          <p className="font-mono text-ui-xs uppercase tracking-[0.16em] text-tl-faint">
            Head two
          </p>
          <h3 className="type-display mt-3 text-[26px] font-bold leading-tight">
            Priority
          </h3>
          <p className="mt-3 text-[15px] leading-relaxed text-tl-ink-soft">
            Four levels, ordered. This is the head that decides what your team
            sees first thing in the morning, so it is scored and calibrated on
            its own rather than inferred from the class.
          </p>
          <ul className="mt-6 space-y-2.5">
            {PRIORITIES.map((p) => (
              <li key={p.level} className="flex items-center gap-3">
                <span
                  className={cn("h-1.5 w-10 shrink-0 rounded-full", p.bar)}
                  aria-hidden
                />
                <span className="font-mono text-[13px] text-tl-ink-soft">
                  {p.level}
                </span>
              </li>
            ))}
          </ul>
        </div>
      </div>
    </Section>
  );
}

/* -------------------------------------------------------------------------- */

function Taxonomy() {
  return (
    <Section muted>
      <SectionHead
        eyebrow="The class list"
        title={
          <>
            Ten classes.
            <br />
            No &ldquo;other&rdquo;.
          </>
        }
        lede="An escape-hatch class is the easiest thing in the world to add and it ruins the model. Every ambiguous ticket collects in it, the class stops meaning anything, and the confidence score you needed most is the one you destroyed. So there is no bucket for uncertainty. Uncertainty gets a flag instead."
      />

      <dl className="mt-14 grid gap-x-10 gap-y-8 sm:grid-cols-2">
        {CATEGORIES.map((c) => (
          <div key={c.slug} className="border-t border-tl-line pt-4">
            <dt className="font-mono text-[13px] text-tl-blue">{c.slug}</dt>
            <dd className="mt-1.5 text-[14px] leading-relaxed text-tl-muted">
              {c.blurb}
            </dd>
          </div>
        ))}
      </dl>
    </Section>
  );
}

/* -------------------------------------------------------------------------- */

/*
  The three facts are labelled with the real column names rather than 01/02/03.
  They are an unordered set, not a sequence, and the identifiers are what a
  reader would actually go looking for in the API.
*/
const UNCERTAINTY = [
  {
    field: "needs_human_review",
    title: "It flags what it cannot call.",
    body: "The threshold is not a round number somebody picked. Confidence is temperature-scaled against a held-out split and the cut-off is swept for the trade-off you want between tickets auto-routed and tickets sent to a person. Below the line, the ticket goes to a human and no prediction is recorded against it.",
  },
  {
    field: "mapping_fallback",
    title: "The model's opinion is stored apart from your org chart.",
    body: "Categories are model-wide; departments are yours. A demo organization runs four departments against ten classes, so six classes have no team to land in and fall through to the default one. Those tickets are marked and excluded from the accept rate — otherwise 'the model was wrong' and 'you have no team for this' would be the same number.",
  },
  {
    field: "priority_overridden",
    title: "Every correction is counted against it.",
    body: "When somebody changes the priority or moves the ticket to another team, the original prediction stays on the record and the ticket is marked as overridden. The accept rate on your dashboard is that arithmetic and nothing else, which is why it is allowed to go down.",
  },
];

function Uncertainty() {
  return (
    <Section>
      <SectionHead
        eyebrow="Uncertainty"
        title="Confidence you can audit."
        lede="A triage model is only worth installing if you can tell when it is wrong. Three things are on the record for every ticket it touches."
      />

      <div className="mt-14 space-y-px overflow-hidden rounded-card border border-tl-line bg-tl-line">
        {UNCERTAINTY.map((item) => (
          <div
            key={item.field}
            className="grid gap-4 bg-tl-card p-7 sm:p-9 md:grid-cols-[minmax(0,15rem)_1fr] md:gap-10"
          >
            <p className="font-mono text-[13px] leading-relaxed text-tl-blue">
              {item.field}
            </p>
            <div>
              <h3 className="text-[19px] font-semibold leading-snug tracking-[-0.01em]">
                {item.title}
              </h3>
              <p className="mt-2.5 max-w-[62ch] text-[15px] leading-relaxed text-tl-ink-soft">
                {item.body}
              </p>
            </div>
          </div>
        ))}
      </div>

      <div className="mt-10 flex flex-wrap items-center gap-x-2 gap-y-2 text-[13px] text-tl-muted">
        <span className="font-mono text-tl-faint">routes into</span>
        {DEPARTMENTS.map((d) => (
          <span
            key={d.name}
            className="rounded-md border border-tl-line bg-tl-card px-2.5 py-1"
          >
            {d.name}
          </span>
        ))}
        <span className="font-mono text-tl-faint">
          + your default, for everything unmapped
        </span>
      </div>
    </Section>
  );
}

/* -------------------------------------------------------------------------- */

function Close() {
  return (
    <section className="bg-tlm-bench">
      <div className="mx-auto max-w-6xl px-5 py-24 sm:px-8 sm:py-32">
        <h2 className="type-display max-w-[18ch] text-title font-bold leading-[1.02] text-white">
          Point it at your own queue.
        </h2>
        <p className="mt-6 max-w-[52ch] text-lede leading-[1.6] text-tlm-muted">
          The workspace ships with a seeded organization — real tickets, real
          departments, real overrides — so you can watch the accept rate move
          before you connect anything of your own.
        </p>
        <div className="mt-9 flex flex-wrap items-center gap-3">
          <Link
            href="/register"
            className="rounded-btn bg-white px-5 py-3 text-[15px] font-semibold text-tlm-bench transition-colors hover:bg-tlm-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tlm-beam focus-visible:ring-offset-2 focus-visible:ring-offset-tlm-bench"
          >
            Create an account
          </Link>
          <Link
            href="/login"
            className="rounded-btn border border-tlm-bench-line px-5 py-3 text-[15px] font-medium text-tlm-ink transition-colors hover:border-tlm-muted focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tlm-beam focus-visible:ring-offset-2 focus-visible:ring-offset-tlm-bench"
          >
            Sign in
          </Link>
        </div>
      </div>
    </section>
  );
}

function Footer() {
  return (
    <footer className="bg-tlm-bench">
      <div className="mx-auto max-w-6xl border-t border-tlm-bench-line px-5 py-8 sm:px-8">
        <p className="font-mono text-ui-xs text-tlm-muted">
          TicketLens
          <span className="mx-2 text-tlm-bench-line">·</span>
          AI-assisted support ticket triage
        </p>
      </div>
    </footer>
  );
}

/* -------------------------------------------------------------------------- */

/**
 * The page's one rhythm. Every light section is this shell, so vertical spacing
 * is set in a single place instead of drifting section by section.
 */
function Section({
  children,
  muted,
}: {
  children: React.ReactNode;
  muted?: boolean;
}) {
  return (
    <section className={muted ? "bg-tl-canvas" : "bg-white"}>
      <div className="mx-auto max-w-6xl px-5 py-24 sm:px-8 sm:py-32">
        {children}
      </div>
    </section>
  );
}

function SectionHead({
  eyebrow,
  title,
  lede,
}: {
  eyebrow: string;
  title: React.ReactNode;
  lede: string;
}) {
  return (
    <div className="max-w-[68ch]">
      <p className="font-mono text-ui-xs uppercase tracking-[0.2em] text-tl-faint">
        {eyebrow}
      </p>
      <h2 className="type-display mt-5 max-w-[20ch] text-title font-bold leading-[1.02]">
        {title}
      </h2>
      <p className="mt-6 text-[17px] leading-[1.65] text-tl-ink-soft">{lede}</p>
    </div>
  );
}
