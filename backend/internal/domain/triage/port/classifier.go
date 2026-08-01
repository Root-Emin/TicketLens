// Package port holds the outbound interfaces the triage context depends on.
// Nothing here may import an infrastructure package: adapters live under
// internal/infrastructure and depend on these definitions, never the reverse.
package port

import (
	"context"
	"encoding/json"
)

// ClassifyInput is everything the classifier sees about a ticket.
type ClassifyInput struct {
	Subject string
	Body    string
}

// ClassifyResult is one classification.
//
// Category comes from the fixed taxonomy in the model package; the classifier
// never names a department, because departments differ per organization.
// CategoryConfidence therefore describes the label, not the routing.
type ClassifyResult struct {
	Priority           string // low | normal | high | urgent
	PriorityConfidence float64
	Category           string // one of model.AllCategories
	CategoryConfidence float64
	ModelName          string
	ModelVersion       string
	Raw                json.RawMessage
}

// Classifier turns a ticket into a priority and a category.
//
// Implementations must be safe for concurrent use. The HTTP adapter for the
// real model will sit beside the stub without this interface changing.
type Classifier interface {
	Classify(ctx context.Context, in ClassifyInput) (ClassifyResult, error)
}
