-- +goose Up
-- +goose StatementBegin
-- author_id is polymorphic (users.id or customers.id depending on author_type),
-- so it carries no foreign key. It is NULL for author_type = 'system'.
CREATE TABLE IF NOT EXISTS ticket_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    ticket_id       UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    author_type     VARCHAR(50) NOT NULL
        CHECK (author_type IN ('customer', 'agent', 'system')),
    author_id       UUID,
    body            TEXT NOT NULL,
    is_internal     BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ticket_messages_org_id ON ticket_messages(organization_id);
CREATE INDEX IF NOT EXISTS idx_ticket_messages_ticket_id ON ticket_messages(ticket_id);
CREATE INDEX IF NOT EXISTS idx_ticket_messages_author_id ON ticket_messages(author_id);

-- Thread view: messages of one ticket in ascending order.
CREATE INDEX IF NOT EXISTS idx_ticket_messages_ticket_created_at
    ON ticket_messages(ticket_id, created_at);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_ticket_messages_ticket_created_at;
DROP INDEX IF EXISTS idx_ticket_messages_author_id;
DROP INDEX IF EXISTS idx_ticket_messages_ticket_id;
DROP INDEX IF EXISTS idx_ticket_messages_org_id;
DROP TABLE IF EXISTS ticket_messages;
-- +goose StatementEnd
