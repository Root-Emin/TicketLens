import type {
  Agent,
  Customer,
  Message,
  Notification,
  Priority,
  TeamId,
  Ticket,
  TicketStatus,
  TimelineEvent,
} from "./types";

/*
  Fixtures for the staff panel.

  Every timestamp derives from NOW_ISO, a frozen anchor, rather than Date.now().
  Two reasons, both learned from the panel this replaces: the demo always looks
  identical, and the server and the client render the same relative string
  instead of disagreeing by a few milliseconds and triggering a hydration
  mismatch.

  Nothing outside lib/staff imports this file — screens call queries.ts, so
  swapping in the Go backend later means rewriting that one module.
*/

/** The frozen "now" the whole panel renders against: 09:50 in Istanbul. */
export const NOW_ISO = "2026-08-03T06:50:00.000Z";
const NOW_MS = Date.parse(NOW_ISO);

const MIN = 1;
const HOUR = 60;
const DAY = 24 * HOUR;

/** ago builds an ISO timestamp N minutes before the anchor. */
function ago(minutes: number): string {
  return new Date(NOW_MS - minutes * 60_000).toISOString();
}

/** ahead builds an ISO timestamp N minutes after the anchor. */
function ahead(minutes: number): string {
  return new Date(NOW_MS + minutes * 60_000).toISOString();
}

/* ----------------------------------------------------------------- people */

/*
  Every agent belongs to one team, and that team decides what they can see.
  Ahmet is on Backend, so the Finance, Sales, IT and HR tickets below exist
  purely to prove the scoping works — he should never receive one.
*/
export const currentAgent: Agent = {
  id: "u-ahmet",
  name: "Ahmet Yılmaz",
  email: "ahmet.yilmaz@ticketlens.io",
  initials: "AY",
  team: "backend",
};

export const agents: Agent[] = [
  currentAgent,
  {
    id: "u-elif",
    name: "Elif Demir",
    email: "elif.demir@ticketlens.io",
    initials: "ED",
    team: "it",
  },
  {
    id: "u-marco",
    name: "Marco Rossi",
    email: "marco.rossi@ticketlens.io",
    initials: "MR",
    team: "finance",
  },
  {
    id: "u-deniz",
    name: "Deniz Arslan",
    email: "deniz.arslan@ticketlens.io",
    initials: "DA",
    team: "backend",
  },
];

export const teams: { id: TeamId; label: string }[] = [
  { id: "backend", label: "Backend" },
  { id: "finance", label: "Finance" },
  { id: "sales", label: "Sales" },
  { id: "it", label: "IT Support" },
  { id: "hr", label: "HR" },
];

export const teamLabel: Record<TeamId, string> = {
  backend: "Backend Team",
  finance: "Finance Team",
  sales: "Sales Team",
  it: "IT Support",
  hr: "HR Team",
};

function customer(
  id: string,
  name: string,
  email: string,
  company: string,
  ticketCount: number,
  sinceDays: number,
): Customer {
  return {
    id,
    name,
    email,
    company,
    ticketCount,
    initials: name
      .split(" ")
      .map((p) => p[0])
      .join(""),
    since: ago(sinceDays * DAY),
  };
}

const C = {
  john: customer("c-john", "John Doe", "john.doe@email.com", "Northwind Ltd.", 14, 420),
  sarah: customer("c-sarah", "Sarah Johnson", "sarah.j@brightpay.io", "BrightPay", 6, 210),
  michael: customer("c-michael", "Michael Brown", "m.brown@lumen.co", "Lumen Co.", 3, 95),
  emily: customer("c-emily", "Emily Davis", "emily.davis@corex.dev", "Corex", 9, 300),
  david: customer("c-david", "David Wilson", "d.wilson@finhub.com", "FinHub", 21, 640),
  lisa: customer("c-lisa", "Lisa Anderson", "lisa@anderson.design", "Anderson Design", 2, 40),
  robert: customer("c-robert", "Robert Taylor", "r.taylor@atlas-io.com", "Atlas IO", 11, 380),
  nina: customer("c-nina", "Nina Petrova", "nina.p@vektor.eu", "Vektor", 5, 150),
  omar: customer("c-omar", "Omar Haddad", "omar@sandcastle.sa", "Sandcastle", 7, 260),
  yuki: customer("c-yuki", "Yuki Tanaka", "yuki.tanaka@kaido.jp", "Kaido", 4, 120),
};

/* ------------------------------------------------------------ ticket seed */

interface Seed {
  n: number;
  subject: string;
  cust: Customer;
  team: TeamId;
  priority: Priority;
  status: TicketStatus;
  category: string;
  subcategory: string;
  tags: string[];
  /** Minutes since the ticket was created. */
  age: number;
  /** Minutes until the SLA expires; negative means already breached. */
  sla: number;
  assignee: Agent | null;
  unread?: boolean;
  /** The customer's opening message. */
  opening: string;
  /** Optional agent reply, then optional internal note. */
  reply?: string;
  note?: string;
  sentiment?: "positive" | "neutral" | "negative";
  confidence?: number;
}

const ME = currentAgent;
const ELIF = agents[1];
const MARCO = agents[2];
const DENIZ = agents[3];

const seeds: Seed[] = [
  {
    n: 2501,
    subject: "API returns 500 error on checkout",
    cust: C.john,
    team: "backend",
    priority: "high",
    status: "open",
    category: "Technical Issue",
    subcategory: "API / Integration",
    tags: ["api", "checkout", "error"],
    age: 10 * MIN,
    sla: 45 * MIN,
    assignee: ME,
    unread: true,
    opening:
      "Hi, I'm trying to complete a payment using the API but I'm getting a 500 Internal Server Error. This was working fine yesterday.",
    reply:
      "Hello John, thank you for reaching out. I'm checking this for you. Could you please share the request ID and the time this issue occurred?",
    note: "Looks like a database connection issue. Escalating to backend team.",
    sentiment: "negative",
    confidence: 0.92,
  },
  {
    n: 2498,
    subject: "Payment not reflected in account",
    cust: C.sarah,
    team: "finance",
    priority: "medium",
    status: "open",
    category: "Billing",
    subcategory: "Payments",
    tags: ["billing", "payment"],
    age: 15 * MIN,
    sla: 3 * HOUR,
    assignee: MARCO,
    opening:
      "I paid the annual invoice two days ago but my account still shows as unpaid. The bank confirms the transfer went through.",
    reply:
      "Hi Sarah, I can see the transfer is still settling on our side. I've flagged it with our finance team and will update you today.",
    sentiment: "neutral",
    confidence: 0.81,
  },
  {
    n: 2495,
    subject: "Request to cancel subscription",
    cust: C.michael,
    team: "sales",
    priority: "low",
    status: "open",
    category: "Account",
    subcategory: "Subscription",
    tags: ["churn", "subscription"],
    age: 30 * MIN,
    sla: 8 * HOUR,
    assignee: null,
    opening:
      "We'd like to cancel our subscription at the end of the current billing period. What's the process?",
    sentiment: "neutral",
    confidence: 0.74,
  },
  {
    n: 2490,
    subject: "Unable to reset password",
    cust: C.emily,
    team: "it",
    priority: "high",
    status: "open",
    category: "Account",
    subcategory: "Access",
    tags: ["auth", "password"],
    age: 1 * HOUR,
    sla: 20 * MIN,
    assignee: ELIF,
    opening:
      "The reset link in the email takes me to a page that says the token has expired, even when I click it immediately.",
    reply:
      "Hi Emily, thanks for the detail — that narrows it down a lot. Could you tell me which browser you're using?",
    note: "Third report of expired reset tokens this week. Possible clock skew on the auth service.",
    sentiment: "negative",
    confidence: 0.88,
  },
  {
    n: 2487,
    subject: "Invoice not received",
    cust: C.david,
    team: "finance",
    priority: "medium",
    status: "pending",
    category: "Billing",
    subcategory: "Invoicing",
    tags: ["billing", "invoice"],
    age: 2 * HOUR,
    sla: 5 * HOUR,
    assignee: MARCO,
    opening:
      "We never received the invoice for July. Could you resend it to our accounts inbox?",
    reply:
      "Of course — could you confirm the billing address on file is still correct so it reaches the right inbox?",
    sentiment: "neutral",
    confidence: 0.69,
  },
  {
    n: 2483,
    subject: "Feature request: Dark mode",
    cust: C.lisa,
    team: "sales",
    priority: "low",
    status: "open",
    category: "Feature Request",
    subcategory: "UI",
    tags: ["feature", "ui"],
    age: 3 * HOUR,
    sla: 2 * DAY,
    assignee: null,
    opening:
      "Our team works late and the bright interface is hard on the eyes. Any plans for a dark theme?",
    sentiment: "positive",
    confidence: 0.55,
  },
  {
    n: 2479,
    subject: "Database connection timeout",
    cust: C.robert,
    team: "backend",
    priority: "high",
    status: "open",
    category: "Technical Issue",
    subcategory: "Infrastructure",
    tags: ["database", "timeout"],
    age: 4 * HOUR,
    sla: -25 * MIN,
    assignee: ME,
    opening:
      "Our nightly sync job has been failing with connection timeouts for the past three runs. Logs attached.",
    note: "SLA already breached. Needs an owner on the backend side this morning.",
    sentiment: "negative",
    confidence: 0.94,
  },
  {
    n: 2476,
    subject: "Webhook deliveries silently dropped",
    cust: C.nina,
    team: "backend",
    priority: "high",
    status: "open",
    category: "Technical Issue",
    subcategory: "API / Integration",
    tags: ["webhook", "api"],
    age: 5 * HOUR,
    sla: -2 * HOUR,
    assignee: null,
    unread: true,
    opening:
      "About one in twenty webhook deliveries never arrives. We see no retry and no error in the dashboard.",
    sentiment: "negative",
    confidence: 0.9,
  },
  {
    n: 2470,
    subject: "SSO login loops back to sign-in page",
    cust: C.omar,
    team: "it",
    priority: "high",
    status: "open",
    category: "Account",
    subcategory: "Access",
    tags: ["sso", "auth"],
    age: 6 * HOUR,
    sla: 30 * MIN,
    assignee: ELIF,
    opening:
      "After authenticating with Okta we get redirected straight back to the sign-in screen. Started this morning.",
    sentiment: "negative",
    confidence: 0.86,
  },
  {
    n: 2466,
    subject: "Export to CSV truncates long descriptions",
    cust: C.yuki,
    team: "backend",
    priority: "medium",
    status: "open",
    category: "Technical Issue",
    subcategory: "Reporting",
    tags: ["export", "csv"],
    age: 8 * HOUR,
    sla: 6 * HOUR,
    assignee: null,
    opening:
      "Descriptions longer than 255 characters get cut off in the CSV export but display fine in the UI.",
    sentiment: "neutral",
    confidence: 0.78,
  },
  {
    n: 2461,
    subject: "Duplicate charge on annual plan",
    cust: C.david,
    team: "finance",
    priority: "high",
    status: "open",
    category: "Billing",
    subcategory: "Payments",
    tags: ["billing", "refund"],
    age: 9 * HOUR,
    sla: 1 * HOUR,
    assignee: MARCO,
    opening:
      "We were charged twice for the annual plan this month. Please refund the duplicate.",
    sentiment: "negative",
    confidence: 0.91,
  },
  {
    n: 2455,
    subject: "Add seats to existing workspace",
    cust: C.sarah,
    team: "sales",
    priority: "low",
    status: "pending",
    category: "Account",
    subcategory: "Subscription",
    tags: ["seats", "upgrade"],
    age: 11 * HOUR,
    sla: 1 * DAY,
    assignee: null,
    opening: "We'd like to add five more seats. Can you send an updated quote?",
    reply:
      "Happy to help — I've sent a revised quote to your billing contact for review.",
    sentiment: "positive",
    confidence: 0.83,
  },
  {
    n: 2449,
    subject: "Rate limit hit on bulk import",
    cust: C.nina,
    team: "backend",
    priority: "medium",
    status: "on_hold",
    category: "Technical Issue",
    subcategory: "API / Integration",
    tags: ["api", "rate-limit"],
    age: 14 * HOUR,
    sla: 10 * HOUR,
    assignee: ME,
    opening:
      "Importing 50k records trips the rate limiter halfway through and the job can't resume.",
    note: "On hold until the customer confirms which API key the job uses.",
    sentiment: "neutral",
    confidence: 0.72,
  },
  {
    n: 2444,
    subject: "Onboarding call follow-up",
    cust: C.michael,
    team: "sales",
    priority: "low",
    status: "pending",
    category: "Onboarding",
    subcategory: "Training",
    tags: ["onboarding"],
    age: 16 * HOUR,
    sla: 2 * DAY,
    assignee: null,
    opening:
      "Thanks for the walkthrough yesterday. Could you share the recording and the slide deck?",
    sentiment: "positive",
    confidence: 0.87,
  },
  {
    n: 2438,
    subject: "Two-factor codes rejected on iOS",
    cust: C.emily,
    team: "it",
    priority: "high",
    status: "open",
    category: "Account",
    subcategory: "Access",
    tags: ["2fa", "mobile"],
    age: 19 * HOUR,
    sla: -40 * MIN,
    assignee: null,
    unread: true,
    opening:
      "Authenticator codes are rejected on the iOS app but the same code works on desktop.",
    sentiment: "negative",
    confidence: 0.84,
  },
  {
    n: 2431,
    subject: "Update company billing address",
    cust: C.omar,
    team: "finance",
    priority: "low",
    status: "resolved",
    category: "Billing",
    subcategory: "Invoicing",
    tags: ["billing"],
    age: 1 * DAY,
    sla: 12 * HOUR,
    assignee: MARCO,
    opening: "We've moved offices. Could you update the billing address on our invoices?",
    reply: "Updated — the next invoice will carry the new address. Anything else I can help with?",
    sentiment: "neutral",
    confidence: 0.95,
  },
  {
    n: 2425,
    subject: "Slow dashboard load for large workspaces",
    cust: C.robert,
    team: "backend",
    priority: "medium",
    status: "on_hold",
    category: "Technical Issue",
    subcategory: "Performance",
    tags: ["performance"],
    age: 1 * DAY + 4 * HOUR,
    sla: 8 * HOUR,
    assignee: ME,
    opening:
      "The dashboard takes upwards of twelve seconds to load for our largest workspace.",
    sentiment: "negative",
    confidence: 0.66,
  },
  {
    n: 2419,
    subject: "Request GDPR data export",
    cust: C.yuki,
    team: "hr",
    priority: "medium",
    status: "open",
    category: "Compliance",
    subcategory: "Privacy",
    tags: ["gdpr", "compliance"],
    age: 1 * DAY + 8 * HOUR,
    sla: 3 * DAY,
    assignee: null,
    opening:
      "We need a full export of personal data held for one of our former employees.",
    sentiment: "neutral",
    confidence: 0.79,
  },
  {
    n: 2412,
    subject: "Notification emails going to spam",
    cust: C.lisa,
    team: "it",
    priority: "medium",
    status: "resolved",
    category: "Technical Issue",
    subcategory: "Email",
    tags: ["email", "deliverability"],
    age: 2 * DAY,
    sla: 6 * HOUR,
    assignee: ELIF,
    opening: "All your notification emails land in our spam folder since last week.",
    reply:
      "We've updated our sending domain's DMARC policy. Could you confirm whether new mail arrives correctly?",
    sentiment: "neutral",
    confidence: 0.8,
  },
  {
    n: 2406,
    subject: "Mobile app crashes on attachment upload",
    cust: C.john,
    team: "backend",
    priority: "high",
    status: "resolved",
    category: "Technical Issue",
    subcategory: "Mobile",
    tags: ["mobile", "crash"],
    age: 2 * DAY + 6 * HOUR,
    sla: 2 * HOUR,
    assignee: ME,
    opening: "The Android app closes itself whenever I attach a photo over about 5 MB.",
    reply: "Fixed in build 4.2.1, which is rolling out now. Thanks for the clear repro steps.",
    sentiment: "negative",
    confidence: 0.89,
  },
  {
    n: 2398,
    subject: "Clarify enterprise SLA terms",
    cust: C.david,
    team: "sales",
    priority: "low",
    status: "closed",
    category: "Sales",
    subcategory: "Contract",
    tags: ["sla", "contract"],
    age: 3 * DAY,
    sla: 2 * DAY,
    assignee: null,
    opening: "Could you clarify what the four-hour response window covers on the enterprise plan?",
    reply: "Sent over the SLA appendix with the response and resolution targets broken out.",
    sentiment: "neutral",
    confidence: 0.93,
  },
  {
    n: 2390,
    subject: "Timezone wrong on scheduled reports",
    cust: C.nina,
    team: "backend",
    priority: "low",
    status: "closed",
    category: "Technical Issue",
    subcategory: "Reporting",
    tags: ["reports", "timezone"],
    age: 4 * DAY,
    sla: 1 * DAY,
    assignee: DENIZ,
    opening: "Our scheduled reports arrive at 3am local instead of the 9am we configured.",
    reply: "The scheduler was reading UTC rather than the workspace timezone. Corrected.",
    sentiment: "neutral",
    confidence: 0.85,
  },
  {
    n: 2381,
    subject: "Offboarding checklist for departing staff",
    cust: C.omar,
    team: "hr",
    priority: "low",
    status: "closed",
    category: "Onboarding",
    subcategory: "Offboarding",
    tags: ["hr"],
    age: 5 * DAY,
    sla: 3 * DAY,
    assignee: null,
    opening: "What's the recommended process for revoking access when someone leaves?",
    reply: "Shared our offboarding runbook — revoke SSO first, then rotate any shared API keys.",
    sentiment: "positive",
    confidence: 0.9,
  },
  {
    n: 2374,
    subject: "Sandbox environment returns stale data",
    cust: C.yuki,
    team: "backend",
    priority: "medium",
    status: "closed",
    category: "Technical Issue",
    subcategory: "API / Integration",
    tags: ["sandbox", "api"],
    age: 6 * DAY,
    sla: 1 * DAY,
    assignee: DENIZ,
    opening: "Records created in sandbox don't appear in reads for several minutes.",
    reply: "Sandbox replication lag was misconfigured. It now matches production behaviour.",
    sentiment: "neutral",
    confidence: 0.77,
  },
];

/* ------------------------------------------------------------- expansion */

const ATTACHMENTS_BY_TICKET: Record<number, Ticket["messages"][number]["attachments"]> =
  {
    2501: [{ id: "a1", name: "checkout-error.har", size: "248 KB", kind: "log" }],
    2479: [
      { id: "a2", name: "sync-job.log", size: "1.2 MB", kind: "log" },
      { id: "a3", name: "timeout-graph.png", size: "86 KB", kind: "image" },
    ],
  };

function buildMessages(seed: Seed): Message[] {
  const list: Message[] = [];
  const openedAt = ago(seed.age);

  list.push({
    id: `${seed.n}-m1`,
    kind: "customer",
    authorId: seed.cust.id,
    authorName: seed.cust.name,
    authorInitials: seed.cust.initials,
    body: seed.opening,
    sentAt: openedAt,
    attachments: ATTACHMENTS_BY_TICKET[seed.n],
  });

  if (seed.reply) {
    const agent = seed.assignee ?? ME;
    list.push({
      id: `${seed.n}-m2`,
      kind: "agent",
      authorId: agent.id,
      authorName: agent.name,
      authorInitials: agent.initials,
      body: seed.reply,
      sentAt: ago(Math.max(1, Math.round(seed.age * 0.6))),
      delivered: true,
    });
  }

  // TK-2501 carries the follow-up the design shows verbatim.
  if (seed.n === 2501) {
    list.push({
      id: "2501-m3",
      kind: "customer",
      authorId: seed.cust.id,
      authorName: seed.cust.name,
      authorInitials: seed.cust.initials,
      body: "Sure, request ID: req_9f8d7g6h5j4k\nTime: 2024-05-20 09:40:12 UTC",
      sentAt: ago(6 * MIN),
    });
  }

  if (seed.note) {
    list.push({
      id: `${seed.n}-note`,
      kind: "note",
      authorId: ME.id,
      authorName: ME.name,
      authorInitials: ME.initials,
      body: seed.note,
      sentAt: ago(Math.max(1, Math.round(seed.age * 0.4))),
    });
  }

  return list;
}

function buildTimeline(seed: Seed): TimelineEvent[] {
  const events: TimelineEvent[] = [
    {
      id: `${seed.n}-t1`,
      kind: "created",
      summary: `${seed.cust.name} opened this ticket`,
      actor: seed.cust.name,
      at: ago(seed.age),
    },
  ];

  if (seed.assignee) {
    events.push({
      id: `${seed.n}-t2`,
      kind: "assigned",
      summary: `Assigned to ${seed.assignee.name}`,
      actor: "Triage bot",
      at: ago(Math.max(1, Math.round(seed.age * 0.9))),
    });
  }

  if (seed.priority === "high") {
    events.push({
      id: `${seed.n}-t3`,
      kind: "priority_changed",
      summary: "Priority raised to High",
      actor: "TicketLens AI",
      at: ago(Math.max(1, Math.round(seed.age * 0.85))),
    });
  }

  if (seed.note) {
    events.push({
      id: `${seed.n}-t4`,
      kind: "note_added",
      summary: "Internal note added",
      actor: ME.name,
      at: ago(Math.max(1, Math.round(seed.age * 0.4))),
    });
  }

  if (seed.status === "resolved" || seed.status === "closed") {
    events.push({
      id: `${seed.n}-t5`,
      kind: "status_changed",
      summary: `Marked as ${seed.status}`,
      actor: (seed.assignee ?? ME).name,
      at: ago(Math.max(1, Math.round(seed.age * 0.2))),
    });
  }

  return events.sort((a, b) => Date.parse(a.at) - Date.parse(b.at));
}

const SUGGESTED: Record<number, string> = {
  2501:
    "This seems to be a temporary server issue. Our team is investigating this right now. We will update you as soon as we have more information.",
};

function buildTicket(seed: Seed): Ticket {
  const messages = buildMessages(seed);
  const similar = seeds
    .filter(
      (s) =>
        s.n !== seed.n &&
        s.subcategory === seed.subcategory &&
        (s.status === "resolved" || s.status === "closed"),
    )
    .slice(0, 2)
    .map((s) => ({ id: `TK-${s.n}`, subject: s.subject, status: s.status }));

  return {
    id: `TK-${seed.n}`,
    subject: seed.subject,
    customer: seed.cust,
    assignee: seed.assignee,
    team: seed.team,
    priority: seed.priority,
    status: seed.status,
    category: seed.category,
    subcategory: seed.subcategory,
    tags: seed.tags,
    createdAt: ago(seed.age),
    updatedAt: messages[messages.length - 1].sentAt,
    slaDueAt: seed.sla >= 0 ? ahead(seed.sla) : ago(-seed.sla),
    unread: seed.unread ?? false,
    messages,
    timeline: buildTimeline(seed),
    ai: {
      category: seed.category,
      subcategory: seed.subcategory,
      priority: seed.priority,
      sentiment: seed.sentiment ?? "neutral",
      confidence: seed.confidence ?? 0.75,
      suggestedReply:
        SUGGESTED[seed.n] ??
        `Thanks for flagging this. I've picked up your ticket about "${seed.subject.toLowerCase()}" and will come back to you with an update shortly.`,
      similar,
    },
  };
}

export const tickets: Ticket[] = seeds.map(buildTicket);

/* ---------------------------------------------------------- notifications */

export const notifications: Notification[] = [
  {
    id: "n1",
    title: "SLA breached",
    body: "TK-2479 · Database connection timeout has passed its response target.",
    at: ago(25 * MIN),
    read: false,
    ticketId: "TK-2479",
  },
  {
    id: "n2",
    title: "New reply from John Doe",
    body: "TK-2501 · Sure, request ID: req_9f8d7g6h5j4k",
    at: ago(6 * MIN),
    read: false,
    ticketId: "TK-2501",
  },
  {
    id: "n3",
    title: "Ticket assigned to you",
    body: "TK-2455 · Add seats to existing workspace",
    at: ago(11 * HOUR),
    read: false,
    ticketId: "TK-2455",
  },
  {
    id: "n4",
    title: "SLA breached",
    body: "TK-2438 · Two-factor codes rejected on iOS.",
    at: ago(40 * MIN),
    read: false,
    ticketId: "TK-2438",
  },
  {
    id: "n5",
    title: "Mention from Elif Demir",
    body: "TK-2470 · Can you take a look at the Okta config?",
    at: ago(2 * HOUR),
    read: false,
    ticketId: "TK-2470",
  },
  {
    id: "n6",
    title: "Ticket resolved",
    body: "TK-2412 · Notification emails going to spam.",
    at: ago(2 * DAY),
    read: true,
    ticketId: "TK-2412",
  },
];
