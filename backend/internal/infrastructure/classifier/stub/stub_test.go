package stub

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/port"
)

// The assertions here are about the contract every Classifier must honour —
// in-taxonomy labels, bounded confidence, determinism — rather than about which
// keyword table a particular phrase happens to land in. The real model has to
// satisfy the same contract, so these carry over; the keyword tables will not.

func classify(t *testing.T, subject, body string) port.ClassifyResult {
	t.Helper()
	result, err := New().Classify(context.Background(), port.ClassifyInput{
		Subject: subject,
		Body:    body,
	})
	require.NoError(t, err, "the stub classifier never fails")
	return result
}

var sampleTickets = []port.ClassifyInput{
	{Subject: "Site down, cannot take orders", Body: "our store is completely unreachable"},
	{Subject: "Invoice question", Body: "I need a copy of last month's invoice"},
	{Subject: "How do I add a user?", Body: "looking for documentation"},
	{Subject: "Trendyol integration not syncing", Body: "products are not transferred"},
	{Subject: "", Body: ""},
	{Subject: "ǧarbled ünïcode ✓", Body: "12345 !@#$%"},
	{Subject: "password reset", Body: "cannot login, 2fa keeps failing"},
	{Subject: "settlement missing", Body: "payout did not reach my account"},
}

func TestClassify_AlwaysReturnsLabelsInsideTheTaxonomy(t *testing.T) {
	// A label outside the taxonomy is rejected downstream by AnalyzeTicketUseCase
	// and would leave the ticket unclassified, so this is the load-bearing
	// guarantee of the port.
	for _, in := range sampleTickets {
		t.Run(in.Subject, func(t *testing.T) {
			result := classify(t, in.Subject, in.Body)

			assert.True(t, model.ValidTicketPriority(model.TicketPriority(result.Priority)),
				"priority %q is not in the taxonomy", result.Priority)
			assert.True(t, model.ValidCategory(model.Category(result.Category)),
				"category %q is not in the taxonomy", result.Category)
		})
	}
}

func TestClassify_ConfidenceStaysWithinBounds(t *testing.T) {
	for _, in := range sampleTickets {
		t.Run(in.Subject, func(t *testing.T) {
			result := classify(t, in.Subject, in.Body)

			assert.GreaterOrEqual(t, result.PriorityConfidence, minConfidence)
			assert.LessOrEqual(t, result.PriorityConfidence, maxConfidence)
			assert.GreaterOrEqual(t, result.CategoryConfidence, minConfidence)
			assert.LessOrEqual(t, result.CategoryConfidence, maxConfidence)
		})
	}
}

func TestClassify_IsDeterministic(t *testing.T) {
	// Seed data and end-to-end expectations depend on this: the same ticket must
	// always produce the same row. Map iteration order is the usual way it breaks.
	for _, in := range sampleTickets {
		t.Run(in.Subject, func(t *testing.T) {
			first := classify(t, in.Subject, in.Body)
			for i := 0; i < 25; i++ {
				assert.Equal(t, first, classify(t, in.Subject, in.Body))
			}
		})
	}
}

func TestClassify_UnmatchedTicketLandsAtFloorConfidence(t *testing.T) {
	// There is no "other" class, so an unrecognised ticket must instead be
	// unmistakably low-confidence — that is what routes it to a human.
	result := classify(t, "zzzz", "qqqq")

	assert.Equal(t, minConfidence, result.CategoryConfidence)
	assert.Equal(t, minConfidence, result.PriorityConfidence)
	assert.Equal(t, string(model.CategoryHowTo), result.Category,
		"the broadest label is the documented fallback")
	assert.Equal(t, string(model.TicketPriorityNormal), result.Priority)
}

func TestClassify_SubjectOutweighsBody(t *testing.T) {
	// The subject is the strongest signal a ticket has. A category named in the
	// subject must beat a competing category mentioned only in the body.
	result := classify(t,
		"Invoice and subscription billing question",
		"this started after we set up the trendyol integration",
	)

	assert.Equal(t, string(model.CategoryBilling), result.Category)
}

func TestClassify_MoreEvidenceRaisesConfidence(t *testing.T) {
	weak := classify(t, "invoice", "")
	strong := classify(t, "invoice subscription billing commission plan", "")

	assert.Greater(t, strong.CategoryConfidence, weak.CategoryConfidence,
		"a ticket matching more of the taxonomy should score higher")
}

func TestClassify_PriorityFollowsBusinessImpact(t *testing.T) {
	tests := []struct {
		name     string
		subject  string
		body     string
		expected model.TicketPriority
	}{
		{"stopped business is urgent", "site down", "we cannot sell anything", model.TicketPriorityUrgent},
		{"visible breakage is high", "error on export", "the sync failed", model.TicketPriorityHigh},
		{"a question is low", "how to configure webhooks", "", model.TicketPriorityLow},
		{"no signal is normal", "quick note", "just checking in", model.TicketPriorityNormal},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := classify(t, tc.subject, tc.body)
			assert.Equal(t, string(tc.expected), result.Priority)
		})
	}
}

func TestClassify_UrgentOutranksHigh(t *testing.T) {
	// A ticket usually trips several tables at once. The most severe must win,
	// otherwise an outage could be filed as ordinary breakage.
	result := classify(t, "error: site down", "everything failed and we cannot sell")

	assert.Equal(t, string(model.TicketPriorityUrgent), result.Priority)
}

func TestClassify_IsCaseAndWhitespaceInsensitive(t *testing.T) {
	lower := classify(t, "invoice question", "about my subscription")
	upper := classify(t, "  INVOICE Question  ", "  About My SUBSCRIPTION  ")

	assert.Equal(t, lower.Category, upper.Category)
	assert.Equal(t, lower.Priority, upper.Priority)
	assert.Equal(t, lower.CategoryConfidence, upper.CategoryConfidence)
}

func TestClassify_ReportsModelIdentityAndParseableRaw(t *testing.T) {
	// The analysis row stores model_name/model_version so a prediction can be
	// attributed to what produced it once several models are in play.
	result := classify(t, "invoice question", "")

	assert.Equal(t, modelName, result.ModelName)
	assert.Equal(t, modelVersion, result.ModelVersion)

	var raw map[string]any
	require.NoError(t, json.Unmarshal(result.Raw, &raw),
		"raw_response is persisted as jsonb and must be valid JSON")
	assert.Equal(t, modelName, raw["engine"])
}

func TestClassify_EveryCategoryIsReachable(t *testing.T) {
	// A label with no keywords could never be predicted, which would silently
	// shrink the taxonomy the model is meant to cover.
	for _, category := range model.AllCategories {
		keywords, ok := categoryKeywords[category]
		assert.True(t, ok, "category %q has no keyword table", category)
		assert.NotEmpty(t, keywords, "category %q has an empty keyword table", category)
	}
}
