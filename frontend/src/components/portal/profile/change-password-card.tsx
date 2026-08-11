"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { PasswordField } from "@/components/auth/fields";
import { ActionButton, FormError, useToast } from "@/components/portal/primitives";
import { Panel, PanelHeader, PanelSection } from "@/components/staff/primitives";
import { Spinner } from "@/components/ui/spinner";
import { ApiError } from "@/lib/api/client";
import { useChangePassword } from "@/lib/portal/hooks";

/*
  Changing a password.

  The backend exposes no password endpoint at all right now — /auth carries
  register and login and nothing else — so this posts to the assumed
  POST /auth/change-password and reports exactly what came back. That is the
  honest failure mode: a form that pretended to succeed would leave someone
  believing their password had changed when it had not.

  The 8-character minimum mirrors the `min=8` on the register DTO so both
  places agree on what a valid password is.
*/

const schema = z
  .object({
    current_password: z.string().min(1, "Enter your current password"),
    new_password: z.string().min(8, "Use at least 8 characters"),
    confirm: z.string().min(1, "Confirm your new password"),
  })
  .refine((values) => values.new_password === values.confirm, {
    path: ["confirm"],
    message: "Passwords do not match",
  })
  .refine((values) => values.new_password !== values.current_password, {
    path: ["new_password"],
    message: "Choose a password you haven't used here before",
  });

type FormValues = z.infer<typeof schema>;

export function ChangePasswordCard() {
  const toast = useToast();
  const changePassword = useChangePassword();
  const [serverError, setServerError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    reset,
    formState: { errors },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { current_password: "", new_password: "", confirm: "" },
  });

  async function onSubmit(values: FormValues) {
    setServerError(null);
    try {
      await changePassword.mutateAsync({
        current_password: values.current_password,
        new_password: values.new_password,
      });
      reset();
      toast.success("Password updated.");
    } catch (error) {
      setServerError(describe(error));
    }
  }

  const submitting = changePassword.isPending;

  return (
    <Panel>
      <PanelHeader title="Change password" />
      <PanelSection>
        <form
          onSubmit={handleSubmit(onSubmit)}
          noValidate
          className="max-w-md space-y-4"
        >
          <PasswordField
            label="Current password"
            autoComplete="current-password"
            error={errors.current_password?.message}
            {...register("current_password")}
          />
          <PasswordField
            label="New password"
            autoComplete="new-password"
            placeholder="At least 8 characters"
            error={errors.new_password?.message}
            {...register("new_password")}
          />
          <PasswordField
            label="Confirm new password"
            autoComplete="new-password"
            error={errors.confirm?.message}
            {...register("confirm")}
          />

          {serverError && <FormError>{serverError}</FormError>}

          <ActionButton type="submit" disabled={submitting} aria-busy={submitting}>
            {submitting && <Spinner className="size-4" />}
            {submitting ? "Updating…" : "Update password"}
          </ActionButton>
        </form>
      </PanelSection>
    </Panel>
  );
}

function describe(error: unknown): string {
  if (error instanceof ApiError) {
    if (error.status === 401) return "Your current password is not correct.";
    if (error.status === 404 || error.status === 405) {
      return "Password changes aren't available yet. Contact support and we'll reset it for you.";
    }
    if (error.status >= 500) {
      return "We couldn't update your password just now. Please try again shortly.";
    }
    return error.message;
  }
  return "We couldn't update your password. Please try again.";
}
