package usecase

import (
	"context"
	"errors"

	iamService "github.com/Root-Emin/TicketLens/internal/domain/iam/service"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/google/uuid"
)

/*
	Who is asking, and how much of the organization they get to see.

	Every ticket read and write in this package takes a Scope. It is a value, not
	a flag, so the two questions that have to be answered together — "is this
	caller limited to their own tickets?" and "which customer are they?" — cannot
	drift apart. A boolean `isCustomer` plus a separate id is how a handler ends
	up scoping the list and forgetting the detail.

	Scope is produced only by ResolveScopeUseCase, from the token and the live
	RBAC grants. Nothing accepts a customer id from a request body or a query
	parameter: an owner filter the client can set is not a filter.
*/

// permissionReadAll is the organization-wide read. A caller who holds it sees
// every ticket in their tenant; a caller who does not is restricted to their
// own, which is what ticket:read_own means in docs/api-contract.md §2.
const permissionReadAll = "ticket:read"

// Scope describes the slice of the organization a caller may act on.
//
// The zero value is the *narrowest* useful thing it could be — a customer scope
// with no customer — and it is unusable: OwnerID returns uuid.Nil and Allows
// rejects everything. Forgetting to pass a real scope therefore denies rather
// than admits.
type Scope struct {
	// customerID pins reads and writes to one customer's tickets.
	customerID *uuid.UUID
	// orgWide is set for callers holding ticket:read; they see everything and
	// carry no customer of their own.
	orgWide bool
}

// OrgWideScope is the staff view: the whole tenant, internal notes included.
func OrgWideScope() Scope {
	return Scope{orgWide: true}
}

// CustomerScope restricts every operation to one customer's own tickets.
func CustomerScope(customerID uuid.UUID) Scope {
	id := customerID
	return Scope{customerID: &id}
}

// IsCustomer reports whether this caller is limited to their own tickets.
func (s Scope) IsCustomer() bool { return !s.orgWide }

// OwnerID is the customer every read and write is pinned to, or uuid.Nil for a
// staff caller.
func (s Scope) OwnerID() uuid.UUID {
	if s.customerID == nil {
		return uuid.Nil
	}
	return *s.customerID
}

// CustomerFilter returns the value to force onto TicketFilter.CustomerID: the
// caller's own id for a customer, nil for staff.
func (s Scope) CustomerFilter() *uuid.UUID {
	if s.orgWide {
		return nil
	}
	return s.customerID
}

// IncludesInternalNotes reports whether this caller may see internal notes.
//
// Derived from the scope rather than passed alongside it, because those are the
// same question: internal notes are staff-only, and a customer must never be
// able to receive one no matter which endpoint they came in through.
func (s Scope) IncludesInternalNotes() bool { return s.orgWide }

// Allows reports whether a ticket is inside this scope.
func (s Scope) Allows(ticket *model.Ticket) bool {
	if ticket == nil {
		return false
	}
	if s.orgWide {
		return true
	}
	return s.customerID != nil && ticket.CustomerID == *s.customerID
}

// Guard returns not-found for a ticket outside the scope, and nil otherwise.
//
// Not-found rather than forbidden, deliberately: a 403 on someone else's ticket
// confirms that the id exists, which is enough to enumerate a tenant's ticket
// ids one guess at a time. The API answers exactly as it would for an id that
// was never issued.
func (s Scope) Guard(ticket *model.Ticket) error {
	if s.Allows(ticket) {
		return nil
	}
	return domainErr.New(domainErr.ErrNotFound, "ticket not found", nil)
}

// ResolveScopeUseCase turns an authenticated caller into a Scope.
type ResolveScopeUseCase struct {
	rbac         iamService.RBACService
	customerRepo repository.CustomerRepository
}

// NewResolveScopeUseCase creates a new ResolveScopeUseCase.
func NewResolveScopeUseCase(
	rbac iamService.RBACService,
	customerRepo repository.CustomerRepository,
) *ResolveScopeUseCase {
	return &ResolveScopeUseCase{rbac: rbac, customerRepo: customerRepo}
}

// Execute decides what the caller may see.
//
// The test is permission-based, never `if role == "customer"`: a caller holding
// ticket:read reads the organization, and everybody else is narrowed to their
// own tickets. That keeps the rule true for any future role — a read-only
// auditor, a partner account — without this function learning its name.
//
// A caller with no ticket:read and no customer record of their own gets
// not-found rather than an empty list. They are not a customer and not staff;
// answering "you have zero tickets" would be a polite lie about an account that
// is simply misconfigured.
func (uc *ResolveScopeUseCase) Execute(ctx context.Context, orgID, userID uuid.UUID) (Scope, error) {
	if orgID == uuid.Nil || userID == uuid.Nil {
		return Scope{}, domainErr.New(domainErr.ErrUnauthorized, "caller is not authenticated", nil)
	}

	readsAll, err := uc.rbac.HasPermission(ctx, userID, orgID, permissionReadAll)
	if err != nil {
		return Scope{}, domainErr.New(domainErr.ErrInternal, "failed to resolve caller scope", err)
	}
	if readsAll {
		return OrgWideScope(), nil
	}

	customer, err := uc.customerRepo.GetByUserID(ctx, orgID, userID)
	if err != nil {
		if errors.Is(err, domainErr.ErrNotFound) {
			return Scope{}, domainErr.New(domainErr.ErrForbidden,
				"this account is not linked to a customer record", nil)
		}
		return Scope{}, err
	}

	return CustomerScope(customer.ID), nil
}

// EnsureCustomer resolves the caller's customer record, creating one from their
// account when they have none.
//
// This is what makes "register, then open a ticket" work end to end: a fresh
// account has a users row and nothing else, and the alternative is telling
// somebody their first request cannot be filed until an agent creates a record
// for them.
//
// If an unlinked customer already exists on the same email — an agent filed a
// ticket for this person before they ever signed in — that row is claimed
// rather than duplicated, so their history carries over.
//
// SECURITY: that claim trusts the email on the account. Registration performs
// no email verification today, so anyone who registers as alice@acme.com
// inherits Alice's tickets. This is safe for the seeded demo and NOT safe for
// production; email verification at registration is the prerequisite, and it is
// called out in the handover notes. Claiming is confined to this function so it
// is one edit to gate or remove.
func (uc *ResolveScopeUseCase) EnsureCustomer(
	ctx context.Context,
	orgID, userID uuid.UUID,
	email, fullName string,
) (*model.Customer, error) {
	existing, err := uc.customerRepo.GetByUserID(ctx, orgID, userID)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, domainErr.ErrNotFound) {
		return nil, err
	}

	byEmail, err := uc.customerRepo.GetByEmail(ctx, orgID, email)
	switch {
	case err == nil && !byEmail.HasLogin():
		byEmail.UserID = &userID
		if err := uc.customerRepo.Update(ctx, byEmail); err != nil {
			return nil, err
		}
		return byEmail, nil
	case err == nil:
		// The email belongs to a different login. Refuse rather than hand over
		// somebody else's ticket history.
		return nil, domainErr.New(domainErr.ErrConflict,
			"another account already uses this email", nil)
	case !errors.Is(err, domainErr.ErrNotFound):
		return nil, err
	}

	customer := &model.Customer{
		OrganizationID: orgID,
		UserID:         &userID,
		Email:          email,
		FullName:       fullName,
	}
	if err := uc.customerRepo.Create(ctx, customer); err != nil {
		return nil, err
	}
	return customer, nil
}
