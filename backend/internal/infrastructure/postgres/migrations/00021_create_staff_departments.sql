-- +goose Up
-- +goose StatementBegin
-- Which department a person works in.
--
-- Until now this relation did not exist anywhere. The admin panel answered
-- "which team is this agent on" by looking at the departments of the tickets
-- assigned to them, which is a derivation and not a record: somebody holding no
-- tickets had no team, and a newly hired agent was invisible to the roster until
-- their first assignment. Rosters, workload balancing and "who covers Payments
-- while Selin is away" all need the fact itself.
--
-- Its own table rather than a column, for two reasons.
--
-- organization_users would have been the obvious home and is the wrong one:
-- nothing writes it. Membership is read from user_roles instead (see
-- shared/middleware/org_scope.go and its test, which says so outright), so a
-- column there would never be populated.
--
-- users.department_id is wrong for a different reason: users are global to the
-- platform while departments belong to one organization, so the column would
-- claim that a person has one department everywhere. The organization has to be
-- part of the key.
CREATE TABLE IF NOT EXISTS staff_departments (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    -- CASCADE, not SET NULL: with (organization_id, user_id) as the key, the
    -- absence of a row already means "no department". Deleting a department
    -- therefore unassigns its people rather than silently moving them into
    -- General — a manager should place them deliberately, and they show up in
    -- the roster's Unassigned bucket until somebody does.
    department_id   UUID NOT NULL REFERENCES departments(id) ON DELETE CASCADE,
    assigned_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- One department per person per organization. A support agent splitting
    -- their week across two teams is a real thing, but it is a scheduling
    -- concept and modelling it now would make "the Payments roster" ambiguous
    -- before anything can consume the ambiguity.
    PRIMARY KEY (organization_id, user_id)
);

-- The roster query: everybody in one department.
CREATE INDEX IF NOT EXISTS idx_staff_departments_department
    ON staff_departments(organization_id, department_id);

COMMENT ON TABLE staff_departments IS
    'Support staff to department membership. No row means the person is on the roster but unassigned.';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_staff_departments_department;
DROP TABLE IF EXISTS staff_departments;
-- +goose StatementEnd
