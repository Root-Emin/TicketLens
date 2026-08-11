import "server-only";

import { apiBase } from "./backend";

/*
  A standing customer session for the portal, in development only.

  The portal is still being built, so /portal is open without signing in (see
  src/proxy.ts). But the screens behind it read live, per-account data, and
  /api/proxy refuses any call without a token — so an open route on its own
  would render the shell over a wall of 401s. This module fills that gap: when a
  dev request arrives with no session cookie, the proxy borrows a token minted
  here and the portal comes up with real data.

  It signs in as a *customer*, never the admin. The whole point of looking at
  the portal is to see what one account can see, and an admin token would quietly
  widen every list to the entire organization — which is exactly the bug the
  backend's scope resolution exists to prevent. Signing in as a customer means
  the screens stay honest: whatever shows up is what that customer is entitled
  to.

  Seeded by cmd/seed/main.go, which gives the first few demo customers a login
  under the `customer` role. Override with DEV_PORTAL_EMAIL / DEV_PORTAL_PASSWORD
  to look at a different account.

  Never active in a production build: every entry point returns null there, so
  the fallback cannot survive a deploy even if the proxy forgot to check.
*/

const DEV_EMAIL = process.env.DEV_PORTAL_EMAIL || "alice.morgan@modaboutique.com";
const DEV_PASSWORD = process.env.DEV_PORTAL_PASSWORD || "Demo1234!";

/** Re-login this many seconds before the token's own expiry. */
const RENEW_MARGIN_SECONDS = 60;

interface CachedSession {
  token: string;
  /** Unix seconds, taken from the token's `exp`. */
  expiresAt: number;
}

/*
  Module state, so the demo account is not re-authenticated on every request.
  A dev server is a single process; if it reloads, the cache goes with it and
  the next call simply signs in again.
*/
let cached: CachedSession | null = null;
let inFlight: Promise<string | null> | null = null;

/** devPortalToken returns a customer JWT for dev, or null when unavailable. */
export async function devPortalToken(): Promise<string | null> {
  if (process.env.NODE_ENV === "production") return null;

  const now = Math.floor(Date.now() / 1000);
  if (cached && cached.expiresAt - RENEW_MARGIN_SECONDS > now) {
    return cached.token;
  }

  // Collapse concurrent misses into one login. The portal fires several
  // queries as it mounts, and without this every one of them would open its
  // own session on the first paint.
  inFlight ??= login().finally(() => {
    inFlight = null;
  });
  return inFlight;
}

/** Drop the cached session so the next call re-authenticates. */
export function clearDevPortalToken(): void {
  cached = null;
}

async function login(): Promise<string | null> {
  let res: Response;
  try {
    res = await fetch(`${apiBase()}/auth/login`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ email: DEV_EMAIL, password: DEV_PASSWORD }),
      cache: "no-store",
    });
  } catch {
    // Backend not up yet. Returning null lets the caller answer 401 as usual.
    return null;
  }

  if (!res.ok) return null;

  const data: unknown = await res.json().catch(() => null);
  const token =
    typeof data === "object" && data !== null
      ? (data as { token?: unknown }).token
      : undefined;
  if (typeof token !== "string" || !token) return null;

  cached = { token, expiresAt: expiryOf(token) };
  return token;
}

/*
  Reads `exp` straight off the payload. The signature is irrelevant here: this
  only decides when to ask for a new token, and the backend verifies the real
  thing on every request. A token we cannot read is treated as short-lived
  rather than rejected.
*/
function expiryOf(token: string): number {
  const fallback = Math.floor(Date.now() / 1000) + 300;
  const payload = token.split(".")[1];
  if (!payload) return fallback;

  try {
    const normalized = payload.replace(/-/g, "+").replace(/_/g, "/");
    const padded = normalized.padEnd(
      normalized.length + ((4 - (normalized.length % 4)) % 4),
      "=",
    );
    const json: unknown = JSON.parse(
      Buffer.from(padded, "base64").toString("utf8"),
    );
    if (typeof json !== "object" || json === null) return fallback;

    const exp = (json as { exp?: unknown }).exp;
    return typeof exp === "number" ? exp : fallback;
  } catch {
    return fallback;
  }
}

/** The account the dev fallback signs in as, for logging. */
export const devPortalEmail = DEV_EMAIL;
