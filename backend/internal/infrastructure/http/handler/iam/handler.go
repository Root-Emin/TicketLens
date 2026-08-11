package iam

import (
	"net/http"

	"github.com/Root-Emin/TicketLens/internal/application/iam/dto"
	"github.com/Root-Emin/TicketLens/internal/application/iam/usecase"
	"github.com/Root-Emin/TicketLens/internal/shared/middleware"
	"github.com/Root-Emin/TicketLens/internal/shared/pagination"
	"github.com/Root-Emin/TicketLens/internal/shared/response"
	"github.com/Root-Emin/TicketLens/internal/shared/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	iamRepo "github.com/Root-Emin/TicketLens/internal/domain/iam/repository"
)

// Handler provides IAM HTTP handlers.
type Handler struct {
	registerUC         *usecase.RegisterUseCase
	loginUC            *usecase.LoginUseCase
	assignRoleUC       *usecase.AssignRoleUseCase
	changePasswordUC   *usecase.ChangePasswordUseCase
	createInvitationUC *usecase.CreateInvitationUseCase
	acceptInvitationUC *usecase.AcceptInvitationUseCase
	listInvitationsUC  *usecase.ListInvitationsUseCase
	revokeInvitationUC *usecase.RevokeInvitationUseCase
	listRolesUC        *usecase.ListRolesUseCase
	userRepo           iamRepo.UserRepository
}

// NewHandler creates a new IAM handler.
func NewHandler(
	registerUC *usecase.RegisterUseCase,
	loginUC *usecase.LoginUseCase,
	assignRoleUC *usecase.AssignRoleUseCase,
	changePasswordUC *usecase.ChangePasswordUseCase,
	createInvitationUC *usecase.CreateInvitationUseCase,
	acceptInvitationUC *usecase.AcceptInvitationUseCase,
	listInvitationsUC *usecase.ListInvitationsUseCase,
	revokeInvitationUC *usecase.RevokeInvitationUseCase,
	listRolesUC *usecase.ListRolesUseCase,
	userRepo iamRepo.UserRepository,
) *Handler {
	return &Handler{
		registerUC:         registerUC,
		loginUC:            loginUC,
		assignRoleUC:       assignRoleUC,
		changePasswordUC:   changePasswordUC,
		createInvitationUC: createInvitationUC,
		acceptInvitationUC: acceptInvitationUC,
		listInvitationsUC:  listInvitationsUC,
		revokeInvitationUC: revokeInvitationUC,
		listRolesUC:        listRolesUC,
		userRepo:           userRepo,
	}
}

// ChangePassword replaces the signed-in user's own password.
//
// Mounted under the authenticated routes, not under /auth, even though the path
// reads that way: it needs a validated token to know whose password to change,
// and /auth/* is the public subtree.
func (h *Handler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	var req dto.ChangePasswordRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := h.changePasswordUC.Execute(r.Context(), userID, req); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// Register handles user registration.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	user, err := h.registerUC.Execute(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, user)
}

// Login handles user authentication.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.loginUC.Execute(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, result)
}

// AssignRole handles role assignment within the caller's organization.
func (h *Handler) AssignRole(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	var req dto.AssignRoleRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	if err := h.assignRoleUC.Execute(r.Context(), orgID, req); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// orgFromToken reads the caller's organization from the validated claims and
// answers 403 when there is none.
//
// Every organization-scoped handler in this package goes through it rather than
// reading the context inline, so none of them can accidentally fall back to a
// value supplied by the request.
func orgFromToken(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok || orgID == uuid.Nil {
		response.JSON(w, http.StatusForbidden, map[string]string{"error": "no organization in token"})
		return uuid.Nil, false
	}
	return orgID, true
}

// GetMe returns the current authenticated user.
func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.UserIDFromContext(r.Context())
	if !ok {
		response.JSON(w, http.StatusUnauthorized, map[string]string{"error": "not authenticated"})
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), userID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt,
	})
}

// GetUser returns a user by ID.
func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid user id"})
		return
	}

	user, err := h.userRepo.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.UserInfo{
		ID:        user.ID,
		Email:     user.Email,
		FirstName: user.FirstName,
		LastName:  user.LastName,
		Status:    string(user.Status),
		CreatedAt: user.CreatedAt,
	})
}

// ListUsers returns a paginated list of users in the caller's organization.
//
// Scoped deliberately: the unscoped repository List spans every tenant, so
// serving it here disclosed the email address of every account on the platform
// to anyone holding user:read in any organization.
func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	params := pagination.FromRequest(r)

	users, total, err := h.userRepo.ListByOrganization(r.Context(), orgID, params.Offset(), params.Limit())
	if err != nil {
		response.Error(w, err)
		return
	}

	var infos []dto.UserInfo
	for _, u := range users {
		infos = append(infos, dto.UserInfo{
			ID:        u.ID,
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Status:    string(u.Status),
			CreatedAt: u.CreatedAt,
		})
	}

	response.JSON(w, http.StatusOK, pagination.NewResult(infos, params, total))
}
