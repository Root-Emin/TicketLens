package main

import (
	"context"
	"testing"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/port"
	stubClassifier "github.com/Root-Emin/TicketLens/internal/infrastructure/classifier/stub"
)

// reviewThreshold mirrors the default in config (CLASSIFIER_REVIEW_THRESHOLD).
// A category confidence below it forces needs_human_review in the analyzer.
const reviewThreshold = 0.60

// TestSeedCorpusClassifies is the acceptance guard for the language switch.
//
// Every non-ambiguous ticket must classify to the category its text was written
// to trigger, and every ambiguous ticket must land under the review threshold so
// the low-confidence path stays exercised. If a future edit to the corpus or the
// stub keyword tables breaks either property, the demo's AI metrics stop meaning
// what the docs say they mean, and this fails loudly instead.
func TestSeedCorpusClassifies(t *testing.T) {
	classifier := stubClassifier.New()

	var ambiguous int
	for _, ticket := range demoTickets {
		result, err := classifier.Classify(context.Background(), port.ClassifyInput{
			Subject: ticket.Subject,
			Body:    ticket.Body,
		})
		if err != nil {
			t.Fatalf("classify %q: %v", ticket.Subject, err)
		}

		if ticket.Category == ambiguousCategory {
			ambiguous++
			if result.CategoryConfidence >= reviewThreshold {
				t.Errorf("ambiguous ticket %q scored %.2f, expected < %.2f (must need review)",
					ticket.Subject, result.CategoryConfidence, reviewThreshold)
			}
			continue
		}

		if result.Category != string(ticket.Category) {
			t.Errorf("ticket %q classified as %q, expected %q",
				ticket.Subject, result.Category, ticket.Category)
		}
	}

	if ambiguous != 7 {
		t.Errorf("expected 7 ambiguous tickets, found %d", ambiguous)
	}
}
