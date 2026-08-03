import Link from "next/link";
import type { LucideIcon } from "lucide-react";

/*
  An honest placeholder.

  Knowledge Base, Reports and AI Insights are real destinations in the
  navigation but are not built products yet. Rather than mock up screens full of
  invented data that would read as finished, each says plainly what it will do
  and points at the work that exists today.
*/

export function PlaceholderPage({
  icon: Icon,
  title,
  description,
  planned,
}: {
  icon: LucideIcon;
  title: string;
  description: string;
  planned: string[];
}) {
  return (
    <div className="h-full overflow-y-auto">
      <div className="mx-auto flex max-w-[720px] flex-col items-center px-6 py-16 text-center">
        <span className="flex size-14 items-center justify-center rounded-2xl border border-tl-line bg-tl-card shadow-panel">
          <Icon className="size-6 text-tl-blue" strokeWidth={1.8} aria-hidden />
        </span>

        <h1 className="mt-5 text-ui-xl font-bold tracking-[-0.02em] text-tl-ink">
          {title}
        </h1>
        <p className="mt-2 max-w-[46ch] text-ui-md leading-relaxed text-tl-muted">
          {description}
        </p>

        <div className="mt-8 w-full rounded-card border border-tl-line bg-tl-card p-5 text-left shadow-panel">
          <h2 className="text-ui-sm font-semibold uppercase tracking-[0.06em] text-tl-faint">
            Planned
          </h2>
          <ul className="mt-3 space-y-2">
            {planned.map((item) => (
              <li
                key={item}
                className="flex gap-2.5 text-ui-base text-tl-ink-soft"
              >
                <span
                  className="mt-2 size-1.5 shrink-0 rounded-full bg-tl-faint-soft"
                  aria-hidden
                />
                {item}
              </li>
            ))}
          </ul>
        </div>

        <Link
          href="/staff/tickets?view=assigned"
          className="mt-6 inline-flex h-10 items-center rounded-btn bg-tl-blue px-4 text-ui-base font-semibold text-white transition-colors duration-150 hover:bg-blue-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/40"
        >
          Back to my queue
        </Link>
      </div>
    </div>
  );
}
