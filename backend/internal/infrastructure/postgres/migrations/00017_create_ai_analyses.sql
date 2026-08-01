-- +goose Up
-- +goose StatementBegin
-- Append-only. A ticket may have several analyses (re-runs, model comparisons);
-- "the" analysis for display purposes is the most recent one.
CREATE TABLE IF NOT EXISTS ai_analyses (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id         UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    ticket_id               UUID NOT NULL REFERENCES tickets(id) ON DELETE CASCADE,
    predicted_priority      VARCHAR(50) NOT NULL
        CHECK (predicted_priority IN ('low', 'normal', 'high', 'urgent')),
    priority_confidence     DOUBLE PRECISION NOT NULL
        CHECK (priority_confidence >= 0 AND priority_confidence <= 1),
    -- NULL when the classifier returned a department name with no match in this
    -- organization; the application falls back to the default department.
    predicted_department_id UUID REFERENCES departments(id) ON DELETE SET NULL,
    department_confidence   DOUBLE PRECISION NOT NULL
        CHECK (department_confidence >= 0 AND department_confidence <= 1),
    needs_human_review      BOOLEAN NOT NULL DEFAULT FALSE,
    model_name              VARCHAR(255) NOT NULL,
    model_version           VARCHAR(100) NOT NULL,
    raw_response            JSONB,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ai_analyses_org_id ON ai_analyses(organization_id);
CREATE INDEX IF NOT EXISTS idx_ai_analyses_ticket_id ON ai_analyses(ticket_id);
CREATE INDEX IF NOT EXISTS idx_ai_analyses_needs_human_review ON ai_analyses(needs_human_review);
CREATE INDEX IF NOT EXISTS idx_ai_analyses_predicted_department_id
    ON ai_analyses(predicted_department_id);

-- Latest-analysis lookup per ticket (list view, needs_review filter).
CREATE INDEX IF NOT EXISTS idx_ai_analyses_ticket_created_at
    ON ai_analyses(ticket_id, created_at DESC);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_ai_analyses_ticket_created_at;
DROP INDEX IF EXISTS idx_ai_analyses_predicted_department_id;
DROP INDEX IF EXISTS idx_ai_analyses_needs_human_review;
DROP INDEX IF EXISTS idx_ai_analyses_ticket_id;
DROP INDEX IF EXISTS idx_ai_analyses_org_id;
DROP TABLE IF EXISTS ai_analyses;
-- +goose StatementEnd
