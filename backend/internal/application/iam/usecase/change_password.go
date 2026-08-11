package usecase

import (
	"context"

	"github.com/Root-Emin/TicketLens/internal/application/iam/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/service"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
	"github.com/google/uuid"
)

// ChangePasswordUseCase lets a signed-in user replace their own password.
type ChangePasswordUseCase struct {
	userRepo repository.UserRepository
	auth     service.AuthService
}

// NewChangePasswordUseCase creates a new ChangePasswordUseCase.
func NewChangePasswordUseCase(
	userRepo repository.UserRepository,
	auth service.AuthService,
) *ChangePasswordUseCase {
	return &ChangePasswordUseCase{userRepo: userRepo, auth: auth}
}

// Execute verifies the current password and stores a new hash.
//
// userID comes from the validated token. Nothing in the request body names an
// account, so there is no shape of this call that changes somebody else's
// password.
//
// Re-checking the current password is the point of the endpoint: possession of
// a live session is a weaker claim than knowledge of the password, and the two
// must not be treated as equivalent when the outcome is permanent lockout.
//
// Existing tokens keep working afterwards. There is no revocation list and JWTs
// are stateless, so a session opened before the change survives until it
// expires — worth knowing before this is relied on to end a compromise.
func (uc *ChangePasswordUseCase) Execute(ctx context.Context, userID uuid.UUID, req dto.ChangePasswordRequest) error {
	if userID == uuid.Nil {
		return domainErr.New(domainErr.ErrUnauthorized, "not authenticated", nil)
	}

	user, err := uc.userRepo.GetByID(ctx, userID)
	if err != nil {
		return err
	}

	// VerifyPassword already returns ErrUnauthorized, which the HTTP layer maps
	// to 401 — the same answer a wrong password gets at login.
	if err := uc.auth.VerifyPassword(user.PasswordHash, req.CurrentPassword); err != nil {
		return err
	}

	if req.NewPassword == req.CurrentPassword {
		return domainErr.New(domainErr.ErrValidation,
			"the new password must differ from the current one", nil)
	}

	hash, err := uc.auth.HashPassword(req.NewPassword)
	if err != nil {
		return domainErr.New(domainErr.ErrInternal, "failed to hash password", err)
	}

	// UpdatePassword, not Update: the latter writes name, email and status and
	// leaves password_hash alone, so this would have returned 204 having
	// changed nothing at all.
	return uc.userRepo.UpdatePassword(ctx, user.ID, hash)
}
