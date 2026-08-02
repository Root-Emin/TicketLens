import { NextResponse } from "next/server";
import { TOKEN_COOKIE } from "@/lib/server/backend";

export async function POST() {
  const response = NextResponse.json({ ok: true });
  // The attributes have to match the ones login set, otherwise the browser
  // treats this as a different cookie and the session survives the logout.
  response.cookies.set(TOKEN_COOKIE, "", {
    httpOnly: true,
    sameSite: "lax",
    secure: process.env.NODE_ENV === "production",
    path: "/",
    maxAge: 0,
  });
  return response;
}
