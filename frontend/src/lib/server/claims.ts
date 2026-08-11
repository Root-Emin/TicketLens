import "server-only";

import type { RoutingClaims } from "@/lib/auth/roles";

/*
  Reads the routing claims out of a freshly issued token.

  The signature is deliberately *not* verified here: the backend signed this
  token seconds ago and verifies it again on every proxied request. All this
  read decides is which panel to open, so a tampered payload buys nothing —
  the API still answers according to the real claims.
*/

/** decodeClaims returns the JWT payload's roles/permissions, or null. */
export function decodeClaims(token: string): RoutingClaims | null {
  const payload = token.split(".")[1];
  if (!payload) return null;

  try {
    // JWT uses base64url; Buffer's base64 decoder accepts it once - and _ are
    // mapped back and the padding is restored.
    const normalized = payload.replace(/-/g, "+").replace(/_/g, "/");
    const padded = normalized.padEnd(
      normalized.length + ((4 - (normalized.length % 4)) % 4),
      "=",
    );
    const json: unknown = JSON.parse(
      Buffer.from(padded, "base64").toString("utf8"),
    );

    if (typeof json !== "object" || json === null) return null;
    const record = json as Record<string, unknown>;

    return {
      roles: stringArray(record.roles),
      permissions: stringArray(record.permissions),
    };
  } catch {
    return null;
  }
}

function stringArray(value: unknown): string[] {
  if (!Array.isArray(value)) return [];
  return value.filter((item): item is string => typeof item === "string");
}
