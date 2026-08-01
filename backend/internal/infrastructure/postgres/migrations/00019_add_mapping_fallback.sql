-- +goose Up
-- +goose StatementBegin
-- Records that the category -> department mapping fell back to the default
-- department because the organization has none for the predicted category.
--
-- Without this flag the accept rate lies: the fallback writes the default
-- department into predicted_department_id, the ticket sits in that same
-- department, and the comparison reports a match. Nothing was accepted — the
-- system simply could not route. Organizations that define no departments would
-- score a perfect department_accept_rate while being maximally wrong.
ALTER TABLE ai_analyses ADD COLUMN IF NOT EXISTS mapping_fallback BOOLEAN NOT NULL DEFAULT FALSE;

CREATE INDEX IF NOT EXISTS idx_ai_analyses_mapping_fallback ON ai_analyses(mapping_fallback);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_ai_analyses_mapping_fallback;
ALTER TABLE ai_analyses DROP COLUMN IF EXISTS mapping_fallback;
-- +goose StatementEnd
