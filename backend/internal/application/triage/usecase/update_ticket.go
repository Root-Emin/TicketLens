package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	auditModel "github.com/Root-Emin/TicketLens/internal/domain/audit/model"
	auditRepo "github.com/Root-Emin/TicketLens/internal/domain/audit/repository"
	iamRepo "github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	triageEvent "github.com/Root-Emin/TicketLens/internal/domain/triage/event"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/Root-Emin/TicketLens/internal/shared/events"
	"github.com/google/uuid"
)

// checkCustomerPatch narrows PATCH /tickets/{id} to the one thing a customer is
// allowed to do with it: reopen their own resolved request.
//
// The endpoint exists for agents, who use it to correct the classifier's
// priority and routing. Those are exactly the fields a customer must not touch
// — a self-service priority field makes the whole triage meaningless, because
// everything arrives urgent. So rather than granting customers ticket:update,
// they get ticket:reopen_own, and this function enforces what that name
// promises: one transition, resolved or closed back to open, and nothing else.
//
// Staff callers pass straight through; their limits are the ones already
// encoded below.
func checkCustomerPatch(ticket *model.Ticket, req dto.UpdateTicketRequest, scope Scope) error {
	if !scope.IsCustomer() {
		return nil
	}

	if req.Priority != nil || req.DepartmentID != nil {
		return domainErr.New(domainErr.ErrForbidden,
			"priority and department are set by triage, not by the requester", nil)
	}
	if req.Status == nil {
		return domainErr.New(domainErr.ErrForbidden,
			"you can only reopen your own resolved request", nil)
	}
	if model.TicketStatus(*req.Status) != model.TicketStatusOpen {
		return domainErr.New(domainErr.ErrForbidden,
			"a request can only be moved back to open", nil)
	}
	if !ticket.IsResolved() {
		return domainErr.New(domainErr.ErrConflict,
			"this request is not resolved, so there is nothing to reopen", nil)
	}
	return nil
}

// UpdateTicketUseCase applies a human's corrections to a ticket.
type UpdateTicketUseCase struct {
	ticketRepo     repository.TicketRepository
	analysisRepo   repository.AIAnalysisRepository
	departmentRepo repository.DepartmentRepository
	auditRepo      auditRepo.AuditRepository
	eventBus       events.EventBus
	assembler      *ticketAssembler
}

// NewUpdateTicketUseCase creates a new UpdateTicketUseCase.
func NewUpdateTicketUseCase(
	ticketRepo repository.TicketRepository,
	analysisRepo repository.AIAnalysisRepository,
	departmentRepo repository.DepartmentRepository,
	customerRepo repository.CustomerRepository,
	messageRepo repository.TicketMessageRepository,
	userRepo iamRepo.UserRepository,
	audit auditRepo.AuditRepository,
	eventBus events.EventBus,
) *UpdateTicketUseCase {
	return &UpdateTicketUseCase{
		ticketRepo:     ticketRepo,
		analysisRepo:   analysisRepo,
		departmentRepo: departmentRepo,
		auditRepo:      audit,
		eventBus:       eventBus,
		assembler:      newTicketAssembler(departmentRepo, customerRepo, messageRepo, analysisRepo, userRepo),
	}
}

// Execute patches priority, department and/or status.
//
// The override flags are the point of this use case. They compare the value a
// human is setting against the LATEST analysis' prediction:
//
//   - different from the prediction  -> the flag becomes true (human disagreed)
//   - equal to the prediction        -> the flag becomes false (human agreed)
//
// Setting the flag back to false matters: if someone corrects a value and then
// restores the model's answer, the model was in fact right, and the accept-rate
// metric must say so. The flags therefore describe the current state, not the
// edit history.
//
// A ticket with no analysis yet, or an analysis that predicted no department,
// leaves the corresponding flag untouched: there is no prediction to override.
func (uc *UpdateTicketUseCase) Execute(ctx context.Context, orgID, ticketID, actorID uuid.UUID, req dto.UpdateTicketRequest, scope Scope) (*dto.TicketDetail, error) {
	ticket, err := uc.ticketRepo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}
	if err := scope.Guard(ticket); err != nil {
		return nil, err
	}
	if err := checkCustomerPatch(ticket, req, scope); err != nil {
		return nil, err
	}

	latest, err := uc.analysisRepo.GetLatestByTicket(ctx, orgID, ticketID)
	if err != nil {
		if !errors.Is(err, domainErr.ErrNotFound) {
			return nil, err
		}
		latest = nil // classification still pending
	}

	if req.Priority != nil {
		priority := model.TicketPriority(*req.Priority)
		if !model.ValidTicketPriority(priority) {
			return nil, domainErr.New(domainErr.ErrValidation,
				fmt.Sprintf("invalid priority: %s", *req.Priority), nil)
		}
		ticket.Priority = priority
		if latest != nil {
			ticket.PriorityOverridden = !latest.PriorityAccepted(priority)
		}
	}

	if req.DepartmentID != nil {
		// Org-scoped lookup: another tenant's department reads as not found.
		department, err := uc.departmentRepo.GetByID(ctx, orgID, *req.DepartmentID)
		if err != nil {
			return nil, err
		}
		ticket.DepartmentID = department.ID
		if latest != nil && latest.PredictedDepartmentID != nil {
			ticket.DepartmentOverridden = !latest.DepartmentAccepted(department.ID)
		}
	}

	if req.Status != nil {
		status := model.TicketStatus(*req.Status)
		if !model.ValidTicketStatus(status) {
			return nil, domainErr.New(domainErr.ErrValidation,
				fmt.Sprintf("invalid status: %s", *req.Status), nil)
		}
		if status == model.TicketStatusResolved {
			// Stamp on the way in, and keep the original timestamp if the ticket
			// was already resolved.
			if ticket.ResolvedAt == nil {
				now := time.Now().UTC()
				ticket.ResolvedAt = &now
			}
		} else {
			ticket.ResolvedAt = nil
		}
		ticket.Status = status
	}

	if err := uc.ticketRepo.Update(ctx, ticket); err != nil {
		return nil, err
	}

	uc.writeAudit(ctx, orgID, ticketID, actorID)

	_ = uc.eventBus.Publish(ctx, events.TopicTriage, triageEvent.TicketUpdated{
		TicketID:       ticket.ID,
		OrganizationID: orgID,
		Timestamp:      time.Now().UTC(),
	})

	return uc.assembler.toDetail(ctx, orgID, ticket, true)
}

// writeAudit records the change on a best-effort basis: an audit failure must
// not undo an update the caller already succeeded in making.
func (uc *UpdateTicketUseCase) writeAudit(ctx context.Context, orgID, ticketID, actorID uuid.UUID) {
	if uc.auditRepo == nil {
		return
	}
	var userID *uuid.UUID
	if actorID != uuid.Nil {
		userID = &actorID
	}
	_ = uc.auditRepo.Create(ctx, &auditModel.AuditLog{
		OrganizationID: orgID,
		UserID:         userID,
		Action:         "ticket.updated",
		ResourceType:   "ticket",
		ResourceID:     ticketID.String(),
	})
}
