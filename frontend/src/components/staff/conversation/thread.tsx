"use client";

import { useEffect, useLayoutEffect, useRef } from "react";

import { dayHeading } from "@/lib/staff/format";
import type { Message } from "@/lib/staff/types";
import { InitialsAvatar } from "../primitives";
import { MessageGroup } from "./message-bubble";

/*
  The message thread.

  Two behaviours worth spelling out:

  1. Grouping — consecutive messages from the same author within five minutes
     render as one run with a single avatar and name. Notes never group with
     replies.

  2. Scroll anchoring — the thread jumps to the newest message on open, but a
     new message only pulls the view down if the agent was already near the
     bottom. Yanking someone away from history they are reading is the classic
     chat bug, and it is entirely avoidable.
*/

const GROUP_WINDOW_MS = 5 * 60 * 1000;
const NEAR_BOTTOM_PX = 120;

interface Block {
  key: string;
  day: string;
  groups: Message[][];
}

/** Buckets messages by day, then into same-author runs within each day. */
function buildBlocks(messages: Message[]): Block[] {
  const blocks: Block[] = [];

  for (const message of messages) {
    const day = dayHeading(message.sentAt);
    let block = blocks.at(-1);

    if (!block || block.day !== day) {
      block = { key: `${day}-${message.id}`, day, groups: [] };
      blocks.push(block);
    }

    const lastGroup = block.groups.at(-1);
    const previous = lastGroup?.at(-1);

    const continues =
      previous !== undefined &&
      previous.authorId === message.authorId &&
      previous.kind === message.kind &&
      message.kind !== "note" &&
      Date.parse(message.sentAt) - Date.parse(previous.sentAt) < GROUP_WINDOW_MS;

    if (continues && lastGroup) lastGroup.push(message);
    else block.groups.push([message]);
  }

  return blocks;
}

export function Thread({
  messages,
  ownId,
  draftAuthor,
}: {
  messages: Message[];
  ownId: string;
  /** Shows a live indicator while the agent has an unsent draft. */
  draftAuthor?: { name: string; initials: string } | null;
}) {
  const scroller = useRef<HTMLDivElement>(null);
  const wasNearBottom = useRef(true);
  const blocks = buildBlocks(messages);

  // Record whether we were pinned to the bottom *before* React commits the new
  // message, since the measurement is meaningless afterwards.
  useLayoutEffect(() => {
    const el = scroller.current;
    if (!el) return;
    wasNearBottom.current =
      el.scrollHeight - el.scrollTop - el.clientHeight < NEAR_BOTTOM_PX;
  });

  // Jump to the newest message on first paint, without animating.
  useEffect(() => {
    const el = scroller.current;
    if (el) el.scrollTop = el.scrollHeight;
  }, []);

  useEffect(() => {
    const el = scroller.current;
    if (!el || !wasNearBottom.current) return;
    el.scrollTo({ top: el.scrollHeight, behavior: "smooth" });
  }, [messages.length]);

  return (
    <div
      ref={scroller}
      className="min-h-0 flex-1 overflow-y-auto overscroll-contain px-4 py-5 lg:px-5"
    >
      <div className="mx-auto flex max-w-[860px] flex-col gap-6">
        {blocks.map((block) => (
          <section key={block.key} className="flex flex-col gap-5">
            <div className="flex items-center gap-3">
              <span className="h-px flex-1 bg-tl-line" aria-hidden />
              <span className="text-ui-xs font-medium text-tl-faint">
                {block.day}
              </span>
              <span className="h-px flex-1 bg-tl-line" aria-hidden />
            </div>

            {block.groups.map((group) => (
              <MessageGroup key={group[0].id} messages={group} ownId={ownId} />
            ))}
          </section>
        ))}

        {draftAuthor && <DraftIndicator author={draftAuthor} />}
      </div>
    </div>
  );
}

/**
 * Shows where the message being typed will land. Driven by the composer's
 * actual draft rather than a simulated presence signal, so it never claims
 * something the app does not know.
 */
function DraftIndicator({
  author,
}: {
  author: { name: string; initials: string };
}) {
  return (
    <div className="flex flex-row-reverse items-center gap-3" aria-live="polite">
      <InitialsAvatar name={author.name} initials={author.initials} size={32} />
      <div className="flex items-center gap-1.5 rounded-[14px] rounded-tr-[4px] bg-tl-blue-soft px-4 py-3">
        <span className="sr-only">You are writing a reply</span>
        {[0, 1, 2].map((i) => (
          <span
            key={i}
            className="size-1.5 animate-bounce rounded-full bg-tl-blue/60"
            style={{ animationDelay: `${i * 120}ms`, animationDuration: "900ms" }}
            aria-hidden
          />
        ))}
      </div>
    </div>
  );
}
