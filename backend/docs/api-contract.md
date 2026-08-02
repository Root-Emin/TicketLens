# TicketLens — API Contract (v0)

Scope of this document: the **ticketing** bounded context that we add on top of the
masterfabric-go template. IAM (auth, users, roles, permissions), tenancy and audit
already exist in the template and are referenced here, not redefined.

This file is the single source of truth for both the Go handlers and the Next.js
API client. If an endpoint changes, it changes here first.

---

## 1. Conventions

- Base path: `/api/v1`
- Auth: `Authorization: Bearer <access_token>` on every endpoint except `/auth/*`
  and `/health/*`.
- Every resource is scoped to the caller's `organization_id`, taken from the JWT
  claims — **never** from a request body, query parameter or path segment. Cross-org
  reads must return `404`, not `403` (do not leak existence).
- Where a path does carry an id (the template's `/organizations/{orgId}/apps/{appId}/…`
  routes), the id is checked against the token before the handler runs, and a child
  id is checked against its parent. A permission is granted *within* an
  organization, so RBAC alone never authorizes a path naming a different one.
- `X-Organization-ID` may only repeat the organization already in the token.
  A disagreeing value is `403`; the header cannot establish a tenant by itself.
- IDs are UUID v4.
- Timestamps are RFC3339 UTC (`2026-07-31T14:05:00Z`).
- Error envelope: reuse the template's existing error response shape. Do not
  invent a second one. If the template returns
  `{"error": {"code": "...", "message": "..."}}`, all new handlers match it.
- List endpoints are paginated:
  `?page=1&page_size=25` → `{"data": [...], "meta": {"page":1,"page_size":25,"total":137}}`
  `page_size` max 100.

### Enums

| Field | Values |
|---|---|
| `priority` | `low`, `normal`, `high`, `urgent` |
| `status` | `open`, `in_progress`, `pending_customer`, `resolved`, `closed` |
| `author_type` (message) | `customer`, `agent`, `system` |

---

## 2. Roles and permissions

Checks are permission-based, never `if role == "owner"`.

| Permission | Super Admin | Owner | Agent | Customer |
|---|:-:|:-:|:-:|:-:|
| `ticket:create` | ✓ | ✓ | ✓ | ✓ |
| `ticket:read` (all in org) | ✓ | ✓ | ✓ | — |
| `ticket:read_own` | — | — | — | ✓ |
| `ticket:update` | ✓ | ✓ | ✓ | — |
| `ticket:assign` | ✓ | ✓ | ✓ | — |
| `ticket:delete` | ✓ | ✓ | — | — |
| `message:create` | ✓ | ✓ | ✓ | ✓ (own ticket) |
| `department:manage` | ✓ | ✓ | — | — |
| `customer:manage` | ✓ | ✓ | ✓ | — |
| `analysis:read` | ✓ | ✓ | ✓ | — |
| `stats:read` | ✓ | ✓ | — | — |

Customers are **not** platform users. They live in a separate `customers` table
and authenticate through a separate portal flow. A Customer token carries
`customer_id` instead of `user_id`.

---

## 3. Data model

Only the columns the API depends on. Full DDL lives in the migrations.

### `departments`
`id`, `organization_id`, `name`, `description`, `is_default` (bool), `created_at`, `updated_at`

Exactly one `is_default = true` row per organization ("General"). Seeded on org
creation. Cannot be deleted; tickets of a deleted department fall back to it.

### `customers`
`id`, `organization_id`, `email`, `full_name`, `company`, `created_at`
Unique on `(organization_id, email)`.

### `tickets`
`id`, `organization_id`, `customer_id`, `department_id`, `assignee_id` (nullable,
FK users), `subject`, `status`, `priority`, `priority_overridden` (bool, default
false), `department_overridden` (bool, default false), `created_at`, `updated_at`,
`resolved_at` (nullable)

No `description` column — the first `ticket_message` is the description.

`priority` and `department_id` hold the **current effective** value. The model's
original guess stays in `ai_analyses`. When a human changes either field, the
corresponding `*_overridden` flag flips to true. This is what powers the
"X% of AI predictions were accepted" metric.

### `ticket_messages`
`id`, `ticket_id`, `author_type`, `author_id` (user or customer id), `body`,
`is_internal` (bool — internal notes are hidden from the customer portal),
`created_at`

### `ai_analyses`
`id`, `ticket_id`, `predicted_priority`, `priority_confidence` (0–1),
`predicted_department_id`, `department_confidence` (0–1), `needs_human_review`
(bool), `model_name`, `model_version`, `raw_response` (JSONB), `created_at`

Append-only. A ticket can have several analyses (re-runs, model comparisons).
"The" analysis for display purposes is the most recent one.

---

## 4. Endpoints

### 4.1 Departments

**`GET /departments`** — `department:manage` or `ticket:read`
```json
{"data":[{"id":"...","name":"Billing","description":"","is_default":false,"ticket_count":12}]}
```

**`POST /departments`** — `department:manage`
```json
{"name":"Billing","description":"Payment and invoice issues"}
```
→ `201` with the created department.

**`PATCH /departments/{id}`** — `department:manage`. Body: any of `name`, `description`.

**`DELETE /departments/{id}`** — `department:manage`.
`409` if `is_default`. Otherwise reassigns its tickets to the default department.

---

### 4.2 Customers

**`GET /customers`** — `customer:manage`. Query: `q` (matches email or name).

**`POST /customers`** — `customer:manage`
```json
{"email":"alice@acme.com","full_name":"Alice Morgan","company":"Acme"}
```
`409` if the email already exists in this organization.

**`GET /customers/{id}`** — `customer:manage`. Includes `ticket_count` and the
5 most recent tickets.

---

### 4.3 Tickets

**`POST /tickets`** — `ticket:create`

```json
{
  "subject": "Cannot log in after password reset",
  "body": "I reset my password yesterday and now the app says...",
  "customer_id": "uuid"
}
```

- `customer_id` is required when an Agent/Owner creates the ticket, and ignored
  when a Customer creates it (taken from the token).
- `department_id` and `priority` are **not** accepted here. They are set by the
  classifier. Until the first analysis arrives the ticket sits in the default
  department at `normal`.
- Creates the ticket plus the first `ticket_message` (`author_type: customer`).
- Publishes a `ticket.created` event so the classification consumer can pick it up.

→ `201`, full ticket object (see GET below), `ai_analysis: null`.

---

**`GET /tickets`** — `ticket:read`, or `ticket:read_own` (auto-filtered to the
caller's own tickets)

Query parameters, all optional and combinable:

| Param | Notes |
|---|---|
| `status` | repeatable: `?status=open&status=in_progress` |
| `priority` | repeatable |
| `department_id` | |
| `assignee_id` | `unassigned` is a valid literal |
| `needs_review` | `true` → only tickets whose latest analysis has `needs_human_review = true` |
| `overridden` | `true` → only tickets a human corrected |
| `q` | substring match on subject |
| `sort` | `created_at`, `-created_at` (default), `priority`, `-priority` |
| `page`, `page_size` | |

Each list item carries just enough analysis data to render the queue without an
N+1:

```json
{
  "data": [
    {
      "id": "uuid",
      "subject": "Cannot log in after password reset",
      "status": "open",
      "priority": "high",
      "priority_overridden": false,
      "department": {"id":"uuid","name":"Technical"},
      "customer": {"id":"uuid","full_name":"Alice Morgan","email":"alice@acme.com"},
      "assignee": null,
      "message_count": 3,
      "latest_analysis": {
        "predicted_category": "integration",
        "predicted_priority": "high",
        "priority_confidence": 0.94,
        "department_confidence": 0.71,
        "needs_human_review": false,
        "mapping_fallback": false,
        "model_name": "ticketlens-berturk"
      },
      "created_at": "2026-07-31T09:12:00Z",
      "updated_at": "2026-07-31T09:40:00Z"
    }
  ],
  "meta": {"page":1,"page_size":25,"total":137}
}
```

`latest_analysis` is `null` while classification is pending — the UI renders a
"analyzing" state rather than a zero confidence.

---

**`GET /tickets/{id}`** — `ticket:read` / `ticket:read_own`

Same shape as the list item, plus `messages` (ascending) and `analyses`
(descending, full objects including `raw_response`).

Also carries `review_threshold` — the confidence cutoff currently configured via
`CLASSIFIER_REVIEW_THRESHOLD`. It is published so a client can draw the cutoff on
a confidence meter without hardcoding its own copy. It is the *current* setting,
not the one each analysis was judged under; the recorded verdict is
`needs_human_review` on the analysis itself. Every endpoint returning a ticket
detail (`POST /tickets`, `GET/PATCH /tickets/{id}`, assign, re-analyze) includes
it.

---

**`PATCH /tickets/{id}`** — `ticket:update`

```json
{"priority":"urgent","department_id":"uuid","status":"in_progress"}
```

Rules:
- Changing `priority` to a value different from the latest analysis'
  `predicted_priority` sets `priority_overridden = true`. Same rule for
  `department_id` / `department_overridden`.
- Setting `status` to `resolved` stamps `resolved_at`. Moving away from
  `resolved` clears it.
- Writes an audit log entry using the template's existing audit service.

---

**`POST /tickets/{id}/assign`** — `ticket:assign`

```json
{"assignee_id": "uuid"}
```
`null` unassigns. `422` if the user is not in this organization.

---

**`GET /tickets/{id}/messages`** — `ticket:read` / `ticket:read_own`
Customers never receive messages with `is_internal = true`.

**`POST /tickets/{id}/messages`** — `message:create`
```json
{"body":"We've reset the session, please try again.","is_internal":false}
```
`author_type` and `author_id` come from the token, never from the body.

---

**`GET /tickets/{id}/analyses`** — `analysis:read`
Full history, newest first. This is the "lens" view: what the model predicted,
how sure it was, and whether a human disagreed.

**`POST /tickets/{id}/analyses`** — `analysis:read` + `ticket:update`
Re-runs classification on demand. Useful for the live demo. Returns `202` and
the new analysis once the classifier responds.

---

### 4.4 Stats

**`GET /stats/overview`** — `stats:read`. Query: `from`, `to` (RFC3339, default last 30 days).

```json
{
  "tickets": {"total":137,"open":41,"in_progress":22,"resolved":74},
  "by_priority": {"low":18,"normal":63,"high":41,"urgent":15},
  "by_department": [{"department_id":"uuid","name":"Technical","count":58}],
  "ai": {
    "analyzed": 130,
    "priority_accept_rate": 0.86,
    "department_accept_rate": 0.79,
    "avg_priority_confidence": 0.88,
    "needs_review": 11
  }
}
```

`priority_accept_rate` = tickets with an analysis and `priority_overridden = false`,
divided by tickets with an analysis. This is the headline number of the project —
compute it in SQL, not in the frontend.

---

## 5. Classifier integration

The Go side depends on a port, not on HTTP:

```go
// internal/domain/triage/port/classifier.go
type Classification struct {
    Priority           string
    PriorityConfidence float64
    Department         string
    DepartmentConfidence float64
    NeedsHumanReview   bool
    ModelName          string
    ModelVersion       string
    Raw                json.RawMessage
}

type Classifier interface {
    Classify(ctx context.Context, subject, body string) (Classification, error)
}
```

Adapters (the env var is `CLASSIFIER_URL`; there is no `ML_SERVICE_URL`):
- `infrastructure/classifier/http` (`httpclassifier`) → `POST {CLASSIFIER_URL}/classify`
- `infrastructure/classifier/stub` → deterministic keyword fake, used in tests
  and whenever `CLASSIFIER_URL` is empty. **The backend must start and work
  without the Python service running.**

The classifier is called from an event consumer on `ticket.created`, never
inline in the HTTP handler — a cold Render instance takes 30+ seconds to wake and
that must not block ticket creation.

`Department` comes back as a name string; the adapter maps it to a
`department_id` within the organization, falling back to the default department
when there is no match.

---

## 6. Seed data requirements

For the demo to be legible, `scripts/seed.go` should produce:

- 2 organizations, each with an Owner, 2 Agents, 4 departments (one default)
- 8–10 customers
- 15–20 tickets spread across every status and priority
- An `ai_analysis` for every ticket **except two** (so the pending state is visible)
- Confidence values spread realistically: mostly 0.85–0.98, three or four in the
  0.45–0.65 band with `needs_human_review = true`
- 3–4 tickets with `priority_overridden = true`, so the accept-rate metric is not
  a suspiciously round 100%

Analyses are seeded, not generated live. The live classifier call is demonstrated
on a single ticket during the presentation.