package dto

import (
	"time"

	"github.com/google/uuid"
)

// CreateCustomerRequest is the input for creating a customer.
type CreateCustomerRequest struct {
	Email    string `json:"email" validate:"required,email"`
	FullName string `json:"full_name" validate:"required"`
	Company  string `json:"company"`
}

// CustomerInfo is the public customer representation.
type CustomerInfo struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FullName  string    `json:"full_name"`
	Company   string    `json:"company,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// CustomerDetail is GET /customers/{id}: the customer plus their ticket count
// and five most recent tickets.
type CustomerDetail struct {
	CustomerInfo
	TicketCount   int              `json:"ticket_count"`
	RecentTickets []TicketListItem `json:"recent_tickets"`
}
