-- +goose Up
-- +goose StatementBegin
-- No description column: the first ticket_message is the description.
-- priority and department_id hold the CURRENT EFFECTIVE value; the model's
-- original guess lives in ai_analyses. The *_overridden flags record whether a
-- human changed the value, which is what powers the AI accept-rate metric.
CREATE TABLE IF NOT EXISTS tickets (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id       UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    customer_id           UUID NOT NULL REFERENCES customers(id) ON DELETE CASCADE,
    -- No ON DELETE action: a department with tickets cannot be dropped until the
    -- application reassigns them to the organization's default department.
    department_id         UUID NOT NULL REFERENCES departments(id),
    assignee_id           UUID REFERENCES users(id) ON DELETE SET NULL,
    subject               VARCHAR(500) NOT NULL,
    status                VARCHAR(50) NOT NULL DEFAULT 'open'
        CHECK (status IN ('open', 'in_progress', 'pending_customer', 'resolved', 'closed')),
    priority              VARCHAR(50) NOT NULL DEFAULT 'normal'
        CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    priority_overridden   BOOLEAN NOT NULL DEFAULT FALSE,
    department_overridden BOOLEAN NOT NULL DEFAULT FALSE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at           TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_tickets_org_id ON tickets(organization_id);
CREATE INDEX IF NOT EXISTS idx_tickets_customer_id ON tickets(customer_id);
CREATE INDEX IF NOT EXISTS idx_tickets_department_id ON tickets(department_id);
CREATE INDEX IF NOT EXISTS idx_tickets_assignee_id ON tickets(assignee_id);
CREATE INDEX IF NOT EXISTS idx_tickets_status ON tickets(status);
CREATE INDEX IF NOT EXISTS idx_tickets_priority ON tickets(priority);
CREATE INDEX IF NOT EXISTS idx_tickets_created_at ON tickets(created_at);

-- Queue listing is always organization-scoped and sorted by recency.
CREATE INDEX IF NOT EXISTS idx_tickets_org_created_at ON tickets(organization_id, created_at DESC);
-- Backs the ?overridden=true filter and the accept-rate aggregation.
CREATE INDEX IF NOT EXISTS idx_tickets_org_overridden
    ON tickets(organization_id, priority_overridden, department_overridden);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_tickets_org_overridden;
DROP INDEX IF EXISTS idx_tickets_org_created_at;
DROP INDEX IF EXISTS idx_tickets_created_at;
DROP INDEX IF EXISTS idx_tickets_priority;
DROP INDEX IF EXISTS idx_tickets_status;
DROP INDEX IF EXISTS idx_tickets_assignee_id;
DROP INDEX IF EXISTS idx_tickets_department_id;
DROP INDEX IF EXISTS idx_tickets_customer_id;
DROP INDEX IF EXISTS idx_tickets_org_id;
DROP TABLE IF EXISTS tickets;
-- +goose StatementEnd
