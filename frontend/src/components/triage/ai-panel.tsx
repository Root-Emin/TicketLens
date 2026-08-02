"use client";

import { RefreshCw, Sparkles } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardBody, CardHeader, CardTitle } from "@/components/ui/card";
import { Spinner } from "@/components/ui/spinner";
import { CategoryBadge, PriorityBadge } from "@/components/triage/badges";
import { ConfidenceBar } from "@/components/triage/confidence-bar";
import { useReanalyze } from "@/lib/api/hooks";
import { CATEGORY_LABELS, PRIORITY_LABELS } from "@/lib/api/labels";
import type { Category, TicketDetail, TicketPriority } from "@/lib/api/types";
import { formatDate } from "@/lib/utils";

/*
  The AI panel is the heart of the product: it makes the model's decision, its
  confidence, and the human's correction all legible at a glance. Predicted and
  effective values sit side by side, priority and category confidence are shown
  as separate bars (never a single blended score), and the append-only analysis
  history lets model versions be compared over time.
*/
export function AIPanel({ ticket }: { ticket: TicketDetail }) {
  const reanalyze = useReanalyze(ticket.id);
  const latest = ticket.analyses[0] ?? null;

  return (
    <Card>
      <CardHeader className="flex items-center justify-between">
        <CardTitle className="flex items-center gap-2">
          <Sparkles className="h-4 w-4 text-accent" />
          AI Analysis
        </CardTitle>
        <Button
          variant="secondary"
          size="sm"
          onClick={() => reanalyze.mutate()}
          disabled={reanalyze.isPending}
        >
          {reanalyze.isPending ? <Spinner /> : <RefreshCw className="h-3.5 w-3.5" />}
          Re-analyze
        </Button>
      </CardHeader>

      <CardBody className="space-y-5">
        {!latest ? (
          <p className="text-sm text-muted-foreground">
            No analysis yet. The classifier runs asynchronously after a ticket is
            created; re-analyze to trigger it now.
          </p>
        ) : (
          <>
            {latest.needs_human_review && (
              <div className="rounded-lg bg-amber-500/10 px-3 py-2 text-sm text-amber-700 dark:text-amber-400">
                This ticket was flagged for human review — a confidence fell below
                the threshold{latest.mapping_fallback ? " and it could not be routed to a department" : ""}.
              </div>
            )}

            {/* Category: predicted vs effective */}
            <Comparison
              label="Category → Department"
              predicted={<CategoryBadge category={latest.predicted_category} />}
              effective={
                <span className="text-sm font-medium text-foreground">
                  {ticket.department.name}
                </span>
              }
              overridden={ticket.department_overridden}
            />

            {/* Priority: predicted vs effective */}
            <Comparison
              label="Priority"
              predicted={
                <PriorityBadge
                  priority={latest.predicted_priority as TicketPriority}
                />
              }
              effective={<PriorityBadge priority={ticket.priority} />}
              overridden={ticket.priority_overridden}
            />

            <div className="space-y-3 border-t border-border pt-4">
              {/* The flag is the backend's verdict for the whole analysis, so
                  both bars carry it; the banner above says why. The threshold
                  only positions the marker line and comes from the API, so it
                  tracks the backend setting instead of a hardcoded copy. */}
              <ConfidenceBar
                label="Priority confidence"
                value={latest.priority_confidence}
                flagged={latest.needs_human_review}
                threshold={ticket.review_threshold}
              />
              <ConfidenceBar
                label="Category confidence"
                value={latest.department_confidence}
                flagged={latest.needs_human_review}
                threshold={ticket.review_threshold}
              />
            </div>

            <div className="flex flex-wrap items-center gap-2 border-t border-border pt-4 text-xs text-muted-foreground">
              <span>Model:</span>
              <Badge>
                {latest.model_name} {latest.model_version}
              </Badge>
              {latest.mapping_fallback && (
                <Badge tone="bg-amber-500/15 text-amber-600 dark:text-amber-400 ring-amber-500/30">
                  Unrouted (fallback)
                </Badge>
              )}
            </div>

            {ticket.analyses.length > 1 && (
              <AnalysisHistory ticket={ticket} />
            )}
          </>
        )}
      </CardBody>
    </Card>
  );
}

function Comparison({
  label,
  predicted,
  effective,
  overridden,
}: {
  label: string;
  predicted: React.ReactNode;
  effective: React.ReactNode;
  overridden: boolean;
}) {
  return (
    <div>
      <div className="mb-2 text-xs font-medium text-muted-foreground">{label}</div>
      <div className="flex items-center gap-3">
        <div className="flex-1">
          <div className="mb-1 text-[10px] uppercase tracking-wide text-muted-foreground">
            AI predicted
          </div>
          {predicted}
        </div>
        <div className="text-muted-foreground">→</div>
        <div className="flex-1">
          <div className="mb-1 text-[10px] uppercase tracking-wide text-muted-foreground">
            Effective
          </div>
          <div className="flex items-center gap-1.5">
            {effective}
            {overridden && (
              <Badge tone="bg-indigo-500/15 text-indigo-600 dark:text-indigo-400 ring-indigo-500/30">
                overridden
              </Badge>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

function AnalysisHistory({ ticket }: { ticket: TicketDetail }) {
  return (
    <div className="border-t border-border pt-4">
      <div className="mb-2 text-xs font-medium text-muted-foreground">
        History ({ticket.analyses.length})
      </div>
      <ul className="space-y-2">
        {ticket.analyses.map((a) => (
          <li
            key={a.id}
            className="flex items-center justify-between rounded-lg bg-surface-muted/50 px-3 py-2 text-xs"
          >
            <div className="flex items-center gap-2">
              <span className="text-foreground">
                {a.predicted_category
                  ? CATEGORY_LABELS[a.predicted_category as Category]
                  : "Unclassified"}
              </span>
              <span className="text-muted-foreground">·</span>
              <span className="text-muted-foreground">
                {PRIORITY_LABELS[a.predicted_priority as TicketPriority] ??
                  a.predicted_priority}
              </span>
            </div>
            <div className="flex items-center gap-2 text-muted-foreground">
              <span className="font-mono">
                {a.model_name} {a.model_version}
              </span>
              <span>{formatDate(a.created_at)}</span>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
