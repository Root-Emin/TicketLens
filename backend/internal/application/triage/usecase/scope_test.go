package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
)

/*
	The two properties the customer portal rests on:

	  (a) customer A can reach none of customer B's tickets, through any route,
	      and the refusal is indistinguishable from a ticket that never existed;
	  (b) an internal note never reaches a customer.

	Both are asserted here rather than only at the HTTP layer, because that is
	where they are decided — a handler that forgot to pass a scope would not
	compile, but a use case that ignored one would fail these.
*/

// ── fixtures ────────────────────────────────────────────────────────────────

// fakeRBAC answers permission checks from a fixed grant list, mirroring
// matchesPermission's exact-match branch. The wildcard forms are covered by
// the real service's own tests.
type fakeRBAC struct {
	permissions map[uuid.UUID][]string
	err         error
}

func (r *fakeRBAC) HasPermission(_ context.Context, userID, _ uuid.UUID, permission string) (bool, error) {
	if r.err != nil {
		return false, r.err
	}
	for _, granted := range r.permissions[userID] {
		if granted == permission || granted == "*" {
			return true, nil
		}
	}
	return false, nil
}

func (r *fakeRBAC) HasAnyPermission(ctx context.Context, userID, orgID uuid.UUID, permissions []string) (bool, error) {
	for _, p := range permissions {
		has, err := r.HasPermission(ctx, userID, orgID, p)
		if err != nil || has {
			return has, err
		}
	}
	return false, nil
}

func (r *fakeRBAC) GetUserPermissions(_ context.Context, userID, _ uuid.UUID) ([]string, error) {
	return r.permissions[userID], nil
}

func (r *fakeRBAC) InvalidateCache(context.Context, uuid.UUID, uuid.UUID) error { return nil }

// twoCustomerWorld is the smallest setup that can express "somebody else's
// ticket": one organization, two customers with logins, one ticket each.
type twoCustomerWorld struct {
	orgID  uuid.UUID
	deptID uuid.UUID

	aliceUserID, aliceCustomerID, aliceTicketID uuid.UUID
	bobUserID, bobCustomerID, bobTicketID       uuid.UUID
	agentUserID                                 uuid.UUID

	ticketRepo   *fakeTicketRepo
	messageRepo  *fakeMessageRepo
	customerRepo *fakeCustomerRepo
	resolver     *ResolveScopeUseCase
}

func newTwoCustomerWorld() *twoCustomerWorld {
	w := &twoCustomerWorld{
		orgID:           uuid.New(),
		deptID:          uuid.New(),
		aliceUserID:     uuid.New(),
		aliceCustomerID: uuid.New(),
		aliceTicketID:   uuid.New(),
		bobUserID:       uuid.New(),
		bobCustomerID:   uuid.New(),
		bobTicketID:     uuid.New(),
		agentUserID:     uuid.New(),
	}

	w.ticketRepo = newFakeTicketRepo(
		&model.Ticket{
			ID: w.aliceTicketID, OrganizationID: w.orgID, CustomerID: w.aliceCustomerID,
			DepartmentID: w.deptID, Subject: "Alice cannot pay", Status: model.TicketStatusOpen,
			Priority: model.TicketPriorityNormal,
		},
		&model.Ticket{
			ID: w.bobTicketID, OrganizationID: w.orgID, CustomerID: w.bobCustomerID,
			DepartmentID: w.deptID, Subject: "Bob cannot log in", Status: model.TicketStatusResolved,
			Priority: model.TicketPriorityHigh,
		},
	)

	w.messageRepo = newFakeMessageRepo()
	w.customerRepo = &fakeCustomerRepo{customers: []*model.Customer{
		{ID: w.aliceCustomerID, OrganizationID: w.orgID, UserID: &w.aliceUserID,
			Email: "alice@example.com", FullName: "Alice Morgan"},
		{ID: w.bobCustomerID, OrganizationID: w.orgID, UserID: &w.bobUserID,
			Email: "bob@example.com", FullName: "Bob Reed"},
	}}

	w.resolver = NewResolveScopeUseCase(&fakeRBAC{permissions: map[uuid.UUID][]string{
		// The agent holds the organization-wide read; the customers do not.
		// That absence is the whole mechanism.
		w.agentUserID: {"ticket:read", "ticket:update", "message:create"},
		w.aliceUserID: {"ticket:create", "ticket:read_own", "ticket:reopen_own", "message:create"},
		w.bobUserID:   {"ticket:create", "ticket:read_own", "ticket:reopen_own", "message:create"},
	}}, w.customerRepo)

	return w
}

func (w *twoCustomerWorld) departments() *fakeDepartmentRepo {
	return &fakeDepartmentRepo{departments: []*model.Department{
		{ID: w.deptID, OrganizationID: w.orgID, Name: "General", IsDefault: true},
	}}
}

func (w *twoCustomerWorld) scopeFor(t *testing.T, userID uuid.UUID) Scope {
	t.Helper()
	scope, err := w.resolver.Execute(context.Background(), w.orgID, userID)
	if err != nil {
		t.Fatalf("resolving scope: %v", err)
	}
	return scope
}

func (w *twoCustomerWorld) getTicketUC() *GetTicketUseCase {
	return NewGetTicketUseCase(w.ticketRepo, w.departments(), w.customerRepo,
		w.messageRepo, &fakeAnalysisRepo{}, &fakeUserRepo{})
}

func (w *twoCustomerWorld) listTicketsUC() *ListTicketsUseCase {
	return NewListTicketsUseCase(w.ticketRepo, w.departments(), w.customerRepo,
		w.messageRepo, &fakeAnalysisRepo{}, &fakeUserRepo{})
}

// ── scope resolution ────────────────────────────────────────────────────────

func TestResolveScope_ReadAllGrantsOrgWide(t *testing.T) {
	w := newTwoCustomerWorld()

	scope := w.scopeFor(t, w.agentUserID)

	if scope.IsCustomer() {
		t.Error("a caller holding ticket:read must not be narrowed to one customer")
	}
	if !scope.IncludesInternalNotes() {
		t.Error("staff must keep seeing internal notes")
	}
	if scope.CustomerFilter() != nil {
		t.Error("an org-wide scope must not pin a customer filter")
	}
}

func TestResolveScope_WithoutReadAllPinsToOwnCustomer(t *testing.T) {
	w := newTwoCustomerWorld()

	scope := w.scopeFor(t, w.aliceUserID)

	if !scope.IsCustomer() {
		t.Fatal("a caller without ticket:read must be narrowed to their own tickets")
	}
	if scope.OwnerID() != w.aliceCustomerID {
		t.Errorf("scope owner = %s, want Alice's customer id %s", scope.OwnerID(), w.aliceCustomerID)
	}
	if scope.IncludesInternalNotes() {
		t.Error("a customer scope must never include internal notes")
	}
}

func TestResolveScope_UnlinkedAccountIsRefused(t *testing.T) {
	w := newTwoCustomerWorld()
	stranger := uuid.New() // authenticated, no permissions, no customer record

	_, err := w.resolver.Execute(context.Background(), w.orgID, stranger)

	if !errors.Is(err, domainErr.ErrForbidden) {
		t.Fatalf("error = %v, want ErrForbidden — an account with no customer record "+
			"must be refused rather than shown an empty list", err)
	}
}

// The zero Scope is what a caller gets if somebody adds a code path that forgets
// to resolve one. It must deny, not admit.
func TestZeroScope_DeniesEverything(t *testing.T) {
	var scope Scope

	if !scope.IsCustomer() {
		t.Error("the zero scope must not read as staff")
	}
	if scope.IncludesInternalNotes() {
		t.Error("the zero scope must not expose internal notes")
	}
	if scope.Allows(&model.Ticket{CustomerID: uuid.New()}) {
		t.Error("the zero scope must allow no ticket")
	}
	if scope.Allows(nil) {
		t.Error("the zero scope must reject a nil ticket")
	}
}

// ── (a) one customer cannot reach another's ticket ──────────────────────────

func TestGetTicket_ForeignTicketReadsAsNotFound(t *testing.T) {
	w := newTwoCustomerWorld()
	alice := w.scopeFor(t, w.aliceUserID)

	_, err := w.getTicketUC().Execute(context.Background(), w.orgID, w.bobTicketID, alice)

	// Not-found, not forbidden: a 403 would confirm the id exists, which is
	// enough to enumerate a tenant's tickets one guess at a time.
	if !errors.Is(err, domainErr.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound for another customer's ticket", err)
	}
}

func TestGetTicket_OwnTicketIsReadable(t *testing.T) {
	w := newTwoCustomerWorld()
	alice := w.scopeFor(t, w.aliceUserID)

	detail, err := w.getTicketUC().Execute(context.Background(), w.orgID, w.aliceTicketID, alice)
	if err != nil {
		t.Fatalf("Alice must be able to read her own ticket: %v", err)
	}
	if detail.ID != w.aliceTicketID {
		t.Errorf("got ticket %s, want %s", detail.ID, w.aliceTicketID)
	}
}

func TestListTickets_CustomerSeesOnlyTheirOwn(t *testing.T) {
	w := newTwoCustomerWorld()
	alice := w.scopeFor(t, w.aliceUserID)

	page, err := w.listTicketsUC().Execute(context.Background(), w.orgID,
		repositoryFilterNamingBob(w), dto.PageParams{Page: 1, PageSize: 25}, alice)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(page.Data) != 1 || page.Data[0].ID != w.aliceTicketID {
		t.Fatalf("Alice's queue = %d ticket(s), want exactly her own", len(page.Data))
	}
	// The request named Bob's customer id. It must have been overwritten, not
	// merged: honouring a client-supplied owner filter is the vulnerability.
	if got := w.ticketRepo.lastFilter.CustomerID; got == nil || *got != w.aliceCustomerID {
		t.Errorf("filter.CustomerID = %v, want it forced to Alice's id %s", got, w.aliceCustomerID)
	}
}

func TestListTickets_StaffSeesEveryone(t *testing.T) {
	w := newTwoCustomerWorld()
	agent := w.scopeFor(t, w.agentUserID)

	page, err := w.listTicketsUC().Execute(context.Background(), w.orgID,
		repositoryFilterEmpty(), dto.PageParams{Page: 1, PageSize: 25}, agent)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("agent queue = %d tickets, want both", len(page.Data))
	}
}

func TestCreateMessage_OnForeignTicketReadsAsNotFound(t *testing.T) {
	w := newTwoCustomerWorld()
	alice := w.scopeFor(t, w.aliceUserID)
	uc := NewCreateMessageUseCase(w.ticketRepo, w.messageRepo)

	_, err := uc.Execute(context.Background(), w.orgID, w.bobTicketID, uuid.Nil,
		dto.CreateMessageRequest{Body: "let me in"}, alice)

	if !errors.Is(err, domainErr.ErrNotFound) {
		t.Fatalf("error = %v, want ErrNotFound when writing into another customer's thread", err)
	}
}

func TestCreateTicket_CustomerCannotFileAsSomebodyElse(t *testing.T) {
	w := newTwoCustomerWorld()
	alice := w.scopeFor(t, w.aliceUserID)

	uc := NewCreateTicketUseCase(w.ticketRepo, w.messageRepo, w.customerRepo,
		w.departments(), &fakeAnalysisRepo{}, &fakeUserRepo{}, &fakeTxManager{}, &noopBus{})

	detail, err := uc.Execute(context.Background(), w.orgID, dto.CreateTicketRequest{
		Subject: "Filed under the wrong name",
		Body:    "The body claims to be Bob's.",
		// Supplied by the client and must be ignored outright.
		CustomerID: w.bobCustomerID,
	}, alice)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if detail.Customer.ID != w.aliceCustomerID {
		t.Errorf("ticket owner = %s, want Alice %s — the body's customer_id must be ignored",
			detail.Customer.ID, w.aliceCustomerID)
	}
}

func TestCreateTicket_StaffMustNameACustomer(t *testing.T) {
	w := newTwoCustomerWorld()
	agent := w.scopeFor(t, w.agentUserID)

	uc := NewCreateTicketUseCase(w.ticketRepo, w.messageRepo, w.customerRepo,
		w.departments(), &fakeAnalysisRepo{}, &fakeUserRepo{}, &fakeTxManager{}, &noopBus{})

	_, err := uc.Execute(context.Background(), w.orgID, dto.CreateTicketRequest{
		Subject: "On behalf of nobody",
		Body:    "No customer named.",
	}, agent)

	if !errors.Is(err, domainErr.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation — customer_id stays required for staff", err)
	}
}

// ── (b) internal notes never reach a customer ───────────────────────────────

func TestGetTicket_CustomerNeverReceivesInternalNotes(t *testing.T) {
	w := newTwoCustomerWorld()
	w.messageRepo.byTicket[w.aliceTicketID] = []*model.TicketMessage{
		{ID: uuid.New(), OrganizationID: w.orgID, TicketID: w.aliceTicketID,
			AuthorType: model.AuthorTypeCustomer, Body: "My card is declined."},
		{ID: uuid.New(), OrganizationID: w.orgID, TicketID: w.aliceTicketID,
			AuthorType: model.AuthorTypeAgent, Body: "Escalating to payments.", IsInternal: true},
		{ID: uuid.New(), OrganizationID: w.orgID, TicketID: w.aliceTicketID,
			AuthorType: model.AuthorTypeAgent, Body: "We are on it."},
	}

	alice := w.scopeFor(t, w.aliceUserID)
	detail, err := w.getTicketUC().Execute(context.Background(), w.orgID, w.aliceTicketID, alice)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if len(detail.Messages) != 2 {
		t.Fatalf("customer received %d messages, want the 2 non-internal ones", len(detail.Messages))
	}
	for _, m := range detail.Messages {
		if m.IsInternal {
			t.Errorf("internal note %q reached the customer", m.Body)
		}
	}

	// And the same ticket, read by staff, still carries all three.
	agent := w.scopeFor(t, w.agentUserID)
	staffView, err := w.getTicketUC().Execute(context.Background(), w.orgID, w.aliceTicketID, agent)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if len(staffView.Messages) != 3 {
		t.Errorf("agent received %d messages, want all 3", len(staffView.Messages))
	}
}

func TestListMessages_CustomerNeverReceivesInternalNotes(t *testing.T) {
	w := newTwoCustomerWorld()
	w.messageRepo.byTicket[w.aliceTicketID] = []*model.TicketMessage{
		{ID: uuid.New(), TicketID: w.aliceTicketID, AuthorType: model.AuthorTypeCustomer, Body: "Hello"},
		{ID: uuid.New(), TicketID: w.aliceTicketID, AuthorType: model.AuthorTypeAgent,
			Body: "Internal: refund approved", IsInternal: true},
	}

	alice := w.scopeFor(t, w.aliceUserID)
	uc := NewListMessagesUseCase(w.ticketRepo, w.messageRepo)

	messages, err := uc.Execute(context.Background(), w.orgID, w.aliceTicketID, alice)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	for _, m := range messages {
		if m.IsInternal {
			t.Fatalf("internal note %q reached the customer through /messages", m.Body)
		}
	}
}

func TestCreateMessage_CustomerCannotWriteAnInternalNote(t *testing.T) {
	w := newTwoCustomerWorld()
	alice := w.scopeFor(t, w.aliceUserID)
	uc := NewCreateMessageUseCase(w.ticketRepo, w.messageRepo)

	message, err := uc.Execute(context.Background(), w.orgID, w.aliceTicketID, uuid.Nil,
		dto.CreateMessageRequest{Body: "sneaky", IsInternal: true}, alice)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if message.IsInternal {
		t.Error("a customer's is_internal flag must be discarded, not honoured")
	}
	if message.AuthorType != string(model.AuthorTypeCustomer) {
		t.Errorf("author_type = %q, want customer — authorship comes from the scope",
			message.AuthorType)
	}
	if message.AuthorID == nil || *message.AuthorID != w.aliceCustomerID {
		t.Errorf("author_id = %v, want Alice's customer id", message.AuthorID)
	}
}

func TestCreateMessage_StaffWritesAsAgentAndKeepsInternalFlag(t *testing.T) {
	w := newTwoCustomerWorld()
	agent := w.scopeFor(t, w.agentUserID)
	uc := NewCreateMessageUseCase(w.ticketRepo, w.messageRepo)

	message, err := uc.Execute(context.Background(), w.orgID, w.aliceTicketID, w.agentUserID,
		dto.CreateMessageRequest{Body: "escalating", IsInternal: true}, agent)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if !message.IsInternal {
		t.Error("staff must still be able to write internal notes")
	}
	if message.AuthorType != string(model.AuthorTypeAgent) {
		t.Errorf("author_type = %q, want agent", message.AuthorType)
	}
}

// ── reopen ──────────────────────────────────────────────────────────────────

func TestCustomerPatch_AllowsOnlyReopen(t *testing.T) {
	w := newTwoCustomerWorld()
	alice := w.scopeFor(t, w.aliceUserID)

	resolved := &model.Ticket{ID: w.aliceTicketID, CustomerID: w.aliceCustomerID,
		Status: model.TicketStatusResolved}
	open := &model.Ticket{ID: w.aliceTicketID, CustomerID: w.aliceCustomerID,
		Status: model.TicketStatusOpen}

	status := func(s model.TicketStatus) *string { v := string(s); return &v }
	priority := "urgent"
	department := uuid.New()

	cases := []struct {
		name   string
		ticket *model.Ticket
		req    dto.UpdateTicketRequest
		want   error
	}{
		{"reopen a resolved ticket", resolved,
			dto.UpdateTicketRequest{Status: status(model.TicketStatusOpen)}, nil},
		{"raise own priority", resolved,
			dto.UpdateTicketRequest{Priority: &priority}, domainErr.ErrForbidden},
		{"reroute own ticket", resolved,
			dto.UpdateTicketRequest{DepartmentID: &department}, domainErr.ErrForbidden},
		{"mark own ticket resolved", open,
			dto.UpdateTicketRequest{Status: status(model.TicketStatusResolved)}, domainErr.ErrForbidden},
		{"empty patch", resolved,
			dto.UpdateTicketRequest{}, domainErr.ErrForbidden},
		{"reopen a ticket that is already open", open,
			dto.UpdateTicketRequest{Status: status(model.TicketStatusOpen)}, domainErr.ErrConflict},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := checkCustomerPatch(tc.ticket, tc.req, alice)
			if tc.want == nil {
				if err != nil {
					t.Fatalf("err = %v, want nil", err)
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("err = %v, want %v", err, tc.want)
			}
		})
	}
}

func TestStaffPatch_IsUnrestricted(t *testing.T) {
	w := newTwoCustomerWorld()
	agent := w.scopeFor(t, w.agentUserID)
	priority := "urgent"

	if err := checkCustomerPatch(
		&model.Ticket{Status: model.TicketStatusOpen},
		dto.UpdateTicketRequest{Priority: &priority},
		agent,
	); err != nil {
		t.Fatalf("staff patches must pass through untouched, got %v", err)
	}
}

// repositoryFilterNamingBob builds the filter a malicious request would send:
// a customer id that is not the caller's. The use case must overwrite it.
func repositoryFilterNamingBob(w *twoCustomerWorld) repository.TicketFilter {
	bob := w.bobCustomerID
	return repository.TicketFilter{CustomerID: &bob}
}

func repositoryFilterEmpty() repository.TicketFilter {
	return repository.TicketFilter{}
}
