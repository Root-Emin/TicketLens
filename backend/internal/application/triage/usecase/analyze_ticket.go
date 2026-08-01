package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	iamRepo "github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/port"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// DefaultReviewThreshold is used when none is configured.
const DefaultReviewThreshold = 0.60

// AnalyzeTicketUseCase runs the classifier over a ticket and records the result.
type AnalyzeTicketUseCase struct {
	ticketRepo      repository.TicketRepository
	messageRepo     repository.TicketMessageRepository
	departmentRepo  repository.DepartmentRepository
	analysisRepo    repository.AIAnalysisRepository
	classifier      port.Classifier
	reviewThreshold float64
	assembler       *ticketAssembler
}

// NewAnalyzeTicketUseCase creates a new AnalyzeTicketUseCase.
func NewAnalyzeTicketUseCase(
	ticketRepo repository.TicketRepository,
	messageRepo repository.TicketMessageRepository,
	departmentRepo repository.DepartmentRepository,
	customerRepo repository.CustomerRepository,
	analysisRepo repository.AIAnalysisRepository,
	userRepo iamRepo.UserRepository,
	classifier port.Classifier,
	reviewThreshold float64,
) *AnalyzeTicketUseCase {
	if reviewThreshold <= 0 {
		reviewThreshold = DefaultReviewThreshold
	}
	return &AnalyzeTicketUseCase{
		ticketRepo:      ticketRepo,
		messageRepo:     messageRepo,
		departmentRepo:  departmentRepo,
		analysisRepo:    analysisRepo,
		classifier:      classifier,
		reviewThreshold: reviewThreshold,
		assembler:       newTicketAssembler(departmentRepo, customerRepo, messageRepo, analysisRepo, userRepo),
	}
}

// HasAnalysis reports whether a ticket has already been classified. The event
// consumer uses it to stay idempotent under at-least-once delivery; the manual
// re-run endpoint deliberately skips this check.
func (uc *AnalyzeTicketUseCase) HasAnalysis(ctx context.Context, orgID, ticketID uuid.UUID) bool {
	_, err := uc.analysisRepo.GetLatestByTicket(ctx, orgID, ticketID)
	return err == nil
}

// Execute classifies a ticket, stores the analysis, and applies the prediction
// to any field a human has not already corrected.
func (uc *AnalyzeTicketUseCase) Execute(ctx context.Context, orgID, ticketID uuid.UUID) (*dto.TicketDetail, error) {
	ticket, err := uc.ticketRepo.GetByID(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}

	body, err := uc.firstMessageBody(ctx, orgID, ticketID)
	if err != nil {
		return nil, err
	}

	result, err := uc.classifier.Classify(ctx, port.ClassifyInput{
		Subject: ticket.Subject,
		Body:    body,
	})
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "classification failed", err)
	}

	priority := model.TicketPriority(result.Priority)
	if !model.ValidTicketPriority(priority) {
		return nil, domainErr.New(domainErr.ErrInternal,
			"classifier returned an unknown priority: "+result.Priority, nil)
	}

	category := model.Category(result.Category)
	if !model.ValidCategory(category) {
		return nil, domainErr.New(domainErr.ErrInternal,
			"classifier returned an unknown category: "+result.Category, nil)
	}

	departmentID, mapped, err := uc.mapCategory(ctx, orgID, category)
	if err != nil {
		return nil, err
	}

	// Uncertainty on either axis, or a category this organization has no
	// department for, sends the ticket to a human.
	needsReview := result.PriorityConfidence < uc.reviewThreshold ||
		result.CategoryConfidence < uc.reviewThreshold ||
		!mapped

	analysis := &model.AIAnalysis{
		OrganizationID:        orgID,
		TicketID:              ticketID,
		PredictedPriority:     priority,
		PriorityConfidence:    result.PriorityConfidence,
		PredictedCategory:     &category,
		PredictedDepartmentID: &departmentID,
		DepartmentConfidence:  result.CategoryConfidence,
		NeedsHumanReview:      needsReview,
		MappingFallback:       !mapped,
		ModelName:             result.ModelName,
		ModelVersion:          result.ModelVersion,
		RawResponse:           result.Raw,
	}
	if err := uc.analysisRepo.Create(ctx, analysis); err != nil {
		return nil, err
	}

	// Apply the prediction to fields nobody has corrected. A field whose
	// *_overridden flag is true carries a human decision and is left alone —
	// a re-run must never overwrite it. Without this the accept rate would be
	// meaningless, because the prediction would never reach the ticket.
	changed := false
	if !ticket.PriorityOverridden && ticket.Priority != priority {
		ticket.Priority = priority
		changed = true
	}
	if !ticket.DepartmentOverridden && ticket.DepartmentID != departmentID {
		ticket.DepartmentID = departmentID
		changed = true
	}
	if changed {
		if err := uc.ticketRepo.Update(ctx, ticket); err != nil {
			return nil, err
		}
	}

	return uc.assembler.toDetail(ctx, orgID, ticket, true)
}

// mapCategory resolves a classifier label onto one of the organization's
// departments. The bool reports whether a real match was found; false means the
// ticket fell back to the default department.
func (uc *AnalyzeTicketUseCase) mapCategory(ctx context.Context, orgID uuid.UUID, category model.Category) (uuid.UUID, bool, error) {
	department, err := uc.departmentRepo.GetByCategory(ctx, orgID, category)
	if err == nil {
		return department.ID, true, nil
	}

	fallback, fallbackErr := uc.departmentRepo.GetDefault(ctx, orgID)
	if fallbackErr != nil {
		return uuid.Nil, false, fallbackErr
	}
	return fallback.ID, false, nil
}

// firstMessageBody returns the ticket's description, which is its first message.
func (uc *AnalyzeTicketUseCase) firstMessageBody(ctx context.Context, orgID, ticketID uuid.UUID) (string, error) {
	messages, err := uc.messageRepo.ListByTicket(ctx, orgID, ticketID, true)
	if err != nil {
		return "", err
	}
	if len(messages) == 0 {
		return "", nil
	}
	return messages[0].Body, nil
}
