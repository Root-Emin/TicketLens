package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	iamRepo "github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	"github.com/google/uuid"
)

// ticketAssembler turns tickets into their API representation.
//
// It exists so the list and detail use cases share one enrichment path. A whole
// page is resolved in a handful of queries: departments and customers come back
// in bulk, message counts and latest analyses have dedicated batch methods.
// Assignees are looked up individually but de-duplicated first, since a queue
// page usually has very few distinct assignees.
type ticketAssembler struct {
	departmentRepo repository.DepartmentRepository
	customerRepo   repository.CustomerRepository
	messageRepo    repository.TicketMessageRepository
	analysisRepo   repository.AIAnalysisRepository
	userRepo       iamRepo.UserRepository
}

func newTicketAssembler(
	departmentRepo repository.DepartmentRepository,
	customerRepo repository.CustomerRepository,
	messageRepo repository.TicketMessageRepository,
	analysisRepo repository.AIAnalysisRepository,
	userRepo iamRepo.UserRepository,
) *ticketAssembler {
	return &ticketAssembler{
		departmentRepo: departmentRepo,
		customerRepo:   customerRepo,
		messageRepo:    messageRepo,
		analysisRepo:   analysisRepo,
		userRepo:       userRepo,
	}
}

// snippetLength is how much of the opening message a list item carries. Long
// enough for two lines in a card, short enough that a page of 25 tickets does
// not ship 100KB of description nobody will read.
const snippetLength = 240

// toListItems enriches a page of tickets.
func (a *ticketAssembler) toListItems(ctx context.Context, orgID uuid.UUID, tickets []*model.Ticket) ([]dto.TicketListItem, error) {
	if len(tickets) == 0 {
		return []dto.TicketListItem{}, nil
	}

	ticketIDs := make([]uuid.UUID, 0, len(tickets))
	customerIDs := make([]uuid.UUID, 0, len(tickets))
	seenCustomer := make(map[uuid.UUID]struct{})
	for _, t := range tickets {
		ticketIDs = append(ticketIDs, t.ID)
		if _, dup := seenCustomer[t.CustomerID]; !dup {
			seenCustomer[t.CustomerID] = struct{}{}
			customerIDs = append(customerIDs, t.CustomerID)
		}
	}

	departments, err := a.departmentRepo.ListByOrg(ctx, orgID)
	if err != nil {
		return nil, err
	}
	departmentByID := make(map[uuid.UUID]*model.Department, len(departments))
	for _, d := range departments {
		departmentByID[d.ID] = d
	}

	customerByID, err := a.customerRepo.ListByIDs(ctx, orgID, customerIDs)
	if err != nil {
		return nil, err
	}

	messageCounts, err := a.messageRepo.CountByTickets(ctx, orgID, ticketIDs)
	if err != nil {
		return nil, err
	}

	latestAnalyses, err := a.analysisRepo.LatestForTickets(ctx, orgID, ticketIDs)
	if err != nil {
		return nil, err
	}

	previews, err := a.messageRepo.PreviewByTickets(ctx, orgID, ticketIDs, snippetLength)
	if err != nil {
		return nil, err
	}

	firstResponses, err := a.messageRepo.FirstResponseByTickets(ctx, orgID, ticketIDs)
	if err != nil {
		return nil, err
	}

	assignees, err := a.resolveAssignees(ctx, tickets)
	if err != nil {
		return nil, err
	}

	items := make([]dto.TicketListItem, 0, len(tickets))
	for _, t := range tickets {
		item := dto.TicketListItem{
			ID:                   t.ID,
			Subject:              t.Subject,
			Status:               string(t.Status),
			Priority:             string(t.Priority),
			PriorityOverridden:   t.PriorityOverridden,
			DepartmentOverridden: t.DepartmentOverridden,
			MessageCount:         messageCounts[t.ID],
			CreatedAt:            t.CreatedAt,
			UpdatedAt:            t.UpdatedAt,
			ResolvedAt:           t.ResolvedAt,
			Snippet:              previews[t.ID],
		}

		// Absent until support answers. Nil rather than the zero time so a
		// client can tell "not answered yet" from "answered at year zero".
		if at, ok := firstResponses[t.ID]; ok {
			respondedAt := at
			item.FirstResponseAt = &respondedAt
		}

		if d, ok := departmentByID[t.DepartmentID]; ok {
			item.Department = dto.DepartmentRef{ID: d.ID, Name: d.Name}
		} else {
			item.Department = dto.DepartmentRef{ID: t.DepartmentID}
		}

		if c, ok := customerByID[t.CustomerID]; ok {
			item.Customer = dto.CustomerRef{ID: c.ID, FullName: c.FullName, Email: c.Email}
		} else {
			item.Customer = dto.CustomerRef{ID: t.CustomerID}
		}

		if t.AssigneeID != nil {
			if ref, ok := assignees[*t.AssigneeID]; ok {
				assignee := ref
				item.Assignee = &assignee
			}
		}

		// Left nil while classification is pending: the UI shows an "analyzing"
		// state instead of a misleading zero confidence.
		if analysis, ok := latestAnalyses[t.ID]; ok {
			item.LatestAnalysis = &dto.LatestAnalysis{
				PredictedCategory:    analysis.PredictedCategory,
				PredictedPriority:    string(analysis.PredictedPriority),
				PriorityConfidence:   analysis.PriorityConfidence,
				DepartmentConfidence: analysis.DepartmentConfidence,
				NeedsHumanReview:     analysis.NeedsHumanReview,
				MappingFallback:      analysis.MappingFallback,
				ModelName:            analysis.ModelName,
			}
		}

		items = append(items, item)
	}

	return items, nil
}

// toDetail enriches one ticket and attaches its thread and analysis history.
// includeInternal is false for customer-facing callers.
func (a *ticketAssembler) toDetail(ctx context.Context, orgID uuid.UUID, ticket *model.Ticket, includeInternal bool) (*dto.TicketDetail, error) {
	items, err := a.toListItems(ctx, orgID, []*model.Ticket{ticket})
	if err != nil {
		return nil, err
	}

	messages, err := a.messageRepo.ListByTicket(ctx, orgID, ticket.ID, includeInternal)
	if err != nil {
		return nil, err
	}

	analyses, err := a.analysisRepo.ListByTicket(ctx, orgID, ticket.ID)
	if err != nil {
		return nil, err
	}

	return &dto.TicketDetail{
		TicketListItem: items[0],
		Messages:       ToMessageInfos(messages),
		Analyses:       ToAnalysisInfos(analyses),
	}, nil
}

func (a *ticketAssembler) resolveAssignees(ctx context.Context, tickets []*model.Ticket) (map[uuid.UUID]dto.UserRef, error) {
	refs := make(map[uuid.UUID]dto.UserRef)
	for _, t := range tickets {
		if t.AssigneeID == nil || *t.AssigneeID == uuid.Nil {
			continue
		}
		if _, done := refs[*t.AssigneeID]; done {
			continue
		}
		user, err := a.userRepo.GetByID(ctx, *t.AssigneeID)
		if err != nil {
			// A stale assignee must not break the whole listing.
			continue
		}
		refs[user.ID] = dto.UserRef{ID: user.ID, FullName: user.FullName(), Email: user.Email}
	}
	return refs, nil
}

// ToMessageInfos maps domain messages onto their API representation.
func ToMessageInfos(messages []*model.TicketMessage) []dto.MessageInfo {
	out := make([]dto.MessageInfo, 0, len(messages))
	for _, m := range messages {
		out = append(out, dto.MessageInfo{
			ID:         m.ID,
			TicketID:   m.TicketID,
			AuthorType: string(m.AuthorType),
			AuthorID:   m.AuthorID,
			Body:       m.Body,
			IsInternal: m.IsInternal,
			CreatedAt:  m.CreatedAt,
		})
	}
	return out
}

// ToAnalysisInfos maps domain analyses onto their API representation.
func ToAnalysisInfos(analyses []*model.AIAnalysis) []dto.AnalysisInfo {
	out := make([]dto.AnalysisInfo, 0, len(analyses))
	for _, a := range analyses {
		out = append(out, dto.AnalysisInfo{
			ID:                    a.ID,
			TicketID:              a.TicketID,
			PredictedPriority:     string(a.PredictedPriority),
			PriorityConfidence:    a.PriorityConfidence,
			PredictedCategory:     a.PredictedCategory,
			PredictedDepartmentID: a.PredictedDepartmentID,
			MappingFallback:       a.MappingFallback,
			DepartmentConfidence:  a.DepartmentConfidence,
			NeedsHumanReview:      a.NeedsHumanReview,
			ModelName:             a.ModelName,
			ModelVersion:          a.ModelVersion,
			RawResponse:           a.RawResponse,
			CreatedAt:             a.CreatedAt,
		})
	}
	return out
}
