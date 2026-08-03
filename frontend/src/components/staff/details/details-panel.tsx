"use client";

import { useState } from "react";
import { Calendar, Plus, Timer, X } from "lucide-react";

import { cn } from "@/lib/utils";
import { fullDate, slaRemaining } from "@/lib/staff/format";
import { teamLabel } from "@/lib/staff/fixtures";
import type { Priority, Ticket, TicketStatus } from "@/lib/staff/types";
import {
  InitialsAvatar,
  PriorityBadge,
  PRIORITY_LABEL,
  StatusBadge,
  STATUS_LABEL,
} from "../primitives";
import { AiCard } from "./ai-card";
import { CustomerCard, HistoryCard } from "./customer-card";
import { EditableValue, Row, Section } from "./field-row";

/*
  The details rail.

  Grouped into labelled sections — Status, Assignment, Classification, SLA,
  Tags — instead of the flat run of nine rows this replaces. Each group is a
  divider apart rather than a box inside a box, which is most of what made the
  old panel feel crowded.
*/

const PRIORITIES: Priority[] = ["high", "medium", "low"];
const STATUSES: TicketStatus[] = [
  "open",
  "pending",
  "on_hold",
  "resolved",
  "closed",
];

export function DetailsPanel({ ticket }: { ticket: Ticket }) {
  // Session-only edits: there is no write endpoint behind the fixtures yet.
  // They are real state rather than a fake animation — the panel reflects them
  // until the page reloads, and this is the seam a mutation slots into.
  const [status, setStatus] = useState(ticket.status);
  const [priority, setPriority] = useState(ticket.priority);
  const [tags, setTags] = useState(ticket.tags);

  const sla = slaRemaining(ticket.createdAt, ticket.slaDueAt);

  return (
    <div className="h-full min-h-0 overflow-y-auto overscroll-contain bg-tl-canvas">
      <div className="space-y-4 p-4">
        <section className="rounded-card border border-tl-line bg-tl-card shadow-panel">
          <Section label="Status">
            <Row label="Status">
              <EditableValue
                current={STATUS_LABEL[status]}
                options={STATUSES.map((s) => ({ id: s, label: STATUS_LABEL[s] }))}
                onSelect={(id) => setStatus(id as TicketStatus)}
              >
                <StatusBadge status={status} />
              </EditableValue>
            </Row>
            <Row label="Priority">
              <EditableValue
                current={PRIORITY_LABEL[priority]}
                options={PRIORITIES.map((p) => ({
                  id: p,
                  label: PRIORITY_LABEL[p],
                }))}
                onSelect={(id) => setPriority(id as Priority)}
              >
                <PriorityBadge priority={priority} />
              </EditableValue>
            </Row>
          </Section>

          <Section label="Assignment">
            <Row label="Assignee">
              {ticket.assignee ? (
                <>
                  <InitialsAvatar
                    name={ticket.assignee.name}
                    initials={ticket.assignee.initials}
                    size={20}
                  />
                  <span className="truncate text-ui-sm font-medium text-tl-ink">
                    {ticket.assignee.name}
                  </span>
                </>
              ) : (
                <span className="text-ui-sm font-medium text-tl-orange-ink">
                  Unassigned
                </span>
              )}
            </Row>
            <Row label="Team">
              <span className="truncate text-ui-sm font-medium text-tl-ink">
                {teamLabel[ticket.team]}
              </span>
            </Row>
          </Section>

          <Section label="Classification">
            <Row label="Category">
              <span className="truncate text-ui-sm font-medium text-tl-ink">
                {ticket.category}
              </span>
            </Row>
            <Row label="Subcategory">
              <span className="truncate text-ui-sm font-medium text-tl-ink">
                {ticket.subcategory}
              </span>
            </Row>
          </Section>

          <Section label="SLA">
            <Row label="Target">
              <span className="inline-flex items-center gap-1.5 text-ui-sm font-medium text-tl-ink">
                <Calendar className="size-3.5 text-tl-faint" aria-hidden />
                {fullDate(ticket.slaDueAt)}
              </span>
            </Row>

            <div className="pt-1">
              <div className="mb-1.5 flex items-center justify-between">
                <span className="inline-flex items-center gap-1.5 text-ui-sm text-tl-muted">
                  <Timer className="size-3.5" aria-hidden />
                  Time remaining
                </span>
                <span
                  className={cn(
                    "text-ui-sm font-semibold",
                    sla.breached ? "text-tl-red-ink" : "text-tl-green-ink",
                  )}
                >
                  {sla.label}
                </span>
              </div>
              <div
                role="progressbar"
                aria-valuenow={Math.round(sla.ratio * 100)}
                aria-valuemin={0}
                aria-valuemax={100}
                aria-label="SLA window elapsed"
                className="h-1.5 overflow-hidden rounded-full bg-tl-line-soft"
              >
                <div
                  className={cn(
                    "h-full rounded-full transition-[width] duration-200",
                    sla.breached
                      ? "bg-tl-red"
                      : sla.ratio > 0.75
                        ? "bg-tl-orange"
                        : "bg-tl-green",
                  )}
                  style={{ width: `${Math.max(4, sla.ratio * 100)}%` }}
                />
              </div>
            </div>
          </Section>

          <Section label="Tags">
            <div className="flex flex-wrap gap-1.5">
              {tags.map((tag) => (
                <span
                  key={tag}
                  className="inline-flex items-center gap-1 rounded-md bg-tl-line-soft px-2 py-1 text-ui-xs font-medium text-tl-ink-soft"
                >
                  {tag}
                  <button
                    type="button"
                    onClick={() =>
                      setTags((prev) => prev.filter((t) => t !== tag))
                    }
                    aria-label={`Remove tag ${tag}`}
                    className="tap-target rounded text-tl-faint transition-colors duration-150 hover:text-tl-red focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
                  >
                    <X className="size-3" strokeWidth={2.4} aria-hidden />
                  </button>
                </span>
              ))}
              <button
                type="button"
                aria-label="Add tag"
                className="tap-target inline-flex size-[26px] items-center justify-center rounded-md border border-dashed border-tl-line text-tl-muted transition-colors duration-150 hover:bg-tl-line-soft hover:text-tl-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
              >
                <Plus className="size-3.5" strokeWidth={2.4} aria-hidden />
              </button>
            </div>
          </Section>
        </section>

        <CustomerCard customer={ticket.customer} />
        <AiCard ticket={ticket} />
        <HistoryCard timeline={ticket.timeline} />
      </div>
    </div>
  );
}
