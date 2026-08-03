import { CheckCheck, FileText, ImageIcon, Lock, Paperclip } from "lucide-react";

import { cn } from "@/lib/utils";
import { timeOnly } from "@/lib/staff/format";
import type { Attachment, Message } from "@/lib/staff/types";
import { InitialsAvatar } from "../primitives";

/*
  Message rendering.

  Bodies are plain strings and wrap naturally — `whitespace-pre-line` honours
  the deliberate newlines in things like a pasted request ID without hardcoding
  where a sentence breaks. The previous build stored one array entry per visual
  line, which looked right at exactly one width and nowhere else.
*/

const ATTACHMENT_ICON = {
  image: ImageIcon,
  document: FileText,
  log: Paperclip,
} as const;

function AttachmentPreview({ attachment }: { attachment: Attachment }) {
  const Icon = ATTACHMENT_ICON[attachment.kind];

  // A button, not a link: there is no download endpoint behind the fixtures,
  // and an anchor to "#" is announced as a link and focusable for nothing.
  return (
    <button
      type="button"
      aria-label={`Download ${attachment.name}, ${attachment.size}`}
      className="flex w-full items-center gap-2.5 rounded-btn border border-tl-line bg-white px-3 py-2 text-left transition-colors duration-150 hover:bg-tl-line-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
    >
      <span className="flex size-8 shrink-0 items-center justify-center rounded-md bg-tl-line-soft text-tl-muted">
        <Icon className="size-4" aria-hidden />
      </span>
      <span className="min-w-0">
        <span className="block truncate text-ui-sm font-medium text-tl-ink">
          {attachment.name}
        </span>
        <span className="block text-ui-xs text-tl-muted">{attachment.size}</span>
      </span>
    </button>
  );
}

/**
 * A run of consecutive messages from the same author.
 *
 * Grouping is what stops a three-line exchange from repeating the same avatar
 * and name three times; only the first bubble in a run is labelled.
 */
export function MessageGroup({
  messages,
  ownId,
}: {
  messages: Message[];
  /** The signed-in agent, whose messages align right. */
  ownId: string;
}) {
  const first = messages[0];

  if (first.kind === "note") {
    return (
      <>
        {messages.map((message) => (
          <InternalNote key={message.id} message={message} />
        ))}
      </>
    );
  }

  const outgoing = first.kind === "agent";

  return (
    <div className={cn("flex items-start gap-3", outgoing && "flex-row-reverse")}>
      <InitialsAvatar
        name={first.authorName}
        initials={first.authorInitials}
        size={32}
        className="mt-5"
      />

      <div
        className={cn(
          "flex min-w-0 max-w-[min(560px,80%)] flex-col gap-1",
          outgoing && "items-end",
        )}
      >
        <span className="px-1 text-ui-sm font-semibold text-tl-ink">
          {first.authorName}
          {outgoing && first.authorId === ownId && (
            <span className="ml-1.5 font-normal text-tl-faint">you</span>
          )}
        </span>

        {messages.map((message) => (
          <div
            key={message.id}
            className={cn(
              "w-fit max-w-full rounded-[14px] px-4 py-2.5",
              outgoing
                ? "rounded-tr-[4px] bg-tl-blue-soft text-tl-ink"
                : "rounded-tl-[4px] bg-tl-line-soft text-tl-ink",
            )}
          >
            <p className="whitespace-pre-line break-words text-ui-base leading-[1.6]">
              {message.body}
            </p>

            {message.attachments && message.attachments.length > 0 && (
              <div className="mt-2.5 space-y-1.5">
                {message.attachments.map((attachment) => (
                  <AttachmentPreview key={attachment.id} attachment={attachment} />
                ))}
              </div>
            )}

            <div
              className={cn(
                "mt-1 flex items-center gap-1 text-ui-xs text-tl-faint",
                outgoing && "justify-end",
              )}
            >
              <time dateTime={message.sentAt}>{timeOnly(message.sentAt)}</time>
              {message.delivered && (
                <CheckCheck className="size-3.5 text-tl-blue" aria-label="Delivered" />
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

/**
 * Internal notes never reach the customer, so they break the bubble pattern
 * entirely: full width, amber, padlocked. The visual difference is the safety
 * feature — an agent must never mistake one for a reply.
 */
export function InternalNote({ message }: { message: Message }) {
  return (
    <div className="rounded-[10px] border border-l-[3px] border-amber-200/70 border-l-tl-orange bg-amber-50/70 px-4 py-3">
      <div className="flex items-center gap-1.5">
        <Lock className="size-3 shrink-0 text-amber-700" aria-hidden />
        <span className="text-ui-xs text-amber-800">
          Internal note by{" "}
          <span className="font-semibold text-tl-ink">{message.authorName}</span>
        </span>
        <time
          dateTime={message.sentAt}
          className="ml-auto shrink-0 text-ui-xs text-tl-faint"
        >
          {timeOnly(message.sentAt)}
        </time>
      </div>
      <p className="mt-1.5 whitespace-pre-line break-words text-ui-base leading-[1.6] text-tl-ink">
        {message.body}
      </p>
    </div>
  );
}
