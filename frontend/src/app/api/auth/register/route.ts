import { NextRequest, NextResponse } from "next/server";
import { homeFor, ROLE_COOKIE, roleFromClaims } from "@/lib/auth/roles";
import { apiBase, TOKEN_COOKIE } from "@/lib/server/backend";
import { decodeClaims } from "@/lib/server/claims";

/*
  Sign-up, then sign in.

  POST /auth/register returns a user and no token, so a new account would land
  back on the login form to type the same password again. This handler does the
  second call itself and sets the session cookies exactly as the login route
  does — same attributes, or the browser treats them as different cookies.

  The password is used once here and never stored or logged.
*/

const MAX_AGE_SECONDS = 24 * 60 * 60;

interface RegisterBody {
  first_name?: unknown;
  last_name?: unknown;
  email?: unknown;
  password?: unknown;
}

export async function POST(req: NextRequest) {
  let body: RegisterBody;
  try {
    body = (await req.json()) as RegisterBody;
  } catch {
    return NextResponse.json({ error: "invalid request body" }, { status: 400 });
  }

  const payload = {
    first_name: asString(body.first_name),
    last_name: asString(body.last_name),
    email: asString(body.email),
    password: asString(body.password),
  };

  if (!payload.email || !payload.password || !payload.first_name || !payload.last_name) {
    return NextResponse.json({ error: "all fields are required" }, { status: 400 });
  }

  let created: Response;
  try {
    created = await fetch(`${apiBase()}/auth/register`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(payload),
      cache: "no-store",
    });
  } catch {
    return NextResponse.json(
      { error: "cannot reach the server, please try again" },
      { status: 502 },
    );
  }

  if (!created.ok) {
    const data = await created.json().catch(() => null);
    return NextResponse.json(
      { error: data?.message || data?.error || "registration failed" },
      { status: created.status },
    );
  }

  // Registration succeeded. Exchange the same credentials for a token so the
  // new account arrives signed in.
  const session = await fetch(`${apiBase()}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email: payload.email, password: payload.password }),
    cache: "no-store",
  }).catch(() => null);

  const data = session ? await session.json().catch(() => null) : null;
  if (!session?.ok || !data?.token) {
    // The account exists; only the automatic sign-in failed. Sending them to
    // the login form is honest and recoverable.
    return NextResponse.json({ user: null, redirect_to: "/login" });
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
