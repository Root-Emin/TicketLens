package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/port"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
)

// analyzeFixture is one organization with a default department, a billing
// department, and a single open ticket sitting in the default department.
type analyzeFixture struct {
	orgID       uuid.UUID
	ticketID    uuid.UUID
	defaultDept *model.Department
	billingDept *model.Department

	tickets     *fakeTicketRepo
	messages    *fakeMessageRepo
	departments *fakeDepartmentRepo
	analyses    *fakeAnalysisRepo
	classifier  *stubbedClassifier
}

func newAnalyzeFixture(t *testing.T) *analyzeFixture {
	t.Helper()

	orgID := uuid.New()
	billingCategory := model.CategoryBilling

	defaultDept := &model.Department{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "General",
		IsDefault:      true,
	}
	billingDept := &model.Department{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           "Billing",
		Category:       &billingCategory,
	}

	ticket := &model.Ticket{
		ID:             uuid.New(),
		OrganizationID: orgID,
		CustomerID:     uuid.New(),
		DepartmentID:   defaultDept.ID,
		Subject:        "I was charged twice",
		Status:         model.TicketStatusOpen,
		Priority:       model.TicketPriorityNormal,
	}

	f := &analyzeFixture{
		orgID:       orgID,
		ticketID:    ticket.ID,
		defaultDept: defaultDept,
		billingDept: billingDept,
		tickets:     newFakeTicketRepo(ticket),
		messages:    newFakeMessageRepo(),
		departments: &fakeDepartmentRepo{departments: []*model.Department{defaultDept, billingDept}},
		analyses:    &fakeAnalysisRepo{},
		classifier:  &stubbedClassifier{},
	}
	f.messages.add(ticket.ID, orgID, "My card was billed twice for the same invoice.")

	f.classifier.result = port.ClassifyResult{
		Priority:           string(model.TicketPriorityHigh),
		PriorityConfidence: 0.91,
		Category:           string(model.CategoryBilling),
		CategoryConfidence: 0.88,
		ModelName:          "test-model",
		ModelVersion:       "1.0.0",
	}

	return f
}

// useCase builds the subject under test with the fixture's doubles.
func (f *analyzeFixture) useCase(threshold float64) *AnalyzeTicketUseCase {
	return NewAnalyzeTicketUseCase(
		f.tickets,
		f.messages,
		f.departments,
		&fakeCustomerRepo{},
		f.analyses,
		&fakeUserRepo{},
		f.classifier,
		threshold,
		nil,
	)
}

// storedTicket reads the ticket back out of the repository, which is what the
// next request would see.
func (f *analyzeFixture) storedTicket() *model.Ticket {
	return f.tickets.tickets[f.ticketID]
}

func TestAnalyzeTicket_ConfidentPredictionRoutesTicket(t *testing.T) {
	f := newAnalyzeFixture(t)

	_, err := f.useCase(0.60).Execute(context.Background(), f.orgID, f.ticketID)
	require.NoError(t, err)

	analysis := f.analyses.latestFor(f.ticketID)
	require.NotNil(t, analysis)
	assert.Equal(t, model.TicketPriorityHigh, analysis.PredictedPriority)
	assert.Equal(t, f.billingDept.ID, *analysis.PredictedDepartmentID)
	assert.False(t, analysis.MappingFallback)
	assert.False(t, analysis.NeedsHumanReview, "both scores clear the threshold and the category maps")

	// The prediction must actually reach the ticket, otherwise the accept rate
	// reported by the stats endpoint would be measuring nothing.
	stored := f.storedTicket()
	assert.Equal(t, model.TicketPriorityHigh, stored.Priority)
	assert.Equal(t, f.billingDept.ID, stored.DepartmentID)
}

func TestAnalyzeTicket_ClassifierReceivesSubjectAndFirstMessage(t *testing.T) {
	f := newAnalyzeFixture(t)

	_, err := f.useCase(0.60).Execute(context.Background(), f.orgID, f.ticketID)
	require.NoError(t, err)

	assert.Equal(t, "I was charged twice", f.classifier.lastInput.Subject)
	assert.Equal(t, "My card was billed twice for the same invoice.", f.classifier.lastInput.Body)
}

func TestAnalyzeTicket_LowConfidenceFlagsForReview(t *testing.T) {
	tests := []struct {
		name               string
		priorityConfidence float64
		categoryConfidence float64
	}{
		{"priority below threshold", 0.41, 0.95},
		{"category below threshold", 0.95, 0.39},
		{"both below threshold", 0.10, 0.20},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newAnalyzeFixture(t)
			f.classifier.result.PriorityConfidence = tc.priorityConfidence
			f.classifier.result.CategoryConfidence = tc.categoryConfidence

			_, err := f.useCase(0.60).Execute(context.Background(), f.orgID, f.ticketID)
			require.NoError(t, err)

			analysis := f.analyses.latestFor(f.ticketID)
			require.NotNil(t, analysis)
			assert.True(t, analysis.NeedsHumanReview)
		})
	}
}

func TestAnalyzeTicket_ConfidenceAtThresholdIsAccepted(t *testing.T) {
	// The comparison is strictly-less-than, so a score exactly on the threshold
	// counts as confident. Pinning it keeps the boundary from drifting silently.
	f := newAnalyzeFixture(t)
	f.classifier.result.PriorityConfidence = 0.60
	f.classifier.result.CategoryConfidence = 0.60

	_, err := f.useCase(0.60).Execute(context.Background(), f.orgID, f.ticketID)
	require.NoError(t, err)

	assert.False(t, f.analyses.latestFor(f.ticketID).NeedsHumanReview)
}

func TestAnalyzeTicket_UnmappedCategoryFallsBackAndFlagsReview(t *testing.T) {
	f := newAnalyzeFixture(t)
	// No department claims this category, but the confidence is high: the
	// prediction is trustworthy, the organization simply has nowhere to route it.
	f.classifier.result.Category = string(model.CategoryHowTo)

	_, err := f.useCase(0.60).Execute(context.Background(), f.orgID, f.ticketID)
	require.NoError(t, err)

	analysis := f.analyses.latestFor(f.ticketID)
	require.NotNil(t, analysis)
	assert.Equal(t, f.defaultDept.ID, *analysis.PredictedDepartmentID)
	assert.True(t, analysis.MappingFallback)
	assert.True(t, analysis.NeedsHumanReview, "a category with no department needs a human")
	assert.Equal(t, model.CategoryHowTo, *analysis.PredictedCategory,
		"the predicted category is recorded even when it routes nowhere")
}

func TestAnalyzeTicket_HumanOverridesSurviveRerun(t *testing.T) {
	f := newAnalyzeFixture(t)

	// A human already set both fields and the override flags record that.
	agentChosenDept := f.defaultDept.ID
	stored := f.storedTicket()
	stored.Priority = model.TicketPriorityUrgent
	stored.PriorityOverridden = true
	stored.DepartmentID = agentChosenDept
	stored.DepartmentOverridden = true

	_, err := f.useCase(0.60).Execute(context.Background(), f.orgID, f.ticketID)
	require.NoError(t, err)

	after := f.storedTicket()
	assert.Equal(t, model.TicketPriorityUrgent, after.Priority,
		"a re-run must not overwrite a priority a human corrected")
	assert.Equal(t, agentChosenDept, after.DepartmentID,
		"a re-run must not overwrite a department a human corrected")
	assert.Zero(t, f.tickets.updates, "nothing changed, so the ticket should not be rewritten")

	// The prediction is still recorded, which is what makes the override
	// measurable as a disagreement with the model.
	analysis := f.analyses.latestFor(f.ticketID)
	require.NotNil(t, analysis)
	assert.Equal(t, model.TicketPriorityHigh, analysis.PredictedPriority)
	assert.Equal(t, f.billingDept.ID, *analysis.PredictedDepartmentID)
}

func TestAnalyzeTicket_PartialOverrideAppliesOnlyTheFreeField(t *testing.T) {
	f := newAnalyzeFixture(t)

	stored := f.storedTicket()
	stored.Priority = model.TicketPriorityLow
	stored.PriorityOverridden = true

	_, err := f.useCase(0.60).Execute(context.Background(), f.orgID, f.ticketID)
	require.NoError(t, err)

	after := f.storedTicket()
	assert.Equal(t, model.TicketPriorityLow, after.Priority, "overridden priority is left alone")
	assert.Equal(t, f.billingDept.ID, after.DepartmentID, "department was never overridden, so it routes")
}

func TestAnalyzeTicket_RejectsUnknownLabels(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*port.ClassifyResult)
	}{
		{"unknown priority", func(r *port.ClassifyResult) { r.Priority = "catastrophic" }},
		{"empty priority", func(r *port.ClassifyResult) { r.Priority = "" }},
		{"unknown category", func(r *port.ClassifyResult) { r.Category = "not_a_category" }},
		{"empty category", func(r *port.ClassifyResult) { r.Category = "" }},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := newAnalyzeFixture(t)
			tc.mutate(&f.classifier.result)

			_, err := f.useCase(0.60).Execute(context.Background(), f.orgID, f.ticketID)

			require.Error(t, err, "an out-of-taxonomy label must not be persisted")
			assert.Nil(t, f.analyses.latestFor(f.ticketID))
			assert.Equal(t, model.TicketPriorityNormal, f.storedTicket().Priority,
				"the ticket is left untouched")
		})
	}
}

func TestAnalyzeTicket_ClassifierFailureLeavesTicketIntact(t *testing.T) {
	f := newAnalyzeFixture(t)
	f.classifier.err = errors.New("model service unreachable")

	_, err := f.useCase(0.60).Execute(context.Background(), f.orgID, f.ticketID)

	require.Error(t, err)
	assert.True(t, errors.Is(err, domainErr.ErrInternal))
	assert.Nil(t, f.analyses.latestFor(f.ticketID))
	assert.Equal(t, model.TicketPriorityNormal, f.storedTicket().Priority)
	assert.Equal(t, f.defaultDept.ID, f.storedTicket().DepartmentID)
}

func TestAnalyzeTicket_DepartmentLookupFailureIsNotTreatedAsUnmapped(t *testing.T) {
	// A failing query previously fell through to the default department, which
	// reported a database problem as a taxonomy gap.
	f := newAnalyzeFixture(t)
	f.departments.byCategoryErr = domainErr.New(domainErr.ErrInternal, "connection reset", nil)

	_, err := f.useCase(0.60).Execute(context.Background(), f.orgID, f.ticketID)

	require.Error(t, err)
	assert.True(t, errors.Is(err, domainErr.ErrInternal))
	assert.Nil(t, f.analyses.latestFor(f.ticketID),
		"no analysis should be recorded when routing could not be determined")
}

func TestAnalyzeTicket_MissingDefaultDepartmentIsAnError(t *testing.T) {
	f := newAnalyzeFixture(t)
	f.departments.departments = []*model.Department{} // no default, no mapping
	f.classifier.result.Category = string(model.CategoryHowTo)

	_, err := f.useCase(0.60).Execute(context.Background(), f.orgID, f.ticketID)

	require.Error(t, err)
	assert.Nil(t, f.analyses.latestFor(f.ticketID))
}

func TestAnalyzeTicket_TicketFromAnotherOrganizationIsNotFound(t *testing.T) {
	f := newAnalyzeFixture(t)

	_, err := f.useCase(0.60).Execute(context.Background(), uuid.New(), f.ticketID)

	require.Error(t, err)
	assert.True(t, errors.Is(err, domainErr.ErrNotFound))
	assert.Zero(t, f.classifier.calls, "the classifier must not see another tenant's ticket")
}

func TestAnalyzeTicket_TicketWithNoMessagesStillClassifies(t *testing.T) {
	// The subject alone is enough input; a ticket created without a description
	// must not block classification.
	f := newAnalyzeFixture(t)
	f.messages.byTicket = map[uuid.UUID][]*model.TicketMessage{}

	_, err := f.useCase(0.60).Execute(context.Background(), f.orgID, f.ticketID)
	require.NoError(t, err)

	assert.Empty(t, f.classifier.lastInput.Body)
	assert.NotNil(t, f.analyses.latestFor(f.ticketID))
}

func TestAnalyzeTicket_ZeroThresholdFallsBackToDefault(t *testing.T) {
	f := newAnalyzeFixture(t)
	f.classifier.result.PriorityConfidence = 0.50 // under DefaultReviewThreshold
	f.classifier.result.CategoryConfidence = 0.95

	uc := f.useCase(0)

	_, err := uc.Execute(context.Background(), f.orgID, f.ticketID)
	require.NoError(t, err)

	assert.Equal(t, DefaultReviewThreshold, uc.reviewThreshold)
	assert.True(t, f.analyses.latestFor(f.ticketID).NeedsHumanReview)
}

func TestAnalyzeTicket_HasAnalysis(t *testing.T) {
	f := newAnalyzeFixture(t)
	uc := f.useCase(0.60)
	ctx := context.Background()

	assert.False(t, uc.HasAnalysis(ctx, f.orgID, f.ticketID),
		"a freshly created ticket has no analysis, so the consumer should proceed")

	_, err := uc.Execute(ctx, f.orgID, f.ticketID)
	require.NoError(t, err)

	assert.True(t, uc.HasAnalysis(ctx, f.orgID, f.ticketID),
		"a redelivered event should be recognised as already handled")
	assert.False(t, uc.HasAnalysis(ctx, uuid.New(), f.ticketID),
		"the check is organization-scoped")
}
