import type { Metadata } from "next";
import Link from "next/link";
import { cookies } from "next/headers";

import { AcceptInvitationForm } from "@/components/auth/accept-invitation-form";
import { AuthShell } from "@/components/auth/auth-shell";
import { ActionLink } from "@/components/portal/primitives";
import { TOKEN_COOKIE } from "@/lib/server/backend";
import { decodeClaims } from "@/lib/server/claims";
import { fetchInvitationPreview } from "@/lib/server/invitations";

export const metadata: Metadata = {
  title: "Accept your invitation — TicketLens",
  description: "Join your team on TicketLens.",
};

// The token is a single-use credential and this page is per-token by
// definition; caching it anywhere would be caching the credential.
export const dynamic = "force-dynamic";

/**
 * Accepting a staff invitation.
 *
 * Public: whoever opens this has no account yet, which is the whole point. The
 * token in the path is the credential and the backend validates it — this
 * screen only renders what it is told.
 *
 * The preview is read server-side rather than from the browser. /api/proxy
 * attaches the session cookie and answers 401 without one, so it cannot serve a
 * page whose entire audience is signed out.
 */
export default async function InvitePage({
  params,
}: {
  params: Promise<{ token: string }>;
}) {
  const { token } = await params;
  const lookup = await fetchInvitationPreview(token);

  if (!lookup.ok) {
    return lookup.reason === "unreachable" ? <Unreachable /> : <NotValid />;
  }

  const { preview } = lookup;

  // Who is signed in on this browser, if anybody. Display only — nothing here
  // is authorized on it, and the session plays no part in accepting.
  const jar = await cookies();
  const sessionEmail =
    decodeClaims(jar.get(TOKEN_COOKIE)?.value ?? "")?.email ?? null;

  return (
    <AuthShell
      title={`Join ${preview.organization_name}`}
      subtitle={
        preview.has_account
          ? `You have been invited as ${preview.role_name}. Sign in to accept.`
          : `You have been invited as ${preview.role_name}. Choose a password to activate your account.`
      }
      footer={
        <p className="text-ui-sm text-tl-ink-soft">
          Not expecting this?{" "}
          <Link href="/" className="font-medium text-tl-blue hover:underline">
            Go to TicketLens
          </Link>
        </p>
      }
    >
      <AcceptInvitationForm
        token={token}
        email={preview.email}
        organization={preview.organization_name}
        role={preview.role_name}
        hasAccount={preview.has_account}
        sessionEmail={sessionEmail}
      />
    </AuthShell>
  );
}

/**
 * The one failure screen.
 *
 * Unknown, expired, revoked and already-used tokens all land here, because the
 * backend answers all four identically on purpose — distinguishing them tells
 * somebody probing which tokens were once real. Four tailored messages would
 * undo that from the client side, so there is deliberately only this one.
 */
function NotValid() {
  return (
    <AuthShell
      title="This invitation is no longer valid"
      subtitle="It may have been used already, withdrawn, or it may have expired."
      footer={
        <p className="text-ui-sm text-tl-ink-soft">
          Already have an account?{" "}
          <Link href="/login" className="font-medium text-tl-blue hover:underline">
            Sign in
          </Link>
        </p>
      }
    >
      <p className="text-ui-sm leading-relaxed text-tl-ink-soft">
        Please contact the person who invited you and ask them to send a new
        invitation.
      </p>
    </AuthShell>
  );
}

/** Our fault, not the link's — so it must not be reported as a dead invitation. */
function Unreachable() {
  return (
    <AuthShell
      title="Something went wrong"
      subtitle="We could not check your invitation just now."
      footer={
        <p className="text-ui-sm text-tl-ink-soft">
          If this keeps happening, contact the person who invited you.
        </p>
      }
    >
      <div className="space-y-4">
        <p className="text-ui-sm leading-relaxed text-tl-ink-soft">
          Your invitation is probably fine — we just could not reach the server.
          Try again in a moment.
        </p>
        <ActionLink href="/login">Go to sign in</ActionLink>
      </div>
    </AuthShell>
  );
}
