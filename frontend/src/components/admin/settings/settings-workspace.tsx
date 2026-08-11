"use client";

import { Building2, Inbox, ShieldCheck } from "lucide-react";

import { ErrorState, PageHeader } from "@/components/portal/primitives";
import { ChangePasswordCard } from "@/components/portal/profile/change-password-card";
import {
  Panel,
  PanelHeader,
  PanelSection,
} from "@/components/staff/primitives";
import { Skeleton } from "@/components/shadcn/skeleton";
import { AdminPage, Footnote, ForbiddenState } from "@/components/admin/primitives";
import { ApiError } from "@/lib/api/client";
import { useMe } from "@/lib/api/hooks";
import { useOrganization, useWorkforce } from "@/lib/admin/hooks";
import { PERMISSION, roleLabel } from "@/lib/auth/permissions";
import { longDate } from "@/lib/portal/format";

/*
  Organization settings.

  Two cards, because there are exactly two things here the backend supports:
  reading the organization (GET /organizations) and changing your own password
  (POST /auth/change-password). PATCH /organizations/{id} is not registered in
  router.go, so the identity card is a description list rather than a form —
  which is the same call the portal's AccountCard makes, for the same reason.

  The change-password card is the portal's, imported rather than copied. It
  posts to the same endpoint with the same validation, and a second copy would
  be two places to fix the day the minimum length changes.

  Deliberately absent: SLA policy, business hours, ticket templates, branding,
  the classifier's review threshold. All of them are real settings a support
  organization wants and none of them are writable — the threshold, for one,
  arrives as a read-only echo on ticket detail responses
  (CLASSIFIER_REVIEW_THRESHOLD, set as an environment variable on the server).
  Drawing them as disabled inputs would be a promise this screen cannot keep.
*/

export function SettingsWorkspace({ roles }: { roles: string[] }) {
  const { data: org, isLoading, isError, error, refetch, isRefetching } =
    useOrganization();
  const { data: me } = useMe();
  const { derived } = useWorkforce();

  const forbidden = error instanceof ApiError && error.status === 403;

  return (
    <AdminPage className="max-w-[900px]">
      <PageHeader
        title="Organization"
        description="Workspace details, and the account you're signed in with."
      />

      {forbidden ? (
        <Panel>
          <ForbiddenState
            title="You can't read this organization"
            description="Organization details need org:read. Your password can still be changed below — that endpoint takes the account from your token and needs no permission."
            permission={PERMISSION.readOrg}
          />
        </Panel>
      ) : isError ? (
        <Panel>
          <ErrorState
            title="Couldn't load the organization"
            onRetry={() => void refetch()}
            retrying={isRefetching}
          />
        </Panel>
      ) : (
        <Panel>
          <PanelHeader title="Workspace" />
          <PanelSection>
            {isLoading || !org ? (
              <div className="grid gap-4 sm:grid-cols-2">
                {Array.from({ length: 4 }).map((_, index) => (
                  <div key={index} className="space-y-2">
                    <Skeleton className="h-3 w-20" />
                    <Skeleton className="h-4 w-36" />
                  </div>
                ))}
              </div>
            ) : (
              <dl className="grid gap-4 sm:grid-cols-2">
                <Row label="Name" value={org.name} />
                <Row label="Slug" value={org.slug} mono />
                <Row label="Status" value={titleCase(org.status)} />
                <Row label="Created" value={longDate(org.created_at)} />
              </dl>
            )}

            <p className="mt-5 text-ui-xs text-tl-faint">
              Read-only. There is no route for updating an organization, so
              renaming one is a database change today.
            </p>
          </PanelSection>

          {derived && (
            <PanelSection>
              <div className="grid gap-4 sm:grid-cols-3">
                <Metric
                  icon={Building2}
                  label="Departments"
                  value={derived.summary.departments}
                />
                <Metric
                  icon={ShieldCheck}
                  label="Staff"
                  value={derived.summary.total}
                />
                <Metric
                  icon={Inbox}
                  label="Unassigned tickets"
                  value={derived.summary.unassignedTickets}
                />
              </div>
            </PanelSection>
          )}
        </Panel>
      )}

      <Panel>
        <PanelHeader title="Your account" />
        <PanelSection>
          {me ? (
            <dl className="grid gap-4 sm:grid-cols-2">
              <Row
                label="Signed in as"
                value={`${me.first_name} ${me.last_name}`.trim() || me.email}
              />
              <Row label="Email" value={me.email} />
              <Row label="Role" value={roleLabel(roles)} />
              <Row label="Member since" value={longDate(me.created_at)} />
            </dl>
          ) : (
            <div className="grid gap-4 sm:grid-cols-2">
              {Array.from({ length: 4 }).map((_, index) => (
                <div key={index} className="space-y-2">
                  <Skeleton className="h-3 w-20" />
                  <Skeleton className="h-4 w-36" />
                </div>
              ))}
            </div>
          )}

          <Footnote className="mt-5">
            Role is the token&apos;s own <code className="font-mono">roles</code>{" "}
            claim, issued at sign-in. It is not re-read while you are signed in,
            so a role granted to you just now appears after your next sign-in.
          </Footnote>
        </PanelSection>
      </Panel>

      <ChangePasswordCard />
    </AdminPage>
  );
}

function Row({
  label,
  value,
  mono,
}: {
  label: string;
  value: string;
  mono?: boolean;
}) {
  return (
    <div className="min-w-0">
      <dt className="text-ui-xs font-semibold uppercase tracking-[0.06em] text-tl-faint">
        {label}
      </dt>
      <dd
        className={
          mono
            ? "mt-1 truncate font-mono text-ui-md text-tl-ink"
            : "mt-1 truncate text-ui-md text-tl-ink"
        }
      >
        {value}
      </dd>
    </div>
  );
}

function Metric({
  icon: Icon,
  label,
  value,
}: {
  icon: typeof Building2;
  label: string;
  value: number;
}) {
  return (
    <div className="flex items-center gap-3">
      <span className="flex size-9 shrink-0 items-center justify-center rounded-[10px] bg-tl-line-soft text-tl-ink-soft">
        <Icon className="size-[18px]" strokeWidth={2} aria-hidden />
      </span>
      <div className="min-w-0">
        <div className="text-ui-xl font-bold leading-none tracking-[-0.02em] tabular-nums text-tl-ink">
          {value}
        </div>
        <div className="mt-1 truncate text-ui-sm text-tl-muted">{label}</div>
      </div>
    </div>
  );
}

function titleCase(value: string): string {
  if (!value) return "—";
  return value.charAt(0).toUpperCase() + value.slice(1);
}
