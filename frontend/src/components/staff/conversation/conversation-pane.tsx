"use client";

import { useState } from "react";
import Link from "next/link";
import { ArrowLeft, PanelRightOpen } from "lucide-react";

import { cn } from "@/lib/utils";
import { relativeAge } from "@/lib/staff/format";
import type { Message, MessageKind, Ticket, TicketQuery } from "@/lib/staff/types";
import { inboxHref } from "@/lib/staff/url";
import { PriorityBadge, StatusBadge } from "../primitives";
import { Composer } from "./composer";
import { Thread } from "./thread";

/*
  Conversation pane: header, thread, composer.

  Owns the messages so the composer can actually post. New messages are local —
  there is no write endpoint behind the fixtures — but they behave like the real
  thing: they group, they scroll into view, and notes render as notes.
*/

const TABS = ["Conversation", "Internal notes", "Attachments", "Timeline"] as const;
type Tab = (typeof TABS)[number];

export function ConversationPane({
  ticket,
  query,
  ownId,
  ownName,
  ownInitials,
  onOpenDetails,
}: {
  ticket: Ticket;
  query: TicketQuery;
  ownId: string;
  ownName: string;
  ownInitials: string;
  /** Opens the details drawer on laptop and below. */
  onOpenDetails?: () => void;
}) {
  const [tab, setTab] = useState<Tab>("Conversation");
  const [extra, setExtra] = useState<Message[]>([]);
  const [draft, setDraft] = useState("");

  const messages = [...ticket.messages, ...extra];
  const notes = messages.filter((m) => m.kind === "note");
  const attachments = messages.flatMap((m) =>
    (m.attachments ?? []).map((a) => ({ attachment: a, message: m })),
  );

  const send = (body: string, kind: MessageKind) => {
    setExtra((prev) => [
      ...prev,
      {
        id: `local-${Date.now()}`,
        kind,
        authorId: ownId,
        authorName: ownName,
        authorInitials: ownInitials,
        body,
        sentAt: new Date().toISOString(),
        delivered: kind === "agent",
      },
    ]);
    setDraft("");
  };

  const visible =
    tab === "Internal notes"
      ? notes
      : tab === "Conversation"
        ? messages
        : [];

  return (
    <div className="flex h-full min-h-0 flex-col bg-tl-card">
      {/* Header */}
      <div className="shrink-0 border-b border-tl-line px-4 pt-3 lg:px-5">
        <div className="flex items-center gap-2.5">
          <Link
            href={inboxHref(query)}
            aria-label="Back to ticket list"
            className="-ml-1 inline-flex size-9 items-center justify-center rounded-btn text-tl-muted transition-colors duration-150 hover:bg-tl-line-soft hover:text-tl-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30 md:hidden"
          >
            <ArrowLeft className="size-5" aria-hidden />
          </Link>

          <span className="text-ui-base font-semibold text-tl-ink">
            {ticket.id}
          </span>
          <PriorityBadge priority={ticket.priority} />
          <StatusBadge status={ticket.status} />

          <button
            type="button"
            onClick={onOpenDetails}
            aria-label="Open ticket details"
            className="ml-auto inline-flex size-9 items-center justify-center rounded-btn border border-tl-line text-tl-muted transition-colors duration-150 hover:bg-tl-line-soft hover:text-tl-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30 xl:hidden"
          >
            <PanelRightOpen className="size-4" aria-hidden />
          </button>
        </div>

        <h1 className="mt-2 text-ui-xl font-bold leading-tight tracking-[-0.02em] text-tl-ink">
          {ticket.subject}
        </h1>

        <p className="mt-1.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-ui-sm text-tl-muted">
          <span className="font-medium text-tl-ink">{ticket.customer.name}</span>
          <span className="text-tl-faint-soft" aria-hidden>·</span>
          <span className="truncate">{ticket.customer.email}</span>
          <span className="text-tl-faint-soft" aria-hidden>·</span>
          <span>Opened {relativeAge(ticket.createdAt)}</span>
        </p>

        <div role="tablist" aria-label="Ticket sections" className="mt-3 flex gap-5 overflow-x-auto">
          {TABS.map((name) => {
            const active = name === tab;
            const count =
              name === "Internal notes"
                ? notes.length
                : name === "Attachments"
                  ? attachments.length
                  : undefined;

            return (
              <button
                key={name}
                type="button"
                role="tab"
                aria-selected={active}
                onClick={() => setTab(name)}
                className={cn(
                  "-mb-px flex shrink-0 items-center gap-1.5 border-b-2 pb-2.5 pt-1 text-ui-sm transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30",
                  active
                    ? "border-tl-blue font-semibold text-tl-blue"
                    : "border-transparent font-medium text-tl-muted hover:text-tl-ink",
                )}
              >
                {name}
                {count !== undefined && count > 0 && (
                  <span className="inline-flex h-[17px] min-w-[17px] items-center justify-center rounded-full bg-tl-line-soft px-1 text-ui-xs font-semibold text-tl-muted tabular-nums">
                    {count}
                  </span>
                )}
              </button>
            );
          })}
        </div>
      </div>

      {/* Body */}
      {tab === "Timeline" ? (
        <TimelinePanel ticket={ticket} />
      ) : tab === "Attachments" ? (
        <AttachmentsPanel items={attachments} />
      ) : (
        <Thread
          messages={visible}
          ownId={ownId}
          draftAuthor={
            draft.trim() ? { name: ownName, initials: ownInitials } : null
          }
        />
      )}

      {tab !== "Timeline" && tab !== "Attachments" && (
        <Composer
          customerName={ticket.customer.name.split(" ")[0]}
          onSend={send}
          onDraftChange={setDraft}
        />
      )}
    </div>
  );
}

function TimelinePanel({ ticket }: { ticket: Ticket }) {
  return (
    <div className="min-h-0 flex-1 overflow-y-auto px-5 py-6">
      <ol className="mx-auto max-w-[640px] space-y-0">
        {ticket.timeline.map((event, i) => (
          <li key={event.id} className="flex gap-3">
            <div className="flex flex-col items-center">
              <span className="mt-1.5 size-2 shrink-0 rounded-full bg-tl-blue" aria-hidden />
              {i < ticket.timeline.length - 1 && (
                <span className="w-px flex-1 bg-tl-line" aria-hidden />
              )}
            </div>
            <div className="pb-6">
              <p className="text-ui-base font-medium text-tl-ink">
                {event.summary}
              </p>
              <p className="mt-0.5 text-ui-xs text-tl-muted">
                {event.actor} · {relativeAge(event.at)}
              </p>
            </div>
          </li>
        ))}
      </ol>
    </div>
  );
}

function AttachmentsPanel({
  items,
}: {
  items: { attachment: { id: string; name: string; size: string }; message: Message }[];
}) {
  if (items.length === 0) {
    return (
      <div className="flex min-h-0 flex-1 items-center justify-center px-5 py-16 text-center">
        <p className="text-ui-base text-tl-muted">
          No attachments on this ticket.
        </p>
      </div>
    );
  }

  return (
    <div className="min-h-0 flex-1 overflow-y-auto px-5 py-6">
      <ul className="mx-auto grid max-w-[640px] gap-2">
        {items.map(({ attachment, message }) => (
          <li
            key={attachment.id}
            className="flex items-center gap-3 rounded-btn border border-tl-line px-4 py-3"
          >
            <div className="min-w-0 flex-1">
              <p className="truncate text-ui-base font-medium text-tl-ink">
                {attachment.name}
              </p>
              <p className="text-ui-xs text-tl-muted">
                {attachment.size} · from {message.authorName}
              </p>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
