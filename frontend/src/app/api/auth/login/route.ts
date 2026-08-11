import { NextRequest, NextResponse } from "next/server";
import { homeFor, ROLE_COOKIE, roleFromClaims } from "@/lib/auth/roles";
import { apiBase, TOKEN_COOKIE } from "@/lib/server/backend";
import { decodeClaims } from "@/lib/server/claims";

// Token lifetime mirrors the backend default (JWT_EXPIRATION_HOURS=24). There is
// no refresh endpoint, so the cookie simply expires with the token and the user
// logs in again.
const MAX_AGE_SECONDS = 24 * 60 * 60;

/**
 * Body accepted from the login form.
 *
 * `remember` never reaches the backend: it only decides whether the cookie
 * survives the browser closing. Unchecked means a session cookie, which is the
 * safer default on a shared machine.
 */
interface LoginBody {
  email?: unknown;
  password?: unknown;
  remember?: unknown;
}

export async function POST(req: NextRequest) {
  let body: LoginBody;
  try {
    body = (await req.json()) as LoginBody;
  } catch {
    return NextResponse.json({ error: "invalid request body" }, { status: 400 });
  }

  const email = typeof body.email === "string" ? body.email : "";
  const password = typeof body.password === "string" ? body.password : "";
  const remember = body.remember === true;

  if (!email || !password) {
    return NextResponse.json(
      { error: "email and password are required" },
      { status: 400 },
    );
  }

  let res: Response;
  try {
    res = await fetch(`${apiBase()}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email, password }),
      cache: "no-store",
    });
  } catch {
    // The Go service is unreachable. 502 lets the form tell the difference
    // between "wrong password" and "nothing is listening".
    return NextResponse.json(
      { error: "cannot reach the server, please try again" },
      { status: 502 },
    );
  }

  const data = await res.json().catch(() => null);
  if (!res.ok || !data?.token) {
    return NextResponse.json(
      { error: data?.message || data?.error || "login failed" },
      { status: res.status || 401 },
    );
  }

  const role = roleFromClaims(decodeClaims(data.token));

  // Return the user, but never the token: the client stores nothing.
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
    ...(remember ? { maxAge: MAX_AGE_SECONDS } : {}),
  };

  response.cookies.set(TOKEN_COOKIE, data.token, cookie);
  // Read by src/proxy.ts to pick a panel. Advisory only — see lib/auth/roles.ts.
  response.cookies.set(ROLE_COOKIE, role, cookie);
  return response;
}
