// Package httpclassifier is a port.Classifier that calls the Python inference
// service over HTTP. The request/response fields match ClassifyInput /
// ClassifyResult exactly so the Go use case never learns about transport.
package httpclassifier

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/port"
	stubClassifier "github.com/Root-Emin/TicketLens/internal/infrastructure/classifier/stub"
)

// Config drives the HTTP classifier adapter.
type Config struct {
	URL            string
	Timeout        time.Duration
	MaxRetries     int
	FallbackToStub bool
}

// Client implements port.Classifier against the FastAPI /classify endpoint.
type Client struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
	fallback   port.Classifier
	logger     *slog.Logger

	// hooks for tests / metrics
	onFailure func(attempt int, err error)
	onSuccess func(attempts int)
}

// New builds a Client. When FallbackToStub is true, a local keyword stub is
// used after exhausted retries so ticket creation still yields an analysis.
func New(cfg Config, logger *slog.Logger) *Client {
	if cfg.Timeout <= 0 {
		cfg.Timeout = 5 * time.Second
	}
	if cfg.MaxRetries < 0 {
		cfg.MaxRetries = 0
	}
	c := &Client{
		baseURL: strings.TrimRight(cfg.URL, "/"),
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		maxRetries: cfg.MaxRetries,
		logger:     logger,
	}
	if cfg.FallbackToStub {
		c.fallback = stubClassifier.New()
	}
	return c
}

// Verify interface compliance at compile time.
var _ port.Classifier = (*Client)(nil)

type classifyRequest struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}

type classifyResponse struct {
	Priority           string          `json:"priority"`
	PriorityConfidence float64         `json:"priority_confidence"`
	Category           string          `json:"category"`
	CategoryConfidence float64         `json:"category_confidence"`
	ModelName          string          `json:"model_name"`
	ModelVersion       string          `json:"model_version"`
	Raw                json.RawMessage `json:"raw"`
}

// Classify posts the ticket text to the model service.
func (c *Client) Classify(ctx context.Context, in port.ClassifyInput) (port.ClassifyResult, error) {
	var lastErr error
	attempts := c.maxRetries + 1
	for attempt := 1; attempt <= attempts; attempt++ {
		result, err := c.once(ctx, in)
		if err == nil {
			if c.onSuccess != nil {
				c.onSuccess(attempt)
			}
			return result, nil
		}
		lastErr = err
		if c.onFailure != nil {
			c.onFailure(attempt, err)
		}
		if c.logger != nil {
			c.logger.Warn("classifier HTTP call failed",
				"attempt", attempt,
				"max_attempts", attempts,
				"error", err,
			)
		}
		if attempt < attempts {
			// Simple linear backoff; keeps the consumer from stampeding a
			// cold-starting model pod.
			select {
			case <-ctx.Done():
				return port.ClassifyResult{}, ctx.Err()
			case <-time.After(time.Duration(attempt) * 200 * time.Millisecond):
			}
		}
	}

	if c.fallback != nil {
		if c.logger != nil {
			c.logger.Error("classifier HTTP exhausted retries; falling back to stub",
				"error", lastErr)
		}
		return c.fallback.Classify(ctx, in)
	}
	return port.ClassifyResult{}, fmt.Errorf("classifier HTTP failed after %d attempts: %w", attempts, lastErr)
}

func (c *Client) once(ctx context.Context, in port.ClassifyInput) (port.ClassifyResult, error) {
	payload, err := json.Marshal(classifyRequest{Subject: in.Subject, Body: in.Body})
	if err != nil {
		return port.ClassifyResult{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/classify", bytes.NewReader(payload))
	if err != nil {
		return port.ClassifyResult{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return port.ClassifyResult{}, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return port.ClassifyResult{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return port.ClassifyResult{}, fmt.Errorf("classifier returned %d: %s", resp.StatusCode, truncate(string(body), 200))
	}

	var parsed classifyResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return port.ClassifyResult{}, fmt.Errorf("decode classifier response: %w", err)
	}

	raw := parsed.Raw
	if len(raw) == 0 {
		raw = json.RawMessage(`{}`)
	}

	return port.ClassifyResult{
		Priority:           parsed.Priority,
		PriorityConfidence: parsed.PriorityConfidence,
		Category:           parsed.Category,
		CategoryConfidence: parsed.CategoryConfidence,
		ModelName:          parsed.ModelName,
		ModelVersion:       parsed.ModelVersion,
		Raw:                raw,
	}, nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
