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
	The support roster.

	Two properties are worth asserting here rather than only at the HTTP layer,
	because this is where they are decided:

	  (a) a department assignment cannot cross an organization boundary, in
	      either direction — not by naming somebody else's user, and not by
	      naming somebody else's department for one of your own people. The
	      foreign keys do not catch the second: both rows exist, they just do not
	      belong together.

	  (b) a portal customer is not a colleague. They hold a role in the
	      organization exactly like an agent does, so nothing but the roster
	      query distinguishes them, and putting one on a support team would be a
	      silent nonsense that every scoped read then filters back out.

	The listing tests cover the filter the department detail screen is built on:
	"unassigned" has to mean the absence of a row, which no equality comparison
	can express.
*/

// ── fixtures ────────────────────────────────────────────────────────────────

type staffFixture struct {
	orgID      uuid.UUID
	otherOrgID uuid.UUID

	support  uuid.UUID // an agent on a team
	unplaced uuid.UUID // an agent on no team
	outsider uuid.UUID // a member of otherOrg

	techID  uuid.UUID
	billing uuid.UUID
	// otherDeptID belongs to otherOrg.
	otherDeptID uuid.UUID

	staff       *fakeStaffRepo
	departments *fakeDepartmentRepo
}

func newStaffFixture() *staffFixture {
	f := &staffFixture{
		orgID:       uuid.New(),
		otherOrgID:  uuid.New(),
		support:     uuid.New(),
		unplaced:    uuid.New(),
		outsider:    uuid.New(),
		techID:      uuid.New(),
		billing:     uuid.New(),
		otherDeptID: uuid.New(),
	}

	tech := f.techID
	f.staff = &fakeStaffRepo{
		members: []*model.StaffMember{
			{
				UserID: f.support, OrganizationID: f.orgID,
				Email: "selin@example.com", FirstName: "Selin", LastName: "Aydın",
				Status: "active", DepartmentID: &tech, DepartmentName: "Technical Support",
			},
			{
				UserID: f.unplaced, OrganizationID: f.orgID,
				Email: "can@example.com", FirstName: "Can", LastName: "Öztürk",
				Status: "active",
			},
			{
				UserID: f.outsider, OrganizationID: f.otherOrgID,
				Email: "rival@example.com", FirstName: "Rival", LastName: "Person",
				Status: "active",
			},
		},
	}

	f.departments = &fakeDepartmentRepo{
		departments: []*model.Department{
			{ID: f.techID, OrganizationID: f.orgID, Name: "Technical Support"},
			{ID: f.billing, OrganizationID: f.orgID, Name: "Billing"},
			{ID: f.otherDeptID, OrganizationID: f.otherOrgID, Name: "Someone Else's Team"},
		},
	}

	return f
}

func (f *staffFixture) assignUC() *AssignStaffDepartmentUseCase {
	return NewAssignStaffDepartmentUseCase(f.staff, f.departments)
}

func (f *staffFixture) listUC() *ListStaffUseCase {
	return NewListStaffUseCase(f.staff)
}

func page() dto.PageParams {
	return dto.PageParams{Page: 1, PageSize: 25}
}

// ── assignment ──────────────────────────────────────────────────────────────

func TestAssignStaffDepartment_PlacesOnTeam(t *testing.T) {
	f := newStaffFixture()

	got, err := f.assignUC().Execute(context.Background(), f.orgID, f.unplaced,
		dto.AssignStaffDepartmentRequest{DepartmentID: &f.billing})
	if err != nil {
		t.Fatalf("assign: %v", err)
	}

	if got.Department == nil {
		t.Fatal("expected a department on the response, got none")
	}
	if got.Department.ID != f.billing {
		t.Errorf("department = %s, want %s", got.Department.ID, f.billing)
	}
	if got.FullName != "Can Öztürk" {
		t.Errorf("full_name = %q, want %q", got.FullName, "Can Öztürk")
	}
}

func TestAssignStaffDepartment_NilRemovesFromTeam(t *testing.T) {
	f := newStaffFixture()

	got, err := f.assignUC().Execute(context.Background(), f.orgID, f.support,
		dto.AssignStaffDepartmentRequest{DepartmentID: nil})
	if err != nil {
		t.Fatalf("unassign: %v", err)
	}

	// Null is a value here, not an omission — it is the only way to take
	// somebody off a team, so it must not be read as "leave unchanged".
	if got.Department != nil {
		t.Errorf("department = %+v, want nil after unassigning", got.Department)
	}
}

func TestAssignStaffDepartment_RejectsUserFromAnotherOrganization(t *testing.T) {
	f := newStaffFixture()

	_, err := f.assignUC().Execute(context.Background(), f.orgID, f.outsider,
		dto.AssignStaffDepartmentRequest{DepartmentID: &f.techID})

	if !errors.Is(err, domainErr.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	// Not found rather than forbidden: whether that id names a real person in
	// some other tenant is not something this organization gets to learn.
	if f.staff.setCalls != 0 {
		t.Errorf("SetDepartment called %d times, want 0 — refuse before writing",
			f.staff.setCalls)
	}
}

func TestAssignStaffDepartment_RejectsDepartmentFromAnotherOrganization(t *testing.T) {
	f := newStaffFixture()

	_, err := f.assignUC().Execute(context.Background(), f.orgID, f.unplaced,
		dto.AssignStaffDepartmentRequest{DepartmentID: &f.otherDeptID})

	if !errors.Is(err, domainErr.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	// The foreign key would have accepted this: both rows exist. Only the
	// organization check catches a cross-tenant pairing.
	if f.staff.setCalls != 0 {
		t.Errorf("SetDepartment called %d times, want 0", f.staff.setCalls)
	}
}

// A portal customer is not on the roster, so the use case cannot see them at
// all — the same refusal as a user from another tenant, which is the point:
// GetByUser is the one gate, and it is scoped to the roster rather than to
// users.
func TestAssignStaffDepartment_RejectsSomebodyNotOnTheRoster(t *testing.T) {
	f := newStaffFixture()
	customer := uuid.New() // holds a role in orgID, but is a customer

	_, err := f.assignUC().Execute(context.Background(), f.orgID, customer,
		dto.AssignStaffDepartmentRequest{DepartmentID: &f.techID})

	if !errors.Is(err, domainErr.ErrNotFound) {
		t.Fatalf("err = %v, want ErrNotFound", err)
	}
	if f.staff.setCalls != 0 {
		t.Errorf("SetDepartment called %d times, want 0", f.staff.setCalls)
	}
}

func TestAssignStaffDepartment_IsIdempotent(t *testing.T) {
	f := newStaffFixture()

	// Moving somebody onto the team they are already on is a no-op, not a
	// duplicate-key error: the UI offers the control regardless of current
	// state, and a manager clicking it twice should not see a failure.
	for i := 0; i < 2; i++ {
		got, err := f.assignUC().Execute(context.Background(), f.orgID, f.support,
			dto.AssignStaffDepartmentRequest{DepartmentID: &f.techID})
		if err != nil {
			t.Fatalf("assign #%d: %v", i+1, err)
		}
		if got.Department == nil || got.Department.ID != f.techID {
			t.Fatalf("assign #%d: department = %+v, want %s", i+1, got.Department, f.techID)
		}
	}
}

func TestAssignStaffDepartment_PropagatesWriteFailure(t *testing.T) {
	f := newStaffFixture()
	f.staff.setErr = errors.New("connection reset")

	_, err := f.assignUC().Execute(context.Background(), f.orgID, f.unplaced,
		dto.AssignStaffDepartmentRequest{DepartmentID: &f.billing})
	if err == nil {
		t.Fatal("expected the write failure to surface, got nil")
	}
}

// ── listing ─────────────────────────────────────────────────────────────────

func TestListStaff_ScopesToOrganization(t *testing.T) {
	f := newStaffFixture()

	got, err := f.listUC().Execute(context.Background(), f.orgID,
		repository.StaffFilter{}, page())
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if got.Meta.Total != 2 {
		t.Fatalf("total = %d, want 2 (the other tenant's member must not appear)",
			got.Meta.Total)
	}
	for _, s := range got.Data {
		if s.ID == f.outsider {
			t.Error("another organization's member leaked into the roster")
		}
	}
}

func TestListStaff_FiltersByDepartment(t *testing.T) {
	f := newStaffFixture()

	got, err := f.listUC().Execute(context.Background(), f.orgID,
		repository.StaffFilter{DepartmentID: &f.techID}, page())
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(got.Data) != 1 || got.Data[0].ID != f.support {
		t.Fatalf("data = %+v, want only the Technical Support member", got.Data)
	}
	if got.Data[0].Department == nil || got.Data[0].Department.Name != "Technical Support" {
		t.Errorf("department = %+v, want the name joined in", got.Data[0].Department)
	}
}

func TestListStaff_UnassignedMeansNoRow(t *testing.T) {
	f := newStaffFixture()

	got, err := f.listUC().Execute(context.Background(), f.orgID,
		repository.StaffFilter{Unassigned: true}, page())
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(got.Data) != 1 || got.Data[0].ID != f.unplaced {
		t.Fatalf("data = %+v, want only the member on no team", got.Data)
	}
	if got.Data[0].Department != nil {
		t.Errorf("department = %+v, want nil", got.Data[0].Department)
	}
}

// Unassigned beats a department id, mirroring how ?assignee_id=unassigned beats
// a UUID on the ticket queue. Without this a caller sending both would silently
// get one of the two behaviours.
func TestListStaff_UnassignedBeatsDepartmentFilter(t *testing.T) {
	f := newStaffFixture()

	got, err := f.listUC().Execute(context.Background(), f.orgID,
		repository.StaffFilter{DepartmentID: &f.techID, Unassigned: true}, page())
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	if len(got.Data) != 1 || got.Data[0].ID != f.unplaced {
		t.Fatalf("data = %+v, want the unassigned member", got.Data)
	}
}

func TestListStaff_EmptyRosterSerialisesAsEmptyList(t *testing.T) {
	f := newStaffFixture()

	got, err := f.listUC().Execute(context.Background(), uuid.New(),
		repository.StaffFilter{}, page())
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	// A nil slice marshals to `null`, which every client then has to guard.
	// NewListResponse renders an empty collection instead.
	if got.Data == nil {
		t.Error("data = nil, want an empty slice so the JSON is [] and not null")
	}
	if got.Meta.Total != 0 {
		t.Errorf("total = %d, want 0", got.Meta.Total)
	}
}

// ── department headcounts ───────────────────────────────────────────────────

// The number beside a department and the list behind it are read from the same
// roster, so they cannot disagree. This is the join ListDepartments performs.
func TestListDepartments_CountsStaffPerDepartment(t *testing.T) {
	f := newStaffFixture()

	uc := NewListDepartmentsUseCase(f.departments, &fakeTicketRepo{}, f.staff)
	got, err := uc.Execute(context.Background(), f.orgID)
	if err != nil {
		t.Fatalf("list departments: %v", err)
	}

	counts := make(map[uuid.UUID]int, len(got))
	for _, d := range got {
		counts[d.ID] = d.StaffCount
	}

	if counts[f.techID] != 1 {
		t.Errorf("Technical Support staff_count = %d, want 1", counts[f.techID])
	}
	// A department nobody is on reports zero rather than being absent, so the
	// UI can mark it as unstaffed instead of rendering a blank cell.
	if counts[f.billing] != 0 {
		t.Errorf("Billing staff_count = %d, want 0", counts[f.billing])
	}
}

// A staff repository is optional on this use case, so a caller that wires it
// without one still gets departments and ticket counts.
func TestListDepartments_WithoutStaffRepo(t *testing.T) {
	f := newStaffFixture()

	uc := NewListDepartmentsUseCase(f.departments, &fakeTicketRepo{}, nil)
	got, err := uc.Execute(context.Background(), f.orgID)
	if err != nil {
		t.Fatalf("list departments: %v", err)
	}

	if len(got) == 0 {
		t.Fatal("expected departments, got none")
	}
	for _, d := range got {
		if d.StaffCount != 0 {
			t.Errorf("%s staff_count = %d, want 0 with no staff repository",
				d.Name, d.StaffCount)
		}
	}
}
