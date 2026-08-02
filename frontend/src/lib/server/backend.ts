import "server-only";

/*
  Server-only helpers shared by the auth and proxy route handlers. The JWT lives
  in an httpOnly cookie the browser JS never reads; every call to the Go backend
  is made from the Next server with the token attached, so CORS never enters the
  picture and a stolen token cannot be exfiltrated from the client.

  The `server-only` import above turns "please don't import this from a client
  component" into a build error. Next resolves it internally, so the npm package
  is only needed to keep lint rules about extraneous dependencies quiet.
*/

export const TOKEN_COOKIE = "tl_token";

/**
 * backendURL is the Go API base, resolved server-side only.
 *
 * Deliberately not NEXT_PUBLIC_: the browser never calls the backend directly,
 * it calls /api/proxy/*. A NEXT_PUBLIC_ name would inline the internal address
 * into the client bundle and buy nothing.
 */
export function backendURL(): string {
  return process.env.API_URL || "http://localhost:8080";
}

/** apiBase is the versioned prefix every backend route is mounted under. */
export function apiBase(): string {
  return `${backendURL()}/api/v1`;
}
