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
| `ticket:reopen_own` | — | — | — | ✓ |
| `ticket:assign` | ✓ | ✓ | ✓ | — |
| `ticket:delete` | ✓ | ✓ | — | — |
| `message:create` | ✓ | ✓ | ✓ | ✓ (own ticket) |
| `department:manage` | ✓ | ✓ | — | — |
| `customer:manage` | ✓ | ✓ | ✓ | — |
| `analysis:read` | ✓ | ✓ | ✓ | — |
| `stats:read` | ✓ | ✓ | — | — |
| `user:read` (staff roster, invitation list) | ✓ | ✓ | — | — |
| `user:write` (assign staff team, invite staff) | ✓ | ✓ | — | — |

Seeded role names that carry these grants (see `cmd/seed` `templateRoleDefs`):

| Seed role | Contract column | Notes |
|---|---|---|
| `admin` | Super Admin | `*` |
| `org_admin` | Owner | org/user/ticket/department/stats grants |
| `agent` | Agent | queue work; `ticket:read` lists departments for routing only |
| `customer` | Customer | own-ticket scope |
| `viewer` | — | `*:read` (read-only management views) |

### How own-scope is decided

The scoping test is **the absence of `ticket:read`**, never `if role ==
"customer"`. A caller holding `ticket:read` sees the organization; anyone else
is narrowed to the customer record their login is attached to. That keeps the
rule correct for any future role without the code learning its name.

The narrowing is applied in `usecase.Scope` and covers, together:

- `GET /tickets` — `customer_id` is **overwritten**, not defaulted. A
  client-supplied owner filter is ignored outright.
- `GET /tickets/{id}`, `GET|POST /tickets/{id}/messages`, `PATCH /tickets/{id}`
  — a ticket belonging to another customer answers **404**, never 403. A 403
  would confirm the id exists, which is enough to enumerate a tenant's tickets.
- internal notes — dropped server-side for a customer scope. The portal filters
  them again, but that is cosmetic; the cut happens here.

### Customers and logins

Customers live in the `customers` table and are **not** platform users. A
customer may exist with no login at all — an agent can create one for somebody
who has never signed in.

`customers.user_id` (nullable, unique per organization) links a customer to the
`users` row they authenticate as. That link is how a portal request resolves
"which customer am I"; the token carries `user_id` and `roles`, and the
`customer_id` is resolved from the link rather than trusted from the client.

> **Deviation from the original draft:** an earlier version of this document
> said a Customer token carries `customer_id` instead of `user_id`. It does not.
> Customers authenticate through the same `/auth/login` as staff and are told
> apart by their permissions.

---

## 3. Data model

Only the columns the API depends on. Full DDL lives in the migrations.

### `departments`
`id`, `organization_id`, `name`, `description`, `is_default` (bool), `created_at`, `updated_at`

Exactly one `is_default = true` row per organization ("General"). Seeded on org
creation. Cannot be deleted; tickets of a deleted department fall back to it.

### `staff_departments`
`organization_id`, `user_id`, `department_id`, `assigned_at`, `updated_at`
Primary key `(organization_id, user_id)` — one department per person per
organization.

Which support team somebody works on. Its own table rather than a column:
`organization_users` is never written (membership is read from `user_roles`),
and `users` is global to the platform while a department belongs to one
organization, so the organization has to be part of the key.

**The absence of a row means "on the roster but on no team"** — a normal state,
not a broken one. `department_id` cascades on delete, so removing a department
returns its people to that state rather than moving them somewhere they were
not chosen for.

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
{"data":[{"id":"...","name":"Billing","description":"","category":null,
          "is_default":false,"ticket_count":12,"staff_count":3}]}
```
`ticket_count` is every ticket ever routed here. `staff_count` is how many
people are assigned to it, counted over the same roster `GET /staff` serves —
so the number beside a department always matches the list behind it. Portal
customers are excluded from both the count and the list.

**`POST /departments`** — `department:manage`
```json
{"name":"Billing","description":"Payment and invoice issues"}
```
→ `201` with the created department.

**`PATCH /departments/{id}`** — `department:manage`. Body: any of `name`, `description`.

**`DELETE /departments/{id}`** — `department:manage`.
`409` if `is_default`. Otherwise reassigns its tickets to the default
department, and **unassigns** its staff — the two are deliberately different. A
ticket has to belong somewhere or it falls out of every queue; a person placed
on a team nobody chose for them is a staffing decision made by a delete button.

---

### 4.2 Staff

The support roster: who works here and on which team.

Distinct from `GET /users`, which returns every account holding a role in the
organization — including portal customers, who hold one exactly like an agent
does. These routes exclude them in SQL via `customers.user_id`, so "staff" here
means colleagues and nothing else.

**`GET /staff`** — `user:read`

Query: `department_id` (a UUID, or the literal `unassigned`), `q` (matches name
or email), `page`, `page_size`.

```json
{"data":[{"id":"...","email":"selin@acme.com","first_name":"Selin",
          "last_name":"Aydın","full_name":"Selin Aydın","status":"active",
          "department":{"id":"...","name":"Technical Support"},
          "created_at":"2026-08-01T09:00:00Z"}],
 "meta":{"page":1,"page_size":25,"total":9}}
```

`department` is `null` for somebody on no team. `full_name` is computed
server-side and falls back to the email, since first and last name are nullable.

`?department_id=unassigned` returns the people on no team — the absence of a
row, which no equality comparison can express. It beats a UUID if both are
sent, mirroring `?assignee_id=unassigned` on the ticket queue.

**`PUT /staff/{userId}/department`** — `user:write`
```json
{"department_id":"..."}
```
→ `200` with the updated `StaffInfo`.

`{"department_id": null}` takes the person off every team. A PUT rather than a
PATCH for that reason: the request replaces the assignment wholesale, so "no
department" can be stated rather than implied by omitting a field.

Idempotent — assigning somebody to the team they are already on succeeds.

`404` when the user is not staff in the caller's organization. That covers three
cases deliberately made indistinguishable: no such user, a user in another
tenant, and one of this organization's own portal customers.

`404` likewise when `department_id` names a department outside the caller's
organization. The foreign keys do not catch this — both rows exist, they just do
not belong together.

`user:write`, not `department:manage`: the subject of the change is the person,
and the seeded `agent` role holds `department:manage`, which would otherwise let
an agent move themselves onto whichever team they liked.

---

### 4.3 Customers

**`GET /customers`** — `customer:manage`. Query: `q` (matches email or name).

**`POST /customers`** — `customer:manage`
```json
{"email":"alice@acme.com","full_name":"Alice Morgan","company":"Acme"}
```
`409` if the email already exists in this organization.

**`GET /customers/{id}`** — `customer:manage`. Includes `ticket_count` and the
5 most recent tickets.

---

### 4.4 Tickets

**`POST /tickets`** — `ticket:create`

```json
{
  "subject": "Cannot log in after password reset",
  "body": "I reset my password yesterday and now the app says...",
  "customer_id": "uuid"
}
```

- `customer_id` is required when an Agent/Owner creates the ticket, and ignored
  when a Customer creates it — theirs is resolved from the token. A body that
  names a different customer is not rejected, it is discarded: the field was
  never the client's to set.
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
| `sort` | `created_at`, `-created_at` (default), `priority`, `-priority`, `updated_at`, `-updated_at` |
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
      "snippet": "I reset my password yesterday and now the app says...",
      "first_response_at": "2026-07-31T09:31:00Z",
      "created_at": "2026-07-31T09:12:00Z",
      "updated_at": "2026-07-31T09:40:00Z"
    }
  ],
  "meta": {"page":1,"page_size":25,"total":137}
}
```

`snippet` is the opening message truncated to 240 characters — tickets have no
description column, so this is the only way a list renders a preview without a
query per row. `first_response_at` is the first non-internal reply that was not
the customer's own; absent while nobody has answered.

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

**`PATCH /tickets/{id}`** — `ticket:update`, or `ticket:reopen_own`

```json
{"priority":"urgent","department_id":"uuid","status":"in_progress"}
```

A caller holding only `ticket:reopen_own` may send exactly one thing: `status`
set to `open`, on their own ticket, when it is currently `resolved` or `closed`.
`priority` or `department_id` in the body is `403`; any other status is `403`;
a ticket that is not resolved is `409`.

That narrowness is the product decision, not an oversight. A customer who could
set their own priority would make triage meaningless — every request would
arrive urgent — which is why reopening gets its own grant instead of
`ticket:update`.

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

**`GET /tickets/summary`** — `ticket:read` or `ticket:read_own`

The portal dashboard's counters, scoped exactly like the list: an agent gets the
organization's figures, a customer gets their own.

```json
{
  "total": 7,
  "open": 5,
  "waiting_customer": 0,
  "resolved": 2,
  "by_status": {"open":3,"in_progress":2,"resolved":1,"closed":1},
  "avg_first_response_minutes": 47,
  "avg_resolution_minutes": 2730
}
```

`open` folds `open` + `in_progress`; `resolved` folds `resolved` + `closed`.
Both averages are `null` until there is something to average — a customer whose
first ticket is still open has no average response time, and `0` would read as
an instant reply.

Separate from `/stats/overview` on purpose: that one is gated on `stats:read`
and reports on the triage queue, which is not a customer's business even about
their own tickets.

---

### 4.5 Stats

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

### 4.6 Account

**`POST /auth/change-password`** — any authenticated caller, own account only.

```json
{"current_password":"...","new_password":"..."}
```

→ `204`. `401` if `current_password` does not verify. The account is taken from
the token; no body field names a user.

Mounted under the authenticated routes despite the `/auth` path — `/auth/*` is
otherwise the public subtree.

> Existing tokens keep working after a change. JWTs are stateless and there is
> no revocation list, so this does not end a session opened before it.

---

### 4.7 Staff invitations

How somebody becomes staff. Before this existed the only path was for the person
to self-register on the public `POST /auth/register` — which attaches them to no
organization and grants nothing — after which an administrator had to find their
user id and call `POST /roles/assign`. That forced registration to stay open on
the internet.

`POST /auth/register` is now mounted only when `ALLOW_PUBLIC_REGISTRATION` is
set (true in development, false otherwise). When it is off the route does not
exist and answers `404` — not `403`, which would advertise it.

> **Breaking change to `POST /roles/assign`.** It no longer takes
> `organization_id`; the organization is the caller's, from the token, and a
> body field is ignored. It also now verifies the role belongs to that
> organization, answering `404` otherwise.
>
> As written before, a caller holding `user:write` in any organization could
> write a `user_roles` row into any tenant whose role id they could obtain —
> neither the organization nor the role's ownership was checked. The ids being
> unguessable is not an authorization check. This is the rule §1 already states:
> the organization comes from the claims, never from a request body.

**`POST /invitations`** — `user:write`.

```json
{"email":"ada@acme.com","role_id":"<uuid>","department_id":"<uuid>|null"}
```

→ `201`:

```json
{"id":"...","email":"ada@acme.com","role_id":"...","role_name":"agent",
 "department_id":"...","status":"pending","expires_at":"...","created_at":"...",
 "accept_url":"https://app.example.com/invite/inv_9f2c…"}
```

- The organization comes from the token. There is no `organization_id` field.
- `role_id` must name a role in the caller's organization; one from another
  tenant answers `404`, never `403`.
- `department_id` is optional. When set, the invitee lands on that team on
  acceptance instead of in the roster's Unassigned bucket.
- `accept_url` appears **only here**. Only the token's SHA-256 is stored, so the
  link cannot be reproduced afterwards — `GET /invitations` never returns it.
- `409` if the address already holds a live invitation (revoke it first) or is
  already a member.
- A mail delivery failure does **not** fail the call. The invitation is the
  record and the link is in this response; the failure is logged.

**`GET /invitations`** — `user:read`. Paginated. Includes spent, revoked and
expired invitations; `status` is derived from the timestamps, not stored.

**`DELETE /invitations/{invitationId}`** — `user:write` → `204`. Scoped to the
caller's organization inside the update, so another tenant's id answers `404`.
Refuses an already-accepted invitation: revoking one would read as withdrawing
the membership, which it does not do.

**`GET /roles`** — `user:read`. The organization's assignable roles:

```json
{"data":[{"id":"...","name":"agent","description":"..."}]}
```

Exists because a role id was otherwise unobtainable: the ids are
per-organization clones, and both inviting and `POST /roles/assign` take one.
The seeded `customer` role is **excluded** — these are staff roles, and offering
a portal role on a staffing screen puts somebody in the organization but on no
roster. Filtered server-side so two clients cannot disagree about it.

**`GET /auth/invitations/{token}`** — public. What the acceptance screen shows:

```json
{"email":"ada@acme.com","organization_name":"Acme","role_name":"agent",
 "expires_at":"...","has_account":false}
```

`has_account` says the address already has a TicketLens login. The screen must
then **not** ask for a password: acceptance joins that account and leaves its
credentials alone, so anything typed would be discarded and the person could not
sign in with it. It discloses only whether the address the invitation already
names has an account, so it is not an oracle for arbitrary addresses.

**`POST /auth/invitations/{token}/accept`** — public.

```json
{"first_name":"Ada","last_name":"Lovelace","password":"..."}
```

All three fields are required only when an account has to be created. When
`has_account` is true they are ignored and may be omitted entirely; when it is
false, omitting them is `422`.

→ `201` with the user. No token: the caller has just chosen a password and signs
in with it, as registration already does. Issuing a session straight from a
link-bearing request would make the invitation email a one-click login for
anyone who read it.

There is no `email` field — the address is fixed by the invitation. Accepting
under an address of the caller's choosing would let a token holder open an
account as anybody.

If the address already has an account it is **joined, not duplicated**, and its
password is untouched: users are global, roles are per-organization, and setting
the password here would turn inviting a known address into a password reset for
it.

> **Every failed redemption gives the same answer.** Unknown, expired, revoked
> and already-spent tokens all return the identical `404` with the identical
> message. Distinguishing them confirms which tokens were once real, which is
> what somebody probing for live invitations wants to learn.

Acceptance is one transaction: account, role assignment, department placement
and spending the invitation commit together or not at all. A partial acceptance
would leave a login belonging to no organization whose address can no longer be
invited.

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