/*
  Which panel a session belongs to.

  Three panels, one per tier of account:

    owner     /dashboard, /tickets   the workspace — everything not claimed below
    staff     /staff/*               the support agent's queue
    customer  /portal/*              the portal, one account's own tickets

  The role is resolved from the token's `roles` claim at login and cached in a
  cookie, so the proxy can pick a panel without decoding a JWT on every
  navigation.

  This is routing, not authorization. Nothing here decides what an account may
  read — the Go handlers do, from the token itself. Forging the cookie gets you
  a panel whose every request still comes back 401/403.
*/

export type AppRole = "owner" | "staff" | "customer";

/** Cookie holding the resolved role. Set at login, cleared at logout. */
export const ROLE_COOKIE = "tl_role";

/*
  The support agent. `agent` is the only role the seed gives to somebody who
  works the queue without running the place (cmd/seed/main.go), and the staff
  panel is built for exactly that person.
*/
const STAFF_ROLE_NAMES = ["agent", "support", "support_agent"];

/*
  The portal account. Named separately from the staff list because a customer is
  not a colleague with fewer buttons — it is a different product surface.
*/
const CUSTOMER_ROLE_NAMES = ["customer"];

/** The claims the routing layer cares about. Everything else is ignored. */
export interface RoutingClaims {
  roles?: string[];
  permissions?: string[];
  /**
   * The signed-in account's address, for display only.
   *
   * Used by /invite/[token] to say who the current session belongs to when it
   * is not the person the invitation names. Advisory like everything else read
   * out of this payload — nothing is authorized on it.
   */
  email?: string;
}

/**
 * roleFromClaims maps a token to the panel it should open.
 *
 * Customer and agent are matched by name because they are the two roles with a
 * panel of their own. Everything else — admin, org_admin, app_admin, developer,
 * viewer, and any role added later — is an owner: the workspace is the default
 * home, and narrowing it is what the two named lists above are for.
 *
 * The fallback being the widest panel is deliberate and safe: this only decides
 * which screen opens. An account that lands in the workspace without the
 * permissions to use it gets a workspace full of 403s, which is a visible bug.
 * The reverse — silently demoting a real owner — would look like data loss.
 */
export function roleFromClaims(claims: RoutingClaims | null): AppRole {
  const roles = (claims?.roles ?? []).map((role) => role.toLowerCase());

  if (roles.some((role) => CUSTOMER_ROLE_NAMES.includes(role))) return "customer";
  if (roles.some((role) => STAFF_ROLE_NAMES.includes(role))) return "staff";

  return "owner";
}

/** Where each role lands when it has no destination of its own. */
const HOME: Record<AppRole, string> = {
  owner: "/tickets",
  staff: "/staff",
  customer: "/portal",
};

/** homeFor returns the default landing route for a role. */
export function homeFor(role: AppRole): string {
  return HOME[role];
}

/** isAppRole narrows an untrusted cookie value. */
export function isAppRole(value: string | undefined): value is AppRole {
  return value === "owner" || value === "staff" || value === "customer";
}

/** Routes that belong to the customer portal. */
export function isPortalPath(pathname: string): boolean {
  return pathname === "/portal" || pathname.startsWith("/portal/");
}

/** Routes that belong to the support agent's panel. */
export function isStaffPath(pathname: string): boolean {
  return pathname === "/staff" || pathname.startsWith("/staff/");
}

/*
  The management workspace's routes.

  Written as a list rather than as `!isStaffPath && !isPortalPath` so that a new
  top-level route is not silently absorbed into the owner's panel: an unlisted
  path belongs to nobody, and canAccess lets only the owner near it.

  /team and /departments are the workforce administration screens. They are
  named here, and not merely rendered, because that listing is what stops an
  agent reaching them: the panel is chosen from this list, and an `agent` token
  resolves to "staff", which canAccess confines to /staff/*. The backend agrees
  independently — the seeded agent role holds no user:read, so GET /staff
  answers a 403 even if somebody edits the cookie (cmd/seed/main.go).

  /team rather than /staff: /staff/* is already the agent's own panel, and one
  path cannot mean "the queue I work" to an agent and "the people I manage" to
  an administrator.
*/
const OWNER_ROUTES = [
  "/dashboard",
  "/tickets",
  "/team",
  "/departments",
  "/settings",
];

export function isOwnerPath(pathname: string): boolean {
  return OWNER_ROUTES.some(
    (route) => pathname === route || pathname.startsWith(`${route}/`),
  );
}

/**
 * canAccess reports whether a role may open a path.
 *
 * The owner holds `*` in the backend, so the two narrower panels are readable
 * to them as well — being able to open the agent's queue and the customer's
 * portal is how you check what your own people see. Staff and customer stay
 * inside their own surface.
 */
export function canAccess(role: AppRole, pathname: string): boolean {
  if (role === "owner") return true;
  if (role === "staff") return isStaffPath(pathname);
  return isPortalPath(pathname);
}

/**
 * safeRedirect keeps a `?from=` value honest: it has to be a relative path, and
 * it has to belong to the role that just signed in. A customer bounced off
 * /staff/tickets must not be sent back there by their own login.
 */
export function safeRedirect(from: string | null, role: AppRole): string {
  if (!from || !from.startsWith("/") || from.startsWith("//")) {
    return homeFor(role);
  }
  return canAccess(role, from) ? from : homeFor(role);
}
