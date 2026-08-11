"use client";

import { Panel, PanelHeader, PanelSection, InitialsAvatar } from "@/components/staff/primitives";
import { Skeleton } from "@/components/shadcn/skeleton";
import { initialsOf, longDate } from "@/lib/portal/format";
import type { UserInfo } from "@/lib/api/types";

/*
  Who is signed in.

  Read-only, because it is: there is no PATCH /me on the backend, so an editable
  name field would be a form with nowhere to submit. The fields are shown as a
  description list with a line saying where a change has to go instead — which
  is more useful than an input that silently does nothing.
*/

export function AccountCard({ user }: { user: UserInfo }) {
  const name = `${user.first_name} ${user.last_name}`.trim() || user.email;

  return (
    <Panel>
      <PanelHeader title="Account" />
      <PanelSection>
        <div className="flex items-center gap-4">
          <InitialsAvatar name={name} initials={initialsOf(name)} size={56} />
          <div className="min-w-0">
            <p className="truncate text-ui-lg font-semibold tracking-[-0.01em] text-tl-ink">
              {name}
            </p>
            <p className="truncate text-ui-md text-tl-muted">{user.email}</p>
          </div>
        </div>

        <dl className="mt-5 grid gap-4 sm:grid-cols-2">
          <Row label="Full name" value={name} />
          <Row label="Email" value={user.email} />
          <Row label="Member since" value={longDate(user.created_at)} />
          <Row label="Account status" value={capitalize(user.status)} />
        </dl>

        <p className="mt-5 text-ui-xs text-tl-faint">
          Need your name or email changed? Open a ticket and support will update
          it for you.
        </p>
      </PanelSection>
    </Panel>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="min-w-0">
      <dt className="text-ui-xs font-semibold uppercase tracking-[0.06em] text-tl-faint">
        {label}
      </dt>
      <dd className="mt-1 truncate text-ui-md text-tl-ink">{value}</dd>
    </div>
  );
}

function capitalize(value: string): string {
  if (!value) return "—";
  return value.charAt(0).toUpperCase() + value.slice(1);
}

export function AccountCardSkeleton() {
  return (
    <Panel aria-hidden>
      <div className="px-5 py-4">
        <Skeleton className="h-5 w-24" />
      </div>
      <PanelSection>
        <div className="flex items-center gap-4">
          <Skeleton className="size-14 rounded-full" />
          <div className="space-y-2">
            <Skeleton className="h-4 w-40" />
            <Skeleton className="h-3.5 w-52" />
          </div>
        </div>
        <div className="mt-5 grid gap-4 sm:grid-cols-2">
          {Array.from({ length: 4 }).map((_, index) => (
            <div key={index} className="space-y-2">
              <Skeleton className="h-3 w-20" />
              <Skeleton className="h-4 w-32" />
            </div>
          ))}
        </div>
      </PanelSection>
    </Panel>
  );
}
