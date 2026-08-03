"use client";

import { useRef, useState } from "react";
import { ImageIcon, Lock, Paperclip, Send, Smile } from "lucide-react";

import { cn } from "@/lib/utils";
import type { MessageKind } from "@/lib/staff/types";

/*
  The reply composer.

  Two modes, and they look different on purpose: a reply goes to the customer,
  a note does not. Switching to Note tints the whole surface amber so the
  distinction is visible while typing, not just in the tab label.

  Not pinned to the viewport — it sits at the end of the thread pane and scrolls
  with its own column, which is what the brief asked for.
*/

const MODES: { id: Exclude<MessageKind, "customer">; label: string }[] = [
  { id: "agent", label: "Reply" },
  { id: "note", label: "Internal note" },
];

export function Composer({
  customerName,
  onSend,
  onDraftChange,
}: {
  customerName: string;
  onSend?: (body: string, kind: MessageKind) => void;
  /** Lets the thread show where an in-progress reply will land. */
  onDraftChange?: (draft: string) => void;
}) {
  const [mode, setMode] = useState<Exclude<MessageKind, "customer">>("agent");
  const [value, setValue] = useState("");
  const textarea = useRef<HTMLTextAreaElement>(null);
  const isNote = mode === "note";

  const update = (next: string) => {
    setValue(next);
    onDraftChange?.(next);
  };

  const submit = () => {
    const body = value.trim();
    if (!body) return;
    onSend?.(body, mode);
    update("");
    textarea.current?.focus();
  };

  // ⌘/Ctrl+Enter sends, matching every other tool an agent has open.
  const onKeyDown = (event: React.KeyboardEvent<HTMLTextAreaElement>) => {
    if (event.key === "Enter" && (event.metaKey || event.ctrlKey)) {
      event.preventDefault();
      submit();
    }
  };

  return (
    <div
      className={cn(
        "shrink-0 border-t px-4 py-3 transition-colors duration-200",
        isNote ? "border-amber-200 bg-amber-50/50" : "border-tl-line bg-tl-card",
      )}
    >
      <div
        role="tablist"
        aria-label="Reply mode"
        className="mb-2 flex items-center gap-1"
      >
        {MODES.map((option) => {
          const active = option.id === mode;
          return (
            <button
              key={option.id}
              type="button"
              role="tab"
              aria-selected={active}
              onClick={() => setMode(option.id)}
              className={cn(
                "inline-flex items-center gap-1.5 rounded-lg px-2.5 py-1 text-ui-sm font-medium transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30",
                active
                  ? option.id === "note"
                    ? "bg-amber-100 text-amber-800"
                    : "bg-tl-blue-soft text-tl-blue"
                  : "text-tl-muted hover:bg-tl-line-soft hover:text-tl-ink",
              )}
            >
              {option.id === "note" && <Lock className="size-3" aria-hidden />}
              {option.label}
            </button>
          );
        })}
      </div>

      <textarea
        ref={textarea}
        rows={3}
        value={value}
        onChange={(e) => update(e.target.value)}
        onKeyDown={onKeyDown}
        placeholder={
          isNote
            ? "Add a note only your team can see…"
            : `Reply to ${customerName}…`
        }
        aria-label={isNote ? "Internal note" : `Reply to ${customerName}`}
        className={cn(
          "block w-full resize-none rounded-btn border bg-white px-3 py-2.5 text-ui-base leading-[1.6] text-tl-ink outline-none transition-colors duration-150 placeholder:text-tl-faint focus:ring-2",
          isNote
            ? "border-amber-300 focus:border-amber-400 focus:ring-amber-200/40"
            : "border-tl-line focus:border-tl-blue focus:ring-tl-blue/15",
        )}
      />

      <div className="mt-2 flex items-center gap-1">
        <ToolbarButton label="Attach file">
          <Paperclip className="size-[18px]" strokeWidth={1.9} aria-hidden />
        </ToolbarButton>
        <ToolbarButton label="Insert emoji">
          <Smile className="size-[18px]" strokeWidth={1.9} aria-hidden />
        </ToolbarButton>
        <ToolbarButton label="Insert image">
          <ImageIcon className="size-[18px]" strokeWidth={1.9} aria-hidden />
        </ToolbarButton>

        <span className="ml-auto hidden text-ui-xs text-tl-faint sm:inline">
          ⌘↵ to send
        </span>

        <button
          type="button"
          onClick={submit}
          disabled={!value.trim()}
          className={cn(
            "ml-2 inline-flex h-9 items-center gap-2 rounded-btn px-4 text-ui-base font-semibold text-white transition-colors duration-150 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/40",
            "disabled:cursor-not-allowed disabled:opacity-40",
            isNote ? "bg-amber-600 hover:bg-amber-700" : "bg-tl-blue hover:bg-blue-700",
          )}
        >
          <Send className="size-4" strokeWidth={2} aria-hidden />
          {isNote ? "Add note" : "Send"}
        </button>
      </div>
    </div>
  );
}

function ToolbarButton({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      aria-label={label}
      className="tap-target inline-flex size-9 items-center justify-center rounded-lg text-tl-muted transition-colors duration-150 hover:bg-tl-line-soft hover:text-tl-ink focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
    >
      {children}
    </button>
  );
}
