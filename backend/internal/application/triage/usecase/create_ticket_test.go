package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/shared/events"
)

// fakeTxManager runs the unit of work and records that it was asked to. Because
// the in-memory repositories are not transactional, this double cannot undo a
// write; the atomicity guarantee itself lives in the postgres TxManager. What it
// proves here is that CreateTicketUseCase performs both writes inside the unit of
// work and surfaces a failure from within it.
type fakeTxManager struct {
	calls int
}

func (m *fakeTxManager) WithinTx(ctx context.Context, fn func(ctx context.Context) error) error {
	m.calls++
	return fn(ctx)
}

// noopBus discards published events.
type noopBus struct{ published int }

func (b *noopBus) Publish(context.Context, string, events.Event) error { b.published++; return nil }
func (b *noopBus) Subscribe(string, events.Handler)                    {}
func (b *noopBus) Close() error                                        { return nil }

func newCreateTicketFixture() (*CreateTicketUseCase, *fakeTicketRepo, *fakeMessageRepo, *fakeTxManager, *noopBus, uuid.UUID, uuid.UUID) {
	orgID := uuid.New()
	customerID := uuid.New()
	deptID := uuid.New()

	ticketRepo := newFakeTicketRepo()
	messageRepo := newFakeMessageRepo()
	customerRepo := &fakeCustomerRepo{customers: []*model.Customer{
		{ID: customerID, OrganizationID: orgID, Email: "c@example.com", FullName: "Customer"},
	}}
	departmentRepo := &fakeDepartmentRepo{departments: []*model.Department{
		{ID: deptID, OrganizationID: orgID, Name: "General", IsDefault: true},
	}}
	analysisRepo := &fakeAnalysisRepo{}
	userRepo := &fakeUserRepo{}
	tx := &fakeTxManager{}
	bus := &noopBus{}

	uc := NewCreateTicketUseCase(
		ticketRepo, messageRepo, customerRepo, departmentRepo, analysisRepo, userRepo, tx, bus,
	)
	return uc, ticketRepo, messageRepo, tx, bus, orgID, customerID
}

func TestCreateTicket_WritesBothInsideTransaction(t *testing.T) {
	uc, ticketRepo, messageRepo, tx, bus, orgID, customerID := newCreateTicketFixture()

	detail, err := uc.Execute(context.Background(), orgID, dto.CreateTicketRequest{
		Subject:    "Cannot log in",
		Body:       "The password reset email never arrives.",
		CustomerID: customerID,
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if tx.calls != 1 {
		t.Errorf("expected the write to run inside exactly one transaction, got %d", tx.calls)
	}
	if len(ticketRepo.tickets) != 1 {
		t.Errorf("expected 1 ticket stored, got %d", len(ticketRepo.tickets))
	}

	msgs := messageRepo.byTicket[detail.ID]
	if len(msgs) != 1 {
		t.Fatalf("expected 1 first message, got %d", len(msgs))
	}
	if msgs[0].TicketID != detail.ID {
		t.Errorf("first message points at %s, want ticket %s", msgs[0].TicketID, detail.ID)
	}
	if msgs[0].AuthorType != model.AuthorTypeCustomer {
		t.Errorf("first message author = %q, want customer", msgs[0].AuthorType)
	}
	if bus.published != 1 {
		t.Errorf("expected ticket.created to be published once, got %d", bus.published)
	}
}

// errMessageRepo fails the message write, standing in for a failure on the
// second statement of the pair.
type errMessageRepo struct {
	*fakeMessageRepo
}

func (r *errMessageRepo) Create(context.Context, *model.TicketMessage) error {
	return errors.New("message insert failed")
}

func TestCreateTicket_FailedMessageWriteSurfacesAndSkipsEvent(t *testing.T) {
	uc, _, _, _, bus, orgID, customerID := newCreateTicketFixture()
	// Swap in a message repo that fails, so the transaction body returns an error.
	uc.messageRepo = &errMessageRepo{fakeMessageRepo: newFakeMessageRepo()}

	_, err := uc.Execute(context.Background(), orgID, dto.CreateTicketRequest{
		Subject:    "Broken export",
		Body:       "Excel export throws an error.",
		CustomerID: customerID,
	})
	if err == nil {
		t.Fatal("expected an error when the first message cannot be written")
	}
	if bus.published != 0 {
		t.Errorf("no ticket.created must be published when the transaction failed, got %d", bus.published)
	}
}
