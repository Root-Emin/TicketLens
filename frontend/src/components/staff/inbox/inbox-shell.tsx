import { cn } from "@/lib/utils";
import { Skeleton } from "@/components/shadcn/skeleton";

/*
  The inbox's two-pane frame.

  This is the one screen that keeps independent scrolling, because a queue you
  scroll separately from the conversation you are reading is the entire point of
  an inbox — every comparable product works this way.

  Below md the two panes become two routes: /staff/tickets is the queue and
  /staff/tickets/[id] is the conversation. No duplicate mobile layout, just a
  different pane showing at a different URL.
*/

export function InboxShell({
  list,
  detail,
  hasDetail,
}: {
  list: React.ReactNode;
  detail: React.ReactNode;
  hasDetail: boolean;
}) {
  return (
    <div className="flex h-full min-h-0">
      <div
        className={cn(
          "min-h-0 w-full shrink-0 border-r border-tl-line md:w-[320px] xl:w-[360px]",
          hasDetail && "hidden md:block",
        )}
      >
        {list}
      </div>

      <div className={cn("min-h-0 min-w-0 flex-1", !hasDetail && "hidden md:block")}>
        {detail}
      </div>
    </div>
  );
}

export function TicketListSkeleton() {
  return (
    <div className="flex h-full min-h-0 flex-col bg-tl-card">
      <div className="flex shrink-0 items-center gap-2 px-4 pb-3 pt-4">
        <Skeleton className="h-5 w-32" />
        <Skeleton className="h-5 w-8 rounded-full" />
      </div>

      <div className="shrink-0 space-y-3 px-4 pb-3">
        <Skeleton className="h-9 w-full rounded-btn" />
        <div className="flex gap-1.5">
          <Skeleton className="h-7 w-20 rounded-lg" />
          <Skeleton className="h-7 w-20 rounded-lg" />
          <Skeleton className="h-7 w-24 rounded-lg" />
        </div>
      </div>

      <div className="min-h-0 flex-1 border-t border-tl-line-soft">
        {Array.from({ length: 8 }).map((_, i) => (
          <div
            key={i}
            className="flex gap-3 border-b border-tl-line-soft px-4 py-3"
          >
            <Skeleton className="size-8 shrink-0 rounded-full" />
            <div className="min-w-0 flex-1 space-y-2">
              <div className="flex items-center gap-2">
                <Skeleton className="h-3 w-16" />
                <Skeleton className="ml-auto h-4 w-12 rounded-md" />
              </div>
              <Skeleton className="h-3.5 w-11/12" />
              <Skeleton className="h-3 w-1/2" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

export function ConversationSkeleton() {
  return (
    <div className="flex h-full min-h-0 flex-col">
      <div className="shrink-0 space-y-3 border-b border-tl-line px-5 py-4">
        <div className="flex items-center gap-2.5">
          <Skeleton className="h-4 w-20" />
          <Skeleton className="h-5 w-14 rounded-md" />
          <Skeleton className="ml-auto h-8 w-24 rounded-btn" />
        </div>
        <Skeleton className="h-6 w-2/3" />
        <Skeleton className="h-3.5 w-1/2" />
      </div>

      <div className="min-h-0 flex-1 space-y-6 px-5 py-6">
        {[0, 1, 2].map((i) => (
          <div
            key={i}
            className={cn("flex gap-3", i === 1 && "flex-row-reverse")}
          >
            <Skeleton className="size-8 shrink-0 rounded-full" />
            <div className="w-2/3 space-y-2">
              <Skeleton className="h-3 w-24" />
              <Skeleton className="h-16 w-full rounded-[12px]" />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
