package model

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestValidTicketStatus(t *testing.T) {
	for _, status := range []TicketStatus{
		TicketStatusOpen, TicketStatusInProgress, TicketStatusPendingCustomer,
		TicketStatusResolved, TicketStatusClosed,
	} {
		assert.True(t, ValidTicketStatus(status), "%q should be accepted", status)
	}

	// These reach the domain straight from a query string or a model response,
	// so rejecting them is what keeps an unknown value out of the database.
	for _, status := range []TicketStatus{"", "OPEN", "open ", "pending", "deleted", "in-progress"} {
		assert.False(t, ValidTicketStatus(status), "%q should be rejected", status)
	}
}

func TestValidTicketPriority(t *testing.T) {
	for _, priority := range []TicketPriority{
		TicketPriorityLow, TicketPriorityNormal, TicketPriorityHigh, TicketPriorityUrgent,
	} {
		assert.True(t, ValidTicketPriority(priority), "%q should be accepted", priority)
	}

	for _, priority := range []TicketPriority{"", "URGENT", "critical", "medium", "p1"} {
		assert.False(t, ValidTicketPriority(priority), "%q should be rejected", priority)
	}
}

func TestValidCategory(t *testing.T) {
	for _, category := range AllCategories {
		assert.True(t, ValidCategory(category), "%q should be accepted", category)
	}

	// "other" in particular: the taxonomy deliberately has no escape hatch, and
	// a model that invents one must be rejected rather than quietly stored.
	for _, category := range []Category{"", "other", "OTHER", "technical", "TECHNICAL_ISSUE", "unknown"} {
		assert.False(t, ValidCategory(category), "%q should be rejected", category)
	}
}

func TestAllCategoriesIsTheWholeTaxonomyWithoutDuplicates(t *testing.T) {
	// AllCategories drives validation, the zero-filled statistics buckets and the
	// classifier's label set, so a duplicate or a missing entry would spread.
	seen := map[Category]bool{}
	for _, category := range AllCategories {
		assert.False(t, seen[category], "%q appears twice in AllCategories", category)
		seen[category] = true
	}
	assert.Len(t, AllCategories, 10, "the taxonomy is fixed at ten labels")
}

func TestTicketIsResolved(t *testing.T) {
	tests := []struct {
		status   TicketStatus
		resolved bool
	}{
		{TicketStatusOpen, false},
		{TicketStatusInProgress, false},
		{TicketStatusPendingCustomer, false},
		{TicketStatusResolved, true},
		{TicketStatusClosed, true},
	}

	for _, tc := range tests {
		t.Run(string(tc.status), func(t *testing.T) {
			ticket := &Ticket{Status: tc.status}
			assert.Equal(t, tc.resolved, ticket.IsResolved())
		})
	}
}

func TestTicketIsAssigned(t *testing.T) {
	assigneeID := uuid.New()
	nilID := uuid.Nil

	assert.False(t, (&Ticket{}).IsAssigned(), "no assignee")
	assert.True(t, (&Ticket{AssigneeID: &assigneeID}).IsAssigned())
	// A pointer to the zero UUID is not an assignment; it is how an unassigned
	// row can come back from the database.
	assert.False(t, (&Ticket{AssigneeID: &nilID}).IsAssigned())
}

func TestTicketIsOverridden(t *testing.T) {
	// IsOverridden backs the ?overridden= filter and the accept-rate metric:
	// a correction on either axis counts.
	assert.False(t, (&Ticket{}).IsOverridden())
	assert.True(t, (&Ticket{PriorityOverridden: true}).IsOverridden())
	assert.True(t, (&Ticket{DepartmentOverridden: true}).IsOverridden())
	assert.True(t, (&Ticket{PriorityOverridden: true, DepartmentOverridden: true}).IsOverridden())
}
