package dto

import (
	"encoding/json"
	"time"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/google/uuid"
)

// CreateTicketRequest is the input for raising a ticket.
//
// department_id and priority are deliberately absent: they are the classifier's
// job. Until the first analysis arrives the ticket sits in the default
// department at normal priority.
type CreateTicketRequest struct {
	Subject string `json:"subject" validate:"required,min=3"`
	Body    string `json:"body" validate:"required"`
	// CustomerID is required when an agent raises the ticket. A customer portal
	// token would supply it instead, but that flow is not implemented yet.
	CustomerID uuid.UUID `json:"customer_id" validate:"required"`
}

// UpdateTicketRequest is the input for PATCH /tickets/{id}. Pointer fields keep
// "not supplied" distinguishable from "set to zero value".
type UpdateTicketRequest struct {
	Priority     *string    `json:"priority"`
	DepartmentID *uuid.UUID `json:"department_id"`
	Status       *string    `json:"status"`
}

// AssignTicketRequest is the input for POST /tickets/{id}/assign.
// A null assignee_id unassigns the ticket.
type AssignTicketRequest struct {
	AssigneeID *uuid.UUID `json:"assignee_id"`
}

// CreateMessageRequest is the input for POST /tickets/{id}/messages.
// author_type and author_id come from the token, never from the body.
type CreateMessageRequest struct {
	Body       string `json:"body" validate:"required"`
	IsInternal bool   `json:"is_internal"`
}

// DepartmentRef is the inline department of a ticket.
type DepartmentRef struct {
	ID   uuid.UUID `json:"id"`
	Name string    `json:"name"`
}

// CustomerRef is the inline customer of a ticket.
type CustomerRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
}

// UserRef is the inline assignee of a ticket.
type UserRef struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    string    `json:"email"`
}

// LatestAnalysis is the slice of classifier output the queue view needs.
// It is null while classification is pending, so the UI can render an
// "analyzing" state rather than zero confidence.
//
// DepartmentConfidence carries the CATEGORY confidence; the field keeps its
// name to match the stored column (see migration 00018).
type LatestAnalysis struct {
	PredictedCategory *model.Category `json:"predicted_category"`
	// PredictedPriority is the priority the model proposed, kept alongside the
	// category so the queue can show "AI said X, agent set Y" without a second
	// request per ticket for the full analysis history.
	PredictedPriority    string  `json:"predicted_priority"`
	PriorityConfidence   float64 `json:"priority_confidence"`
	DepartmentConfidence float64 `json:"department_confidence"`
	NeedsHumanReview     bool    `json:"needs_human_review"`
	// MappingFallback marks a ticket the system could not route: the predicted
	// category has no department in this organization. The UI renders a
	// "could not be routed" badge from it.
	MappingFallback bool   `json:"mapping_fallback"`
	ModelName       string `json:"model_name"`
}

// TicketListItem is one row of the ticket queue.
type TicketListItem struct {
	ID                   uuid.UUID       `json:"id"`
	Subject              string          `json:"subject"`
	Status               string          `json:"status"`
	Priority             string          `json:"priority"`
	PriorityOverridden   bool            `json:"priority_overridden"`
	DepartmentOverridden bool            `json:"department_overridden"`
	Department           DepartmentRef   `json:"department"`
	Customer             CustomerRef     `json:"customer"`
	Assignee             *UserRef        `json:"assignee"`
	MessageCount         int             `json:"message_count"`
	LatestAnalysis       *LatestAnalysis `json:"latest_analysis"`
	CreatedAt            time.Time       `json:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at"`
	ResolvedAt           *time.Time      `json:"resolved_at,omitempty"`
}

// MessageInfo is one entry of a ticket's conversation.
type MessageInfo struct {
	ID         uuid.UUID  `json:"id"`
	TicketID   uuid.UUID  `json:"ticket_id"`
	AuthorType string     `json:"author_type"`
	AuthorID   *uuid.UUID `json:"author_id,omitempty"`
	Body       string     `json:"body"`
	IsInternal bool       `json:"is_internal"`
	CreatedAt  time.Time  `json:"created_at"`
}

// AnalysisInfo is one full classifier run, raw response included.
//
// PredictedCategory is what the model said; PredictedDepartmentID is what the
// application mapped it onto. Keeping both is what separates "the model was
// wrong" from "this organization has no department for that category".
type AnalysisInfo struct {
	ID                    uuid.UUID       `json:"id"`
	TicketID              uuid.UUID       `json:"ticket_id"`
	PredictedPriority     string          `json:"predicted_priority"`
	PriorityConfidence    float64         `json:"priority_confidence"`
	PredictedCategory     *model.Category `json:"predicted_category"`
	PredictedDepartmentID *uuid.UUID      `json:"predicted_department_id,omitempty"`
	MappingFallback       bool            `json:"mapping_fallback"`
	DepartmentConfidence  float64         `json:"department_confidence"`
	NeedsHumanReview      bool            `json:"needs_human_review"`
	ModelName             string          `json:"model_name"`
	ModelVersion          string          `json:"model_version"`
	RawResponse           json.RawMessage `json:"raw_response,omitempty"`
	CreatedAt             time.Time       `json:"created_at"`
}

// TicketDetail is GET /tickets/{id}: the list item plus the full thread and
// analysis history.
type TicketDetail struct {
	TicketListItem
	Messages []MessageInfo  `json:"messages"`
	Analyses []AnalysisInfo `json:"analyses"`
	// ReviewThreshold is the confidence cutoff currently configured
	// (CLASSIFIER_REVIEW_THRESHOLD). It is published so the UI can draw the
	// cutoff on a confidence meter instead of hardcoding a copy that silently
	// disagrees the moment the setting changes.
	//
	// It is the *current* setting, not the one each analysis was judged under —
	// the threshold is not stored per row. Each analysis carries its own verdict
	// in NeedsHumanReview, so read that for "was this flagged", and read this
	// only to show where the line sits today.
	ReviewThreshold float64 `json:"review_threshold"`
}
