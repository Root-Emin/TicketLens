"use client";

import { Headset } from "lucide-react";

import { InitialsAvatar } from "@/components/staff/primitives";
import { dayHeading, initialsOf, timeOnly } from "@/lib/portal/format";
import type { MessageInfo } from "@/lib/api/types";
import { cn } from "@/lib/utils";

/*
  The conversation.

  Two rules matter here and neither is cosmetic.

  1. Internal notes never render. The backend contract says a customer token
     receives no message with is_internal = true, but this component filters
     again on the way in — the portal is the last place that mistake can be
     caught, and the cost of catching it twice is one predicate. Nothing below
     `visibleMessages` can leak a note, because nothing below ever sees one.

  2. The customer's own messages sit right in the brand tint; support sits left
     on a white card. Side alone carries the authorship, so a bubble never has
     to say "you".

  Consecutive messages from the same author are grouped: a three-line reply
  should not repeat the same avatar three times.
*/

/** Everything the customer is allowed to see, in the order it was written. */
export function visibleMessages(messages: MessageInfo[]): MessageInfo[] {
  return messages
    .filter((message) => !message.is_internal)
    .slice()
    .sort((a, b) => Date.parse(a.created_at) - Date.parse(b.created_at));
}

interface Group {
  key: string;
  outgoing: boolean;
  system: boolean;
  /** The day separator this group opens with, if it starts a new day. */
  dayHeading: string | null;
  messages: MessageInfo[];
}

/**
 * Splits the thread into runs of one author, tagging the first run of each day
 * with its separator.
 *
 * The day is resolved here rather than while rendering: a `let` carried across
 * a `.map()` is a render-order dependency, and React makes no promise about
 * that.
 */
function groupMessages(messages: MessageInfo[]): Group[] {
  const groups: Group[] = [];
  let lastDay: string | null = null;

  for (const message of messages) {
    const outgoing = message.author_type === "customer";
    const system = message.author_type === "system";
    const last = groups[groups.length - 1];

    if (last && last.outgoing === outgoing && last.system === system) {
      last.messages.push(message);
      continue;
    }

    const heading = dayHeading(message.created_at);
    groups.push({
      key: message.id,
      outgoing,
      system,
      dayHeading: heading === lastDay ? null : heading,
      messages: [message],
    });
    lastDay = heading;
  }

  return groups;
}

export function Thread({
  messages,
  customerName,
}: {
  messages: MessageInfo[];
  customerName: string;
}) {
  const groups = groupMessages(visibleMessages(messages));

  return (
    <div className="space-y-5 px-4 py-5 sm:px-5">
      {groups.map((group) => (
        <div key={group.key} className="space-y-5">
          {group.dayHeading && <DaySeparator label={group.dayHeading} />}
          {group.system ? (
            <SystemNotice messages={group.messages} />
          ) : (
            <MessageGroup group={group} customerName={customerName} />
          )}
        </div>
      ))}
    </div>
  );
}

function DaySeparator({ label }: { label: string }) {
  return (
    <div className="flex items-center gap-3">
      <span className="h-px flex-1 bg-tl-line" aria-hidden />
      <span className="text-ui-xs font-medium text-tl-faint">{label}</span>
      <span className="h-px flex-1 bg-tl-line" aria-hidden />
    </div>
  );
}

function MessageGroup({
  group,
  customerName,
}: {
  group: Group;
  customerName: string;
}) {
  const { outgoing, messages } = group;
  const name = outgoing ? customerName : "Support";

  return (
    <div className={cn("flex items-start gap-3", outgoing && "flex-row-reverse")}>
      {outgoing ? (
        <InitialsAvatar
          name={customerName}
          initials={initialsOf(customerName)}
          size={32}
          className="mt-5"
        />
      ) : (
        <span
          className="mt-5 flex size-8 shrink-0 items-center justify-center rounded-full bg-tl-blue-soft text-tl-blue"
          aria-hidden
        >
          <Headset className="size-4" strokeWidth={1.9} />
        </span>
      )}

      <div
        className={cn(
          "flex min-w-0 max-w-[min(560px,84%)] flex-col gap-1",
          outgoing && "items-end",
        )}
      >
        <span className="px-1 text-ui-sm font-semibold text-tl-ink">{name}</span>

        {messages.map((message) => (
          <div
            key={message.id}
            className={cn(
              "w-fit max-w-full rounded-[14px] px-4 py-2.5",
              outgoing
                ? "rounded-tr-[4px] bg-tl-blue text-white"
                : "rounded-tl-[4px] border border-tl-line bg-tl-card text-tl-ink",
            )}
          >
            <p className="whitespace-pre-line break-words text-ui-base leading-[1.6]">
              {message.body}
            </p>
            <time
              dateTime={message.created_at}
              className={cn(
                "mt-1 block text-ui-xs",
                outgoing ? "text-white/70 text-right" : "text-tl-faint",
              )}
            >
              {timeOnly(message.created_at)}
            </time>
          </div>
        ))}
      </div>
    </div>
  );
}

/** Status changes and other automated entries, centred and unattributed. */
function SystemNotice({ messages }: { messages: MessageInfo[] }) {
  return (
    <div className="space-y-2">
      {messages.map((message) => (
        <p
          key={message.id}
          className="mx-auto w-fit max-w-full rounded-full bg-tl-line-soft px-3.5 py-1.5 text-center text-ui-xs text-tl-muted"
        >
          {message.body}
        </p>
      ))}
    </div>
  );
}
