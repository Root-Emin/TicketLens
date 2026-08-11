import { NextResponse } from "next/server";
import { ROLE_COOKIE } from "@/lib/auth/roles";
import { TOKEN_COOKIE } from "@/lib/server/backend";

export async function POST() {
  const response = NextResponse.json({ ok: true });
  // The attributes have to match the ones login set, otherwise the browser
  // treats this as a different cookie and the session survives the logout.
  const expired = {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 0,
  };
  response.cookies.set(TOKEN_COOKIE, "", expired);
  response.cookies.set(ROLE_COOKIE, "", expired);
  return response;
}
