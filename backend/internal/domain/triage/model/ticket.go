package model

import (
	"time"

	"github.com/google/uuid"
)

// TicketStatus represents the lifecycle state of a ticket.
type TicketStatus string

const (
	TicketStatusOpen            TicketStatus = "open"
	TicketStatusInProgress      TicketStatus = "in_progress"
	TicketStatusPendingCustomer TicketStatus = "pending_customer"
	TicketStatusResolved        TicketStatus = "resolved"
	TicketStatusClosed          TicketStatus = "closed"
)

// TicketPriority represents the urgency of a ticket.
type TicketPriority string

const (
	TicketPriorityLow    TicketPriority = "low"
	TicketPriorityNormal TicketPriority = "normal"
	TicketPriorityHigh   TicketPriority = "high"
	TicketPriorityUrgent TicketPriority = "urgent"
)

// Ticket is a customer request handled by the organization.
//
// It has no description field: the first TicketMessage is the description.
//
// Priority and DepartmentID hold the current effective values. The model's
// original guess stays in AIAnalysis. When a human sets either field to
// something other than the prediction, the matching *Overridden flag flips to
// true — that is what the "X% of AI predictions were accepted" metric counts.
type Ticket struct {
	ID                   uuid.UUID      `json:"id"`
	OrganizationID       uuid.UUID      `json:"organization_id"`
	CustomerID           uuid.UUID      `json:"customer_id"`
	DepartmentID         uuid.UUID      `json:"department_id"`
	AssigneeID           *uuid.UUID     `json:"assignee_id,omitempty"`
	Subject              string         `json:"subject"`
	Status               TicketStatus   `json:"status"`
	Priority             TicketPriority `json:"priority"`
	PriorityOverridden   bool           `json:"priority_overridden"`
	DepartmentOverridden bool           `json:"department_overridden"`
	CreatedAt            time.Time      `json:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at"`
	ResolvedAt           *time.Time     `json:"resolved_at,omitempty"`
}

// IsResolved reports whether the ticket has reached a terminal state.
func (t *Ticket) IsResolved() bool {
	return t.Status == TicketStatusResolved || t.Status == TicketStatusClosed
}

// IsAssigned reports whether the ticket has an owner.
func (t *Ticket) IsAssigned() bool {
	return t.AssigneeID != nil && *t.AssigneeID != uuid.Nil
}

// IsOverridden reports whether a human corrected either AI prediction.
func (t *Ticket) IsOverridden() bool {
	return t.PriorityOverridden || t.DepartmentOverridden
}

// ValidTicketStatus reports whether s is a known status value.
func ValidTicketStatus(s TicketStatus) bool {
	switch s {
	case TicketStatusOpen, TicketStatusInProgress, TicketStatusPendingCustomer,
		TicketStatusResolved, TicketStatusClosed:
		return true
	default:
		return false
	}
}

// ValidTicketPriority reports whether p is a known priority value.
func ValidTicketPriority(p TicketPriority) bool {
	switch p {
	case TicketPriorityLow, TicketPriorityNormal, TicketPriorityHigh, TicketPriorityUrgent:
		return true
	default:
		return false
	}
}
