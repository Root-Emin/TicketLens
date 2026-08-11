package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/application/triage/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/repository"
	"github.com/google/uuid"
)

// ListStaffUseCase serves the support roster.
type ListStaffUseCase struct {
	staffRepo repository.StaffRepository
}

// NewListStaffUseCase creates a new ListStaffUseCase.
func NewListStaffUseCase(staffRepo repository.StaffRepository) *ListStaffUseCase {
	return &ListStaffUseCase{staffRepo: staffRepo}
}

// Execute returns one page of the organization's support staff.
//
// Thin on purpose: the interesting decisions — who counts as staff, what an
// unassigned person looks like — are properties of the roster itself and live
// in the repository query, where they are one join rather than a loop over
// results.
func (uc *ListStaffUseCase) Execute(
	ctx context.Context,
	orgID uuid.UUID,
	filter repository.StaffFilter,
	params dto.PageParams,
) (dto.ListResponse[dto.StaffInfo], error) {
	staff, total, err := uc.staffRepo.ListByOrg(ctx, orgID, filter, params.Offset(), params.Limit())
	if err != nil {
		return dto.ListResponse[dto.StaffInfo]{}, err
	}

	infos := make([]dto.StaffInfo, 0, len(staff))
	for _, member := range staff {
		infos = append(infos, toStaffInfo(member))
	}

	return dto.NewListResponse(infos, params, total), nil
}

func toStaffInfo(member *model.StaffMember) dto.StaffInfo {
	info := dto.StaffInfo{
		ID:        member.UserID,
		Email:     member.Email,
		FirstName: member.FirstName,
		LastName:  member.LastName,
		FullName:  member.FullName(),
		Status:    member.Status,
		CreatedAt: member.CreatedAt,
	}

	if member.DepartmentID != nil {
		info.Department = &dto.DepartmentRef{
			ID:   *member.DepartmentID,
			Name: member.DepartmentName,
		}
	}
	return info
}
