package main

import (
	"context"
	"fmt"
	"math/rand"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	iamUC "github.com/Root-Emin/TicketLens/internal/application/iam/usecase"
	triageModel "github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	pgIam "github.com/Root-Emin/TicketLens/internal/infrastructure/postgres/iam"
)

/*
	The support team.

	Before staff_departments existed (migration 00021) the demo tenant had one
	agent and no notion of a team, which made every roster screen a list of one
	and every department a page with nobody on it. A support organization is the
	product's subject; seeding one agent is like seeding one ticket.

	Six agents across four departments, plus one left deliberately unassigned.
	The unassigned one is not an oversight — "somebody is on the roster but on no
	team" is a state the panel has to render and a manager has to resolve, and a
	demo where that bucket is always empty hides the screen that handles it.
*/

// demoAgent is a seeded support account and the team it belongs to.
//
// department is matched by name against the departments the seed just created;
// an empty string means the agent is left off every team.
type demoAgent struct {
	email      string
	fullName   string
	department string
}

// demoAgentDepartment is where the documented sign-in agent
// (demoAgentEmail / Selin Aydın) is placed.
//
// That account is created by seedAgentLogin, not here, because it is the one
// printed at the end of a seed run for somebody to sign in as. It still needs a
// team — an agent whose own roster entry says "no department" is a confusing
// first impression of the panel — so it is placed alongside the rest rather than
// duplicated into the list below.
const demoAgentDepartment = "Technical Support"

var demoAgents = []demoAgent{
	{"emre.demir@ticketlens.dev", "Emre Demir", "Technical Support"},
	{"deniz.kaya@ticketlens.dev", "Deniz Kaya", "Payment Operations"},
	{"merve.arslan@ticketlens.dev", "Merve Arslan", "Payment Operations"},
	{"burak.sahin@ticketlens.dev", "Burak Şahin", "Integration Support"},
	{"elif.yildiz@ticketlens.dev", "Elif Yıldız", "Customer Success"},
	{"kerem.polat@ticketlens.dev", "Kerem Polat", "Customer Success"},
	// No team on purpose: a newly granted account, waiting to be placed.
	{"can.ozturk@ticketlens.dev", "Can Öztürk", ""},
}

// seedSupportTeam creates the agent accounts, grants them the agent role and
// places them on their departments.
//
// Returns the user IDs grouped by department, which the ticket pass uses to
// hand work to somebody who is actually on that team.
func seedSupportTeam(
	ctx context.Context,
	db *pgxpool.Pool,
	userRepo *pgIam.UserRepo,
	registerUC *iamUC.RegisterUseCase,
	orgID uuid.UUID,
	departments []*triageModel.Department,
) (map[uuid.UUID][]uuid.UUID, error) {
	agentRoleID, err := roleIDByName(ctx, db, orgID, "agent")
	if err != nil {
		return nil, err
	}

	departmentByName := make(map[string]uuid.UUID, len(departments))
	for _, d := range departments {
		departmentByName[d.Name] = d.ID
	}

	byDepartment := make(map[uuid.UUID][]uuid.UUID)

	// The sign-in agent first, so they are on a team like everybody else.
	if departmentID, ok := departmentByName[demoAgentDepartment]; ok {
		existing, err := userRepo.GetByEmail(ctx, demoAgentEmail)
		if err != nil || existing == nil {
			return nil, fmt.Errorf("seed the agent login before the support team")
		}
		if err := assignStaffDepartment(ctx, db, orgID, existing.ID, departmentID); err != nil {
			return nil, fmt.Errorf("place %s in %s: %w", demoAgentEmail, demoAgentDepartment, err)
		}
		byDepartment[departmentID] = append(byDepartment[departmentID], existing.ID)
	}

	for _, agent := range demoAgents {
		userID, err := ensureUser(ctx, userRepo, registerUC, agent.email, agent.fullName)
		if err != nil {
			return nil, err
		}
		if err := grantRole(ctx, db, userID, agentRoleID, orgID); err != nil {
			return nil, fmt.Errorf("assign agent role to %s: %w", agent.email, err)
		}

		if agent.department == "" {
			continue
		}

		departmentID, ok := departmentByName[agent.department]
		if !ok {
			return nil, fmt.Errorf("agent %s: no department named %q", agent.email, agent.department)
		}

		if err := assignStaffDepartment(ctx, db, orgID, userID, departmentID); err != nil {
			return nil, fmt.Errorf("place %s in %s: %w", agent.email, agent.department, err)
		}
		byDepartment[departmentID] = append(byDepartment[departmentID], userID)
	}

	return byDepartment, nil
}

// assignStaffDepartment writes one staff_departments row. Idempotent, so a
// re-run of the seed updates rather than failing on the primary key.
func assignStaffDepartment(ctx context.Context, db *pgxpool.Pool, orgID, userID, departmentID uuid.UUID) error {
	_, err := db.Exec(ctx, `
		INSERT INTO staff_departments (organization_id, user_id, department_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (organization_id, user_id)
		DO UPDATE SET department_id = EXCLUDED.department_id, updated_at = NOW()
	`, orgID, userID, departmentID)
	return err
}

/*
	Handing the queue out.

	Every seeded ticket used to arrive unassigned, which made the whole workload
	half of the admin panel measure nothing: six agents each carrying zero, every
	load bar empty, "last active" blank for everybody. Assigning most of the
	open work — to somebody who is actually on the ticket's own department —
	makes those columns say something, and leaves a visible remainder in the
	unassigned queue for the screen that exists to clear it.
*/

// assignedShare is how much of the eligible queue gets an owner. The remainder
// is what a manager sees waiting in the Unassigned view.
const assignedShare = 0.72

// assignTicketsToAgents gives open tickets an assignee from their own
// department, and returns how many it placed.
//
// Resolved and closed tickets are skipped: they are history, and putting them on
// somebody's desk would inflate every workload number with work that is already
// done.
func assignTicketsToAgents(
	ctx context.Context,
	db *pgxpool.Pool,
	orgID uuid.UUID,
	byDepartment map[uuid.UUID][]uuid.UUID,
	rng *rand.Rand,
) (int, error) {
	rows, err := db.Query(ctx,
		`SELECT id, department_id FROM tickets
		 WHERE organization_id = $1 AND status NOT IN ('resolved', 'closed')
		 ORDER BY created_at`,
		orgID,
	)
	if err != nil {
		return 0, fmt.Errorf("read tickets for assignment: %w", err)
	}

	type pending struct {
		ticketID     uuid.UUID
		departmentID uuid.UUID
	}

	var queue []pending
	for rows.Next() {
		var p pending
		if err := rows.Scan(&p.ticketID, &p.departmentID); err != nil {
			rows.Close()
			return 0, fmt.Errorf("scan ticket for assignment: %w", err)
		}
		queue = append(queue, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate tickets for assignment: %w", err)
	}

	assigned := 0
	for _, p := range queue {
		agents := byDepartment[p.departmentID]
		// A department with nobody on it keeps its tickets unassigned, which is
		// exactly the signal the panel should show for an unstaffed team.
		if len(agents) == 0 || rng.Float64() > assignedShare {
			continue
		}

		agent := agents[rng.Intn(len(agents))]
		if _, err := db.Exec(ctx,
			`UPDATE tickets SET assignee_id = $1, updated_at = updated_at WHERE id = $2`,
			agent, p.ticketID,
		); err != nil {
			return 0, fmt.Errorf("assign ticket %s: %w", p.ticketID, err)
		}
		assigned++
	}

	return assigned, nil
}
