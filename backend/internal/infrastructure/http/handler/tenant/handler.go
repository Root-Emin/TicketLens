package tenant

import (
	"net/http"

	"github.com/Root-Emin/TicketLens/internal/application/tenant/dto"
	"github.com/Root-Emin/TicketLens/internal/application/tenant/usecase"
	"github.com/Root-Emin/TicketLens/internal/domain/tenant/repository"
	"github.com/Root-Emin/TicketLens/internal/shared/middleware"
	"github.com/Root-Emin/TicketLens/internal/shared/pagination"
	"github.com/Root-Emin/TicketLens/internal/shared/response"
	"github.com/Root-Emin/TicketLens/internal/shared/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// Handler provides Tenant HTTP handlers.
type Handler struct {
	createOrgUC       *usecase.CreateOrgUseCase
	createAppUC       *usecase.CreateAppUseCase
	manageKeysUC      *usecase.ManageAPIKeysUseCase
	createWorkspaceUC *usecase.CreateWorkspaceUseCase
	listWorkspacesUC  *usecase.ListWorkspacesUseCase
	updateWorkspaceUC *usecase.UpdateWorkspaceUseCase
	orgRepo           repository.OrgRepository
	appRepo           repository.AppRepository
}

// NewHandler creates a new Tenant handler.
func NewHandler(
	createOrgUC *usecase.CreateOrgUseCase,
	createAppUC *usecase.CreateAppUseCase,
	manageKeysUC *usecase.ManageAPIKeysUseCase,
	createWorkspaceUC *usecase.CreateWorkspaceUseCase,
	listWorkspacesUC *usecase.ListWorkspacesUseCase,
	updateWorkspaceUC *usecase.UpdateWorkspaceUseCase,
	orgRepo repository.OrgRepository,
	appRepo repository.AppRepository,
) *Handler {
	return &Handler{
		createOrgUC:       createOrgUC,
		createAppUC:       createAppUC,
		manageKeysUC:      manageKeysUC,
		createWorkspaceUC: createWorkspaceUC,
		listWorkspacesUC:  listWorkspacesUC,
		updateWorkspaceUC: updateWorkspaceUC,
		orgRepo:           orgRepo,
		appRepo:           appRepo,
	}
}

// CreateOrg handles organization creation.
func (h *Handler) CreateOrg(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateOrgRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	org, err := h.createOrgUC.Execute(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, org)
}

// GetOrg returns an organization by ID.
func (h *Handler) GetOrg(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "orgId"))
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid org id"})
		return
	}

	org, err := h.orgRepo.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.OrgInfo{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Status:    string(org.Status),
		CreatedAt: org.CreatedAt,
	})
}

// ListOrgs returns the organizations the caller belongs to.
//
// It used to page over every organization on the platform, which disclosed the
// name and slug of every tenant. A user currently belongs to exactly one
// organization (the JWT claim), so the result holds at most one entry; when
// multi-org membership arrives this reads the membership table instead.
func (h *Handler) ListOrgs(w http.ResponseWriter, r *http.Request) {
	params := pagination.FromRequest(r)

	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok || orgID == uuid.Nil {
		response.JSON(w, http.StatusOK, pagination.NewResult([]dto.OrgInfo{}, params, 0))
		return
	}

	org, err := h.orgRepo.GetByID(r.Context(), orgID)
	if err != nil {
		response.Error(w, err)
		return
	}

	infos := []dto.OrgInfo{{
		ID:        org.ID,
		Name:      org.Name,
		Slug:      org.Slug,
		Status:    string(org.Status),
		CreatedAt: org.CreatedAt,
	}}

	response.JSON(w, http.StatusOK, pagination.NewResult(infos, params, len(infos)))
}

// CreateApp handles app creation within an organization.
func (h *Handler) CreateApp(w http.ResponseWriter, r *http.Request) {
	orgID, ok := middleware.TenantIDFromContext(r.Context())
	if !ok {
		orgIDStr := chi.URLParam(r, "orgId")
		var err error
		orgID, err = uuid.Parse(orgIDStr)
		if err != nil {
			response.JSON(w, http.StatusBadRequest, map[string]string{"error": "organization id required"})
			return
		}
	}

	var req dto.CreateAppRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	app, err := h.createAppUC.Execute(r.Context(), orgID, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, app)
}

// GetApp returns an app by ID.
func (h *Handler) GetApp(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(chi.URLParam(r, "appId"))
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid app id"})
		return
	}

	app, err := h.appRepo.GetByID(r.Context(), id)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, dto.AppInfo{
		ID:             app.ID,
		OrganizationID: app.OrganizationID,
		Name:           app.Name,
		Slug:           app.Slug,
		Status:         string(app.Status),
		SLATier:        app.SLATier,
		CreatedAt:      app.CreatedAt,
	})
}

// ListApps returns apps for an organization.
func (h *Handler) ListApps(w http.ResponseWriter, r *http.Request) {
	orgID, err := uuid.Parse(chi.URLParam(r, "orgId"))
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid org id"})
		return
	}

	params := pagination.FromRequest(r)
	apps, total, err := h.appRepo.ListByOrg(r.Context(), orgID, params.Offset(), params.Limit())
	if err != nil {
		response.Error(w, err)
		return
	}

	var infos []dto.AppInfo
	for _, a := range apps {
		infos = append(infos, dto.AppInfo{
			ID:             a.ID,
			OrganizationID: a.OrganizationID,
			Name:           a.Name,
			Slug:           a.Slug,
			Status:         string(a.Status),
			SLATier:        a.SLATier,
			CreatedAt:      a.CreatedAt,
		})
	}

	response.JSON(w, http.StatusOK, pagination.NewResult(infos, params, total))
}

// CreateAPIKey creates an API key for an app.
func (h *Handler) CreateAPIKey(w http.ResponseWriter, r *http.Request) {
	appID, err := uuid.Parse(chi.URLParam(r, "appId"))
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid app id"})
		return
	}

	var req dto.CreateAPIKeyRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	key, err := h.manageKeysUC.CreateKey(r.Context(), appID, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, key)
}

// RevokeAPIKey revokes an API key.
func (h *Handler) RevokeAPIKey(w http.ResponseWriter, r *http.Request) {
	keyID, err := uuid.Parse(chi.URLParam(r, "keyId"))
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid key id"})
		return
	}

	if err := h.manageKeysUC.RevokeKey(r.Context(), keyID); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// ListAPIKeys lists API keys for an app.
func (h *Handler) ListAPIKeys(w http.ResponseWriter, r *http.Request) {
	appID, err := uuid.Parse(chi.URLParam(r, "appId"))
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid app id"})
		return
	}

	keys, err := h.manageKeysUC.ListKeys(r.Context(), appID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, keys)
}

// CreateWorkspace handles workspace creation.
func (h *Handler) CreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateWorkspaceRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	workspace, err := h.createWorkspaceUC.Execute(r.Context(), req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.Created(w, workspace)
}

// ListWorkspaces lists all workspaces for the current organization.
func (h *Handler) ListWorkspaces(w http.ResponseWriter, r *http.Request) {
	workspaces, err := h.listWorkspacesUC.Execute(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, workspaces)
}

// UpdateWorkspace handles workspace updates.
func (h *Handler) UpdateWorkspace(w http.ResponseWriter, r *http.Request) {
	workspaceID, err := uuid.Parse(chi.URLParam(r, "workspaceId"))
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid workspace id"})
		return
	}

	var req dto.UpdateWorkspaceRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	workspace, err := h.updateWorkspaceUC.Execute(r.Context(), workspaceID, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, workspace)
}
