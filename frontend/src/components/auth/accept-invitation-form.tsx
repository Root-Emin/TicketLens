"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { PasswordField, TextField } from "@/components/auth/fields";
import { ActionButton, FormError } from "@/components/portal/primitives";
import { Spinner } from "@/components/ui/spinner";

/*
  The acceptance form.

  Two shapes, decided by the backend rather than here. When the invited address
  already has a TicketLens account, accepting joins it and leaves its password
  alone — so asking for one would collect a value the API discards, and the
  person would then fail to sign in with what they just typed. That case gets a
  confirm button and no fields.

  The 8-character minimum mirrors AcceptInvitationRequest's `min=8`
  (internal/application/iam/dto/invitation_dto.go), so a short password is
  rejected before the round trip rather than after it. It is the only rule the
  backend enforces; adding others here would reject passwords the API accepts.
*/

const schema = z
  .object({
    first_name: z.string().trim().min(1, "First name is required"),
    last_name: z.string().trim().min(1, "Last name is required"),
    password: z.string().min(8, "Use at least 8 characters"),
    confirm: z.string().min(1, "Confirm your password"),
  })
  .refine((values) => values.password === values.confirm, {
    path: ["confirm"],
    message: "Passwords do not match",
  });

type FormValues = z.infer<typeof schema>;

/** The one message for every unusable token. See the page's NotValid screen. */
const INVALID =
  "This invitation is no longer valid. Please contact the person who invited you.";

export function AcceptInvitationForm({
  token,
  email,
  organization,
  role,
  hasAccount,
  sessionEmail,
}: {
  token: string;
  email: string;
  organization: string;
  role: string;
  hasAccount: boolean;
  sessionEmail: string | null;
}) {
  const router = useRouter();
  const [serverError, setServerError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { first_name: "", last_name: "", password: "", confirm: "" },
  });

  async function accept(values?: FormValues) {
    setServerError(null);
    setSubmitting(true);

    try {
      const res = await fetch("/api/auth/accept-invite", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          token,
          // Omitted entirely for an existing account: the API ignores them, and
          // sending a password it will discard invites the reader of this code
          // to believe it was set.
          ...(values
            ? {
                first_name: values.first_name.trim(),
                last_name: values.last_name.trim(),
                password: values.password,
              }
            : {}),
        }),
      });

      const data: { error?: string; redirect_to?: string } = await res
        .json()
        .catch(() => ({}));

      if (!res.ok) {
        setServerError(
          data.error === "invitation_invalid" || res.status === 404
            ? INVALID
            : data.error || "Could not accept the invitation. Please try again.",
        );
        setSubmitting(false);
        return;
      }

      // The session cookie is set by the route handler, so a full navigation is
      // what makes the app see it. router.refresh() first drops the cached
      // signed-out render of the shell.
      router.refresh();
      router.replace(data.redirect_to ?? "/login");
    } catch {
      setServerError("Cannot reach the server. Please try again.");
      setSubmitting(false);
    }
  }

  const busy = submitting || isSubmitting;

  return (
    <div className="space-y-5">
      <InvitationSummary
        email={email}
        organization={organization}
        role={role}
        sessionEmail={sessionEmail}
      />

      {serverError ? <FormError>{serverError}</FormError> : null}

      {hasAccount ? (
        <div className="space-y-4">
          <p className="text-ui-sm leading-relaxed text-tl-ink-soft">
            This address already has a TicketLens account. Accepting adds{" "}
            {organization} to it — your existing password does not change.
          </p>
          <ActionButton
            type="button"
            onClick={() => void accept()}
            disabled={busy}
            className="w-full"
          >
            {busy ? <Spinner className="size-4" /> : null}
            {busy ? "Accepting…" : "Accept and sign in"}
          </ActionButton>
        </div>
      ) : (
        <form onSubmit={handleSubmit((values) => accept(values))} className="space-y-4" noValidate>
          <div className="grid gap-4 sm:grid-cols-2">
            <TextField
              label="First name"
              autoComplete="given-name"
              error={errors.first_name?.message}
              {...register("first_name")}
            />
            <TextField
              label="Last name"
              autoComplete="family-name"
              error={errors.last_name?.message}
              {...register("last_name")}
            />
          </div>

          <PasswordField
            label="Password"
            autoComplete="new-password"
            error={errors.password?.message}
            {...register("password")}
          />
          <PasswordField
            label="Confirm password"
            autoComplete="new-password"
            error={errors.confirm?.message}
            {...register("confirm")}
          />

          <ActionButton type="submit" disabled={busy} className="w-full">
            {busy ? <Spinner className="size-4" /> : null}
            {busy ? "Creating your account…" : "Create account"}
          </ActionButton>
        </form>
      )}
    </div>
  );
}

/**
 * What is being accepted, and by whom.
 *
 * The address is shown and not editable: it is fixed by the invitation, and the
 * API takes no email here — accepting under an address of your own choosing
 * would be accepting as somebody else.
 *
 * The session notice only appears when a different account is signed in on this
 * browser. That is a real situation — a colleague borrowing a machine, or an
 * administrator opening a link to check it — and silently switching them to the
 * new account afterwards would be the surprising part, not the notice.
 */
function InvitationSummary({
  email,
  organization,
  role,
  sessionEmail,
}: {
  email: string;
  organization: string;
  role: string;
  sessionEmail: string | null;
}) {
  const otherSession =
    sessionEmail && sessionEmail.toLowerCase() !== email.toLowerCase();

  return (
    <div className="space-y-3">
      <dl className="divide-y divide-tl-line rounded-btn border border-tl-line bg-tl-surface-soft px-3.5 text-ui-sm">
        <Row label="Organization" value={organization} />
        <Row label="Role" value={role} />
        <Row label="Email" value={email} />
      </dl>

      {otherSession ? (
        <p className="text-ui-sm leading-relaxed text-tl-ink-soft">
          You are currently signed in as{" "}
          <span className="font-medium text-tl-ink">{sessionEmail}</span>.
          Continuing will replace that session with this one.
        </p>
      ) : null}
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-baseline justify-between gap-4 py-2.5">
      <dt className="text-tl-ink-soft">{label}</dt>
      <dd className="truncate font-medium text-tl-ink">{value}</dd>
    </div>
  );
}
