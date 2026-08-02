import { NextRequest, NextResponse } from "next/server";
import { apiBase, TOKEN_COOKIE } from "@/lib/server/backend";

/*
  Catch-all reverse proxy to the Go backend. The browser calls /api/proxy/<path>
  and this handler attaches the JWT from the httpOnly cookie and forwards to
  ${API}/api/v1/<path>. Keeping every backend call server-side is what lets the
  token stay out of JavaScript and removes the need for any CORS grant.
*/

async function forward(req: NextRequest, path: string[]) {
  const token = req.cookies.get(TOKEN_COOKIE)?.value;
  if (!token) {
    return NextResponse.json({ error: "not authenticated" }, { status: 401 });
  }

  const search = req.nextUrl.search;
  const target = `${apiBase()}/${path.join("/")}${search}`;

  const headers: Record<string, string> = {
    Authorization: `Bearer ${token}`,
  };
  const contentType = req.headers.get("content-type");
  if (contentType) headers["Content-Type"] = contentType;

  const method = req.method.toUpperCase();
  const hasBody = method !== "GET" && method !== "HEAD" && method !== "DELETE";

  const res = await fetch(target, {
    method,
    headers,
    body: hasBody ? await req.text() : undefined,
    cache: "no-store",
  });

  const text = await res.text();
  return new NextResponse(text, {
    status: res.status,
    headers: {
      "Content-Type": res.headers.get("content-type") || "application/json",
    },
  });
}

type Ctx = { params: Promise<{ path: string[] }> };

export async function GET(req: NextRequest, ctx: Ctx) {
  return forward(req, (await ctx.params).path);
}
export async function POST(req: NextRequest, ctx: Ctx) {
  return forward(req, (await ctx.params).path);
}
export async function PATCH(req: NextRequest, ctx: Ctx) {
  return forward(req, (await ctx.params).path);
}
export async function PUT(req: NextRequest, ctx: Ctx) {
  return forward(req, (await ctx.params).path);
}
export async function DELETE(req: NextRequest, ctx: Ctx) {
  return forward(req, (await ctx.params).path);
}
