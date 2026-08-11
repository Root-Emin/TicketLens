import { NextRequest, NextResponse } from "next/server";

import {
  canAccess,
  homeFor,
  isAppRole,
  isPortalPath,
  isStaffPath,
  ROLE_COOKIE,
  type AppRole,
} from "@/lib/auth/roles";

const TOKEN_COOKIE = "tl_token";

/*
  Gate the app routes on the presence of the session cookie. This is a fast
  redirect for UX, not the security boundary — /api/proxy/* still rejects any
  call without a valid token, and the backend still verifies the JWT. An expired
  but still-present cookie is caught there and surfaces as a 401 the client turns
  into a login redirect.

  The role cookie decides which of the three panels a session lands in — owner,
  staff or customer. It is written at login from the token's own claims (see
  lib/auth/roles.ts) and is likewise advisory: a customer who edits it into
  "owner" reaches a workspace whose every request the Go handlers answer with
  403.

  Named `proxy` (in src/proxy.ts) because Next 16 renamed the middleware file
  convention. Not to be confused with the /api/proxy route handler, which is the
  thing that actually forwards requests to the Go backend; this file only
  redirects.
*/
export function proxy(req: NextRequest) {
  const hasToken = Boolean(req.cookies.get(TOKEN_COOKIE)?.value);
  const { pathname } = req.nextUrl;

  const rawRole = req.cookies.get(ROLE_COOKIE)?.value;
  const role: AppRole = isAppRole(rawRole) ? rawRole : "owner";

  // Signing in and signing up. Kept apart from isPublic below because a
  // signed-in customer must still be bounced off "/" to their own panel, and
  // folding the landing page in with the auth screens would exempt it from
  // that too.
  const isAuthScreen = pathname === "/login" || pathname === "/register";

  /*
    Accepting a staff invitation.

    Public because the whole point is that the recipient has no account yet —
    the token in the path is the only credential, and the backend validates it.

    Grouped with the auth screens rather than merely added to isPublic: a
    signed-in visitor must reach it too. The invitation may be for a colleague
    borrowing this browser, or for a second organization the current account is
    joining, and bouncing them to their own workspace would make the link look
    broken to the one person holding it.
  */
  const isInviteScreen = pathname.startsWith("/invite/");

  // What a stranger may open: the marketing page, the auth screens and an
  // invitation link.
  const isPublic = pathname === "/" || isAuthScreen || isInviteScreen;

  /*
    Panels that open without signing in, while they are being built.

    /staff runs entirely on fixtures and never calls the backend. /portal does
    read live data, so /api/proxy signs a cookieless dev request in as a seeded
    customer (lib/server/dev-session.ts) — a customer, not the owner, so every
    list stays scoped to one account and the screens show what a real portal
    user would see.

    Development only: a production build gates both exactly like the workspace.
    The owner's own routes are deliberately absent — /dashboard and /tickets
    always ask for a session, which is what keeps the three panels
    distinguishable in a browser. Delete this once each panel issues its own
    session.
  */
  const isOpenPanel =
    process.env.NODE_ENV !== "production" &&
    (isStaffPath(pathname) || isPortalPath(pathname));

  if (!hasToken && !isPublic && !isOpenPanel) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.searchParams.set("from", pathname);
    return NextResponse.redirect(url);
  }

  /*
    Keep a session inside its own panel.

    The owner passes untouched — they may open the agent's queue and the portal
    as well, which is how you check what your own people see. Staff and customer
    are returned home, including from "/": a signed-in account lands in its
    workspace rather than on the marketing page.

    This also catches a path that belongs to no panel at all. Only the owner
    gets through, so a new top-level route is not exposed to everybody by the
    mere fact that nobody has claimed it yet.
  */
  if (hasToken && !isAuthScreen && !isInviteScreen && !canAccess(role, pathname)) {
    const url = req.nextUrl.clone();
    url.pathname = homeFor(role);
    url.search = "";
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
}

// Protect everything except Next internals, the auth/proxy API routes (which do
// their own token handling), and static assets.
//
// `assets` has to be listed: files under public/ are served at their bare path,
// so without it every /assets/* request matched this proxy and a signed-out
// visitor got a 307 to /login instead of the file. That broke the logo on the
// sign-in and sign-up screens as well as the landing page.
export const config = {
  matcher: ["/((?!api|assets|_next/static|_next/image|favicon.ico).*)"],
};
