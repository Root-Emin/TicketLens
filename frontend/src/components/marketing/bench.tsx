"use client";

import { useEffect, useState } from "react";
import { Pause, Play } from "lucide-react";

import { useMediaQuery } from "@/hooks/use-media-query";
import { CATEGORIES, PRIORITIES, SAMPLES } from "@/lib/marketing/taxonomy";
import { cn } from "@/lib/utils";

/*
  The bench: the page's one instrument, and the only place it spends any budget
  on motion or ornament.

  It draws a classifier run the way a piece of measuring equipment would — a
  fixed set of classes, one of which lights up. Both heads get that same
  treatment at two scales: ten lanes for category, four steps for priority. The
  numbers are read out in mono because they are the machine's words; the ticket
  itself is set in the UI face because it is a person's.

  Timing is two timers rather than one state machine: SCAN_MS is how long the
  read stroke runs, HOLD_MS is how long the settled verdict stays up. Selecting
  a ticket by hand stops the cycle for good — a visitor who has taken control
  should not have it taken back.
*/

const SCAN_MS = 900;
const HOLD_MS = 4200;

export function Bench() {
  const [index, setIndex] = useState(0);
  const [reading, setReading] = useState(false);
  const [paused, setPaused] = useState(false);

  // Honour the OS setting by not running the cycle at all. The base layer
  // already flattens durations; this stops the content from changing under
  // someone who asked for calm, which a duration override cannot do.
  const reduced = useMediaQuery("(prefers-reduced-motion: reduce)");
  const still = reduced || paused;

  useEffect(() => {
    if (reduced) return;
    setReading(true);
    const t = setTimeout(() => setReading(false), SCAN_MS);
    return () => clearTimeout(t);
  }, [index, reduced]);

  useEffect(() => {
    if (still) return;
    const t = setTimeout(
      () => setIndex((i) => (i + 1) % SAMPLES.length),
      SCAN_MS + HOLD_MS,
    );
    return () => clearTimeout(t);
  }, [index, still]);

  const sample = SAMPLES[index];
  const settled = !reading;

  /** Hand control to the visitor and keep it there. */
  function select(next: number) {
    setIndex(next);
    setPaused(true);
  }

  return (
    <figure
      className="overflow-hidden rounded-card border border-tlm-bench-line bg-tlm-bench-raised"
      role="group"
      aria-label="Example classifier runs"
    >
      {/* Rail: instrument label left, ticket selector right. */}
      <div className="flex flex-wrap items-center justify-between gap-3 border-b border-tlm-bench-line px-4 py-3 sm:px-5">
        <p className="font-mono text-ui-xs uppercase tracking-[0.18em] text-tlm-muted">
          Classifier
          <span className="mx-2 text-tlm-bench-line">/</span>
          <span className={cn(reading ? "text-tlm-beam" : "text-tlm-muted")}>
            {reading ? "reading" : "idle"}
          </span>
        </p>

        <div className="flex items-center gap-1.5">
          {SAMPLES.map((s, i) => (
            <button
              key={s.company}
              type="button"
              onClick={() => select(i)}
              aria-pressed={i === index}
              aria-label={`Ticket ${i + 1}: ${s.company}`}
              className={cn(
                "tap-target size-2.5 rounded-full border transition-colors",
                "focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tlm-beam focus-visible:ring-offset-2 focus-visible:ring-offset-tlm-bench-raised",
                i === index
                  ? "border-tlm-beam bg-tlm-beam"
                  : "border-tlm-bench-line bg-transparent hover:border-tlm-muted",
              )}
            />
          ))}

          {!reduced && (
            <button
              type="button"
              onClick={() => setPaused((p) => !p)}
              aria-label={paused ? "Resume cycling tickets" : "Pause cycling tickets"}
              className="tap-target ml-2 rounded-btn p-1 text-tlm-muted transition-colors hover:text-tlm-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tlm-beam focus-visible:ring-offset-2 focus-visible:ring-offset-tlm-bench-raised"
            >
              {paused ? (
                <Play className="size-3.5" aria-hidden />
              ) : (
                <Pause className="size-3.5" aria-hidden />
              )}
            </button>
          )}
        </div>
      </div>

      <div className="grid lg:grid-cols-[1.15fr_1fr]">
        {/* Incoming — the customer's words, in the human face. */}
        {/*
          Bottom-anchored sender line: the readout column is the taller of the
          two, and letting the left cell run short left a void under the ticket
          text. Anchoring the meta to the floor gives the cell two edges instead
          of one.
        */}
        <div className="relative flex flex-col overflow-hidden border-b border-tlm-bench-line p-5 sm:p-6 lg:border-b-0 lg:border-r">
          <Label>Incoming</Label>

          {/* The read stroke. Keyed so it restarts on every ticket. */}
          {reading && (
            <span
              key={index}
              aria-hidden
              className="animate-scan pointer-events-none absolute inset-x-0 top-0 h-px bg-gradient-to-r from-transparent via-tlm-beam to-transparent"
            />
          )}

          <p className="mt-4 font-ui text-[15px] leading-[1.65] text-tlm-ink sm:text-base">
            {sample.body}
          </p>

          <p className="mt-5 pt-4 font-mono text-ui-xs text-tlm-muted lg:mt-auto">
            {sample.from}
            <span className="mx-1.5 text-tlm-bench-line">·</span>
            {sample.company}
            <span className="mx-1.5 text-tlm-bench-line">·</span>
            {sample.received}
          </p>
        </div>

        {/* Reading — the model's words, in mono. */}
        <div className="p-5 sm:p-6">
          <Label>Reading</Label>

          <dl className="mt-4 space-y-5">
            <Readout
              term="category"
              value={sample.category.label}
              confidence={sample.category.confidence}
              settled={settled}
              muted={sample.needsReview}
            />

            {sample.runnerUp && (
              <Readout
                term="runner-up"
                value={sample.runnerUp.label}
                confidence={sample.runnerUp.confidence}
                settled={settled}
                muted
              />
            )}

            <div>
              <Term>priority</Term>
              <PriorityRamp level={sample.priority.level} settled={settled} />
              <p className="mt-2 flex items-baseline justify-between font-mono text-ui-sm">
                <span className="text-tlm-ink">{sample.priority.label}</span>
                <Confidence value={sample.priority.confidence} settled={settled} />
              </p>
            </div>
          </dl>

          {/* The verdict. Either it routes, or it admits it cannot. */}
          <div className="mt-6 border-t border-tlm-bench-line pt-4">
            {sample.needsReview ? (
              <div className="font-mono text-ui-sm">
                <p className="text-priority-high">needs_human_review = true</p>
                <p className="mt-1.5 leading-relaxed text-tlm-muted">
                  Below the review threshold. Sent to a person, and no
                  prediction is recorded against this ticket.
                </p>
              </div>
            ) : (
              <p className="font-mono text-ui-sm text-tlm-ink">
                <span className="text-tlm-beam" aria-hidden>
                  →{" "}
                </span>
                <span className="sr-only">Routed to </span>
                {sample.department}
              </p>
            )}
          </div>
        </div>
      </div>

      {/* The ten lanes. The lens, doing the one thing a lens does. */}
      <div className="border-t border-tlm-bench-line px-4 py-5 sm:px-5">
        <Label>Taxonomy</Label>
        <ul className="mt-3 grid grid-cols-2 gap-x-3 gap-y-2.5 sm:grid-cols-3 md:grid-cols-5 lg:grid-cols-10">
          {CATEGORIES.map((c) => {
            const isTop = c.slug === sample.category.label;
            const isRunnerUp = c.slug === sample.runnerUp?.label;
            const weight = isTop
              ? sample.category.confidence
              : isRunnerUp
                ? (sample.runnerUp?.confidence ?? 0)
                : 0;

            return (
              <li key={c.slug} className="min-w-0">
                {/*
                  Unlit lanes stay at --tlm-muted rather than dropping to the
                  hairline colour: the strip is the taxonomy, and a reader has
                  to be able to read the nine classes this ticket is not. The
                  lit lane separates itself with ink plus its fill bar, which
                  is enough without pushing the rest under AA.
                */}
                <span
                  className={cn(
                    "block truncate font-mono text-[10px] tracking-tight transition-colors duration-200",
                    isTop && settled
                      ? "font-medium text-tlm-ink"
                      : "text-tlm-muted",
                  )}
                  title={c.slug}
                >
                  {c.slug}
                </span>
                <span className="mt-1.5 block h-0.5 w-full bg-tlm-bench-line-soft">
                  <span
                    className={cn(
                      "block h-full transition-[width] duration-500 ease-out",
                      isTop ? "bg-tlm-beam" : "bg-tlm-muted",
                    )}
                    style={{ width: settled ? `${weight * 100}%` : "0%" }}
                  />
                </span>
              </li>
            );
          })}
        </ul>
      </div>

      <figcaption className="border-t border-tlm-bench-line px-4 py-4 font-ui text-[13px] leading-relaxed text-tlm-muted sm:px-5">
        {sample.note}
      </figcaption>
    </figure>
  );
}

function Label({ children }: { children: React.ReactNode }) {
  return (
    <p className="font-mono text-[10px] uppercase tracking-[0.2em] text-tlm-muted">
      {children}
    </p>
  );
}

function Term({ children }: { children: React.ReactNode }) {
  return (
    <dt className="mb-1.5 font-mono text-[10px] uppercase tracking-[0.16em] text-tlm-muted">
      {children}
    </dt>
  );
}

/** Mono numerals, held at two decimals so the column never jitters. */
function Confidence({ value, settled }: { value: number; settled: boolean }) {
  return (
    <span
      className={cn(
        "tabular-nums transition-opacity duration-300",
        settled ? "text-tlm-muted opacity-100" : "opacity-0",
      )}
    >
      {value.toFixed(2)}
    </span>
  );
}

function Readout({
  term,
  value,
  confidence,
  settled,
  muted,
}: {
  term: string;
  value: string;
  confidence: number;
  settled: boolean;
  muted?: boolean;
}) {
  return (
    <div>
      <Term>{term}</Term>
      <dd className="flex items-baseline justify-between gap-3 font-mono text-ui-sm">
        <span className={cn("truncate", muted ? "text-tlm-muted" : "text-tlm-ink")}>
          {value}
        </span>
        <Confidence value={confidence} settled={settled} />
      </dd>
      <span className="mt-1.5 block h-0.5 w-full bg-tlm-bench-line-soft">
        <span
          className={cn(
            "block h-full transition-[width] duration-500 ease-out",
            muted ? "bg-tlm-muted" : "bg-tlm-beam",
          )}
          style={{ width: settled ? `${confidence * 100}%` : "0%" }}
        />
      </span>
    </div>
  );
}

/*
  The priority head, drawn as the four-step ramp rather than a bar: priority is
  ordered, so where the prediction sits among the others is the information. The
  colours are the app's own --priority-* tokens, which is what makes the ramp
  the one piece of colour on the page that is carrying data.
*/
function PriorityRamp({
  level,
  settled,
}: {
  level: string;
  settled: boolean;
}) {
  return (
    <div className="flex gap-1" aria-hidden>
      {PRIORITIES.map((p) => (
        <span
          key={p.level}
          className={cn(
            // Unlit steps sit at --tlm-bench-line, not line-soft: the ramp only
            // says anything if you can see it is a scale of four with the
            // prediction sitting at a particular point on it.
            "h-1 flex-1 rounded-full transition-colors duration-300",
            settled && p.level === level ? p.bar : "bg-tlm-bench-line",
          )}
        />
      ))}
    </div>
  );
}
