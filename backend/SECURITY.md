# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.0.1   | :white_check_mark: |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Instead, please report them via one of the following methods:

- **Email**: security@masterfabric.co
- **GitHub Security Advisory**: Use the [Security tab](https://github.com/masterfabric-go/masterfabric-go/security/advisories/new) to create a private security advisory

### What to Include

When reporting a security vulnerability, please include:

- Description of the vulnerability
- Steps to reproduce
- Potential impact
- Suggested fix (if any)

### Response Timeline

- **Initial Response**: Within 48 hours
- **Status Update**: Within 7 days
- **Fix Timeline**: Depends on severity (typically 30-90 days)

## Trust Model

masterfabric-go is a multi-tenant API management platform. The server is trusted to enforce authentication, authorization, tenant isolation, and policy rules. Clients are untrusted. Administrative operators are trusted to configure secrets, CORS origins, and infrastructure bindings correctly.

## Security Controls Registry v0.1

Baseline security controls implemented on the `security/hardening` branch (July 2026). Each control maps to a confirmed audit finding or a portable hardening pattern from the shared platform layer.

| ID | Category | Control | Implementation | CWE | Status |
| -- | -------- | ------- | -------------- | --- | ------ |
| SC-01 | Toolchain | Pin Go stdlib to a patched release | `go 1.26.4` in `go.mod` | CWE-94 | ✅ Implemented |
| SC-02 | Dependencies | Refresh vulnerable direct modules | pgx v5.10.0, chi v5.3.0, validator v10.30.3, `golang.org/x/crypto` v0.53.0 | CWE-400 | ✅ Implemented |
| SC-03 | Container | Non-root runtime with current base images | `golang:1.26.4-alpine` builder, `alpine:3.24` runtime, `appuser` | CWE-250 | ✅ Implemented |
| SC-04 | Infrastructure | Bind dev service ports to loopback | `DB_HOST_BIND`, `REDIS_HOST_BIND`, `KAFKA_HOST_BIND` default to `127.0.0.1` in `deployments/docker-compose.yml` | CWE-1392 | ✅ Implemented |
| SC-05 | Error handling | Sanitize internal server error responses | `internal/shared/response/json.go` — generic 5xx message, detail via `slog` | CWE-209 | ✅ Implemented |
| SC-06 | Configuration | Escape database DSN credentials | `DatabaseConfig.DSN()` uses `net/url.UserPassword` | CWE-116 | ✅ Implemented |
| SC-07 | Input validation | Clamp pagination page overflow | `MaxPage = 1_000_000` in `internal/shared/pagination/pagination.go` | CWE-190 | ✅ Implemented |
| SC-08 | Configuration | Bounded int32 environment parsing | `envOrDefaultInt32` for `DB_MAX_CONNS` / `DB_MIN_CONNS` | G115 | ✅ Implemented |
| SC-09 | HTTP surface | Safe CORS allow-list | `CORS_ALLOWED_ORIGINS` env + `middleware.CORSOptions`; credentials off for `*` or empty | CWE-942 | ✅ Implemented |
| SC-10 | HTTP surface | Global request body size cap | `MAX_BODY_BYTES` (default 1 MiB) via `middleware.MaxBodyBytes` | CWE-400 | ✅ Implemented |
| SC-11 | Observability | Generic readiness probe responses | `internal/infrastructure/http/handler/health/handler.go` — no raw error strings | CWE-209 | ✅ Implemented |
| SC-12 | Egress | Harden outbound HTTP proxy client | No redirect following, 30s timeout, 1 MiB response cap in `internal/gateway/dynamic_handler.go` | CWE-522 | ✅ Implemented |
| SC-13 | Authentication | Refuse to start on default JWT signing secret | `Config.Validate()` aborts startup outside `APP_ENV=development` when `JWT_SECRET` is the shipped default, is shorter than 32 characters, when `DB_PASSWORD` is the compose default, or when `CORS_ALLOWED_ORIGINS` is empty | CWE-798 | ✅ Implemented |
| SC-14 | Authorization | Enforce RBAC on administrative routes | `RequirePermission` on all `/api/v1` admin routes in `router.go` | CWE-306 | ✅ Implemented |
| SC-15 | Authorization | Wildcard-aware permission matching | `matchesPermission` in `internal/infrastructure/auth/rbac_service.go` (`*`, `org:*`, `*:read`) | CWE-285 | ✅ Implemented |
| SC-16 | Input validation | Sanitize migration script names | `scripts/migrate.sh create` — `[a-zA-Z0-9_]` charset only | CWE-22 | ✅ Implemented |
| SC-17 | Gateway | Suppress internal DB errors in dynamic handler | Generic `"an internal error occurred"` to clients; detail in logs | CWE-209 | ✅ Implemented |
| SC-18 | Gateway | Document intentional proxy SSRF sink | `#nosec G704` on admin-configured outbound proxy; accepted risk entry below | CWE-918 | ✅ Documented |
| SC-19 | Verification | Automated vulnerability scanning gate | `govulncheck` clean; `gosec` clean with 2 audited suppressions | — | ✅ Verified |
| SC-20 | Tenant isolation | Bind a path-addressed organization to the token | `middleware.RequireOrgFromPath` on the `/organizations/{orgId}` subtree; a mismatch answers 404, not 403 | CWE-639 | ✅ Implemented |
| SC-21 | Tenant isolation | Bind a path-addressed app to the caller's organization | `middleware.RequireAppInOrg` guards the app subtree, including its API keys and endpoints, which resolve by app id alone | CWE-639 | ✅ Implemented |
| SC-22 | Tenant isolation | Bind a child resource to its parent in the path | `middleware.RequireChildOfPathResource` confirms `{keyId}` and `{endpointId}` belong to `{appId}`; previously a caller could pair an owned app with another tenant's key and revoke it | CWE-639 | ✅ Implemented |
| SC-23 | Tenant isolation | Make the token authoritative over the tenant header | `middleware.TenantResolverWithWorkspace` reads the JWT claim first and rejects a disagreeing `X-Organization-ID` with 403; the header can no longer establish a tenant on its own | CWE-290 | ✅ Implemented |
| SC-24 | Information disclosure | Scope user and organization listings to the caller | `ListUsers` uses `UserRepo.ListByOrganization`, `ListOrgs` returns only the caller's organization, and `middleware.RequireUserInOrg` guards user-addressed routes | CWE-200 | ✅ Implemented |

### Control summary

| Metric | Value |
| ------ | ----- |
| Registry version | **v0.2** |
| Total controls | **24** |
| Implemented | **23** |
| Documented accepted risk | **1** (SC-18) |
| Go toolchain | **1.26.4** |
| Verification | `go test ./...`, `govulncheck`, `gosec` |

### Environment variables

| Variable | Purpose | Production guidance |
| -------- | ------- | ------------------- |
| `APP_ENV` | Environment gate for insecure defaults | Set to `staging` or `production`; only `development` tolerates the shipped defaults |
| `JWT_SECRET` | HS256 signing key | Required, at least 32 characters; startup fails on the default outside development |
| `CORS_ALLOWED_ORIGINS` | Comma-separated browser origins | Set explicit origins; avoid `*` |
| `MAX_BODY_BYTES` | Request body cap | Keep at or below gateway policy limits |
| `DB_SSLMODE` | PostgreSQL TLS mode | Use `require` or stricter |
| `DB_HOST_BIND` | Compose host bind for Postgres | Keep `127.0.0.1` outside isolated dev machines |
| `CLASSIFIER_URL` | Python inference base URL | Empty = in-process stub; set in production when serving a trained model |
| `CLASSIFIER_TIMEOUT_MS` | Per-call HTTP timeout | Default `5000` |
| `CLASSIFIER_MAX_RETRIES` | HTTP retries before failure/fallback | Default `2` |
| `CLASSIFIER_FALLBACK_TO_STUB` | Use keyword stub after HTTP exhaustion | Default `true` for resilience; set `false` to fail closed |
| `CLASSIFIER_REVIEW_THRESHOLD` | Confidence floor for `needs_human_review` | Default `0.60`; recalibrate after training |

### Security Best Practices

When using TicketLens in production:

- Set `APP_ENV=production`, which makes the checks below fail-closed at startup instead of advisory
- Change default `JWT_SECRET` to a strong, random value of at least 32 characters
- Use SSL/TLS for database connections (`DB_SSLMODE=require`)
- Set `CORS_ALLOWED_ORIGINS` to explicit trusted origins
- Enable rate limiting for production workloads via endpoint policies
- Regularly update dependencies (`go get -u ./...`) and run `govulncheck`
- Review and rotate API keys regularly
- Monitor audit logs for suspicious activity
- Use environment variables for sensitive configuration
- Keep Docker images updated
- Restrict `/metrics` and health endpoints at the network edge

### Accepted Risks

| Risk | Rationale | Mitigation |
| ---- | --------- | ---------- |
| Unauthenticated `/metrics` and `/health/*` | Required for orchestrator probes and Prometheus scraping | Restrict by network policy or reverse-proxy auth |
| HS256 JWT | Simplicity for single-tenant deployments | Rotate secrets; prefer external identity for large fleets |
| Dynamic SQL gateway handler | Admin-defined table names via endpoint configuration | RBAC on endpoint creation; audit endpoint changes |
| Gateway HTTP proxy (gosec G704) | Managed endpoints may proxy to operator-configured backends | RBAC on endpoint creation; redirect refusal; response size cap |
| Development compose credentials | Convenience for local bootstrap | Loopback bind + documented dev-only posture |

### Verification Commands

```bash
go build ./... && go vet ./... && go test ./...
go run golang.org/x/vuln/cmd/govulncheck@latest ./...
go run github.com/securego/gosec/v2/cmd/gosec@latest -quiet ./...
```

### Security Updates

Security updates will be:

- Documented in CHANGELOG.md
- Tagged with security labels
- Released as patch versions

Thank you for helping keep masterfabric-go secure!
