package usecase

import (
	"context"

	"github.com/masterfabric-go/masterfabric/internal/application/iam/dto"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/repository"
	"github.com/masterfabric-go/masterfabric/internal/domain/iam/service"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
)

// LoginUseCase handles user authentication.
type LoginUseCase struct {
	userRepo repository.UserRepository
	roleRepo repository.RoleRepository
	auth     service.AuthService
}

// NewLoginUseCase creates a new LoginUseCase.
func NewLoginUseCase(userRepo repository.UserRepository, roleRepo repository.RoleRepository, auth service.AuthService) *LoginUseCase {
	return &LoginUseCase{userRepo: userRepo, roleRepo: roleRepo, auth: auth}
}

// Execute authenticates a user and returns a JWT token.
func (uc *LoginUseCase) Execute(ctx context.Context, req dto.LoginRequest) (*dto.LoginResponse, error) {
	user, err := uc.userRepo.GetByEmail(ctx, req.Email)
	if err != nil {
		return nil, domainErr.New(domainErr.ErrUnauthorized, "invalid credentials", nil)
	}

	if !user.IsActive() {
		return nil, domainErr.New(domainErr.ErrForbidden, "account is not active", nil)
	}

	if err := uc.auth.VerifyPassword(user.PasswordHash, req.Password); err != nil {
		return nil, err
	}

	// Resolve which organization the user belongs to and put it in the token:
	// permission checks read organization_id straight from the claims. A user
	// with no organization gets uuid.Nil, logs in fine, and is denied by any
	// endpoint that needs an organization.
	orgID, err := uc.roleRepo.GetPrimaryOrganization(ctx, user.ID)
	if err != nil {
		return nil, err
	}

	token, err := uc.auth.GenerateToken(ctx, service.TokenClaims{
		UserID:         user.ID,
		Email:          user.Email,
		OrganizationID: orgID,
	})
	if err != nil {
		return nil, domainErr.New(domainErr.ErrInternal, "failed to generate token", err)
	}

	return &dto.LoginResponse{
		Token: token,
		User: dto.UserInfo{
			ID:        user.ID,
			Email:     user.Email,
			FirstName: user.FirstName,
			LastName:  user.LastName,
			Status:    string(user.Status),
			CreatedAt: user.CreatedAt,
		},
	}, nil
}
