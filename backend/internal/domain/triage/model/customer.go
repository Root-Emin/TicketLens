package model

import (
	"time"

	"github.com/google/uuid"
)

// Customer is the end user who raises tickets.
// Customers are not platform users: they live in their own table and
// authenticate through a separate portal flow.
type Customer struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Email          string    `json:"email"`
	FullName       string    `json:"full_name"`
	Company        string    `json:"company,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}
