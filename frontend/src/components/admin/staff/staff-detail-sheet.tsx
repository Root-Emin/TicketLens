"use client";

import Link from "next/link";
import {
  Activity,
  Building2,
  Inbox,
  Lock,
  ShieldCheck,
  UserRound,
} from "lucide-react";

import {
  Sheet,
  SheetContent,
  SheetDescription,
  SheetHeader,
  SheetTitle,
} from "@/components/shadcn/sheet";
import { ActionButton, ActionLink } from "@/components/portal/primitives";
import { InitialsAvatar } from "@/components/staff/primitives";
import {
  LoadBar,
  StaffStatusBadge,
} from "@/components/admin/primitives";
import type { StaffMember } from "@/lib/admin/types";
import { loadBand } from "@/lib/admin/workforce";
import { longDate } from "@/lib/portal/format";
import { relativeTime } from "@/lib/utils";

/*
  One person, in four questions.

  A side sheet rather than a modal, because the answer to "who is this" is
  usually checked against the row it came from — a dialog that blacks out the
  table makes you close it to compare. The staff panel uses a Sheet for the same
  reason on its details rail, so the gesture is already in the product.

  The four sections are the four things an administrator manages about a person,
  and they are separated because they have different consequences: getting an
  identity wrong is embarrassing, getting an access level wrong is a security
  incident. A flat list of eleven fields hides that distinction.

  Three of the four sections are currently read-only, and each says why in one
  line at the point where the control would be. That is deliberate: a section
  header with nothing under it reads as a missing feature, and a disabled input
  reads as a bug.
*/

export function StaffDetailSheet({
  member,
  busiest,
  isCurrentUser,
  canAssign,
  onChangeDepartment,
  onOpenChange,
}: {
  member: StaffMember | null;
  busiest: number;
  isCurrentUser: boolean;
  canAssign: boolean;
  onChangeDepartment: () => void;
  onOpenChange: (open: boolean) => void;
}) {
  return (
    <Sheet open={Boolean(member)} onOpenChange={onOpenChange}>
      <SheetContent
        side="right"
        /*
          data-[side=right]:w-full rather than w-full: SheetContent's own
          `data-[side=right]:w-3/4` is an attribute selector and outranks a bare
          width class, so a plain w-full would silently lose and leave a 270px
          sheet on a phone. Matching the variant lets tailwind-merge resolve the
          two properly.
        */
        className="gap-0 overflow-y-auto p-0 font-ui data-[side=right]:w-full sm:max-w-[460px]"
      >
        {member && (
          <Body
            member={member}
            busiest={busiest}
            isCurrentUser={isCurrentUser}
            canAssign={canAssign}
            onChangeDepartment={onChangeDepartment}
          />
        )}
      </SheetContent>
    </Sheet>
  );
}

function Body({
  member,
  busiest,
  isCurrentUser,
  canAssign,
  onChangeDepartment,
}: {
  member: StaffMember;
  busiest: number;
  isCurrentUser: boolean;
  canAssign: boolean;
  onChangeDepartment: () => void;
}) {
  return (
    <>
      {/* pr-12 reserves the corner SheetContent's own close button sits in, so a
          long name does not run underneath it. */}
      <SheetHeader className="gap-0 border-b border-tl-line px-6 pb-5 pr-12 pt-6">
        <div className="flex items-center gap-3.5">
          <InitialsAvatar
            name={member.name}
            initials={member.initials}
            size={52}
          />
          <div className="min-w-0">
            <SheetTitle className="truncate text-ui-xl font-bold tracking-[-0.02em] text-tl-ink">
              {member.name}
              {isCurrentUser && (
                <span className="ml-2 align-middle text-ui-xs font-semibold text-tl-muted">
                  (you)
                </span>
              )}
            </SheetTitle>
            <SheetDescription className="truncate text-ui-md text-tl-muted">
              {member.email}
            </SheetDescription>
          </div>
        </div>

        <div className="mt-4 flex flex-wrap items-center gap-2">
          <StaffStatusBadge status={member.status} />
          <ActionLink
            href={`/tickets?assignee_id=${member.id}`}
            variant="secondary"
            size="sm"
            className="ml-auto"
          >
            <Inbox className="size-3.5" aria-hidden />
            View assigned tickets
          </ActionLink>
        </div>
      </SheetHeader>

      <div className="divide-y divide-tl-line-soft">
        <Section icon={UserRound} title="Identity" note="Read-only — there is no PATCH /users route.">
          <Row label="Full name" value={member.name} />
          <Row label="Email" value={member.email} />
          <Row
            label="Member since"
            value={longDate(member.joinedAt)}
            hint="From the account's created_at"
          />
          <Row
            label="Account status"
            value={<StaffStatusBadge status={member.status} />}
          />
        </Section>

        <Section
          icon={ShieldCheck}
          title="Access & role"
          note="The API returns no role for a user, and no endpoint lists the roles to choose from."
        >
          <Row
            label="Role"
            value={
              member.role ?? (
                <span className="text-tl-faint">Not exposed by the API</span>
              )
            }
          />
          <Row
            label="Panel"
            value="Determined at sign-in from the token's roles claim"
            hint="lib/auth/roles.ts maps the claim onto one of three panels"
          />
        </Section>

        <Section icon={Building2} title="Department assignment">
          {member.department ? (
            <>
              <Row
                label="Team"
                value={
                  <Link
                    href={`/departments/${member.department.id}`}
                    className="inline-flex rounded-md bg-tl-blue-soft px-2 py-[3px] text-ui-xs font-semibold text-tl-blue transition-colors duration-150 hover:bg-blue-100"
                  >
                    {member.department.name}
                  </Link>
                }
              />
              <Row
                label="Routed work"
                value="New tickets in this department can be assigned to them"
              />
            </>
          ) : (
            <p className="text-ui-md leading-relaxed text-tl-muted">
              {member.firstName || "This person"} is on the roster but not on a
              team. They can still be assigned individual tickets; they just are
              not part of any department&apos;s rota.
            </p>
          )}

          {canAssign && (
            <ActionButton
              variant="secondary"
              size="sm"
              onClick={onChangeDepartment}
              className="mt-1"
            >
              <Building2 className="size-3.5" aria-hidden />
              {member.department ? "Change department" : "Assign to a department"}
            </ActionButton>
          )}
        </Section>

        <Section
          icon={Activity}
          title="Availability & workload"
          note="Counted over the organization's open tickets — resolved and closed are excluded."
        >
          <Row
            label="Open tickets"
            value={
              <LoadBar
                open={member.openTickets}
                busiest={busiest}
                band={loadBand(member.openTickets, busiest)}
                className="max-w-[180px]"
              />
            }
          />
          <Row
            label="High or urgent"
            value={
              member.pressingTickets > 0 ? (
                <span className="font-semibold text-tl-orange-ink">
                  {member.pressingTickets}
                </span>
              ) : (
                "0"
              )
            }
          />
          <Row
            label="Last active"
            value={
              member.lastActiveAt ? (
                <time dateTime={member.lastActiveAt}>
                  {relativeTime(member.lastActiveAt)}
                </time>
              ) : (
                <span className="text-tl-faint">No ticket activity</span>
              )
            }
            hint="Newest update across their tickets; users carry no last_seen_at"
          />
        </Section>
      </div>

      <div className="flex items-start gap-2 border-t border-tl-line bg-tl-line-soft/50 px-6 py-4">
        <Lock className="mt-px size-4 shrink-0 text-tl-faint" aria-hidden />
        <p className="text-ui-xs leading-relaxed text-tl-muted">
          Department is the only part of a person this panel can change. Names,
          roles and account status need endpoints the backend does not register
          yet, so they are shown as the API reports them.
        </p>
      </div>
    </>
  );
}

function Section({
  icon: Icon,
  title,
  note,
  children,
}: {
  icon: typeof UserRound;
  title: string;
  note?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="px-6 py-5">
      <h3 className="flex items-center gap-2 text-ui-xs font-semibold uppercase tracking-[0.06em] text-tl-faint">
        <Icon className="size-3.5" strokeWidth={2.1} aria-hidden />
        {title}
      </h3>
      <div className="mt-3.5 space-y-3">{children}</div>
      {note && (
        <p className="mt-3.5 text-ui-xs leading-relaxed text-tl-faint">{note}</p>
      )}
    </section>
  );
}

/**
 * A label/value pair.
 *
 * Plain divs rather than dt/dd: two of the four sections render prose or a list
 * instead of pairs, so wrapping every section in a <dl> to satisfy this one
 * would put non-definition content inside a definition list. The label carries
 * its meaning through position and weight, which is what it does in the staff
 * panel's details rail too.
 */
function Row({
  label,
  value,
  hint,
}: {
  label: string;
  value: React.ReactNode;
  hint?: string;
}) {
  return (
    <div className="grid gap-1 sm:grid-cols-[132px_1fr] sm:items-baseline sm:gap-3">
      <span className="text-ui-sm font-medium text-tl-muted">{label}</span>
      <div className="min-w-0">
        <div className="text-ui-md text-tl-ink">{value}</div>
        {hint && <p className="mt-0.5 text-ui-xs text-tl-faint">{hint}</p>}
      </div>
    </div>
  );
}
