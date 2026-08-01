-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS customers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email           VARCHAR(255) NOT NULL,
    full_name       VARCHAR(255) NOT NULL,
    company         VARCHAR(255) NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(organization_id, email)
);

CREATE INDEX IF NOT EXISTS idx_customers_org_id ON customers(organization_id);
CREATE INDEX IF NOT EXISTS idx_customers_email ON customers(email);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_customers_email;
DROP INDEX IF EXISTS idx_customers_org_id;
DROP TABLE IF EXISTS customers;
-- +goose StatementEnd
