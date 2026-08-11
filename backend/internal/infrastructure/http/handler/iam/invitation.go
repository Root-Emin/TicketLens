package iam

import (
	"net/http"

	"github.com/Root-Emin/TicketLens/internal/application/iam/dto"
	"github.com/Root-Emin/TicketLens/internal/shared/pagination"
	"github.com/Root-Emin/TicketLens/internal/shared/response"
	"github.com/Root-Emin/TicketLens/internal/shared/validator"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

// CreateInvitation invites somebody to join the caller's organization as staff.
//
// 201 with the invitation, including accept_url. That link is returned because
// this is the only moment it exists — the token is stored hashed and cannot be
// recovered — so an administrator always has something to pass on even when mail
// delivery is not configured or fails.
func (h *Handler) CreateInvitation(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	var req dto.CreateInvitationRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	result, err := h.createInvitationUC.Execute(r.Context(), orgID, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, result)
}

// ListInvitations returns a page of the organization's invitations.
func (h *Handler) ListInvitations(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	params := pagination.FromRequest(r)

	invitations, total, err := h.listInvitationsUC.Execute(r.Context(), orgID, params.Offset(), params.Limit())
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, pagination.NewResult(invitations, params, total))
}

// RevokeInvitation withdraws a pending invitation.
func (h *Handler) RevokeInvitation(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "invitationId"))
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid invitation id"})
		return
	}

	if err := h.revokeInvitationUC.Execute(r.Context(), orgID, id); err != nil {
		response.Error(w, err)
		return
	}

	response.NoContent(w)
}

// ListRoles returns the roles the caller's organization can assign to staff.
//
// Exists because a role id was otherwise unobtainable by any client: the ids are
// per-organization clones, and both inviting somebody and assigning a role take
// one.
func (h *Handler) ListRoles(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	roles, err := h.listRolesUC.Execute(r.Context(), orgID)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"data": roles})
}

// PreviewInvitation describes an invitation to the person holding its link.
//
// Public, like the acceptance below: the recipient has no account yet, so there
// is nothing to authenticate. It answers only what the screen must render, and
// an unknown, spent or expired token gets the one generic 404 the use case
// produces — distinguishing them would let somebody probe for live invitations.
func (h *Handler) PreviewInvitation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "missing token"})
		return
	}

	preview, err := h.acceptInvitationUC.Preview(r.Context(), token)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusOK, preview)
}

// AcceptInvitation redeems an invitation and creates or joins the account.
//
// Returns the user, not a session: the caller has just chosen a password and the
// frontend signs in with it immediately afterwards, exactly as the registration
// route already does. Issuing a token straight from a link-bearing request would
// also make the invitation email a one-click session for anyone who read it.
func (h *Handler) AcceptInvitation(w http.ResponseWriter, r *http.Request) {
	token := chi.URLParam(r, "token")
	if token == "" {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "missing token"})
		return
	}

	var req dto.AcceptInvitationRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	user, err := h.acceptInvitationUC.Execute(r.Context(), token, req)
	if err != nil {
		response.Error(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, user)
}
