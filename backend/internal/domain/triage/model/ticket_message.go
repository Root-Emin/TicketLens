package model

import (
	"time"

	"github.com/google/uuid"
)

// AuthorType identifies who wrote a ticket message.
type AuthorType string

const (
	AuthorTypeCustomer AuthorType = "customer"
	AuthorTypeAgent    AuthorType = "agent"
	AuthorTypeSystem   AuthorType = "system"
)

// TicketMessage is one entry in a ticket's conversation thread.
// The first message of a ticket is its description.
//
// AuthorID is polymorphic: it points at a user for AuthorTypeAgent and at a
// customer for AuthorTypeCustomer. It is nil for AuthorTypeSystem.
type TicketMessage struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	TicketID       uuid.UUID  `json:"ticket_id"`
	AuthorType     AuthorType `json:"author_type"`
	AuthorID       *uuid.UUID `json:"author_id,omitempty"`
	Body           string     `json:"body"`
	IsInternal     bool       `json:"is_internal"`
	CreatedAt      time.Time  `json:"created_at"`
}

// VisibleToCustomer reports whether the message may be shown in the customer
// portal. Internal notes never are.
func (m *TicketMessage) VisibleToCustomer() bool {
	return !m.IsInternal
}

// ValidAuthorType reports whether a is a known author type.
func ValidAuthorType(a AuthorType) bool {
	switch a {
	case AuthorTypeCustomer, AuthorTypeAgent, AuthorTypeSystem:
		return true
	default:
		return false
	}
}
