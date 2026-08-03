"use client";

import { Check, ChevronDown } from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/shadcn/dropdown-menu";
import { FieldGroupLabel } from "../primitives";

/** A label/value pair in the details rail. */
export function Row({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-h-8 items-center justify-between gap-3">
      <span className="shrink-0 text-ui-sm text-tl-muted">{label}</span>
      <div className="flex min-w-0 items-center gap-1.5">{children}</div>
    </div>
  );
}

/** A titled group of rows, divided from its neighbours by a single hairline. */
export function Section({
  label,
  children,
}: {
  label: string;
  children: React.ReactNode;
}) {
  return (
    <div className="border-b border-tl-line-soft px-5 py-4 last:border-b-0">
      <FieldGroupLabel>{label}</FieldGroupLabel>
      <div className="space-y-1">{children}</div>
    </div>
  );
}

/**
 * A value that opens a menu.
 *
 * The chevron appears here and nowhere else — in the previous build every row
 * wore one, which promised an editor on nine fields and delivered none.
 */
export function EditableValue({
  current,
  options,
  onSelect,
  children,
}: {
  current: string;
  options: { id: string; label: string }[];
  onSelect: (id: string) => void;
  children: React.ReactNode;
}) {
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button
          type="button"
          aria-label={`Change ${current}`}
          className="inline-flex items-center gap-1.5 rounded-md px-1.5 py-1 transition-colors duration-150 hover:bg-tl-line-soft focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
        >
          {children}
          <ChevronDown className="size-3.5 shrink-0 text-tl-faint" aria-hidden />
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-44">
        {options.map((option) => (
          <DropdownMenuItem
            key={option.id}
            onSelect={() => onSelect(option.id)}
            className="justify-between"
          >
            {option.label}
            {option.label === current && <Check className="size-4" aria-hidden />}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
