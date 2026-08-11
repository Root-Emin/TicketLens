-- +goose Up
-- +goose StatementBegin
-- How a person becomes staff in an organization.
--
-- Until now there was no such path. Somebody joined by registering themselves on
-- the public POST /auth/register — which attaches them to no organization and
-- grants them nothing — after which an administrator had to discover their user
-- id by some means outside the product and call POST /roles/assign. That forced
-- the registration endpoint to stay open on the internet for staff onboarding to
-- work at all.
--
-- An invitation carries what the person will become, decided by the inviter
-- before the account exists: the role, and optionally the department. Acceptance
-- is then a single transaction with nothing left for an administrator to finish
-- by hand, and the new hire appears on the roster already on their team rather
-- than in the Unassigned bucket.
--
-- Note what this table is NOT: a membership record. Membership is read from
-- user_roles (see the note in 00021_create_staff_departments.sql on why
-- organization_users is not that record either). An invitation is the pending
-- intent; accepting it writes user_roles, and this row becomes history.
CREATE TABLE IF NOT EXISTS invitations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    -- Who is being invited. Lowercased by the application before it gets here so
    -- the uniqueness below cannot be sidestepped with a different capitalisation.
    email           VARCHAR(255) NOT NULL,

    -- What they become on acceptance. RESTRICT rather than CASCADE: deleting a
    -- role out from under a pending invitation would leave it accepting into
    -- nothing, and refusing the delete surfaces that instead of hiding it.
    role_id         UUID NOT NULL REFERENCES roles(id) ON DELETE RESTRICT,

    -- Optional team placement. SET NULL, not RESTRICT: a department is an
    -- ordinary organizational unit that may legitimately be dissolved while an
    -- invitation is outstanding, and the invitation is still good without it —
    -- the person simply lands unassigned, exactly as staff_departments models.
    department_id   UUID REFERENCES departments(id) ON DELETE SET NULL,

    -- SHA-256 of the token, never the token itself. Same handling as
    -- app_api_keys.key_hash: the raw value is shown once, at creation, and a
    -- database disclosure must not hand over usable invitations.
    token_hash      VARCHAR(255) NOT NULL UNIQUE,

    invited_by      UUID REFERENCES users(id) ON DELETE SET NULL,
    expires_at      TIMESTAMPTZ NOT NULL,

    -- Both NULL for a live invitation. Recorded rather than deleted so an
    -- administrator can see who was invited and what became of it.
    accepted_at     TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,

    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- One live invitation per email per organization. Partial, so the same address
-- can be re-invited after the first is accepted, revoked, or superseded — and so
-- a person may hold live invitations to two different organizations at once.
--
-- Expiry is deliberately not part of the condition: it is time-dependent and a
-- partial index cannot reference now(). An expired invitation therefore still
-- blocks a duplicate, which the application resolves by revoking it first.
CREATE UNIQUE INDEX IF NOT EXISTS idx_invitations_live_email
    ON invitations(organization_id, email)
    WHERE accepted_at IS NULL AND revoked_at IS NULL;

-- The acceptance lookup: by hash alone, since the token is all an unauthenticated
-- caller has. UNIQUE on the column already indexes this; named here only to make
-- the access path explicit for future readers.

-- The administrator's list: pending invitations for one organization, newest first.
CREATE INDEX IF NOT EXISTS idx_invitations_org_created
    ON invitations(organization_id, created_at DESC);

COMMENT ON TABLE invitations IS
    'Pending staff invitations. Accepting one writes user_roles (and staff_departments); this row is then history, not a membership record.';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_invitations_org_created;
DROP INDEX IF EXISTS idx_invitations_live_email;
DROP TABLE IF EXISTS invitations;
-- +goose StatementEnd
