"use client";

import Link from "next/link";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useCallback } from "react";
import { AlertTriangle, Pencil, Search } from "lucide-react";

import { CategoryBadge, PriorityBadge, StatusBadge } from "@/components/triage/badges";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { Button } from "@/components/ui/button";
import { Input, Select } from "@/components/ui/field";
import { CenteredSpinner } from "@/components/ui/spinner";
import { useDepartments, useTickets } from "@/lib/api/hooks";
import {
  ALL_PRIORITIES,
  ALL_STATUSES,
  PRIORITY_LABELS,
  STATUS_LABELS,
} from "@/lib/api/labels";
import type {
  TicketFilters,
  TicketListItem,
  TicketPriority,
  TicketStatus,
} from "@/lib/api/types";
import { pct, relativeTime } from "@/lib/utils";

const PAGE_SIZE = 25;

function parseFilters(sp: URLSearchParams): TicketFilters {
  const num = (v: string | null) => (v ? Number(v) : undefined);
  return {
    status: sp.getAll("status") as TicketStatus[],
    priority: sp.getAll("priority") as TicketPriority[],
    department_id: sp.get("department_id") || undefined,
    assignee_id: sp.get("assignee_id") || undefined,
    needs_review: sp.get("needs_review") === "true" ? true : undefined,
    overridden: sp.get("overridden") === "true" ? true : undefined,
    q: sp.get("q") || undefined,
    sort: sp.get("sort") || "-created_at",
    page: num(sp.get("page")) || 1,
    page_size: PAGE_SIZE,
  };
}

function QueueInner() {
  const router = useRouter();
  const sp = useSearchParams();
  const filters = parseFilters(new URLSearchParams(sp.toString()));
  const { data, isLoading, isError } = useTickets(filters);
  const { data: departments } = useDepartments();

  // setParam rewrites one query param and resets to page 1 (except for page
  // itself), keeping the URL the single source of truth for the filter state.
  const setParam = useCallback(
    (key: string, value: string | null) => {
      const next = new URLSearchParams(sp.toString());
      if (value === null || value === "") next.delete(key);
      else next.set(key, value);
      if (key !== "page") next.delete("page");
      router.push(`/tickets?${next.toString()}`);
    },
    [router, sp],
  );

  const toggleMulti = useCallback(
    (key: string, value: string) => {
      const next = new URLSearchParams(sp.toString());
      const current = next.getAll(key);
      next.delete(key);
      if (current.includes(value)) {
        current.filter((v) => v !== value).forEach((v) => next.append(key, v));
      } else {
        [...current, value].forEach((v) => next.append(key, v));
      }
      next.delete("page");
      router.push(`/tickets?${next.toString()}`);
    },
    [router, sp],
  );

  const total = data?.meta.total ?? 0;
  const page = filters.page ?? 1;
  const totalPages = Math.max(1, Math.ceil(total / PAGE_SIZE));

  return (
    <div className="p-6">
      <div className="mb-5 flex items-center justify-between">
        <div>
          <h1 className="text-xl font-semibold">Ticket Queue</h1>
          <p className="text-sm text-muted-foreground">
            {total} ticket{total === 1 ? "" : "s"}
          </p>
        </div>
      </div>

      {/* Filter bar */}
      <Card className="mb-5 p-4">
        <div className="flex flex-wrap items-end gap-3">
          <div className="relative min-w-56 flex-1">
            <Search className="pointer-events-none absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              defaultValue={filters.q ?? ""}
              placeholder="Search subject or body…"
              className="pl-9"
              onKeyDown={(e) => {
                if (e.key === "Enter")
                  setParam("q", (e.target as HTMLInputElement).value);
              }}
            />
          </div>

          <Select
            value={filters.department_id ?? ""}
            onChange={(e) => setParam("department_id", e.target.value)}
            className="w-44"
          >
            <option value="">All departments</option>
            {departments?.data.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </Select>

          <Select
            value={filters.assignee_id ?? ""}
            onChange={(e) => setParam("assignee_id", e.target.value)}
            className="w-40"
          >
            <option value="">Any assignee</option>
            <option value="unassigned">Unassigned</option>
          </Select>

          <Select
            value={filters.sort ?? "-created_at"}
            onChange={(e) => setParam("sort", e.target.value)}
            className="w-44"
          >
            <option value="-created_at">Newest first</option>
            <option value="created_at">Oldest first</option>
            <option value="-priority">Priority high→low</option>
            <option value="priority">Priority low→high</option>
          </Select>
        </div>

        <div className="mt-3 flex flex-wrap items-center gap-2">
          {ALL_STATUSES.map((s) => (
            <FilterChip
              key={s}
              active={filters.status?.includes(s) ?? false}
              onClick={() => toggleMulti("status", s)}
            >
              {STATUS_LABELS[s]}
            </FilterChip>
          ))}
          <span className="mx-1 h-4 w-px bg-border" />
          {ALL_PRIORITIES.map((p) => (
            <FilterChip
              key={p}
              active={filters.priority?.includes(p) ?? false}
              onClick={() => toggleMulti("priority", p)}
            >
              {PRIORITY_LABELS[p]}
            </FilterChip>
          ))}
          <span className="mx-1 h-4 w-px bg-border" />
          <FilterChip
            active={filters.needs_review ?? false}
            onClick={() =>
              setParam("needs_review", filters.needs_review ? null : "true")
            }
          >
            Needs review
          </FilterChip>
          <FilterChip
            active={filters.overridden ?? false}
            onClick={() =>
              setParam("overridden", filters.overridden ? null : "true")
            }
          >
            Overridden
          </FilterChip>
        </div>
      </Card>

      {isLoading ? (
        <CenteredSpinner label="Loading tickets…" />
      ) : isError ? (
        <Card className="p-8 text-center text-sm text-red-500">
          Failed to load tickets.
        </Card>
      ) : data && data.data.length === 0 ? (
        <Card className="p-12 text-center text-sm text-muted-foreground">
          No tickets match these filters.
        </Card>
      ) : (
        <Card className="overflow-hidden">
          <table className="w-full text-sm">
            <thead>
              <tr className="border-b border-border bg-surface-muted/50 text-left text-xs uppercase tracking-wide text-muted-foreground">
                <th className="px-4 py-3 font-medium">Subject</th>
                <th className="px-4 py-3 font-medium">Priority</th>
                <th className="px-4 py-3 font-medium">Status</th>
                <th className="px-4 py-3 font-medium">Department</th>
                <th className="px-4 py-3 font-medium">AI</th>
                <th className="px-4 py-3 font-medium">Assignee</th>
                <th className="px-4 py-3 font-medium">Updated</th>
              </tr>
            </thead>
            <tbody>
              {data?.data.map((t) => <TicketRow key={t.id} ticket={t} />)}
            </tbody>
          </table>
        </Card>
      )}

      {/* Pagination */}
      {total > PAGE_SIZE && (
        <div className="mt-4 flex items-center justify-between text-sm text-muted-foreground">
          <span>
            Page {page} of {totalPages}
          </span>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={page <= 1}
              onClick={() => setParam("page", String(page - 1))}
            >
              Previous
            </Button>
            <Button
              variant="secondary"
              size="sm"
              disabled={page >= totalPages}
              onClick={() => setParam("page", String(page + 1))}
            >
              Next
            </Button>
          </div>
        </div>
      )}
    </div>
  );
}

function FilterChip({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      onClick={onClick}
      className={
        active
          ? "rounded-full bg-accent px-3 py-1 text-xs font-medium text-accent-foreground"
          : "rounded-full bg-surface-muted px-3 py-1 text-xs font-medium text-muted-foreground hover:text-foreground"
      }
    >
      {children}
    </button>
  );
}

function TicketRow({ ticket }: { ticket: TicketListItem }) {
  const a = ticket.latest_analysis;
  const priorityMismatch =
    a && a.predicted_priority && a.predicted_priority !== ticket.priority;

  return (
    <tr className="border-b border-border last:border-0 hover:bg-surface-muted/40">
      <td className="max-w-xs px-4 py-3">
        <Link
          href={`/tickets/${ticket.id}`}
          className="block truncate font-medium text-foreground hover:text-accent"
        >
          {ticket.subject}
        </Link>
        <div className="truncate text-xs text-muted-foreground">
          {ticket.customer.full_name} · {ticket.message_count} msg
        </div>
      </td>
      <td className="px-4 py-3">
        <div className="flex items-center gap-1">
          <PriorityBadge priority={ticket.priority} />
          {ticket.priority_overridden && (
            <Pencil className="h-3 w-3 text-muted-foreground" aria-label="overridden" />
          )}
        </div>
      </td>
      <td className="px-4 py-3">
        <StatusBadge status={ticket.status} />
      </td>
      <td className="px-4 py-3">
        <div className="flex items-center gap-1">
          <span className="text-foreground">{ticket.department.name}</span>
          {ticket.department_overridden && (
            <Pencil className="h-3 w-3 text-muted-foreground" aria-label="overridden" />
          )}
        </div>
      </td>
      <td className="px-4 py-3">
        {!a ? (
          <span className="text-xs text-muted-foreground">analyzing…</span>
        ) : (
          <div className="flex items-center gap-1.5">
            <CategoryBadge category={a.predicted_category} />
            <span className="font-mono text-xs text-muted-foreground">
              {pct(a.department_confidence)}
            </span>
            {a.needs_human_review && (
              <AlertTriangle
                className="h-3.5 w-3.5 text-amber-500"
                aria-label="needs review"
              />
            )}
            {priorityMismatch && (
              <span
                className="text-xs text-muted-foreground"
                title={`AI predicted ${a.predicted_priority}`}
              >
                (AI: {a.predicted_priority})
              </span>
            )}
          </div>
        )}
      </td>
      <td className="px-4 py-3 text-xs">
        {ticket.assignee ? (
          <span className="text-foreground">{ticket.assignee.full_name}</span>
        ) : (
          <Badge>Unassigned</Badge>
        )}
      </td>
      <td className="px-4 py-3 text-xs text-muted-foreground">
        {relativeTime(ticket.updated_at)}
      </td>
    </tr>
  );
}

export default function TicketsPage() {
  return (
    <Suspense fallback={<CenteredSpinner />}>
      <QueueInner />
    </Suspense>
  );
}
