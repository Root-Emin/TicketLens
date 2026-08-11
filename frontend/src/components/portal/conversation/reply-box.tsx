"use client";

import { useRef, useState } from "react";
import { CheckCircle2, RotateCcw, Send } from "lucide-react";

import { ApiError } from "@/lib/api/client";
import { ActionButton, FormError, useToast } from "@/components/portal/primitives";
import { Spinner } from "@/components/ui/spinner";
import { useReopenTicket, useReplyToTicket } from "@/lib/portal/hooks";
import { cn } from "@/lib/utils";

/*
  The reply box.

  On a resolved ticket the textarea is genuinely disabled rather than hidden:
  the customer needs to see that the conversation is closed and why, and a
  vanished composer reads as a bug. Reopening is offered in its place.

  Reopen calls PATCH /tickets/{id} with a status, which the contract grants to
  `ticket:update` — staff only. Until the backend offers something narrower this
  will answer 403, so the button explains that rather than failing silently.
*/

export function ReplyBox({
  ticketId,
  resolved,
}: {
  ticketId: string;
  resolved: boolean;
}) {
  const [value, setValue] = useState("");
  const [error, setError] = useState<string | null>(null);
  const textarea = useRef<HTMLTextAreaElement>(null);
  const toast = useToast();

  const reply = useReplyToTicket(ticketId);
  const reopen = useReopenTicket(ticketId);

  async function submit() {
    const body = value.trim();
    if (!body || reply.isPending) return;

    setError(null);
    try {
      await reply.mutateAsync(body);
      setValue("");
      textarea.current?.focus();
    } catch (caught) {
      setError(describe(caught, "send your reply"));
    }
  }

  async function onReopen() {
    setError(null);
    try {
      await reopen.mutateAsync();
      toast.success("Ticket reopened. Support will pick it up again.");
    } catch (caught) {
      setError(describe(caught, "reopen this ticket"));
    }
  }

  // ⌘/Ctrl+Enter sends, matching the agent composer and every other tool the
  // person is likely to have open.
  function onKeyDown(event: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      void submit();
    }
  }

  return (
    <div className="border-t border-tl-line bg-tl-card px-4 py-4 sm:px-5">
      {resolved && (
        <div className="mb-3 flex flex-wrap items-center gap-3 rounded-btn bg-tl-green-soft px-3.5 py-2.5">
          <CheckCircle2
            className="size-[18px] shrink-0 text-tl-green-ink"
            strokeWidth={2}
            aria-hidden
          />
          <p className="min-w-0 flex-1 text-ui-sm text-tl-ink-soft">
            This request has been resolved. Reopen it if you still need help.
          </p>
          <ActionButton
            variant="secondary"
            size="sm"
            onClick={onReopen}
            disabled={reopen.isPending}
            aria-busy={reopen.isPending}
          >
            {reopen.isPending ? (
              <Spinner className="size-3.5" />
            ) : (
              <RotateCcw className="size-3.5" aria-hidden />
            )}
            Reopen
          </ActionButton>
        </div>
      )}

      <label htmlFor="ticket-reply" className="sr-only">
        Write a reply
      </label>
      <textarea
        id="ticket-reply"
        ref={textarea}
        rows={3}
        value={value}
        disabled={resolved || reply.isPending}
        onChange={(event) => setValue(event.target.value)}
        onKeyDown={onKeyDown}
        placeholder={
          resolved
            ? "This conversation is closed."
            : "Add anything that might help us…"
        }
        className={cn(
          "block w-full resize-none rounded-btn border bg-white px-3.5 py-2.5 text-ui-base leading-[1.6] text-tl-ink outline-none transition-colors duration-150 placeholder:text-tl-faint focus:ring-2",
          "border-tl-line focus:border-tl-blue focus:ring-tl-blue/15",
          "disabled:cursor-not-allowed disabled:bg-tl-line-soft/60 disabled:text-tl-muted",
        )}
      />

      {error && <div className="mt-3">{<FormError>{error}</FormError>}</div>}

      <div className="mt-2.5 flex items-center gap-2">
        <span className="hidden text-ui-xs text-tl-faint sm:inline">
          ⌘↵ to send
        </span>
        <ActionButton
          onClick={submit}
          disabled={resolved || !value.trim() || reply.isPending}
          aria-busy={reply.isPending}
          className="ml-auto"
        >
          {reply.isPending ? (
            <Spinner className="size-4" />
          ) : (
            <Send className="size-4" strokeWidth={2} aria-hidden />
          )}
          {reply.isPending ? "Sending…" : "Send"}
        </ActionButton>
      </div>
    </div>
  );
}

function describe(error: unknown, action: string): string {
  if (error instanceof ApiError) {
    if (error.status === 403) {
      return `Your account isn't allowed to ${action}. Please contact support.`;
    }
    if (error.status === 404) return "This ticket is no longer available.";
    if (error.status >= 500) {
      return `We couldn't ${action} just now. Please try again shortly.`;
    }
    return error.message;
  }
  return `We couldn't ${action}. Please try again.`;
}
