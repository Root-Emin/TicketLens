// Package stub provides a deterministic, keyword-driven Classifier.
//
// It exists so the backend runs, seeds and tests without the Python model
// service. Determinism is the point: the same ticket always yields the same
// label and the same confidence, so seed data and end-to-end tests are
// reproducible. When the real model lands, an http adapter sits beside this
// package and port.Classifier does not change.
package stub

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/model"
	"github.com/Root-Emin/TicketLens/internal/domain/triage/port"
)

const (
	modelName    = "stub"
	modelVersion = "v0"

	// Confidence is derived from how many keywords matched, bounded to a band
	// that straddles a realistic review threshold so needs_human_review is
	// actually exercised.
	minConfidence  = 0.50
	maxConfidence  = 0.95
	confidenceStep = 0.15

	// subjectWeight makes a signal in the subject count for more than the same
	// signal buried in a long body, where an incidental mention is likelier.
	subjectWeight = 2
)

// Classifier is a keyword-matching implementation of port.Classifier.
type Classifier struct{}

// New creates a new stub classifier.
func New() *Classifier { return &Classifier{} }

// Verify interface compliance at compile time.
var _ port.Classifier = (*Classifier)(nil)

// categoryKeywords maps each taxonomy label to the terms that signal it.
// Order matters only for tie-breaking, which is resolved deterministically by
// the fixed order of model.AllCategories.
var categoryKeywords = map[model.Category][]string{
	model.CategoryIntegration: {
		"integration", "integrations", "trendyol", "hepsiburada", "n11", "marketplace",
		"cargo", "shipping", "erp", "logo", "mikro", "netsis", "connector",
		"api", "sdk", "webhook", "sync", "synchronization", "not transferring", "virtual pos",
	},
	model.CategoryPaymentOps: {
		"payout", "settlement", "reconciliation", "refund", "chargeback", "dispute",
		"payment", "not credited", "transaction", "deposit", "collect payments",
		"not received", "money came back", "money transfer",
	},
	model.CategoryBilling: {
		"invoice", "subscription", "package", "plan", "commission", "billing",
		"cancel subscription", "upgrade", "downgrade", "pricing tier",
	},
	model.CategoryTechnicalIssue: {
		"error", "500", "crashed", "won't open", "will not open", "not loading",
		"slow", "down", "site is down", "timeout", "bug", "white screen",
		"fails", "won't upload",
	},
	model.CategoryOnboarding: {
		"setup", "onboarding", "data migration", "migration", "go live", "go-live",
		"activation", "getting started", "account setup",
	},
	model.CategoryHowTo: {
		"how to", "how do", "how can", "how is", "how should", "training",
		"documentation", "docs", "guide", "manual", "learn", "configure", "tutorial",
	},
	model.CategoryAccountAccess: {
		"password", "reset password", "can't log in", "cannot log in", "log in",
		"login", "session", "permission", "role", "add user", "add a new user",
		"access", "2fa",
	},
	model.CategoryFeatureRequest: {
		"feature request", "roadmap", "suggestion", "would be great",
		"do you support", "can you add", "please add",
	},
	model.CategorySales: {
		"purchase", "buy", "demo", "quote", "pricing", "price list",
		"add-on module", "sales", "new package",
	},
	model.CategoryCompliance: {
		"kvkk", "gdpr", "contract", "data deletion", "delete my data", "audit",
		"compliance", "privacy notice", "document request",
	},
}

// urgentKeywords mark a stopped revenue stream, not an angry tone. Urgency is
// about the business being unable to operate.
var urgentKeywords = []string{
	"site is down", "site down", "crashed", "cannot process payments",
	"can't process payments", "payout", "not credited to my account", "cannot ship",
	"can't ship", "cannot receive payment", "cannot sell", "can't sell",
	"cannot collect", "no orders", "urgent", "production down", "customers are affected",
}

// highKeywords are real breakage that has not stopped the whole business.
var highKeywords = []string{
	"error", "500", "not transferring", "not syncing", "not working",
	"failed", "fails", "broken", "missing", "refund", "chargeback",
}

// lowKeywords are questions and requests: nothing is broken.
var lowKeywords = []string{
	"how to", "how do", "how can", "learn", "documentation", "docs",
	"guide", "training", "suggestion", "feature request", "demo", "quote", "pricing",
}

// Classify scores the ticket text against the keyword tables.
func (c *Classifier) Classify(_ context.Context, in port.ClassifyInput) (port.ClassifyResult, error) {
	subject := normalize(in.Subject)
	body := normalize(in.Body)

	category, categoryHits := c.bestCategory(subject, body)
	priority, priorityHits := c.priority(subject, body)

	raw, _ := json.Marshal(map[string]any{
		"engine":         modelName,
		"category_hits":  categoryHits,
		"priority_hits":  priorityHits,
		"matched_length": len(subject) + len(body),
	})

	return port.ClassifyResult{
		Priority:           string(priority),
		PriorityConfidence: confidence(priorityHits),
		Category:           string(category),
		CategoryConfidence: confidence(categoryHits),
		ModelName:          modelName,
		ModelVersion:       modelVersion,
		Raw:                raw,
	}, nil
}

// bestCategory returns the highest scoring label. Ties break on the fixed order
// of model.AllCategories so the result never depends on map iteration order.
func (c *Classifier) bestCategory(subject, body string) (model.Category, int) {
	type scored struct {
		category model.Category
		hits     int
		rank     int
	}

	results := make([]scored, 0, len(model.AllCategories))
	for rank, category := range model.AllCategories {
		results = append(results, scored{
			category: category,
			hits:     score(subject, body, categoryKeywords[category]),
			rank:     rank,
		})
	}

	sort.SliceStable(results, func(i, j int) bool {
		if results[i].hits != results[j].hits {
			return results[i].hits > results[j].hits
		}
		return results[i].rank < results[j].rank
	})

	best := results[0]
	if best.hits == 0 {
		// Nothing matched. There is no "other" class by design, so fall back to
		// the broadest label at minimum confidence and let needs_human_review
		// flag it.
		return model.CategoryHowTo, 0
	}
	return best.category, best.hits
}

// priority follows revenue impact: a stopped business is urgent, visible
// breakage is high, questions are low.
func (c *Classifier) priority(subject, body string) (model.TicketPriority, int) {
	if hits := score(subject, body, urgentKeywords); hits > 0 {
		return model.TicketPriorityUrgent, hits
	}
	if hits := score(subject, body, highKeywords); hits > 0 {
		return model.TicketPriorityHigh, hits
	}
	if hits := score(subject, body, lowKeywords); hits > 0 {
		return model.TicketPriorityLow, hits
	}
	return model.TicketPriorityNormal, 0
}

// confidence turns a hit count into a bounded score. Zero hits sit at the floor
// so an unmatched ticket falls under any sensible review threshold.
func confidence(hits int) float64 {
	score := minConfidence + float64(hits)*confidenceStep
	if score > maxConfidence {
		return maxConfidence
	}
	return score
}

// score counts the distinct keywords present, crediting the subject more.
//
// Each keyword counts at most once per field, so a word repeated inside one
// ticket does not inflate the score. The subject used to be concatenated twice
// to weight it, which had no effect for exactly that reason.
func score(subject, body string, keywords []string) int {
	return subjectWeight*countMatches(subject, keywords) + countMatches(body, keywords)
}

func countMatches(text string, keywords []string) int {
	count := 0
	for _, keyword := range keywords {
		if strings.Contains(text, normalize(keyword)) {
			count++
		}
	}
	return count
}

func normalize(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
