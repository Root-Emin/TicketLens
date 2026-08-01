-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS departments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name            VARCHAR(255) NOT NULL,
    description     TEXT NOT NULL DEFAULT '',
    is_default      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, name)
);

CREATE INDEX IF NOT EXISTS idx_departments_org_id ON departments(organization_id);
CREATE INDEX IF NOT EXISTS idx_departments_is_default ON departments(is_default);

-- Exactly one default department ("General") per organization.
CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_one_default_per_org
    ON departments(organization_id) WHERE is_default;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_departments_one_default_per_org;
DROP INDEX IF EXISTS idx_departments_is_default;
DROP INDEX IF EXISTS idx_departments_org_id;
DROP TABLE IF EXISTS departments;
-- +goose StatementEnd
