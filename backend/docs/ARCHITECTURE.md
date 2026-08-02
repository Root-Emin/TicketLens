# TicketLens Architecture

TicketLens is a multi-tenant AI triage platform: agents work a ticket queue,
a classifier proposes priority and category, and humans override when confidence
is low. This document describes the running system. The older
[`MIMARI_REHBER.md`](./MIMARI_REHBER.md) describes the masterfabric-go template
TicketLens was forked from; prefer this file for product-specific guidance.

## Bounded contexts

| Context | Owns |
|---|---|
| **IAM** | Users, roles, JWT, RBAC |
| **Tenant** | Organizations, workspaces, apps, API keys |
| **API Management** | Managed endpoints + policies (gateway control plane) |
| **Triage** | Departments, customers, tickets, messages, AI analyses, stats |
| **Realtime** | WebSocket hub + event bridge |

## Request path (agent API)

```
Browser → Next.js (/api/proxy) → Go chi router
  → JWTAuth → TenantResolver → RBAC → triage handler → use case → postgres
```

Organization scope comes from the JWT claim only. Path `{orgId}` / `{appId}`
guards compare against that claim (404 on mismatch). See `SECURITY.md`.

## Classification pipeline

```
POST /tickets
  → CreateTicketUseCase (ticket + first message in one Tx)
  → publish ticket.created
  → TicketConsumer → AnalyzeTicketUseCase
       → port.Classifier (stub or HTTP → Python /classify)
       → map category → department (or default + mapping_fallback)
       → persist ai_analyses (append-only)
       → apply prediction unless *_overridden
       → publish analysis.completed → WebSocket triage channel
```

`port.Classifier` is the only seam. Wire via `CLASSIFIER_URL`; empty keeps the
keyword stub. HTTP adapter retries and can fall back to the stub.

## Taxonomy

Frozen in [`taxonomy.md`](./taxonomy.md). Ten categories, four priorities.
Copies: `backend/ml/src/ticketlens_ml/taxonomy.py` and
`frontend/src/lib/api/labels.ts`. Both are checked against the Go constants by
`backend/ml/tests/test_taxonomy_sync.py` (`make ml-test`).

## Frontend

Next.js App Router agent panel:

- `/login` — httpOnly cookie session
- `/tickets` — queue with filters
- `/tickets/[id]` — thread + AI panel + overrides
- `/dashboard` — `GET /stats/overview`

## ML workspace

`backend/ml/`: generate → split → train → calibrate → serve. The ML workspace
sits inside the backend because everything AI/ML belongs to the backend side of
the system; it is a Python package, so Go tooling never sees it and it carries
its own test target (`make ml-test`).

The inference container is the `classifier` service in
`deployments/docker-compose.yml`, built from `../ml`. It is opt-in — `dev.sh
infra` starts the datastores only, and `dev.sh classifier` builds and starts the
model service.

## Local run

```bash
./start.sh                 # infra + migrations + backend + frontend
cd backend && make seed    # demo org (demo@ticketlens.dev / Demo1234!)
# optional model service
CLASSIFIER_URL=http://localhost:8091 ./start.sh backend
```
