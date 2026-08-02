import { NextRequest, NextResponse } from "next/server";

const TOKEN_COOKIE = "tl_token";

/*
  Gate the app routes on the presence of the session cookie. This is a fast
  redirect for UX, not the security boundary — /api/proxy/* still rejects any
  call without a valid token, and the backend still verifies the JWT. An expired
  but still-present cookie is caught there and surfaces as a 401 the client turns
  into a login redirect.

  Named `proxy` (in src/proxy.ts) because Next 16 renamed the middleware file
  convention. Not to be confused with the /api/proxy route handler, which is the
  thing that actually forwards requests to the Go backend; this file only
  redirects.
*/
export function proxy(req: NextRequest) {
  const hasToken = Boolean(req.cookies.get(TOKEN_COOKIE)?.value);
  const { pathname } = req.nextUrl;

  const isLogin = pathname === "/login";

  if (!hasToken && !isLogin) {
    const url = req.nextUrl.clone();
    url.pathname = "/login";
    url.searchParams.set("from", pathname);
    return NextResponse.redirect(url);
  }

  if (hasToken && isLogin) {
    const url = req.nextUrl.clone();
    url.pathname = "/tickets";
    url.search = "";
    return NextResponse.redirect(url);
  }

  return NextResponse.next();
}

// Protect everything except Next internals, the auth/proxy API routes (which do
// their own token handling), and static assets.
export const config = {
  matcher: ["/((?!api|_next/static|_next/image|favicon.ico).*)"],
};
