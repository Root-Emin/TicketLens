/*
  Domain model for the TicketLens staff panel.

  This replaces the two older models that used to sit side by side: the
  urgency-only classifier type in lib/types.ts and the presentational shape in
  lib/dashboard/types.ts. One model now backs the dashboard, the inbox, the
  conversation and the details panel.

  Deliberately free of any `lucide-react` import. Icons are a rendering concern
  and are mapped in the components; the data layer stays plain.
*/

/** Priority scale, lowest to highest. Three steps, matching the design. */
export type Priority = "low" | "medium" | "high";

/** Lifecycle of a ticket. `pending` means we are waiting on the customer. */
export type TicketStatus =
  | "open"
  | "pending"
  | "on_hold"
  | "resolved"
  | "closed";

export type TeamId = "backend" | "finance" | "sales" | "it" | "hr";

/** Who wrote a message. `note` is staff-only and never reaches the customer. */
export type MessageKind = "customer" | "agent" | "note";

/** Semantic accents. Every colored surface resolves to one of these. */
export type Accent = "blue" | "green" | "orange" | "red" | "neutral";

export interface Person {
  id: string;
  name: string;
  email: string;
  initials: string;
}

/**
 * A staff member, who belongs to exactly one team.
 *
 * That team is the visibility boundary: an agent only ever sees tickets
 * belonging to it. Enforced in lib/staff/queries.ts, not in the components.
 */
export interface Agent extends Person {
  team: TeamId;
}

export interface Customer extends Person {
  company: string;
  /** Total tickets this customer has ever opened, for the customer card. */
  ticketCount: number;
  since: string;
}

export interface Attachment {
  id: string;
  name: string;
  /** Human-readable, e.g. "248 KB" — fixtures carry no real bytes. */
  size: string;
  kind: "image" | "document" | "log";
}

export interface Message {
  id: string;
  kind: MessageKind;
  authorId: string;
  authorName: string;
  authorInitials: string;
  /**
   * Plain text. Stored as one string rather than pre-split lines so it reflows
   * at any container width — the previous model hardcoded the screenshot's
   * line breaks and could not adapt.
   */
  body: string;
  sentAt: string;
  attachments?: Attachment[];
  /** Agent messages show a delivery tick once sent. */
  delivered?: boolean;
}

export interface TimelineEvent {
  id: string;
  kind:
    | "created"
    | "assigned"
    | "priority_changed"
    | "status_changed"
    | "note_added"
    | "escalated";
  summary: string;
  actor: string;
  at: string;
}

/** What the model predicted for a ticket, plus how sure it was. */
export interface AiAnalysis {
  category: string;
  subcategory: string;
  priority: Priority;
  sentiment: "positive" | "neutral" | "negative";
  /** 0..1. */
  confidence: number;
  suggestedReply: string;
  similar: { id: string; subject: string; status: TicketStatus }[];
}

export interface Ticket {
  id: string;
  subject: string;
  customer: Customer;
  assignee: Agent | null;
  team: TeamId;
  priority: Priority;
  status: TicketStatus;
  category: string;
  subcategory: string;
  tags: string[];
  createdAt: string;
  updatedAt: string;
  /** When the SLA clock expires. Drives the SLA card and "needs attention". */
  slaDueAt: string;
  unread: boolean;
  messages: Message[];
  timeline: TimelineEvent[];
  ai: AiAnalysis;
}

export interface Notification {
  id: string;
  title: string;
  body: string;
  at: string;
  read: boolean;
  ticketId?: string;
}

/* ------------------------------------------------------------------ views */

/**
 * The saved views the sidebar exposes. Every one of them is a filter over the
 * same queue rather than a separate screen, which is how Linear and Zendesk
 * model the equivalent navigation.
 */
export type ViewId =
  | "assigned"
  | "attention"
  | "waiting"
  | "resolved"
  | "all"
  | "unassigned"
  | "open"
  | "hold"
  | "closed";

export type SortId = "newest" | "oldest" | "priority" | "sla";

/** Everything that can be expressed in the inbox URL. */
export interface TicketQuery {
  view: ViewId;
  q?: string;
  priority?: Priority;
  status?: TicketStatus;
  sort: SortId;
  page: number;
}

export interface Paged<T> {
  items: T[];
  total: number;
  page: number;
  pageSize: number;
  pageCount: number;
}
