import { NextRequest, NextResponse } from "next/server";
import { apiBase, TOKEN_COOKIE } from "@/lib/server/backend";

// Token lifetime mirrors the backend default (JWT_EXPIRATION_HOURS=24). There is
// no refresh endpoint, so the cookie simply expires with the token and the user
// logs in again.
const MAX_AGE_SECONDS = 24 * 60 * 60;

export async function POST(req: NextRequest) {
  let body: unknown;
  try {
    body = await req.json();
  } catch {
    return NextResponse.json({ error: "invalid request body" }, { status: 400 });
  }

  const res = await fetch(`${apiBase()}/auth/login`, {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
    cache: "no-store",
  });

  const data = await res.json().catch(() => null);
  if (!res.ok || !data?.token) {
    return NextResponse.json(
      { error: data?.message || data?.error || "login failed" },
      { status: res.status || 401 },
    );
  }

  // Return the user, but never the token: the client stores nothing.
  const response = NextResponse.json({ user: data.user });
  response.cookies.set(TOKEN_COOKIE, data.token, {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: MAX_AGE_SECONDS,
  });
  return response;
}
