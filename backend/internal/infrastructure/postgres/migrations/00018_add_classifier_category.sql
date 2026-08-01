-- +goose Up
-- +goose StatementBegin
-- The classifier predicts a CATEGORY from a fixed, model-wide taxonomy; it does
-- not predict a department. Departments are per-organization, so mapping a
-- category onto one is the application's job. Storing both lets us tell a wrong
-- prediction apart from a missing mapping.
ALTER TABLE departments ADD COLUMN IF NOT EXISTS category TEXT;

ALTER TABLE departments DROP CONSTRAINT IF EXISTS departments_category_check;
ALTER TABLE departments ADD CONSTRAINT departments_category_check
    CHECK (category IS NULL OR category IN (
        'technical_issue', 'integration', 'payment_ops', 'billing', 'onboarding',
        'how_to', 'account_access', 'feature_request', 'sales', 'compliance'
    ));

-- One department per category per organization keeps the mapping deterministic.
-- Departments with no category (the default one) are exempt.
CREATE UNIQUE INDEX IF NOT EXISTS idx_departments_org_category
    ON departments(organization_id, category) WHERE category IS NOT NULL;

ALTER TABLE ai_analyses ADD COLUMN IF NOT EXISTS predicted_category TEXT;

ALTER TABLE ai_analyses DROP CONSTRAINT IF EXISTS ai_analyses_predicted_category_check;
ALTER TABLE ai_analyses ADD CONSTRAINT ai_analyses_predicted_category_check
    CHECK (predicted_category IS NULL OR predicted_category IN (
        'technical_issue', 'integration', 'payment_ops', 'billing', 'onboarding',
        'how_to', 'account_access', 'feature_request', 'sales', 'compliance'
    ));

CREATE INDEX IF NOT EXISTS idx_ai_analyses_predicted_category
    ON ai_analyses(predicted_category);

-- NOTE: ai_analyses.department_confidence now carries the CATEGORY confidence.
-- The column keeps its name so existing readers and migrations stay valid; the
-- value it holds is how sure the model was about predicted_category, not about
-- the department the application mapped it onto.
COMMENT ON COLUMN ai_analyses.department_confidence IS
    'Confidence of predicted_category (kept under the old name for compatibility)';
COMMENT ON COLUMN ai_analyses.predicted_department_id IS
    'Department the application mapped predicted_category onto, or the default department on fallback';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_ai_analyses_predicted_category;
ALTER TABLE ai_analyses DROP CONSTRAINT IF EXISTS ai_analyses_predicted_category_check;
ALTER TABLE ai_analyses DROP COLUMN IF EXISTS predicted_category;
DROP INDEX IF EXISTS idx_departments_org_category;
ALTER TABLE departments DROP CONSTRAINT IF EXISTS departments_category_check;
ALTER TABLE departments DROP COLUMN IF EXISTS category;
-- +goose StatementEnd
