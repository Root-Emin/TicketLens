/*
  The landing page's copy, kept in one file because almost all of it is
  restatement of things the backend actually defines.

  CATEGORIES mirrors backend/internal/domain/triage/model/category.go — the same
  ten slugs in the same order, with each blurb condensed from that file's
  doc comments. If the taxonomy changes there, it changes here. (The Go side and
  the Python side already guard this pairing with a test; this copy is a third
  reader of the same list, and a stale marketing page is a cheaper failure than a
  stale classifier, so it is documented rather than tested.)

  DEPARTMENTS matches the four the seed creates in backend/cmd/seed/main.go.
  Four, against ten categories — that gap is real and the page says so rather
  than pretending every class has a team waiting for it.
*/

export interface CategoryDef {
  /** The slug the model emits, shown verbatim and set in mono. */
  slug: string;
  /** One line, condensed from the Go doc comment. */
  blurb: string;
}

export const CATEGORIES: CategoryDef[] = [
  {
    slug: "technical_issue",
    blurb: "The platform itself is broken, erroring or slow.",
  },
  {
    slug: "integration",
    blurb:
      "A third-party link is failing — marketplace, cargo, ERP, virtual POS, webhook.",
  },
  {
    slug: "payment_ops",
    blurb:
      "Money movement: settlement delays, refunds, chargebacks, a transaction that looks missing.",
  },
  {
    slug: "billing",
    blurb:
      "What the customer pays you — invoices, plan changes, commission rates, cancellation.",
  },
  {
    slug: "onboarding",
    blurb: "Setup, data migration, go-live, application and activation.",
  },
  {
    slug: "how_to",
    blurb: "Nothing is broken. The customer asks how to do something.",
  },
  {
    slug: "account_access",
    blurb: "Users, roles, sessions, passwords, panel access.",
  },
  {
    slug: "feature_request",
    blurb: "A capability that does not exist yet. Roadmap.",
  },
  { slug: "sales", blurb: "Pre-sales, add-on modules, demo requests." },
  {
    slug: "compliance",
    blurb: "KVKK and GDPR, contracts, data deletion, audit documents.",
  },
];

/** The four departments the seed maps categories onto. */
export const DEPARTMENTS = [
  { name: "Technical Support", category: "technical_issue" },
  { name: "Integration Support", category: "integration" },
  { name: "Payment Operations", category: "payment_ops" },
  { name: "Customer Success", category: "how_to" },
];

/** The priority head's four levels, in the order the model ranks them. */
export const PRIORITIES = [
  { level: "low", token: "text-tlm-low", bar: "bg-tlm-low" },
  { level: "normal", token: "text-priority-normal", bar: "bg-priority-normal" },
  { level: "high", token: "text-priority-high", bar: "bg-priority-high" },
  { level: "urgent", token: "text-priority-urgent", bar: "bg-priority-urgent" },
] as const;

export type PriorityLevel = (typeof PRIORITIES)[number]["level"];

/** A prediction against one head, as the bench draws it. */
export interface Head {
  label: string;
  confidence: number;
}

export interface BenchSample {
  /** The customer, as a support desk would see them. */
  from: string;
  company: string;
  received: string;
  /** Verbatim ticket text, in the register the training corpus uses. */
  body: string;
  category: Head;
  /** Second-place class. Drawn dim, and it is the whole story on sample 3. */
  runnerUp?: Head;
  priority: Head & { level: PriorityLevel };
  /** Null when the model declined to call it. */
  department: string | null;
  /**
   * True when category confidence fell under the calibrated review threshold.
   * The ticket goes to a person and no prediction is recorded against it.
   */
  needsReview: boolean;
  /** Why this example is on the page. Shown as the bench's footnote. */
  note: string;
}

/*
  Three tickets, chosen to show three different outcomes rather than three wins.

  The third is the point of the whole component: it is genuinely ambiguous to a
  human reader too — plan entitlement, an invoice dispute and a missing module
  in one paragraph — so the model declining to call it reads as correct
  behaviour instead of a staged failure.
*/
export const SAMPLES: BenchSample[] = [
  {
    from: "Deniz Aktaş",
    company: "Kayra Tekstil",
    received: "2 min ago",
    body: "The payout that closed on Friday never reached our account. The dashboard still shows it as settled but nothing has landed at the bank, and we have supplier invoices due Monday.",
    category: { label: "payment_ops", confidence: 0.96 },
    runnerUp: { label: "billing", confidence: 0.02 },
    priority: { label: "urgent", level: "urgent", confidence: 0.88 },
    department: "Payment Operations",
    needsReview: false,
    note: "Money that has left the platform but not arrived. Both heads agree, and it routes without anyone reading it.",
  },
  {
    from: "Mert Yıldırım",
    company: "Orbit Home",
    received: "6 min ago",
    body: "Since this morning our cargo integration has stopped pushing tracking numbers. Orders ship but customers never get the notification. Nothing changed on our side.",
    category: { label: "integration", confidence: 0.91 },
    runnerUp: { label: "technical_issue", confidence: 0.06 },
    priority: { label: "high", level: "high", confidence: 0.79 },
    department: "Integration Support",
    needsReview: false,
    note: "Reads like a platform outage and is not one. integration and technical_issue are separate classes because they are separate teams.",
  },
  {
    from: "Selin Barış",
    company: "Northwind Trade",
    received: "11 min ago",
    body: "We were told our plan included the analytics module but I cannot see it in the panel, and last month's invoice came in higher than the quote. Can someone explain what we are actually paying for?",
    category: { label: "billing", confidence: 0.41 },
    runnerUp: { label: "sales", confidence: 0.33 },
    priority: { label: "normal", level: "normal", confidence: 0.55 },
    department: null,
    needsReview: true,
    note: "Entitlement, an invoice dispute and a missing feature in one paragraph. Under the review threshold, so it goes to a person and no prediction is recorded.",
  },
];
