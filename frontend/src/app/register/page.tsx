"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { AuthShell } from "@/components/auth/auth-shell";
import { PasswordField, TextField } from "@/components/auth/fields";
import { ActionButton, FormError } from "@/components/portal/primitives";
import { Spinner } from "@/components/ui/spinner";

/*
  Sign-up.

  Not in the original page list, but /login links here and a link to nothing is
  a broken screen. It posts to POST /auth/register, which already exists and
  takes exactly these four fields (internal/application/iam/dto/user_dto.go);
  the 8-character minimum below mirrors that DTO's `min=8` so the form rejects
  a short password before the round trip rather than after it.

  Registration returns a user, not a token, so this signs in with the same
  credentials immediately afterwards and lets the login route set the cookies.
*/

const schema = z
  .object({
    first_name: z.string().trim().min(1, "First name is required"),
    last_name: z.string().trim().min(1, "Last name is required"),
    email: z.string().min(1, "Email is required").email("Enter a valid email"),
    password: z.string().min(8, "Use at least 8 characters"),
    confirm: z.string().min(1, "Confirm your password"),
  })
  .refine((values) => values.password === values.confirm, {
    path: ["confirm"],
    message: "Passwords do not match",
  });

type FormValues = z.infer<typeof schema>;

export default function RegisterPage() {
  const [serverError, setServerError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: {
      first_name: "",
      last_name: "",
      email: "",
      password: "",
      confirm: "",
    },
  });

  async function onSubmit(values: FormValues) {
    setServerError(null);

    try {
      const res = await fetch("/api/auth/register", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          first_name: values.first_name.trim(),
          last_name: values.last_name.trim(),
          email: values.email.trim(),
          password: values.password,
        }),
      });

      const data: { error?: string; redirect_to?: string } = await res
        .json()
        .catch(() => ({}));

      if (!res.ok) {
        setServerError(
          res.status === 409
            ? "An account with this email already exists."
            : data.error || "We could not create your account. Please try again.",
        );
        return;
      }

      // Full navigation so src/proxy.ts re-reads the cookies just set.
      window.location.assign(data.redirect_to ?? "/portal");
    } catch {
      setServerError("No connection. Check your network and try again.");
    }
  }

  return (
    <AuthShell
      title="Create your account"
      subtitle="Open a request and follow it from one place."
      footer={
        <>
          Already have an account?{" "}
          <Link
            href="/login"
            className="font-semibold text-tl-blue hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
          >
            Sign in
          </Link>
        </>
      }
    >
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-5">
        <div className="grid gap-5 sm:grid-cols-2">
          <TextField
            label="First name"
            autoComplete="given-name"
            autoFocus
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

        <TextField
          label="Email"
          type="email"
          autoComplete="email"
          placeholder="you@company.com"
          error={errors.email?.message}
          {...register("email")}
        />

        <PasswordField
          label="Password"
          autoComplete="new-password"
          placeholder="At least 8 characters"
          error={errors.password?.message}
          {...register("password")}
        />

        <PasswordField
          label="Confirm password"
          autoComplete="new-password"
          error={errors.confirm?.message}
          {...register("confirm")}
        />

        {serverError && <FormError>{serverError}</FormError>}

        <ActionButton
          type="submit"
          disabled={isSubmitting}
          className="w-full"
          aria-busy={isSubmitting}
        >
          {isSubmitting && <Spinner className="size-4" />}
          {isSubmitting ? "Creating account…" : "Create account"}
        </ActionButton>
      </form>
    </AuthShell>
  );
}
