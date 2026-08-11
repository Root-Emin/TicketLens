-- +goose Up
-- +goose StatementBegin
-- The portal needs a way to answer "which customer is this token?".
--
-- A JWT subject is a `users` row; a ticket belongs to a `customers` row. Until
-- now nothing joined the two, so `customer_id` had to arrive in the request
-- body — which is exactly what let one customer file a ticket as another. This
-- column is the missing edge: it makes ownership a fact of the database rather
-- than a claim of the client.
--
-- Nullable on purpose. Customers created by an agent (POST /customers) have no
-- login yet, and staff accounts are never customers, so most rows stay NULL.
ALTER TABLE customers ADD COLUMN IF NOT EXISTS user_id UUID REFERENCES users(id) ON DELETE SET NULL;

-- One login maps to at most one customer per organization. Partial, because
-- NULL means "no login" and any number of rows may be in that state.
CREATE UNIQUE INDEX IF NOT EXISTS idx_customers_org_user
    ON customers(organization_id, user_id) WHERE user_id IS NOT NULL;

-- The portal's hot path: resolve the caller's customer row on every request.
CREATE INDEX IF NOT EXISTS idx_customers_user_id ON customers(user_id) WHERE user_id IS NOT NULL;

COMMENT ON COLUMN customers.user_id IS
    'Login this customer signs in with. NULL for customers an agent created who have never registered.';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_customers_user_id;
DROP INDEX IF EXISTS idx_customers_org_user;
ALTER TABLE customers DROP COLUMN IF EXISTS user_id;
-- +goose StatementEnd
