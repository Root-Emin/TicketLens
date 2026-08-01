package model

import (
	"time"

	"github.com/google/uuid"
)

// AIAnalysis is one classifier run against a ticket.
//
// The table is append-only: a ticket may hold several analyses (re-runs, model
// comparisons), and the most recent one is "the" analysis for display purposes.
// These fields keep the model's original guess, while the ticket itself carries
// the current effective priority and department.
type AIAnalysis struct {
	ID                 uuid.UUID      `json:"id"`
	OrganizationID     uuid.UUID      `json:"organization_id"`
	TicketID           uuid.UUID      `json:"ticket_id"`
	PredictedPriority  TicketPriority `json:"predicted_priority"`
	PriorityConfidence float64        `json:"priority_confidence"`
	// PredictedCategory is what the model actually returned; it is the honest
	// record of the model's opinion, independent of any organization's setup.
	PredictedCategory *Category `json:"predicted_category,omitempty"`
	// PredictedDepartmentID is the department the application mapped the
	// category onto, or the default department when nothing matched.
	PredictedDepartmentID *uuid.UUID `json:"predicted_department_id,omitempty"`
	// MappingFallback is true when no department claimed the predicted category
	// and the ticket landed in the default one. Such an analysis must be kept
	// out of the department accept rate: routing did not succeed, so counting
	// the match as agreement would inflate the metric.
	MappingFallback bool `json:"mapping_fallback"`
	// DepartmentConfidence carries the CATEGORY confidence. The column kept its
	// original name for compatibility; see migration 00018.
	DepartmentConfidence float64   `json:"department_confidence"`
	NeedsHumanReview     bool      `json:"needs_human_review"`
	ModelName            string    `json:"model_name"`
	ModelVersion         string    `json:"model_version"`
	RawResponse          []byte    `json:"raw_response,omitempty"`
	CreatedAt            time.Time `json:"created_at"`
}

// PriorityAccepted reports whether the ticket's current priority still matches
// what this analysis predicted.
func (a *AIAnalysis) PriorityAccepted(current TicketPriority) bool {
	return a.PredictedPriority == current
}

// DepartmentAccepted reports whether the ticket's current department still
// matches what this analysis predicted. An analysis with no predicted
// department never counts as accepted.
func (a *AIAnalysis) DepartmentAccepted(current uuid.UUID) bool {
	if a.PredictedDepartmentID == nil {
		return false
	}
	return *a.PredictedDepartmentID == current
}
