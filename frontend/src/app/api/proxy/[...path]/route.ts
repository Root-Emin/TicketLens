import { NextRequest, NextResponse } from "next/server";
import { apiBase, TOKEN_COOKIE } from "@/lib/server/backend";
import { devPortalToken } from "@/lib/server/dev-session";

/*
  Catch-all reverse proxy to the Go backend. The browser calls /api/proxy/<path>
  and this handler attaches the JWT from the httpOnly cookie and forwards to
  ${API}/api/v1/<path>. Keeping every backend call server-side is what lets the
  token stay out of JavaScript and removes the need for any CORS grant.

  In development a request with no cookie falls back to a seeded customer
  session (see lib/server/dev-session.ts) so the open portal can load real data.
  In a production build there is no fallback and a missing cookie is a 401,
  exactly as before.
*/

async function forward(req: NextRequest, path: string[]) {
  const token =
    req.cookies.get(TOKEN_COOKIE)?.value ?? (await devPortalToken());
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

  /*
    204, 205 and 304 are null-body statuses: the Response constructor throws if
    one is given a body, and an empty string still counts as a body. Forwarding
    them the same way as everything else turned the backend's 204 into a 500
    from this route — which is what DELETE /departments returns, so the first
    caller to try deleting anything hit it.

    Content-Type is dropped along with the body, since there is no longer
    anything for it to describe.
  */
  if (NULL_BODY_STATUS.has(res.status)) {
    return new NextResponse(null, { status: res.status });
  }

  const text = await res.text();
  return new NextResponse(text, {
    status: res.status,
    headers: {
      "Content-Type": res.headers.get("content-type") || "application/json",
    },
  });
}

const NULL_BODY_STATUS = new Set([204, 205, 304]);

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
