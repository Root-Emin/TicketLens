"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import { Suspense, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { Key, Check } from "lucide-react";

import { AuthShell } from "@/components/auth/auth-shell";
import {
  CheckboxField,
  PasswordField,
  TextField,
} from "@/components/auth/fields";
import { ActionButton, FormError } from "@/components/portal/primitives";
import { Skeleton } from "@/components/shadcn/skeleton";
import { Spinner } from "@/components/ui/spinner";
import { isAppRole, safeRedirect } from "@/lib/auth/roles";

const schema = z.object({
  email: z.string().min(1, "Email is required").email("Enter a valid email"),
  password: z.string().min(1, "Password is required"),
  remember: z.boolean(),
});

type FormValues = z.infer<typeof schema>;

/** What /api/auth/login answers with on success. */
interface LoginResult {
  role?: string;
  redirect_to?: string;
}

/*
  The demo accounts, grouped by the role their token actually carries.

  `role` is the name the seed writes into the database and the backend puts in
  the `roles` claim — not a label invented for this screen. Signing in with each
  one is the only way to see what that role can really do, which is the point of
  listing them separately: the owner passes every permission check, so a bug
  that locks an agent out is invisible until you sign in as the agent.

  Kept in step with cmd/seed/main.go. An account listed here that the seed does
  not create simply fails to sign in.
*/
interface DemoAccount {
  /** The person, as the seed names them. */
  name: string;
  email: string;
}

interface DemoRoleGroup {
  /** How this tier is spoken about in the project. */
  title: string;
  /** The role name in the database and in the JWT. */
  role: string;
  /** Where signing in lands. */
  panel: string;
  /** What the role is allowed to do, in one line. */
  hint: string;
  badgeBg: string;
  accounts: DemoAccount[];
}

const DEMO_ROLES: DemoRoleGroup[] = [
  {
    title: "Owner",
    role: "admin",
    panel: "Staff Panel",
    hint: "Tüm yetkiler (*) — kısıtsız erişim",
    badgeBg: "bg-amber-100 text-amber-800 dark:bg-amber-900/40 dark:text-amber-300",
    accounts: [{ name: "Demo Yönetici", email: "demo@ticketlens.dev" }],
  },
  {
    title: "Staff",
    role: "agent",
    panel: "Staff Panel",
    hint: "Tüm ticket'ları okur ve günceller, yönetim yetkisi yok",
    badgeBg: "bg-blue-100 text-blue-800 dark:bg-blue-900/40 dark:text-blue-300",
    accounts: [{ name: "Selin Aydın", email: "agent@ticketlens.dev" }],
  },
  {
    title: "Customer",
    role: "customer",
    panel: "Customer Portal",
    hint: "Yalnızca kendi ticket'ları",
    badgeBg: "bg-emerald-100 text-emerald-800 dark:bg-emerald-900/40 dark:text-emerald-300",
    accounts: [
      { name: "Alice Morgan", email: "alice.morgan@modaboutique.com" },
      { name: "Michael Reed", email: "michael.reed@acmetrade.com" },
    ],
  },
];

function LoginForm() {
  const params = useSearchParams();
  const from = params.get("from");
  const [serverError, setServerError] = useState<string | null>(null);

  const {
    register,
    handleSubmit,
    setValue,
    watch,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    defaultValues: { email: "", password: "", remember: true },
  });

  const currentEmail = watch("email");

  const fillDemoAccount = (email: string) => {
    setValue("email", email, { shouldValidate: true });
    setValue("password", "Demo1234!", { shouldValidate: true });
    setServerError(null);
  };

  async function onSubmit(values: FormValues) {
    setServerError(null);

    let res: Response;
    try {
      res = await fetch("/api/auth/login", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(values),
      });
    } catch {
      // fetch only rejects when the request never completed: offline, DNS,
      // a dropped connection. Worth its own message — retrying helps here and
      // does not help a wrong password.
      setServerError("No connection. Check your network and try again.");
      return;
    }

    const data: LoginResult & { error?: string } = await res
      .json()
      .catch(() => ({}));

    if (!res.ok) {
      setServerError(messageFor(res.status, data.error));
      return;
    }

    // safeRedirect falls back to the role's home, so `from` can only ever
    // narrow the destination — never send a customer into the staff panel.
    const role = isAppRole(data.role) ? data.role : "owner";
    // A full navigation rather than router.push: src/proxy.ts has to re-read
    // the cookie that was just set, and a client transition would not send it.
    window.location.assign(safeRedirect(from, role));
  }

  return (
    <div className="space-y-6">
      <form onSubmit={handleSubmit(onSubmit)} noValidate className="space-y-5">
        <TextField
          label="Email"
          type="email"
          autoComplete="email"
          autoFocus
          placeholder="you@company.com"
          error={errors.email?.message}
          {...register("email")}
        />

        <div>
          <PasswordField
            label="Password"
            autoComplete="current-password"
            placeholder="••••••••"
            error={errors.password?.message}
            {...register("password")}
          />

          <div className="mt-3 flex items-center justify-between gap-3">
            <CheckboxField label="Remember me" {...register("remember")} />
            {/* No password reset endpoint exists yet, so this is disabled rather
                than a link to a page that cannot do anything. */}
            <button
              type="button"
              disabled
              title="Password reset is not available yet"
              className="text-ui-sm font-medium text-tl-faint disabled:cursor-not-allowed"
            >
              Forgot password?
            </button>
          </div>
        </div>

        {serverError && <FormError>{serverError}</FormError>}

        <ActionButton
          type="submit"
          disabled={isSubmitting}
          className="w-full"
          aria-busy={isSubmitting}
        >
          {isSubmitting && <Spinner className="size-4" />}
          {isSubmitting ? "Signing in…" : "Sign in"}
        </ActionButton>
      </form>

      {/* Demo Test Accounts Section */}
      <div className="rounded-xl border border-tl-border/60 bg-tl-surface-subtle p-4 shadow-sm">
        <div className="flex items-center justify-between mb-2">
          <span className="text-ui-sm font-semibold text-tl-ink flex items-center gap-1.5">
            <Key className="size-4 text-tl-blue" />
            Hazır Test Hesapları (Demo)
          </span>
          <span className="text-[11px] font-medium text-tl-muted bg-tl-border/40 px-2 py-0.5 rounded">
            Şifre: Demo1234!
          </span>
        </div>
        <p className="text-[12px] text-tl-muted mb-3">
          Tıklayarak giriş bilgilerini otomatik doldurabilirsiniz:
        </p>
        <div className="space-y-3.5">
          {DEMO_ROLES.map((group) => (
            <div key={group.role}>
              <div className="flex items-baseline gap-2 mb-1.5">
                <span className="text-ui-sm font-semibold text-tl-ink">
                  {group.title}
                </span>
                <span
                  className={`text-[10px] font-semibold px-1.5 py-0.5 rounded-full shrink-0 ${group.badgeBg}`}
                >
                  {group.role}
                </span>
                <span className="text-[11px] text-tl-muted truncate">
                  → {group.panel}
                </span>
              </div>
              <p className="text-[11px] text-tl-muted mb-1.5">{group.hint}</p>

              <div className="space-y-1.5">
                {group.accounts.map((acc) => {
                  const isSelected = currentEmail === acc.email;
                  return (
                    <button
                      key={acc.email}
                      type="button"
                      onClick={() => fillDemoAccount(acc.email)}
                      className={`w-full text-left transition-all p-2.5 rounded-lg border flex items-center justify-between text-ui-sm ${
                        isSelected
                          ? "border-tl-blue bg-tl-blue/5 ring-1 ring-tl-blue/30"
                          : "border-tl-border/80 bg-tl-surface hover:border-tl-blue/50 hover:bg-tl-surface-subtle"
                      }`}
                    >
                      <div className="min-w-0 flex-1 pr-2">
                        <div className="font-medium text-tl-ink truncate">
                          {acc.name}
                        </div>
                        <div className="text-[12px] text-tl-muted truncate mt-0.5">
                          {acc.email}
                        </div>
                      </div>
                      {isSelected && (
                        <span className="size-5 rounded-full bg-tl-blue text-white flex items-center justify-center shrink-0">
                          <Check className="size-3.5" strokeWidth={2.5} />
                        </span>
                      )}
                    </button>
                  );
                })}
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}

/** Turns a status code into something a person can act on. */
function messageFor(status: number, error?: string): string {
  if (status === 401) return "Email or password is incorrect.";
  if (status === 403) return "This account is not active. Contact support.";
  if (status === 429) return "Too many attempts. Please wait a moment.";
  if (status >= 500) return "The server is unavailable right now. Try again shortly.";
  return error || "Sign in failed. Please try again.";
}

function FormSkeleton() {
  return (
    <div className="space-y-5" aria-hidden>
      <Skeleton className="h-[70px] w-full" />
      <Skeleton className="h-[70px] w-full" />
      <Skeleton className="h-11 w-full" />
    </div>
  );
}

export default function LoginPage() {
  return (
    <AuthShell
      title="Welcome back"
      subtitle="Sign in to track your support requests."
      footer={
        <p>
          Don&apos;t have an account?{" "}
          <Link
            href="/register"
            className="font-semibold text-tl-blue hover:underline focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-tl-blue/30"
          >
            Create one
          </Link>
        </p>
      }
    >
      {/* useSearchParams opts the subtree into client rendering, so the form
          sits behind its own boundary and the page shell still streams. */}
      <Suspense fallback={<FormSkeleton />}>
        <LoginForm />
      </Suspense>
    </AuthShell>
  );
}

