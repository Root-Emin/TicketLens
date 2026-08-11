// Package health provides liveness and readiness HTTP probes.
package health

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/Root-Emin/TicketLens/internal/shared/response"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// Handler provides health check endpoints.
type Handler struct {
	db    dbPinger
	redis redisPinger
}

type dbPinger interface {
	Ping(ctx context.Context) error
}

type redisPinger interface {
	Ping(ctx context.Context) *redis.StatusCmd
}

// NewHandler creates a new health handler.
//
// A nil pool or client is stored as a nil interface rather than a typed nil:
// assigning the typed nil straight through would leave the interface non-nil,
// and the readiness probe would call Ping on it and panic. Readiness reports
// an absent dependency as not_configured instead — see Readiness.
func NewHandler(db *pgxpool.Pool, redisClient *redis.Client) *Handler {
	h := &Handler{}
	if db != nil {
		h.db = db
	}
	if redisClient != nil {
		h.redis = redisClient
	}
	return h
}

// HealthResponse is the JSON structure for health checks.
type HealthResponse struct {
	Status   string            `json:"status"`
	Services map[string]string `json:"services"`
}

// Liveness returns 200 if the server is alive.
func (h *Handler) Liveness(w http.ResponseWriter, r *http.Request) {
	response.JSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

// Readiness checks the database and cache connectivity.
//
// A dependency that was never wired up counts as not ready, not as absent.
// Reporting "ready" for a process that booted without its database would tell a
// load balancer to send it traffic that can only 500.
func (h *Handler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	services := make(map[string]string)
	healthy := true

	// Check Postgres
	switch {
	case h.db == nil:
		services["postgres"] = "not_configured"
		healthy = false
	default:
		if err := h.db.Ping(ctx); err != nil {
			slog.Error("readiness check failed", "service", "postgres", "error", err)
			services["postgres"] = "unhealthy"
			healthy = false
		} else {
			services["postgres"] = "healthy"
		}
	}

	// Check Redis
	switch {
	case h.redis == nil:
		services["redis"] = "not_configured"
		healthy = false
	default:
		if err := h.redis.Ping(ctx).Err(); err != nil {
			slog.Error("readiness check failed", "service", "redis", "error", err)
			services["redis"] = "unhealthy"
			healthy = false
		} else {
			services["redis"] = "healthy"
		}
	}

	status := "ready"
	code := http.StatusOK
	if !healthy {
		status = "not ready"
		code = http.StatusServiceUnavailable
	}

	response.JSON(w, code, HealthResponse{
		Status:   status,
		Services: services,
	})
}
