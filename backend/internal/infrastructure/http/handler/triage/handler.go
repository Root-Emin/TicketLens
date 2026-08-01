package triage

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	"github.com/masterfabric-go/masterfabric/internal/application/triage/usecase"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
	"github.com/masterfabric-go/masterfabric/internal/domain/triage/repository"
	domainErr "github.com/masterfabric-go/masterfabric/internal/shared/errors"
	"github.com/masterfabric-go/masterfabric/internal/shared/middleware"
	"github.com/masterfabric-go/masterfabric/internal/shared/response"
	"github.com/masterfabric-go/masterfabric/internal/shared/validator"
)

// Handler provides Triage (ticketing) HTTP handlers.
type Handler struct {
	listDepartmentsUC  *usecase.ListDepartmentsUseCase
	createDepartmentUC *usecase.CreateDepartmentUseCase
	updateDepartmentUC *usecase.UpdateDepartmentUseCase
	deleteDepartmentUC *usecase.DeleteDepartmentUseCase

	listCustomersUC  *usecase.ListCustomersUseCase
	createCustomerUC *usecase.CreateCustomerUseCase
	getCustomerUC    *usecase.GetCustomerUseCase

	createTicketUC *usecase.CreateTicketUseCase
	listTicketsUC  *usecase.ListTicketsUseCase
	getTicketUC    *usecase.GetTicketUseCase
	updateTicketUC *usecase.UpdateTicketUseCase
	assignTicketUC *usecase.AssignTicketUseCase

	listMessagesUC  *usecase.ListMessagesUseCase
	createMessageUC *usecase.CreateMessageUseCase
	listAnalysesUC  *usecase.ListAnalysesUseCase
	analyzeTicketUC *usecase.AnalyzeTicketUseCase

	statsOverviewUC *usecase.StatsOverviewUseCase
}

// Config holds the handler's use case dependencies.
type Config struct {
	ListDepartments  *usecase.ListDepartmentsUseCase
	CreateDepartment *usecase.CreateDepartmentUseCase
	UpdateDepartment *usecase.UpdateDepartmentUseCase
	DeleteDepartment *usecase.DeleteDepartmentUseCase

	ListCustomers  *usecase.ListCustomersUseCase
	CreateCustomer *usecase.CreateCustomerUseCase
	GetCustomer    *usecase.GetCustomerUseCase

	CreateTicket *usecase.CreateTicketUseCase
	ListTickets  *usecase.ListTicketsUseCase
	GetTicket    *usecase.GetTicketUseCase
	UpdateTicket *usecase.UpdateTicketUseCase
	AssignTicket *usecase.AssignTicketUseCase

	ListMessages  *usecase.ListMessagesUseCase
	CreateMessage *usecase.CreateMessageUseCase
	ListAnalyses  *usecase.ListAnalysesUseCase
	AnalyzeTicket *usecase.AnalyzeTicketUseCase

	StatsOverview *usecase.StatsOverviewUseCase
}

// NewHandler creates a new Triage handler.
func NewHandler(cfg Config) *Handler {
	return &Handler{
		listDepartmentsUC:  cfg.ListDepartments,
		createDepartmentUC: cfg.CreateDepartment,
		updateDepartmentUC: cfg.UpdateDepartment,
		deleteDepartmentUC: cfg.DeleteDepartment,
		listCustomersUC:    cfg.ListCustomers,
		createCustomerUC:   cfg.CreateCustomer,
		getCustomerUC:      cfg.GetCustomer,
		createTicketUC:     cfg.CreateTicket,
		listTicketsUC:      cfg.ListTickets,
		getTicketUC:        cfg.GetTicket,
		updateTicketUC:     cfg.UpdateTicket,
		assignTicketUC:     cfg.AssignTicket,
		listMessagesUC:     cfg.ListMessages,
		createMessageUC:    cfg.CreateMessage,
		listAnalysesUC:     cfg.ListAnalyses,
		analyzeTicketUC:    cfg.AnalyzeTicket,
		statsOverviewUC:    cfg.StatsOverview,
	}
}

// ─── Departments ─────────────────────────────────────────────────────────────

// ListDepartments returns the organization's departments with ticket counts.
func (h *Handler) ListDepartments(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	departments, err := h.listDepartmentsUC.Execute(r.Context(), orgID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewCollectionResponse(departments))
}

// CreateDepartment creates a department.
func (h *Handler) CreateDepartment(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	var req dto.CreateDepartmentRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	department, err := h.createDepartmentUC.Execute(r.Context(), orgID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, department)
}

// UpdateDepartment patches a department's name and/or description.
func (h *Handler) UpdateDepartment(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}
	departmentID, ok := uuidParam(w, r, "departmentId")
	if !ok {
		return
	}

	var req dto.UpdateDepartmentRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	department, err := h.updateDepartmentUC.Execute(r.Context(), orgID, departmentID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, department)
}

// DeleteDepartment removes a department, moving its tickets to the default one.
func (h *Handler) DeleteDepartment(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}
	departmentID, ok := uuidParam(w, r, "departmentId")
	if !ok {
		return
	}

	if err := h.deleteDepartmentUC.Execute(r.Context(), orgID, departmentID); err != nil {
		response.Error(w, err)
		return
	}
	response.NoContent(w)
}

// ─── Customers ───────────────────────────────────────────────────────────────

// ListCustomers returns a page of customers, optionally filtered by ?q=.
func (h *Handler) ListCustomers(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	result, err := h.listCustomersUC.Execute(r.Context(), orgID, r.URL.Query().Get("q"), pageParams(r))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// CreateCustomer creates a customer.
func (h *Handler) CreateCustomer(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	var req dto.CreateCustomerRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	customer, err := h.createCustomerUC.Execute(r.Context(), orgID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, customer)
}

// GetCustomer returns one customer with their ticket summary.
func (h *Handler) GetCustomer(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}
	customerID, ok := uuidParam(w, r, "customerId")
	if !ok {
		return
	}

	customer, err := h.getCustomerUC.Execute(r.Context(), orgID, customerID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, customer)
}

// ─── Tickets ─────────────────────────────────────────────────────────────────

// CreateTicket raises a ticket and its first message.
func (h *Handler) CreateTicket(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	var req dto.CreateTicketRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	ticket, err := h.createTicketUC.Execute(r.Context(), orgID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, ticket)
}

// ListTickets serves the filtered ticket queue.
func (h *Handler) ListTickets(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	filter, err := ticketFilterFromQuery(r)
	if err != nil {
		response.Error(w, err)
		return
	}

	result, err := h.listTicketsUC.Execute(r.Context(), orgID, filter, pageParams(r))
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, result)
}

// GetTicket returns one ticket with its thread and analyses.
func (h *Handler) GetTicket(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}
	ticketID, ok := uuidParam(w, r, "ticketId")
	if !ok {
		return
	}

	// Agents see internal notes; the customer portal (not implemented yet) would
	// pass false here.
	ticket, err := h.getTicketUC.Execute(r.Context(), orgID, ticketID, true)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ticket)
}

// UpdateTicket patches priority, department and/or status.
func (h *Handler) UpdateTicket(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}
	ticketID, ok := uuidParam(w, r, "ticketId")
	if !ok {
		return
	}

	var req dto.UpdateTicketRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	actorID, _ := middleware.UserIDFromContext(r.Context())
	ticket, err := h.updateTicketUC.Execute(r.Context(), orgID, ticketID, actorID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ticket)
}

// AssignTicket sets or clears a ticket's assignee.
func (h *Handler) AssignTicket(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}
	ticketID, ok := uuidParam(w, r, "ticketId")
	if !ok {
		return
	}

	var req dto.AssignTicketRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	ticket, err := h.assignTicketUC.Execute(r.Context(), orgID, ticketID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, ticket)
}

// ─── Messages ────────────────────────────────────────────────────────────────

// ListMessages returns a ticket's conversation.
func (h *Handler) ListMessages(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}
	ticketID, ok := uuidParam(w, r, "ticketId")
	if !ok {
		return
	}

	messages, err := h.listMessagesUC.Execute(r.Context(), orgID, ticketID, true)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewCollectionResponse(messages))
}

// CreateMessage appends a message to a ticket.
func (h *Handler) CreateMessage(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}
	ticketID, ok := uuidParam(w, r, "ticketId")
	if !ok {
		return
	}

	var req dto.CreateMessageRequest
	if err := validator.DecodeAndValidate(r, &req); err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	// Author identity comes from the token. Every authenticated caller here is a
	// platform user, so the message is authored by an agent.
	actorID, _ := middleware.UserIDFromContext(r.Context())
	message, err := h.createMessageUC.Execute(r.Context(), orgID, ticketID, model.AuthorTypeAgent, actorID, req)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, message)
}

// ─── Analyses ────────────────────────────────────────────────────────────────

// ListAnalyses returns a ticket's classifier history, newest first.
func (h *Handler) ListAnalyses(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}
	ticketID, ok := uuidParam(w, r, "ticketId")
	if !ok {
		return
	}

	analyses, err := h.listAnalysesUC.Execute(r.Context(), orgID, ticketID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, dto.NewCollectionResponse(analyses))
}

// RerunAnalysis re-classifies a ticket on demand. It always appends a new
// analysis rather than updating the previous one — the table is append-only and
// the history is the point of the "lens" view. The idempotency guard used by
// the event consumer is deliberately bypassed here.
func (h *Handler) RerunAnalysis(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}
	ticketID, ok := uuidParam(w, r, "ticketId")
	if !ok {
		return
	}

	ticket, err := h.analyzeTicketUC.Execute(r.Context(), orgID, ticketID)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, ticket)
}

// ─── Stats ───────────────────────────────────────────────────────────────────

// StatsOverview returns the dashboard aggregates.
func (h *Handler) StatsOverview(w http.ResponseWriter, r *http.Request) {
	orgID, ok := orgFromToken(w, r)
	if !ok {
		return
	}

	from, err := rfc3339Query(r, "from")
	if err != nil {
		response.Error(w, err)
		return
	}
	to, err := rfc3339Query(r, "to")
	if err != nil {
		response.Error(w, err)
		return
	}

	stats, err := h.statsOverviewUC.Execute(r.Context(), orgID, from, to)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.JSON(w, http.StatusOK, stats)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// orgFromToken reads the organization from the JWT claims. Per the API
// contract it is never taken from the body or the query string. A user with no
// organization is refused here rather than reaching a repository with uuid.Nil.
func orgFromToken(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	orgID, ok := middleware.OrgIDFromContext(r.Context())
	if !ok || orgID == uuid.Nil {
		response.JSON(w, http.StatusForbidden, map[string]string{
			"error": "no organization in token",
		})
		return uuid.Nil, false
	}
	return orgID, true
}

func uuidParam(w http.ResponseWriter, r *http.Request, name string) (uuid.UUID, bool) {
	id, err := uuid.Parse(chi.URLParam(r, name))
	if err != nil {
		response.JSON(w, http.StatusBadRequest, map[string]string{"error": "invalid " + name})
		return uuid.Nil, false
	}
	return id, true
}

func pageParams(r *http.Request) dto.PageParams {
	page := queryInt(r, "page", dto.DefaultPage)
	if page < 1 {
		page = dto.DefaultPage
	}

	size := queryInt(r, "page_size", dto.DefaultPageSize)
	if size < 1 {
		size = dto.DefaultPageSize
	}
	if size > dto.MaxPageSize {
		size = dto.MaxPageSize
	}

	return dto.PageParams{Page: page, PageSize: size}
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func rfc3339Query(r *http.Request, key string) (time.Time, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(key))
	if raw == "" {
		return time.Time{}, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return time.Time{}, domainErr.New(domainErr.ErrBadRequest,
			"invalid "+key+": expected RFC3339", nil)
	}
	return parsed.UTC(), nil
}

// ticketFilterFromQuery maps the documented query parameters onto the
// repository filter. Unknown sort keys fall back to newest-first.
func ticketFilterFromQuery(r *http.Request) (repository.TicketFilter, error) {
	query := r.URL.Query()
	filter := repository.TicketFilter{
		Query: strings.TrimSpace(query.Get("q")),
		Sort:  query.Get("sort"),
	}

	for _, raw := range query["status"] {
		status := model.TicketStatus(raw)
		if !model.ValidTicketStatus(status) {
			return filter, domainErr.New(domainErr.ErrBadRequest, "invalid status: "+raw, nil)
		}
		filter.Statuses = append(filter.Statuses, status)
	}

	for _, raw := range query["priority"] {
		priority := model.TicketPriority(raw)
		if !model.ValidTicketPriority(priority) {
			return filter, domainErr.New(domainErr.ErrBadRequest, "invalid priority: "+raw, nil)
		}
		filter.Priorities = append(filter.Priorities, priority)
	}

	if raw := query.Get("department_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			return filter, domainErr.New(domainErr.ErrBadRequest, "invalid department_id", nil)
		}
		filter.DepartmentID = &id
	}

	// "unassigned" is a documented literal, not a UUID.
	if raw := query.Get("assignee_id"); raw != "" {
		if raw == "unassigned" {
			filter.Unassigned = true
		} else {
			id, err := uuid.Parse(raw)
			if err != nil {
				return filter, domainErr.New(domainErr.ErrBadRequest, "invalid assignee_id", nil)
			}
			filter.AssigneeID = &id
		}
	}

	if raw := query.Get("needs_review"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return filter, domainErr.New(domainErr.ErrBadRequest, "invalid needs_review", nil)
		}
		filter.NeedsReview = &value
	}

	if raw := query.Get("overridden"); raw != "" {
		value, err := strconv.ParseBool(raw)
		if err != nil {
			return filter, domainErr.New(domainErr.ErrBadRequest, "invalid overridden", nil)
		}
		filter.Overridden = &value
	}

	return filter, nil
}
