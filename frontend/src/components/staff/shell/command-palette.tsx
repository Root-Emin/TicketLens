"use client";

import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { Search } from "lucide-react";

import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
  CommandSeparator,
} from "@/components/shadcn/command";
import type { PaletteTicket } from "@/lib/staff/palette";
import { PRIORITY_LABEL, PRIORITY_ACCENT, DOT } from "../primitives";
import { ALL_DESTINATIONS } from "./nav-config";
import { cn } from "@/lib/utils";

export function CommandPalette({
  open,
  onOpenChange,
  tickets,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  tickets: PaletteTicket[];
}) {
  const router = useRouter();

  const go = (href: string) => {
    onOpenChange(false);
    router.push(href);
  };

  return (
    <CommandDialog
      open={open}
      onOpenChange={onOpenChange}
      title="Command palette"
      description="Search tickets and jump to any view"
    >
      <CommandInput placeholder="Search tickets, views and settings…" />
      <CommandList>
        <CommandEmpty>No results found.</CommandEmpty>

        <CommandGroup heading="Tickets">
          {tickets.map((ticket) => (
            <CommandItem
              key={ticket.id}
              // cmdk filters on this string, so it must carry everything a
              // person might type: id, subject and customer name.
              value={`${ticket.id} ${ticket.subject} ${ticket.customer}`}
              onSelect={() => go(`/staff/tickets/${ticket.id}`)}
            >
              <span
                className={cn(
                  "size-1.5 shrink-0 rounded-full",
                  DOT[PRIORITY_ACCENT[ticket.priority]],
                )}
                aria-hidden
              />
              <span className="font-medium">{ticket.id}</span>
              <span className="truncate">{ticket.subject}</span>
              <span className="ml-auto shrink-0 text-ui-xs text-muted-foreground">
                {PRIORITY_LABEL[ticket.priority]}
              </span>
            </CommandItem>
          ))}
        </CommandGroup>

        <CommandSeparator />

        <CommandGroup heading="Go to">
          {ALL_DESTINATIONS.map((link) => {
            const Icon = link.icon;
            return (
              <CommandItem
                key={link.href + link.label}
                value={`go ${link.label}`}
                onSelect={() => go(link.href)}
              >
                <Icon className="size-4 shrink-0" aria-hidden />
                {link.label}
              </CommandItem>
            );
          })}
        </CommandGroup>
      </CommandList>
    </CommandDialog>
  );
}

/**
 * The header's search affordance. It is a button, not an input: typing here
 * always opens the palette, and a text field that silently refuses keystrokes
 * is worse than a control that looks like what it does.
 */
export function CommandTrigger({ onOpen }: { onOpen: () => void }) {
  return (
    <button
      type="button"
      onClick={onOpen}
      className="group flex h-10 w-full items-center gap-2.5 rounded-btn border border-tl-line bg-white px-3 text-left text-ui-base text-tl-faint transition-colors duration-150 hover:border-slate-300 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30 md:w-[240px] lg:w-[300px]"
    >
      <Search className="size-4 shrink-0" aria-hidden />
      <span className="truncate">Search tickets…</span>
      <kbd className="ml-auto hidden shrink-0 rounded border border-tl-line bg-tl-line-soft px-1.5 py-0.5 font-sans text-ui-xs font-medium text-tl-muted lg:inline">
        ⌘K
      </kbd>
    </button>
  );
}

/**
 * Binds ⌘K / Ctrl-K globally and returns the palette's open state.
 *
 * Intentionally unconditional, including while the composer has focus: a
 * global search shortcut that stops working once you start typing is the one
 * time you most want it.
 */
export function useCommandPalette() {
  const [open, setOpen] = useState(false);

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== "k" || !(event.metaKey || event.ctrlKey)) return;
      event.preventDefault();
      setOpen((prev) => !prev);
    };

    document.addEventListener("keydown", onKeyDown);
    return () => document.removeEventListener("keydown", onKeyDown);
  }, []);

  return { open, setOpen };
}
