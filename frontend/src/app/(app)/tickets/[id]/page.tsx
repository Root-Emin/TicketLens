"use client";

import Link from "next/link";
import { use, useState } from "react";
import { ArrowLeft, Lock, Send } from "lucide-react";

import { AIPanel } from "@/components/triage/ai-panel";
import { PriorityBadge, StatusBadge } from "@/components/triage/badges";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardBody, CardHeader, CardTitle } from "@/components/ui/card";
import { Label, Select, Textarea } from "@/components/ui/field";
import { CenteredSpinner, Spinner } from "@/components/ui/spinner";
import {
  useAddMessage,
  useAssignTicket,
  useDepartments,
  useMe,
  useTicket,
  useUpdateTicket,
} from "@/lib/api/hooks";
import {
  ALL_PRIORITIES,
  ALL_STATUSES,
  PRIORITY_LABELS,
  STATUS_LABELS,
} from "@/lib/api/labels";
import type { MessageInfo } from "@/lib/api/types";
import { cn, formatDate } from "@/lib/utils";

export default function TicketDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = use(params);
  const { data: ticket, isLoading, isError } = useTicket(id);

  if (isLoading) return <CenteredSpinner label="Loading ticket…" />;
  if (isError || !ticket)
    return (
      <div className="p-6">
        <Card className="p-8 text-center text-sm text-red-500">
          Ticket not found.
        </Card>
      </div>
    );

  return (
    <div className="p-6">
      <Link
        href="/tickets"
        className="mb-4 inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to queue
      </Link>

      <div className="mb-5">
        <div className="flex items-center gap-2">
          <h1 className="text-xl font-semibold">{ticket.subject}</h1>
        </div>
        <div className="mt-2 flex flex-wrap items-center gap-2 text-sm text-muted-foreground">
          <StatusBadge status={ticket.status} />
          <PriorityBadge priority={ticket.priority} />
          <span>·</span>
          <span>
            {ticket.customer.full_name} ({ticket.customer.email})
          </span>
          <span>·</span>
          <span>opened {formatDate(ticket.created_at)}</span>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        <div className="space-y-4 lg:col-span-2">
          <Thread messages={ticket.messages} />
          <ReplyBox ticketId={id} />
        </div>

        <div className="space-y-4">
          <ControlsCard ticketId={id} ticket={ticket} />
          <AIPanel ticket={ticket} />
        </div>
      </div>
    </div>
  );
}

function Thread({ messages }: { messages: MessageInfo[] }) {
  return (
    <Card>
      <CardHeader>
        <CardTitle>Conversation</CardTitle>
      </CardHeader>
      <CardBody className="space-y-3">
        {messages.length === 0 && (
          <p className="text-sm text-muted-foreground">No messages.</p>
        )}
        {messages.map((m) => (
          <Message key={m.id} message={m} />
        ))}
      </CardBody>
    </Card>
  );
}

function Message({ message }: { message: MessageInfo }) {
  const isCustomer = message.author_type === "customer";

  // Internal notes are styled apart from the customer-visible thread so an agent
  // never confuses a private note with a reply the customer can see.
  if (message.is_internal) {
    return (
      <div className="rounded-lg border border-amber-500/30 bg-amber-500/5 px-4 py-3">
        <div className="mb-1 flex items-center gap-1.5 text-xs font-medium text-amber-600 dark:text-amber-400">
          <Lock className="h-3 w-3" />
          Internal note
          <span className="font-normal text-muted-foreground">
            · {formatDate(message.created_at)}
          </span>
        </div>
        <p className="whitespace-pre-wrap text-sm text-foreground">{message.body}</p>
      </div>
    );
  }

  return (
    <div className={cn("flex", isCustomer ? "justify-start" : "justify-end")}>
      <div
        className={cn(
          "max-w-[80%] rounded-lg px-4 py-3",
          isCustomer
            ? "bg-surface-muted"
            : "bg-accent/10 ring-1 ring-inset ring-accent/20",
        )}
      >
        <div className="mb-1 text-xs font-medium capitalize text-muted-foreground">
          {message.author_type} · {formatDate(message.created_at)}
        </div>
        <p className="whitespace-pre-wrap text-sm text-foreground">{message.body}</p>
      </div>
    </div>
  );
}

function ReplyBox({ ticketId }: { ticketId: string }) {
  const [body, setBody] = useState("");
  const [internal, setInternal] = useState(false);
  const addMessage = useAddMessage(ticketId);

  function submit() {
    if (!body.trim()) return;
    addMessage.mutate(
      { body: body.trim(), is_internal: internal },
      { onSuccess: () => setBody("") },
    );
  }

  return (
    <Card>
      <CardBody className="space-y-3">
        <Textarea
          rows={3}
          value={body}
          onChange={(e) => setBody(e.target.value)}
          placeholder={
            internal ? "Write an internal note…" : "Write a reply to the customer…"
          }
        />
        <div className="flex items-center justify-between">
          <label className="flex items-center gap-2 text-sm text-muted-foreground">
            <input
              type="checkbox"
              checked={internal}
              onChange={(e) => setInternal(e.target.checked)}
              className="h-4 w-4 rounded border-border"
            />
            Internal note
          </label>
          <Button onClick={submit} disabled={addMessage.isPending || !body.trim()}>
            {addMessage.isPending ? <Spinner /> : <Send className="h-3.5 w-3.5" />}
            Send
          </Button>
        </div>
      </CardBody>
    </Card>
  );
}

function ControlsCard({
  ticketId,
  ticket,
}: {
  ticketId: string;
  ticket: import("@/lib/api/types").TicketDetail;
}) {
  const update = useUpdateTicket(ticketId);
  const assign = useAssignTicket(ticketId);
  const { data: departments } = useDepartments();
  const { data: me } = useMe();

  const assignedToMe = Boolean(me && ticket.assignee?.id === me.id);

  return (
    <Card>
      <CardHeader>
        <CardTitle>Manage</CardTitle>
      </CardHeader>
      <CardBody className="space-y-4">
        <div>
          <Label>Status</Label>
          <Select
            value={ticket.status}
            onChange={(e) => update.mutate({ status: e.target.value })}
            disabled={update.isPending}
          >
            {ALL_STATUSES.map((s) => (
              <option key={s} value={s}>
                {STATUS_LABELS[s]}
              </option>
            ))}
          </Select>
        </div>

        <div>
          <Label>Priority</Label>
          <Select
            value={ticket.priority}
            onChange={(e) => update.mutate({ priority: e.target.value })}
            disabled={update.isPending}
          >
            {ALL_PRIORITIES.map((p) => (
              <option key={p} value={p}>
                {PRIORITY_LABELS[p]}
              </option>
            ))}
          </Select>
        </div>

        <div>
          <Label>Department</Label>
          <Select
            value={ticket.department.id}
            onChange={(e) => update.mutate({ department_id: e.target.value })}
            disabled={update.isPending}
          >
            {departments?.data.map((d) => (
              <option key={d.id} value={d.id}>
                {d.name}
              </option>
            ))}
          </Select>
        </div>

        <div className="border-t border-border pt-4">
          <Label>Assignee</Label>
          <div className="flex items-center justify-between">
            <span className="text-sm text-foreground">
              {ticket.assignee ? (
                ticket.assignee.full_name
              ) : (
                <Badge>Unassigned</Badge>
              )}
            </span>
            {assignedToMe ? (
              <Button
                variant="ghost"
                size="sm"
                onClick={() => assign.mutate(null)}
                disabled={assign.isPending}
              >
                Unassign
              </Button>
            ) : (
              <Button
                variant="secondary"
                size="sm"
                onClick={() => me && assign.mutate(me.id)}
                disabled={assign.isPending || !me}
              >
                Assign to me
              </Button>
            )}
          </div>
        </div>
      </CardBody>
    </Card>
  );
}
