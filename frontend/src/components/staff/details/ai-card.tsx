"use client";

import { useState } from "react";
import Link from "next/link";
import { ChevronDown, Sparkles } from "lucide-react";

import { cn } from "@/lib/utils";
import { percent } from "@/lib/staff/format";
import type { Accent, Ticket } from "@/lib/staff/types";
import {
  FieldGroupLabel,
  INK,
  PRIORITY_ACCENT,
  PRIORITY_LABEL,
  StatusBadge,
} from "../primitives";
import { Row } from "./field-row";

const SENTIMENT_ACCENT: Record<Ticket["ai"]["sentiment"], Accent> = {
  positive: "green",
  neutral: "neutral",
  negative: "red",
};

/** Confidence colouring: strong, fair, weak — the bands triage acts on. */
function confidenceBar(confidence: number): string {
  if (confidence >= 0.8) return "bg-tl-green";
  if (confidence >= 0.5) return "bg-tl-orange";
  return "bg-tl-red";
}

export function AiCard({ ticket }: { ticket: Ticket }) {
  const [open, setOpen] = useState(true);
  const [copied, setCopied] = useState(false);
  const { ai } = ticket;

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(ai.suggestedReply);
      setCopied(true);
      window.setTimeout(() => setCopied(false), 1600);
    } catch {
      // Clipboard permission denied or unavailable over http — the text is
      // still selectable, so fail quietly rather than throwing at the user.
    }
  };

  return (
    <section className="overflow-hidden rounded-card border border-tl-line bg-tl-card shadow-panel">
      <div className="flex items-center gap-2 bg-tl-blue-soft px-5 py-3">
        <Sparkles className="size-4 text-tl-blue" aria-hidden />
        <h2 className="text-ui-md font-semibold text-tl-ink">AI Assistant</h2>
        <span className="rounded-md bg-blue-100 px-1.5 py-[2px] text-[9.5px] font-bold uppercase tracking-wide text-tl-blue">
          Beta
        </span>
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          aria-expanded={open}
          aria-label={open ? "Collapse AI assistant" : "Expand AI assistant"}
          className="tap-target ml-auto rounded text-tl-muted transition-colors duration-150 hover:text-tl-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
        >
          <ChevronDown
            className={cn(
              "size-4 transition-transform duration-200",
              open && "rotate-180",
            )}
            aria-hidden
          />
        </button>
      </div>

      {open && (
        <>
          <div className="space-y-1 px-5 py-4">
            <FieldGroupLabel>Prediction</FieldGroupLabel>

            <Row label="Category">
              <span className="truncate text-ui-sm font-medium text-tl-blue">
                {ai.category}
              </span>
            </Row>
            <Row label="Subcategory">
              <span className="truncate text-ui-sm font-medium text-tl-blue">
                {ai.subcategory}
              </span>
            </Row>
            <Row label="Priority">
              <span
                className={cn(
                  "text-ui-sm font-medium",
                  INK[PRIORITY_ACCENT[ai.priority]],
                )}
              >
                {PRIORITY_LABEL[ai.priority]}
              </span>
            </Row>
            <Row label="Sentiment">
              <span
                className={cn(
                  "text-ui-sm font-medium capitalize",
                  INK[SENTIMENT_ACCENT[ai.sentiment]],
                )}
              >
                {ai.sentiment}
              </span>
            </Row>
            <Row label="Confidence">
              <span className="text-ui-sm font-semibold tabular-nums text-tl-ink">
                {percent(ai.confidence)}
              </span>
              <div
                role="progressbar"
                aria-valuenow={Math.round(ai.confidence * 100)}
                aria-valuemin={0}
                aria-valuemax={100}
                aria-label="Prediction confidence"
                className="h-1.5 w-20 overflow-hidden rounded-full bg-tl-line-soft"
              >
                <div
                  className={cn("h-full rounded-full", confidenceBar(ai.confidence))}
                  style={{ width: `${ai.confidence * 100}%` }}
                />
              </div>
            </Row>
          </div>

          <div className="border-t border-tl-line-soft px-5 py-4">
            <FieldGroupLabel>Suggested reply</FieldGroupLabel>
            <p className="rounded-btn border border-tl-line bg-slate-50 px-3 py-2.5 text-ui-sm leading-[1.6] text-tl-ink-soft">
              {ai.suggestedReply}
            </p>
            <div className="mt-3 flex gap-2">
              <button
                type="button"
                className="inline-flex h-8 flex-1 items-center justify-center rounded-btn bg-tl-blue px-3 text-ui-sm font-semibold text-white transition-colors duration-150 hover:bg-blue-700 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/40"
              >
                Apply reply
              </button>
              <button
                type="button"
                onClick={copy}
                className="inline-flex h-8 items-center justify-center rounded-btn border border-tl-line bg-white px-3 text-ui-sm font-semibold text-tl-ink-soft transition-colors duration-150 hover:bg-tl-line-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
              >
                {copied ? "Copied" : "Copy"}
              </button>
            </div>
          </div>

          {ai.similar.length > 0 && (
            <div className="border-t border-tl-line-soft px-5 py-4">
              <FieldGroupLabel>Similar tickets</FieldGroupLabel>
              <ul className="space-y-2">
                {ai.similar.map((similar) => (
                  <li key={similar.id} className="flex items-center gap-2">
                    <Link
                      href={`/staff/tickets/${similar.id}`}
                      className="shrink-0 text-ui-sm font-semibold text-tl-blue hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
                    >
                      {similar.id}
                    </Link>
                    <span className="min-w-0 truncate text-ui-sm text-tl-ink-soft">
                      {similar.subject}
                    </span>
                    <StatusBadge
                      status={similar.status}
                      className="ml-auto shrink-0"
                    />
                  </li>
                ))}
              </ul>
            </div>
          )}
        </>
      )}
    </section>
  );
}
