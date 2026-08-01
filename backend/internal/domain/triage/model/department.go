package model

import (
	"time"

	"github.com/google/uuid"
)

// Department groups tickets within an organization.
// Exactly one department per organization is the default ("General"); it cannot
// be deleted and receives the tickets of any department that is removed.
// Category binds this department to one classifier label. At most one
// department per organization may claim a given category, which keeps the
// mapping from a prediction to a department deterministic. The default
// department carries no category: it is where predictions with no matching
// department land.
type Department struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Name           string    `json:"name"`
	Description    string    `json:"description"`
	Category       *Category `json:"category,omitempty"`
	IsDefault      bool      `json:"is_default"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// IsDeletable reports whether the department may be removed.
func (d *Department) IsDeletable() bool {
	return !d.IsDefault
}
