package usecase

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"

	iamModel "github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/port"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
)

// In-memory doubles for the triage repositories.
//
// These are hand-written rather than generated mocks because the behaviour under
// test is stateful: AnalyzeTicketUseCase writes an analysis and then may write
// the ticket, and the assertions are about what ended up stored. Expectation
// based mocks would describe the calls instead of the outcome.

func notFoundErr(what string) error {
	return domainErr.New(domainErr.ErrNotFound, what+" not found", nil)
}

// ---------------------------------------------------------------- tickets

type fakeTicketRepo struct {
	tickets map[uuid.UUID]*model.Ticket
	// updates counts Update calls, so a test can assert the use case left an
	// unchanged ticket alone instead of rewriting it.
	updates   int
	updateErr error
	// lastFilter is what ListByOrg was called with, so a test can assert on the
	// scoping the use case applied rather than only on the rows it got back.
	lastFilter repository.TicketFilter
}

var _ repository.TicketRepository = (*fakeTicketRepo)(nil)

func newFakeTicketRepo(tickets ...*model.Ticket) *fakeTicketRepo {
	r := &fakeTicketRepo{tickets: make(map[uuid.UUID]*model.Ticket, len(tickets))}
	for _, t := range tickets {
		r.tickets[t.ID] = t
	}
	return r
}

func (r *fakeTicketRepo) Create(_ context.Context, ticket *model.Ticket) error {
	if ticket.ID == uuid.Nil {
		ticket.ID = uuid.New()
	}
	r.tickets[ticket.ID] = ticket
	return nil
}

func (r *fakeTicketRepo) GetByID(_ context.Context, orgID, id uuid.UUID) (*model.Ticket, error) {
	t, ok := r.tickets[id]
	if !ok || t.OrganizationID != orgID {
		return nil, notFoundErr("ticket")
	}
	// Return a copy so a use case mutating the ticket does not retroactively
	// change what the store held before Update was called.
	clone := *t
	return &clone, nil
}

func (r *fakeTicketRepo) Update(_ context.Context, ticket *model.Ticket) error {
	if r.updateErr != nil {
		return r.updateErr
	}
	r.updates++
	clone := *ticket
	r.tickets[ticket.ID] = &clone
	return nil
}

func (r *fakeTicketRepo) Delete(_ context.Context, _, id uuid.UUID) error {
	delete(r.tickets, id)
	return nil
}

// ListByOrg honours TicketFilter.CustomerID and records the filter it was
// given. The own-scope tests are precisely about whether the use case set that
// field, so a fake that ignored it would pass while the real query leaked every
// customer's tickets.
func (r *fakeTicketRepo) ListByOrg(_ context.Context, orgID uuid.UUID, filter repository.TicketFilter, _, _ int) ([]*model.Ticket, int, error) {
	r.lastFilter = filter

	var out []*model.Ticket
	for _, t := range r.tickets {
		if t.OrganizationID != orgID {
			continue
		}
		if filter.CustomerID != nil && t.CustomerID != *filter.CustomerID {
			continue
		}
		out = append(out, t)
	}
	return out, len(out), nil
}

func (r *fakeTicketRepo) CountByDepartment(_ context.Context, orgID uuid.UUID) (map[uuid.UUID]int, error) {
	counts := map[uuid.UUID]int{}
	for _, t := range r.tickets {
		if t.OrganizationID == orgID {
			counts[t.DepartmentID]++
		}
	}
	return counts, nil
}

func (r *fakeTicketRepo) CountByCustomer(_ context.Context, orgID, customerID uuid.UUID) (int, error) {
	count := 0
	for _, t := range r.tickets {
		if t.OrganizationID == orgID && t.CustomerID == customerID {
			count++
		}
	}
	return count, nil
}

func (r *fakeTicketRepo) ReassignDepartment(_ context.Context, orgID, from, to uuid.UUID) (int64, error) {
	var moved int64
	for _, t := range r.tickets {
		if t.OrganizationID == orgID && t.DepartmentID == from {
			t.DepartmentID = to
			moved++
		}
	}
	return moved, nil
}

// ---------------------------------------------------------------- messages

type fakeMessageRepo struct {
	byTicket map[uuid.UUID][]*model.TicketMessage
	listErr  error
}

var _ repository.TicketMessageRepository = (*fakeMessageRepo)(nil)

func newFakeMessageRepo() *fakeMessageRepo {
	return &fakeMessageRepo{byTicket: map[uuid.UUID][]*model.TicketMessage{}}
}

func (r *fakeMessageRepo) add(ticketID uuid.UUID, orgID uuid.UUID, body string) {
	r.byTicket[ticketID] = append(r.byTicket[ticketID], &model.TicketMessage{
		ID:             uuid.New(),
		OrganizationID: orgID,
		TicketID:       ticketID,
		AuthorType:     model.AuthorTypeCustomer,
		Body:           body,
	})
}

func (r *fakeMessageRepo) Create(_ context.Context, m *model.TicketMessage) error {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	r.byTicket[m.TicketID] = append(r.byTicket[m.TicketID], m)
	return nil
}

func (r *fakeMessageRepo) GetByID(_ context.Context, _, id uuid.UUID) (*model.TicketMessage, error) {
	for _, msgs := range r.byTicket {
		for _, m := range msgs {
			if m.ID == id {
				return m, nil
			}
		}
	}
	return nil, notFoundErr("message")
}

func (r *fakeMessageRepo) ListByTicket(_ context.Context, _, ticketID uuid.UUID, includeInternal bool) ([]*model.TicketMessage, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	var out []*model.TicketMessage
	for _, m := range r.byTicket[ticketID] {
		if !includeInternal && m.IsInternal {
			continue
		}
		out = append(out, m)
	}
	return out, nil
}

func (r *fakeMessageRepo) CountByTickets(_ context.Context, _ uuid.UUID, ticketIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	counts := map[uuid.UUID]int{}
	for _, id := range ticketIDs {
		counts[id] = len(r.byTicket[id])
	}
	return counts, nil
}

// PreviewByTickets mirrors the real query: first non-internal message, cut to
// maxRunes. An internal note must never become a ticket's public preview.
func (r *fakeMessageRepo) PreviewByTickets(_ context.Context, _ uuid.UUID, ticketIDs []uuid.UUID, maxRunes int) (map[uuid.UUID]string, error) {
	previews := map[uuid.UUID]string{}
	for _, id := range ticketIDs {
		for _, m := range r.byTicket[id] {
			if m.IsInternal {
				continue
			}
			body := m.Body
			if len(body) > maxRunes {
				body = body[:maxRunes]
			}
			previews[id] = body
			break
		}
	}
	return previews, nil
}

// FirstResponseByTickets mirrors the real query: earliest message that is
// neither the customer's own nor an internal note.
func (r *fakeMessageRepo) FirstResponseByTickets(_ context.Context, _ uuid.UUID, ticketIDs []uuid.UUID) (map[uuid.UUID]time.Time, error) {
	responses := map[uuid.UUID]time.Time{}
	for _, id := range ticketIDs {
		for _, m := range r.byTicket[id] {
			if m.IsInternal || m.AuthorType == model.AuthorTypeCustomer {
				continue
			}
			if at, seen := responses[id]; !seen || m.CreatedAt.Before(at) {
				responses[id] = m.CreatedAt
			}
		}
	}
	return responses, nil
}

// ---------------------------------------------------------------- departments

type fakeDepartmentRepo struct {
	departments []*model.Department
	// byCategoryErr is returned instead of a lookup result, used to prove that a
	// failing query is not mistaken for an absent mapping.
	byCategoryErr error
}

var _ repository.DepartmentRepository = (*fakeDepartmentRepo)(nil)

func (r *fakeDepartmentRepo) Create(_ context.Context, d *model.Department) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}
	r.departments = append(r.departments, d)
	return nil
}

func (r *fakeDepartmentRepo) GetByID(_ context.Context, orgID, id uuid.UUID) (*model.Department, error) {
	for _, d := range r.departments {
		if d.ID == id && d.OrganizationID == orgID {
			return d, nil
		}
	}
	return nil, notFoundErr("department")
}

func (r *fakeDepartmentRepo) GetByName(_ context.Context, orgID uuid.UUID, name string) (*model.Department, error) {
	for _, d := range r.departments {
		if d.Name == name && d.OrganizationID == orgID {
			return d, nil
		}
	}
	return nil, notFoundErr("department")
}

func (r *fakeDepartmentRepo) GetByCategory(_ context.Context, orgID uuid.UUID, category model.Category) (*model.Department, error) {
	if r.byCategoryErr != nil {
		return nil, r.byCategoryErr
	}
	for _, d := range r.departments {
		if d.OrganizationID == orgID && d.Category != nil && *d.Category == category {
			return d, nil
		}
	}
	return nil, notFoundErr("department")
}

func (r *fakeDepartmentRepo) GetDefault(_ context.Context, orgID uuid.UUID) (*model.Department, error) {
	for _, d := range r.departments {
		if d.OrganizationID == orgID && d.IsDefault {
			return d, nil
		}
	}
	return nil, notFoundErr("default department")
}

func (r *fakeDepartmentRepo) ListByOrg(_ context.Context, orgID uuid.UUID) ([]*model.Department, error) {
	var out []*model.Department
	for _, d := range r.departments {
		if d.OrganizationID == orgID {
			out = append(out, d)
		}
	}
	return out, nil
}

func (r *fakeDepartmentRepo) Update(_ context.Context, d *model.Department) error {
	for i, existing := range r.departments {
		if existing.ID == d.ID {
			r.departments[i] = d
			return nil
		}
	}
	return notFoundErr("department")
}

func (r *fakeDepartmentRepo) Delete(_ context.Context, orgID, id uuid.UUID) error {
	for i, d := range r.departments {
		if d.ID == id && d.OrganizationID == orgID {
			r.departments = append(r.departments[:i], r.departments[i+1:]...)
			return nil
		}
	}
	return notFoundErr("department")
}

// ---------------------------------------------------------------- staff

// fakeStaffRepo is an in-memory support roster.
//
// Modelled on what the SQL actually does rather than on the interface alone:
// members are the people the roster query would return (organization members
// who are not customers), and assignments are the staff_departments rows. A
// user absent from members does not exist as far as this repository is
// concerned, which is how the real query treats both another tenant's accounts
// and this tenant's portal customers — and that equivalence is the point of
// several tests below.
type fakeStaffRepo struct {
	members []*model.StaffMember
	// setCalls counts SetDepartment, so a test can assert a use case refused
	// before writing rather than writing and then failing.
	setCalls int
	setErr   error
}

var _ repository.StaffRepository = (*fakeStaffRepo)(nil)

func (r *fakeStaffRepo) find(orgID, userID uuid.UUID) *model.StaffMember {
	for _, m := range r.members {
		if m.UserID == userID && m.OrganizationID == orgID {
			return m
		}
	}
	return nil
}

func (r *fakeStaffRepo) ListByOrg(
	_ context.Context,
	orgID uuid.UUID,
	filter repository.StaffFilter,
	offset, limit int,
) ([]*model.StaffMember, int, error) {
	var matched []*model.StaffMember
	for _, m := range r.members {
		if m.OrganizationID != orgID {
			continue
		}
		if filter.Unassigned && m.DepartmentID != nil {
			continue
		}
		if !filter.Unassigned && filter.DepartmentID != nil {
			if m.DepartmentID == nil || *m.DepartmentID != *filter.DepartmentID {
				continue
			}
		}
		if filter.Query != "" && !strings.Contains(
			strings.ToLower(m.FullName()+" "+m.Email), strings.ToLower(filter.Query),
		) {
			continue
		}
		matched = append(matched, m)
	}

	total := len(matched)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if limit <= 0 || end > total {
		end = total
	}
	return matched[offset:end], total, nil
}

func (r *fakeStaffRepo) GetByUser(_ context.Context, orgID, userID uuid.UUID) (*model.StaffMember, error) {
	if m := r.find(orgID, userID); m != nil {
		return m, nil
	}
	return nil, notFoundErr("staff member")
}

func (r *fakeStaffRepo) SetDepartment(_ context.Context, orgID, userID uuid.UUID, departmentID *uuid.UUID) error {
	r.setCalls++
	if r.setErr != nil {
		return r.setErr
	}
	m := r.find(orgID, userID)
	if m == nil {
		return notFoundErr("staff member")
	}
	m.DepartmentID = departmentID
	m.DepartmentName = ""
	return nil
}

func (r *fakeStaffRepo) CountByDepartment(_ context.Context, orgID uuid.UUID) (map[uuid.UUID]int, error) {
	counts := make(map[uuid.UUID]int)
	for _, m := range r.members {
		if m.OrganizationID == orgID && m.DepartmentID != nil {
			counts[*m.DepartmentID]++
		}
	}
	return counts, nil
}

// ---------------------------------------------------------------- customers

type fakeCustomerRepo struct {
	customers []*model.Customer
}

var _ repository.CustomerRepository = (*fakeCustomerRepo)(nil)

func (r *fakeCustomerRepo) Create(_ context.Context, c *model.Customer) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	r.customers = append(r.customers, c)
	return nil
}

func (r *fakeCustomerRepo) GetByID(_ context.Context, orgID, id uuid.UUID) (*model.Customer, error) {
	for _, c := range r.customers {
		if c.ID == id && c.OrganizationID == orgID {
			return c, nil
		}
	}
	return nil, notFoundErr("customer")
}

func (r *fakeCustomerRepo) GetByEmail(_ context.Context, orgID uuid.UUID, email string) (*model.Customer, error) {
	for _, c := range r.customers {
		if c.Email == email && c.OrganizationID == orgID {
			return c, nil
		}
	}
	return nil, notFoundErr("customer")
}

func (r *fakeCustomerRepo) GetByUserID(_ context.Context, orgID, userID uuid.UUID) (*model.Customer, error) {
	for _, c := range r.customers {
		if c.OrganizationID == orgID && c.UserID != nil && *c.UserID == userID {
			return c, nil
		}
	}
	return nil, notFoundErr("customer")
}

func (r *fakeCustomerRepo) ListByOrg(_ context.Context, orgID uuid.UUID, _ string, _, _ int) ([]*model.Customer, int, error) {
	var out []*model.Customer
	for _, c := range r.customers {
		if c.OrganizationID == orgID {
			out = append(out, c)
		}
	}
	return out, len(out), nil
}

func (r *fakeCustomerRepo) ListByIDs(_ context.Context, orgID uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]*model.Customer, error) {
	wanted := make(map[uuid.UUID]struct{}, len(ids))
	for _, id := range ids {
		wanted[id] = struct{}{}
	}
	out := map[uuid.UUID]*model.Customer{}
	for _, c := range r.customers {
		if c.OrganizationID != orgID {
			continue
		}
		if _, ok := wanted[c.ID]; ok {
			out[c.ID] = c
		}
	}
	return out, nil
}

func (r *fakeCustomerRepo) Update(_ context.Context, c *model.Customer) error {
	for i, existing := range r.customers {
		if existing.ID == c.ID {
			r.customers[i] = c
			return nil
		}
	}
	return notFoundErr("customer")
}

func (r *fakeCustomerRepo) Delete(_ context.Context, orgID, id uuid.UUID) error {
	for i, c := range r.customers {
		if c.ID == id && c.OrganizationID == orgID {
			r.customers = append(r.customers[:i], r.customers[i+1:]...)
			return nil
		}
	}
	return notFoundErr("customer")
}

// ---------------------------------------------------------------- analyses

type fakeAnalysisRepo struct {
	// analyses is append-only, mirroring the real table, newest last.
	analyses  []*model.AIAnalysis
	createErr error
}

var _ repository.AIAnalysisRepository = (*fakeAnalysisRepo)(nil)

func (r *fakeAnalysisRepo) Create(_ context.Context, a *model.AIAnalysis) error {
	if r.createErr != nil {
		return r.createErr
	}
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	clone := *a
	r.analyses = append(r.analyses, &clone)
	return nil
}

func (r *fakeAnalysisRepo) ListByTicket(_ context.Context, orgID, ticketID uuid.UUID) ([]*model.AIAnalysis, error) {
	var out []*model.AIAnalysis
	for i := len(r.analyses) - 1; i >= 0; i-- {
		a := r.analyses[i]
		if a.OrganizationID == orgID && a.TicketID == ticketID {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *fakeAnalysisRepo) GetLatestByTicket(_ context.Context, orgID, ticketID uuid.UUID) (*model.AIAnalysis, error) {
	for i := len(r.analyses) - 1; i >= 0; i-- {
		a := r.analyses[i]
		if a.OrganizationID == orgID && a.TicketID == ticketID {
			return a, nil
		}
	}
	return nil, notFoundErr("analysis")
}

func (r *fakeAnalysisRepo) LatestForTickets(_ context.Context, orgID uuid.UUID, ticketIDs []uuid.UUID) (map[uuid.UUID]*model.AIAnalysis, error) {
	wanted := make(map[uuid.UUID]struct{}, len(ticketIDs))
	for _, id := range ticketIDs {
		wanted[id] = struct{}{}
	}
	out := map[uuid.UUID]*model.AIAnalysis{}
	for _, a := range r.analyses {
		if a.OrganizationID != orgID {
			continue
		}
		if _, ok := wanted[a.TicketID]; ok {
			out[a.TicketID] = a
		}
	}
	return out, nil
}

// latestFor returns the newest stored analysis for a ticket, for assertions.
func (r *fakeAnalysisRepo) latestFor(ticketID uuid.UUID) *model.AIAnalysis {
	for i := len(r.analyses) - 1; i >= 0; i-- {
		if r.analyses[i].TicketID == ticketID {
			return r.analyses[i]
		}
	}
	return nil
}

// ---------------------------------------------------------------- users

type fakeUserRepo struct {
	users []*iamModel.User
}

func (r *fakeUserRepo) Create(_ context.Context, u *iamModel.User) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	r.users = append(r.users, u)
	return nil
}

func (r *fakeUserRepo) GetByID(_ context.Context, id uuid.UUID) (*iamModel.User, error) {
	for _, u := range r.users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, notFoundErr("user")
}

func (r *fakeUserRepo) GetByEmail(_ context.Context, email string) (*iamModel.User, error) {
	for _, u := range r.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, notFoundErr("user")
}

func (r *fakeUserRepo) Update(_ context.Context, u *iamModel.User) error {
	for i, existing := range r.users {
		if existing.ID == u.ID {
			r.users[i] = u
			return nil
		}
	}
	return notFoundErr("user")
}

func (r *fakeUserRepo) UpdatePassword(_ context.Context, userID uuid.UUID, hash string) error {
	for _, u := range r.users {
		if u.ID == userID {
			u.PasswordHash = hash
			return nil
		}
	}
	return notFoundErr("user")
}

func (r *fakeUserRepo) Delete(_ context.Context, id uuid.UUID) error {
	for i, u := range r.users {
		if u.ID == id {
			r.users = append(r.users[:i], r.users[i+1:]...)
			return nil
		}
	}
	return notFoundErr("user")
}

func (r *fakeUserRepo) List(_ context.Context, _, _ int) ([]*iamModel.User, int, error) {
	return r.users, len(r.users), nil
}

func (r *fakeUserRepo) ListByOrganization(_ context.Context, _ uuid.UUID, _, _ int) ([]*iamModel.User, int, error) {
	return r.users, len(r.users), nil
}

// ---------------------------------------------------------------- classifier

// stubbedClassifier returns a canned result, so a test states the model output
// it wants to exercise rather than reverse-engineering keyword tables.
type stubbedClassifier struct {
	result port.ClassifyResult
	err    error
	calls  int
	// lastInput records what the use case actually sent to the model.
	lastInput port.ClassifyInput
}

var _ port.Classifier = (*stubbedClassifier)(nil)

func (c *stubbedClassifier) Classify(_ context.Context, in port.ClassifyInput) (port.ClassifyResult, error) {
	c.calls++
	c.lastInput = in
	if c.err != nil {
		return port.ClassifyResult{}, c.err
	}
	return c.result, nil
}
