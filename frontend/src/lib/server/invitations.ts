import "server-only";

import { apiBase } from "./backend";

/*
  The public half of the invitation flow, read server-side.

  Not through /api/proxy: that route attaches the session cookie and answers 401
  without one, which is exactly the case here — the recipient has no account yet.
  The token in the URL is the credential, and the backend validates it. So this
  goes straight to the API the same way the auth route handlers do.

  The token is never logged, stored, or sent anywhere but this request. It is a
  single-use credential for creating an account: anything that persists it
  persists a way in.
*/

export interface InvitationPreview {
  email: string;
  organization_name: string;
  role_name: string;
  expires_at: string;
  /**
   * The address already has a TicketLens login, so accepting joins that account
   * rather than creating one — and the screen must not ask for a password. The
   * backend would discard it, leaving the person unable to sign in with what
   * they typed.
   */
  has_account: boolean;
}

/**
 * Why a preview could not be shown.
 *
 * `invalid` covers unknown, expired, revoked and already-accepted tokens
 * together, because the backend answers all four identically and on purpose:
 * telling them apart confirms which tokens were once real. The screen has one
 * generic message for this, not four.
 *
 * `unreachable` is kept separate — that one is our fault, not the link's, and
 * telling somebody their invitation is dead when the API was merely down would
 * send them to bother the person who invited them for nothing.
 */
export type InvitationLookup =
  | { ok: true; preview: InvitationPreview }
  | { ok: false; reason: "invalid" | "unreachable" };

export async function fetchInvitationPreview(
  token: string,
): Promise<InvitationLookup> {
  let res: Response;
  try {
    res = await fetch(
      `${apiBase()}/auth/invitations/${encodeURIComponent(token)}`,
      { cache: "no-store" },
    );
  } catch {
    return { ok: false, reason: "unreachable" };
  }

  if (res.status >= 500) return { ok: false, reason: "unreachable" };
  if (!res.ok) return { ok: false, reason: "invalid" };

  const preview = (await res.json().catch(() => null)) as InvitationPreview | null;
  if (!preview?.email) return { ok: false, reason: "invalid" };

  return { ok: true, preview };
}
