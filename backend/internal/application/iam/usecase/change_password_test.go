package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/Root-Emin/TicketLens/internal/application/iam/dto"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/model"
	"github.com/Root-Emin/TicketLens/internal/domain/iam/service"
	domainErr "github.com/Root-Emin/TicketLens/internal/shared/errors"
)

/*
	These exist because of a bug that shipped 204 and changed nothing.

	The use case called UserRepository.Update, which writes email, name and
	status — and not password_hash. Every layer reported success: the handler
	returned No Content, the use case returned nil, the SQL ran. Only logging in
	again revealed it, and by then the user believes their password changed.

	The tests below assert on what the *store* ended up holding, not on the
	returned error, because a silent no-op has no error to assert on.
*/

// passwordStore records which method actually wrote, so a test can tell a real
// password change apart from a full-row Update that dropped the hash.
type passwordStore struct {
	user *model.User

	updateCalls         int
	updatePasswordCalls int
	storedHash          string
}

func (s *passwordStore) GetByID(_ context.Context, id uuid.UUID) (*model.User, error) {
	if s.user == nil || s.user.ID != id {
		return nil, domainErr.New(domainErr.ErrNotFound, "user not found", nil)
	}
	clone := *s.user
	return &clone, nil
}

// Update mirrors the real repository: it writes every column except the
// password hash. A use case that reaches for this to change a credential must
// fail these tests, not pass them.
func (s *passwordStore) Update(_ context.Context, u *model.User) error {
	s.updateCalls++
	s.user.Email = u.Email
	s.user.FirstName = u.FirstName
	s.user.LastName = u.LastName
	s.user.Status = u.Status
	return nil
}

func (s *passwordStore) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	if s.user == nil || s.user.ID != id {
		return domainErr.New(domainErr.ErrNotFound, "user not found", nil)
	}
	s.updatePasswordCalls++
	s.storedHash = hash
	s.user.PasswordHash = hash
	return nil
}

func (s *passwordStore) Create(context.Context, *model.User) error { return nil }
func (s *passwordStore) Delete(context.Context, uuid.UUID) error   { return nil }
func (s *passwordStore) GetByEmail(context.Context, string) (*model.User, error) {
	return nil, domainErr.New(domainErr.ErrNotFound, "user not found", nil)
}
func (s *passwordStore) List(context.Context, int, int) ([]*model.User, int, error) {
	return nil, 0, nil
}
func (s *passwordStore) ListByOrganization(context.Context, uuid.UUID, int, int) ([]*model.User, int, error) {
	return nil, 0, nil
}

// fakeAuth hashes by prefixing, which keeps "was this hashed" checkable without
// paying bcrypt's cost in a unit test.
type fakeAuth struct{}

func (fakeAuth) HashPassword(password string) (string, error) { return "hashed:" + password, nil }

func (fakeAuth) VerifyPassword(hashedPassword, password string) error {
	if hashedPassword != "hashed:"+password {
		return domainErr.New(domainErr.ErrUnauthorized, "invalid credentials", nil)
	}
	return nil
}

func (fakeAuth) GenerateToken(context.Context, service.TokenClaims) (string, error) { return "", nil }
func (fakeAuth) ValidateToken(context.Context, string) (*service.TokenClaims, error) {
	return nil, nil
}

func newPasswordFixture() (*ChangePasswordUseCase, *passwordStore, uuid.UUID) {
	userID := uuid.New()
	store := &passwordStore{user: &model.User{
		ID:           userID,
		Email:        "alice@example.com",
		PasswordHash: "hashed:Current1234!",
		Status:       model.UserStatusActive,
	}}
	return NewChangePasswordUseCase(store, fakeAuth{}), store, userID
}

func TestChangePassword_ActuallyPersistsTheNewHash(t *testing.T) {
	uc, store, userID := newPasswordFixture()

	err := uc.Execute(context.Background(), userID, dto.ChangePasswordRequest{
		CurrentPassword: "Current1234!",
		NewPassword:     "Replacement1234!",
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	if store.updatePasswordCalls != 1 {
		t.Fatalf("UpdatePassword called %d times, want 1 — returning success without "+
			"writing the hash is the bug this guards", store.updatePasswordCalls)
	}
	if store.storedHash != "hashed:Replacement1234!" {
		t.Errorf("stored hash = %q, want the new password hashed", store.storedHash)
	}
	if store.user.PasswordHash == "hashed:Current1234!" {
		t.Error("the old hash is still in the store: the change did not take effect")
	}
}

func TestChangePassword_WrongCurrentPasswordIsRejected(t *testing.T) {
	uc, store, userID := newPasswordFixture()

	err := uc.Execute(context.Background(), userID, dto.ChangePasswordRequest{
		CurrentPassword: "NotTheCurrentOne",
		NewPassword:     "Replacement1234!",
	})

	if !errors.Is(err, domainErr.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
	if store.updatePasswordCalls != 0 {
		t.Error("nothing must be written when the current password does not verify")
	}
}

func TestChangePassword_ReusingTheSamePasswordIsRejected(t *testing.T) {
	uc, store, userID := newPasswordFixture()

	err := uc.Execute(context.Background(), userID, dto.ChangePasswordRequest{
		CurrentPassword: "Current1234!",
		NewPassword:     "Current1234!",
	})

	if !errors.Is(err, domainErr.ErrValidation) {
		t.Fatalf("error = %v, want ErrValidation", err)
	}
	if store.updatePasswordCalls != 0 {
		t.Error("a no-op change must not touch the store")
	}
}

func TestChangePassword_UnauthenticatedCallerIsRejected(t *testing.T) {
	uc, store, _ := newPasswordFixture()

	err := uc.Execute(context.Background(), uuid.Nil, dto.ChangePasswordRequest{
		CurrentPassword: "Current1234!",
		NewPassword:     "Replacement1234!",
	})

	if !errors.Is(err, domainErr.ErrUnauthorized) {
		t.Fatalf("error = %v, want ErrUnauthorized", err)
	}
	if store.updatePasswordCalls != 0 {
		t.Error("an unauthenticated call must not reach the store")
	}
}
