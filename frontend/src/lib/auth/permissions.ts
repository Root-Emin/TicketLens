/*
  What this session is likely to be allowed to do.

  Read that sentence carefully: *likely*. This module decides which controls are
  worth drawing, and nothing else. Authorization happens in the Go handlers,
  from the token itself — every route in router.go sits behind
  RequirePermission, and the seeded `agent` role holds no user:read no matter
  what this file believes (cmd/seed/main.go).

  Why it has to guess at all: the JWT carries `roles` but deliberately not
  `permissions` (application/iam/usecase/login.go documents the reason — a token
  minted yesterday would otherwise still be authorizing with yesterday's grants).
  There is no GET /permissions either. So the only thing the client can do is map
  the role names it was given through the same template the seed writes, and
  treat the answer as a hint.

  Two rules keep that hint from becoming a lie:

    1. An unrecognised role is treated as permitted. Widening the UI for a role
       this file has never heard of shows an administrator a screen full of
       honest 403 states, which is a visible bug. Narrowing it would silently
       hide working features from somebody who has them — the failure nobody
       reports because it looks like the feature was never built.

    2. Every screen gated here still renders a real forbidden state when the API
       says no. The gate saves a wasted click; the state is what actually
       handles the case.
*/

/**
 * The seeded template roles, mirroring templateRoleDefs in
 * backend/cmd/seed/main.go. Kept in the same order and wording so a diff
 * between the two is obvious.
 *
 * Capability intent:
 * - admin      full ( * )
 * - org_admin  owner workspace: departments + staff + tickets + stats
 * - agent      queue only (ticket:read lists departments for routing; no manage)
 * - viewer     read everything including roster; no writes
 * - customer   portal own-tickets only
 */
const ROLE_PERMISSIONS: Record<string, string[]> = {
  admin: ["*"],
  org_admin: [
    "org:*",
    "app:*",
    "user:*",
    "ticket:create",
    "ticket:read",
    "ticket:update",
    "ticket:delete",
    "ticket:assign",
    "message:create",
    "department:manage",
    "customer:manage",
    "analysis:read",
    "stats:read",
  ],
  app_admin: ["app:*", "endpoint:*"],
  developer: ["endpoint:read", "endpoint:write"],
  viewer: ["*:read"],
  agent: [
    "ticket:create",
    "ticket:read",
    "ticket:update",
    "ticket:assign",
    "message:create",
    "customer:manage",
    "analysis:read",
  ],
  customer: [
    "ticket:create",
    "ticket:read_own",
    "ticket:reopen_own",
    "message:create",
  ],
};

/**
 * matchesPermission is a transliteration of matchesPermission in
 * backend/internal/infrastructure/auth/rbac_service.go. It must keep matching
 * it: a wildcard that means one thing here and another there is worse than no
 * client-side check at all.
 */
export function matchesPermission(granted: string, required: string): boolean {
  if (granted === required || granted === "*") return true;

  if (granted.endsWith(":*")) {
    const prefix = granted.slice(0, -2);
    if (required === prefix || required.startsWith(`${prefix}:`)) return true;
  }

  if (granted === "*:read" && required.endsWith(":read")) return true;

  return false;
}

/** True when none of the role names are ones the seed defines. */
function allUnknown(roles: string[]): boolean {
  return roles.every((role) => !(role in ROLE_PERMISSIONS));
}

/**
 * can reports whether these roles probably carry a permission.
 *
 * An empty role list, or one made entirely of roles this file does not know,
 * returns true — see rule 1 above.
 */
export function can(roles: string[] | undefined, permission: string): boolean {
  const names = (roles ?? []).map((role) => role.toLowerCase());
  if (names.length === 0 || allUnknown(names)) return true;

  return names.some((role) =>
    (ROLE_PERMISSIONS[role] ?? []).some((granted) =>
      matchesPermission(granted, permission),
    ),
  );
}

/** canAny mirrors RequireAnyPermission: one match anywhere is enough. */
export function canAny(
  roles: string[] | undefined,
  permissions: string[],
): boolean {
  return permissions.some((permission) => can(roles, permission));
}

/**
 * The permissions the management screens ask for, named once.
 *
 * Each is the exact string the matching route is registered with in
 * internal/infrastructure/http/router/router.go, so the two can be compared by
 * grepping for the literal.
 */
export const PERMISSION = {
  /** GET /staff — the staff roster. */
  readUsers: "user:read",
  /** PUT /staff/{id}/department — place people on teams. */
  writeUsers: "user:write",
  /** POST/PATCH/DELETE /departments. */
  manageDepartments: "department:manage",
  /** GET /tickets. */
  readTickets: "ticket:read",
  /** GET /organizations. */
  readOrg: "org:read",
  /** GET /stats/overview — the dashboard. */
  readStats: "stats:read",
} as const;

/**
 * A role name as it should appear in the account menu.
 *
 * The token's own claim, title-cased — not a label invented for the UI. An
 * account holding several roles shows the first, which is the one
 * CreateOrgUseCase grants the organization's creator.
 */
export function roleLabel(roles: string[] | undefined): string {
  const first = (roles ?? [])[0];
  if (!first) return "Signed in";
  return first
    .split(/[_\s]+/)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(" ");
}
