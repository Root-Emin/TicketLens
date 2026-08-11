package model

import (
	"time"

	"github.com/google/uuid"
)

// Customer is the end user who raises tickets.
//
// A customer is not a platform user: they live in their own table, and an agent
// can create one for somebody who has never signed in. UserID is the optional
// bridge to an account — set once that person registers and signs into the
// portal, and the only thing that makes "my tickets" answerable.
type Customer struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	// UserID is the login this customer signs in with, or nil when they have
	// none. It is what a portal request resolves its own identity through; it is
	// never accepted from a request body.
	UserID    *uuid.UUID `json:"user_id,omitempty"`
	Email     string     `json:"email"`
	FullName  string     `json:"full_name"`
	Company   string     `json:"company,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// HasLogin reports whether this customer can sign into the portal.
func (c *Customer) HasLogin() bool {
	return c.UserID != nil && *c.UserID != uuid.Nil
}
