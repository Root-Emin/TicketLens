# TicketLens Classification Taxonomy (frozen)

This document is the single source of truth for the ten ticket categories and the
four priority levels. It is **frozen**: the category set does not change once
synthetic data generation begins, because splitting or merging a category after
thousands of examples exist means relabelling the entire corpus.

Everything downstream reads from here:

- The Go constants in `internal/domain/triage/model/category.go` and `ticket.go`.
- The stub classifier keyword tables in `internal/infrastructure/classifier/stub/stub.go`.
- The synthetic data generation prompt (`backend/ml/`), which quotes these
  definitions, and its `taxonomy.py` mirror.
- The frontend filter lists in `frontend/src/lib/api/labels.ts`.
- The evaluation relabelling guide, so a human relabelling an external dataset
  applies the same boundaries the generator did.

If any of those drift from this document, they are wrong, not this file.

---

## Categories

The category answers **"which team owns this ticket?"** — it routes work. It is
independent of priority, which answers "how fast?".

There is deliberately **no `other` / `catch-all` category**. A ticket the model
cannot place with confidence is surfaced through `needs_human_review`, not filed
under a junk drawer that would train the model to give up.

| Category | Owns | One-line test |
|---|---|---|
| `technical_issue` | Product is broken for a user | "The software itself is malfunctioning." |
| `integration` | A connection to an external system is broken | "The break is at the boundary with a marketplace, ERP, cargo, or API." |
| `payment_ops` | Money movement after a sale | "A payout, settlement, refund, or reconciliation is wrong." |
| `billing` | What the customer pays *us* | "It is about our invoice, subscription, or commission to the customer." |
| `onboarding` | Getting a new customer live | "They are not operational yet; this is setup or migration." |
| `how_to` | Usage question, nothing is broken | "The answer is documentation, not a fix." |
| `account_access` | Login, permissions, users, roles | "They cannot get in, or cannot grant someone else in." |
| `feature_request` | Something the product does not do yet | "The correct answer is 'that is on the roadmap' or 'noted'." |
| `sales` | Pre-purchase commercial intent | "They want to buy more, or buy at all." |
| `compliance` | Legal, regulatory, data-governance | "A regulation (KVKK/GDPR), contract, or audit drives the request." |

### Definitions and boundaries

Each entry lists what belongs, what does **not** (with the category it goes to
instead), and the boundary rule against the neighbour it is most confused with.

#### `technical_issue`
- **In scope:** the platform's own UI or backend is failing — pages not loading,
  500 errors, timeouts, crashes, broken exports, slowness.
- **Out of scope:** a failure in a link to an external system → `integration`;
  a wrong payout or refund → `payment_ops`.
- **Boundary vs `integration`:** ask *where* the break is. If the failing thing
  is a marketplace sync, ERP transfer, cargo label, webhook, or API call, it is
  `integration` even though it is also technically "broken". `technical_issue` is
  breakage inside our own product surface.

#### `integration`
- **In scope:** anything at the seam with a third party — Trendyol/Hepsiburada/N11
  marketplaces, ERP (Logo, Mikro, Netsis), cargo/shipping connectors, virtual POS
  connectors, webhooks, SDKs, and the public API.
- **Out of scope:** the money being wrong once it did arrive → `payment_ops`; a
  general panel error unrelated to any connector → `technical_issue`.
- **Boundary vs `payment_ops`:** a *virtual POS integration returning an error at
  checkout* is `integration` (the connector is broken). A *payout that did not
  arrive* is `payment_ops` (the connector worked; the money is wrong).

#### `payment_ops`
- **In scope:** operational money movement tied to sales — payouts, settlements,
  reconciliation differences, refunds to customers, chargebacks/disputes, missing
  transactions, delayed deposits.
- **Out of scope:** what the customer owes us → `billing`; a broken payment
  *connector* → `integration`.
- **Boundary vs `billing`:** `payment_ops` is money flowing *to the merchant*
  from their own sales. `billing` is money flowing *from the merchant to us* for
  the service. "My payout is short" is `payment_ops`; "my invoice is wrong" is
  `billing`. A commission that reduces a *payout* is `payment_ops`; a commission
  line on *our invoice* is `billing`.

#### `billing`
- **In scope:** our invoices to the customer, subscription plans, plan
  upgrades/downgrades, cancellations, and the commission/pricing terms we charge.
- **Out of scope:** a pre-sale request to buy more → `sales`; the merchant's own
  sales revenue → `payment_ops`.
- **Boundary vs `sales`:** `billing` is about an *existing* paid relationship
  (an invoice, the current subscription). `sales` is about *acquiring* new
  product (a quote, a demo, a new module). "Upgrade my subscription" is `billing`
  (changing what I already pay). "Quote me the add-on module" is `sales`.

#### `onboarding`
- **In scope:** activities before a customer is operational — initial setup,
  data migration from an old system, go-live steps, activation.
- **Out of scope:** a how-to question from a live customer → `how_to`; a login
  problem during setup → `account_access`.
- **Boundary vs `how_to`:** `onboarding` is the one-time act of becoming
  operational. `how_to` is a recurring usage question from someone already live.
  "How long does data migration take?" is `onboarding`; "How do I filter
  reports?" is `how_to`.

#### `how_to`
- **In scope:** questions where the answer is guidance/documentation and nothing
  is broken — how to create a campaign, where the manual is, requests for
  training.
- **Out of scope:** it's broken → `technical_issue`; it's first-time setup →
  `onboarding`; it's a request for a capability that does not exist →
  `feature_request`.
- **Boundary vs `feature_request`:** `how_to` asks how to use something that
  exists. `feature_request` asks for something that does not. "How do I bulk-edit
  discounts?" is `how_to` if the feature exists; "please add bulk discount
  editing" is `feature_request`.

#### `account_access`
- **In scope:** authentication and authorization — password resets, cannot log
  in, sessions, adding users, roles, permissions, 2FA.
- **Out of scope:** the panel is down for everyone → `technical_issue`.
- **Boundary vs `technical_issue`:** if the blocker is credentials/permissions
  (one user cannot get in, or cannot grant access), it is `account_access`. If the
  system is failing regardless of who logs in, it is `technical_issue`.

#### `feature_request`
- **In scope:** requests for capabilities the product does not have, roadmap
  questions, suggestions, "do you support X?".
- **Out of scope:** the capability exists and they need help using it → `how_to`.
- **Boundary vs `how_to`:** see `how_to` above — existence of the feature decides.

#### `sales`
- **In scope:** pre-purchase commercial intent — quotes, demos, price lists,
  buying add-on modules or new packages.
- **Out of scope:** changing an existing paid plan → `billing`; a capability that
  does not exist yet → `feature_request`.
- **Boundary vs `billing`:** see `billing` above — new acquisition vs existing
  relationship.

#### `compliance`
- **In scope:** legal and regulatory requests — KVKK/GDPR data deletion, privacy
  notices, contracts, audits, document requests for governance.
- **Out of scope:** a general document request unrelated to regulation/audit is
  usually `how_to` or `billing` depending on the document.
- **Boundary rule:** the request must be *driven by* a regulation, contract, or
  audit obligation. "Send me my invoice" is `billing`; "send me the audit
  documents required for KVKK" is `compliance`.

---

## Priority

Priority answers **"how fast?"** and is driven by **business impact**, not tone.
An angry message about a minor question is not urgent; a calm message that "the
site is down" is.

| Priority | Meaning | Signal |
|---|---|---|
| `urgent` | The business cannot operate | Site down, cannot take orders, cannot get paid, customers affected. Revenue is stopped *now*. |
| `high` | Real breakage, business still runs | A feature or integration is failing but there is a workaround or partial function. |
| `normal` | Default | Standard request with no stoppage and no explicit urgency. |
| `low` | No breakage | Questions, how-tos, suggestions, pre-sales — nothing is wrong. |

**Known difficulty:** priority is genuinely harder than category, because the
line between "urgent" and "high" ("completely unable to operate" vs "painful but
has a workaround") is often not stated in the text. Report priority metrics
separately from category metrics; a lower macro-F1 on the priority head is
expected and is a property of the problem, not a training failure.

---

## Changing this document

Do not, once generation has started. If a category genuinely must change:

1. Stop generation.
2. Update this file, then `category.go` / `ticket.go`, then the stub keyword
   tables, then the generator prompt, then the DB `CHECK` constraint migration.
3. Relabel the entire existing corpus and the evaluation set.
4. Re-baseline the stub and retrain.

The cost of that sequence is the reason the set is frozen here.
