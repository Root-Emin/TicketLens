import { NextRequest, NextResponse } from "next/server";
import { homeFor, ROLE_COOKIE, roleFromClaims } from "@/lib/auth/roles";
import { apiBase, TOKEN_COOKIE } from "@/lib/server/backend";
import { decodeClaims } from "@/lib/server/claims";

/*
  Accept a staff invitation, then sign in.

  The same shape as the registration handler next door, for the same reason:
  POST /auth/invitations/{token}/accept returns a user and no token, so the new
  account would otherwise land on the login form and retype the password it just
  chose. This does the second call itself and sets the session cookies exactly as
  the login route does — same attributes, or the browser treats them as different
  cookies.

  The password is used twice here, for the two calls, and is never stored or
  logged. Neither is the token: it arrives in the body, goes into the URL of the
  upstream call, and nothing else.

  One case cannot be signed in automatically. When the invited address already
  has an account, acceptance joins it and leaves its credentials alone — this
  handler has no password for it and must not pretend to. Those callers are sent
  to /login, which is honest and one step.
*/

const MAX_AGE_SECONDS = 24 * 60 * 60;

interface AcceptBody {
  token?: unknown;
  first_name?: unknown;
  last_name?: unknown;
  password?: unknown;
}

export async function POST(req: NextRequest) {
  let body: AcceptBody;
  try {
    body = (await req.json()) as AcceptBody;
  } catch {
    return NextResponse.json({ error: "invalid request body" }, { status: 400 });
  }

  const token = asString(body.token);
  if (!token) {
    return NextResponse.json({ error: "missing invitation" }, { status: 400 });
  }

  const password = typeof body.password === "string" ? body.password : "";
  const payload = {
    first_name: asString(body.first_name),
    last_name: asString(body.last_name),
    password,
  };

  let accepted: Response;
  try {
    accepted = await fetch(
      `${apiBase()}/auth/invitations/${encodeURIComponent(token)}/accept`,
      {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(payload),
        cache: "no-store",
      },
    );
  } catch {
    return NextResponse.json(
      { error: "cannot reach the server, please try again" },
      { status: 502 },
    );
  }

  if (!accepted.ok) {
    /*
      404 means the invitation is not usable — unknown, expired, revoked or
      already accepted, which the backend answers identically on purpose. The
      client turns it into the one generic message; passing the backend's own
      wording through would be the same message anyway, but this keeps the
      screen's copy in one place.
    */
    if (accepted.status === 404) {
      return NextResponse.json({ error: "invitation_invalid" }, { status: 404 });
    }
    const data = await accepted.json().catch(() => null);
    return NextResponse.json(
      { error: data?.message || data?.error || "could not accept the invitation" },
      { status: accepted.status },
    );
  }

  const user = await accepted.json().catch(() => null);

  // Nothing to sign in with: the account existed before this invitation and
  // keeps its own password.
  if (!password) {
    return NextResponse.json({ user, redirect_to: "/login" });
  }

  const session = await fetch(`${apiBase()}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: user?.email, password }),
    cache: "no-store",
  }).catch(() => null);

  const data = session ? await session.json().catch(() => null) : null;
  if (!session?.ok || !data?.token) {
    // The account exists and the invitation is spent; only the automatic
    // sign-in failed. Sending them to the login form is honest and recoverable
    // — retrying the invitation link would now fail, since it is single-use.
    return NextResponse.json({ user, redirect_to: "/login" });
  }

  const role = roleFromClaims(decodeClaims(data.token));
  const response = NextResponse.json({
    user: data.user,
    role,
    redirect_to: homeFor(role),
  });

  const cookie = {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: MAX_AGE_SECONDS,
  };
  response.cookies.set(TOKEN_COOKIE, data.token, cookie);
  response.cookies.set(ROLE_COOKIE, role, cookie);
  return response;
}

function asString(value: unknown): string {
  return typeof value === "string" ? value.trim() : "";
}
