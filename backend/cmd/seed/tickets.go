package main

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	triageDTO "github.com/masterfabric-go/masterfabric/internal/application/triage/dto"
	triageUC "github.com/masterfabric-go/masterfabric/internal/application/triage/usecase"
	triageModel "github.com/masterfabric-go/masterfabric/internal/domain/triage/model"
)

const (
	// Share of tickets a human corrected. These drive the headline metric; at
	// 0% the accept rates would sit at a suspicious 1.00.
	priorityOverrideEvery   = 5 // ~20%
	departmentOverrideEvery = 7 // ~10% of the mapped subset
	threadEvery             = 3 // ~33% of tickets get a conversation

	historyDays = 30
)

type seedDeps struct {
	db          *pgxpool.Pool
	orgID       uuid.UUID
	adminID     uuid.UUID
	customerIDs []uuid.UUID
	departments []*triageModel.Department

	createTicketUC  *triageUC.CreateTicketUseCase
	createMessageUC *triageUC.CreateMessageUseCase
	analyzeUC       *triageUC.AnalyzeTicketUseCase
	updateTicketUC  *triageUC.UpdateTicketUseCase

	rng    *rand.Rand
	logger *slog.Logger
}

type seedResult struct {
	total               int
	analyzed            int
	unmapped            int
	priorityOverrides   int
	departmentOverrides int
	messages            int
}

// seededTicket is what the seed remembers about a ticket after analysis, so the
// override pass can decide what to change.
type seededTicket struct {
	id                uuid.UUID
	customerID        uuid.UUID
	predictedPriority string
	departmentID      uuid.UUID
	mappingFallback   bool
	createdAt         time.Time
	resolvedAt        *time.Time
	messageCount      int
}

func seedTickets(ctx context.Context, d seedDeps) (seedResult, error) {
	var result seedResult

	statuses := statusPlan(len(demoTickets), d.rng)
	seeded := make([]*seededTicket, 0, len(demoTickets))

	// ── Pass 1: create + analyze ────────────────────────────────────────────
	// Analysis runs before any override, which is the order the production flow
	// uses too. Reversing it would let the automatic apply step overwrite a
	// human value and trip the protection rule in AnalyzeTicketUseCase.
	for i, t := range demoTickets {
		customerID := d.customerIDs[i%len(d.customerIDs)]

		detail, err := d.createTicketUC.Execute(ctx, d.orgID, triageDTO.CreateTicketRequest{
			Subject:    t.Subject,
			Body:       t.Body,
			CustomerID: customerID,
		})
		if err != nil {
			return result, fmt.Errorf("create ticket %q: %w", t.Subject, err)
		}
		result.total++

		analyzed, err := d.analyzeUC.Execute(ctx, d.orgID, detail.ID)
		if err != nil {
			return result, fmt.Errorf("analyze ticket %q: %w", t.Subject, err)
		}
		result.analyzed++

		record := &seededTicket{
			id:           detail.ID,
			customerID:   customerID,
			departmentID: analyzed.Department.ID,
			messageCount: 1,
		}
		if len(analyzed.Analyses) > 0 {
			latest := analyzed.Analyses[0]
			record.predictedPriority = latest.PredictedPriority
			record.mappingFallback = latest.MappingFallback
		}
		if record.mappingFallback {
			result.unmapped++
		}
		seeded = append(seeded, record)
	}

	// ── Pass 2: human corrections ───────────────────────────────────────────
	mappedSeen := 0
	for i, record := range seeded {
		if i%priorityOverrideEvery == 0 {
			target := differentPriority(record.predictedPriority)
			if _, err := d.updateTicketUC.Execute(ctx, d.orgID, record.id, d.adminID,
				triageDTO.UpdateTicketRequest{Priority: &target}); err != nil {
				return result, fmt.Errorf("override priority: %w", err)
			}
			result.priorityOverrides++
		}

		// Department overrides only make sense on tickets that were actually
		// routed. A fallback ticket is excluded from the department accept rate
		// anyway, so overriding it would not move the metric.
		if record.mappingFallback {
			continue
		}
		mappedSeen++
		if mappedSeen%departmentOverrideEvery == 0 {
			target := differentDepartment(d.departments, record.departmentID)
			if target == uuid.Nil {
				continue
			}
			if _, err := d.updateTicketUC.Execute(ctx, d.orgID, record.id, d.adminID,
				triageDTO.UpdateTicketRequest{DepartmentID: &target}); err != nil {
				return result, fmt.Errorf("override department: %w", err)
			}
			result.departmentOverrides++
		}
	}

	// ── Pass 3: status ──────────────────────────────────────────────────────
	for i, record := range seeded {
		status := string(statuses[i])
		if triageModel.TicketStatus(status) == triageModel.TicketStatusOpen {
			continue // tickets are created open
		}
		if _, err := d.updateTicketUC.Execute(ctx, d.orgID, record.id, d.adminID,
			triageDTO.UpdateTicketRequest{Status: &status}); err != nil {
			return result, fmt.Errorf("set status: %w", err)
		}
	}

	// ── Pass 4: conversations ───────────────────────────────────────────────
	for i, record := range seeded {
		if i%threadEvery != 0 {
			continue
		}
		extra := 1 + d.rng.Intn(4) // 2–5 messages in total with the description
		for n := 0; n < extra; n++ {
			var err error
			switch {
			case n%3 == 2 && n > 0:
				_, err = d.createMessageUC.Execute(ctx, d.orgID, record.id,
					triageModel.AuthorTypeAgent, d.adminID, triageDTO.CreateMessageRequest{
						Body:       internalNotes[d.rng.Intn(len(internalNotes))],
						IsInternal: true,
					})
			case n%2 == 0:
				_, err = d.createMessageUC.Execute(ctx, d.orgID, record.id,
					triageModel.AuthorTypeAgent, d.adminID, triageDTO.CreateMessageRequest{
						Body: agentReplies[d.rng.Intn(len(agentReplies))],
					})
			default:
				_, err = d.createMessageUC.Execute(ctx, d.orgID, record.id,
					triageModel.AuthorTypeCustomer, record.customerID, triageDTO.CreateMessageRequest{
						Body: customerFollowUps[d.rng.Intn(len(customerFollowUps))],
					})
			}
			if err != nil {
				return result, fmt.Errorf("create message: %w", err)
			}
			record.messageCount++
			result.messages++
		}
	}

	// ── Pass 5: spread over the last 30 days ────────────────────────────────
	// Timestamps are the one thing the repositories cannot express — they always
	// stamp "now" — so the seed rewrites them directly. Everything else above
	// went through the real use cases.
	now := time.Now().UTC()
	for i, record := range seeded {
		record.createdAt = now.
			Add(-time.Duration(d.rng.Intn(historyDays)) * 24 * time.Hour).
			Add(-time.Duration(d.rng.Intn(24)) * time.Hour).
			Add(-time.Duration(d.rng.Intn(60)) * time.Minute)

		if isTerminal(statuses[i]) {
			resolved := record.createdAt.Add(time.Duration(2+d.rng.Intn(94)) * time.Hour)
			if resolved.After(now) {
				resolved = now.Add(-time.Duration(d.rng.Intn(6)) * time.Hour)
			}
			record.resolvedAt = &resolved
		}

		if err := backdate(ctx, d.db, record); err != nil {
			return result, err
		}
	}

	return result, nil
}

// statusPlan builds the requested status mix and shuffles it deterministically.
func statusPlan(n int, rng *rand.Rand) []triageModel.TicketStatus {
	weights := []struct {
		status triageModel.TicketStatus
		share  float64
	}{
		{triageModel.TicketStatusOpen, 0.40},
		{triageModel.TicketStatusInProgress, 0.15},
		{triageModel.TicketStatusPendingCustomer, 0.10},
		{triageModel.TicketStatusResolved, 0.25},
		{triageModel.TicketStatusClosed, 0.10},
	}

	plan := make([]triageModel.TicketStatus, 0, n)
	for _, w := range weights {
		for i := 0; i < int(float64(n)*w.share); i++ {
			plan = append(plan, w.status)
		}
	}
	for len(plan) < n { // rounding leftovers stay open
		plan = append(plan, triageModel.TicketStatusOpen)
	}
	plan = plan[:n]

	rng.Shuffle(len(plan), func(i, j int) { plan[i], plan[j] = plan[j], plan[i] })
	return plan
}

func isTerminal(s triageModel.TicketStatus) bool {
	return s == triageModel.TicketStatusResolved || s == triageModel.TicketStatusClosed
}

// differentPriority returns a valid priority other than the predicted one, so
// the update genuinely flips priority_overridden.
func differentPriority(predicted string) string {
	for _, p := range []triageModel.TicketPriority{
		triageModel.TicketPriorityUrgent,
		triageModel.TicketPriorityHigh,
		triageModel.TicketPriorityNormal,
		triageModel.TicketPriorityLow,
	} {
		if string(p) != predicted {
			return string(p)
		}
	}
	return string(triageModel.TicketPriorityHigh)
}

// differentDepartment picks any department other than the current one.
func differentDepartment(departments []*triageModel.Department, current uuid.UUID) uuid.UUID {
	for _, d := range departments {
		if d.ID != current {
			return d.ID
		}
	}
	return uuid.Nil
}

// backdate rewrites the timestamps of a ticket and everything hanging off it.
//
// NOTE on closed tickets: the API clears resolved_at when a ticket moves away
// from "resolved" (contract §4.3), so a ticket driven to "closed" through PATCH
// ends with a null resolved_at. The demo wants closed tickets to carry a
// completion date, so the seed writes one here directly.
func backdate(ctx context.Context, db *pgxpool.Pool, t *seededTicket) error {
	updatedAt := t.createdAt
	if t.resolvedAt != nil {
		updatedAt = *t.resolvedAt
	}

	if _, err := db.Exec(ctx,
		`UPDATE tickets SET created_at = $1, updated_at = $2, resolved_at = $3 WHERE id = $4`,
		t.createdAt, updatedAt, t.resolvedAt, t.id,
	); err != nil {
		return fmt.Errorf("backdate ticket: %w", err)
	}

	// Messages keep their order, spaced out from the ticket's creation.
	if _, err := db.Exec(ctx,
		// $1 is cast explicitly: without it Postgres infers the parameter's type
		// from the interval arithmetic and rejects the assignment.
		`UPDATE ticket_messages m
		 SET created_at = $1::timestamptz + ((r.rn - 1) * interval '47 minutes')
		 FROM (
		   SELECT id, row_number() OVER (ORDER BY created_at) AS rn
		   FROM ticket_messages WHERE ticket_id = $2
		 ) r
		 WHERE m.id = r.id`,
		t.createdAt, t.id,
	); err != nil {
		return fmt.Errorf("backdate messages: %w", err)
	}

	// Classification follows creation almost immediately, as it does in
	// production where an event consumer picks the ticket up.
	if _, err := db.Exec(ctx,
		`UPDATE ai_analyses SET created_at = $1::timestamptz + interval '2 minutes' WHERE ticket_id = $2`,
		t.createdAt, t.id,
	); err != nil {
		return fmt.Errorf("backdate analyses: %w", err)
	}

	return nil
}
