"use client";

import {
  AccountCard,
  AccountCardSkeleton,
} from "@/components/portal/profile/account-card";
import { ChangePasswordCard } from "@/components/portal/profile/change-password-card";
import { NotificationPreferences } from "@/components/portal/profile/notification-preferences";
import { ErrorState, PageHeader } from "@/components/portal/primitives";
import { usePortalMe } from "@/lib/portal/hooks";

/*
  Profile.

  Three cards, in descending order of how real they are: the account is read
  from /me, the password form posts to an endpoint that does not exist yet, and
  the preferences are stored on the device. Each says so where it matters rather
  than in a footnote at the bottom of the page.
*/

export default function PortalProfilePage() {
  const { data: me, isPending, isError, isFetching, refetch } = usePortalMe();

  return (
    <>
      <PageHeader
        title="Profile"
        description="Your account details and how we reach you."
      />

      {isError ? (
        <ErrorState
          title="We couldn't load your profile"
          onRetry={() => refetch()}
          retrying={isFetching}
          className="rounded-card border border-tl-line bg-tl-card shadow-panel"
        />
      ) : isPending ? (
        <AccountCardSkeleton />
      ) : (
        <AccountCard user={me} />
      )}

      <ChangePasswordCard />
      <NotificationPreferences />
    </>
  );
}
