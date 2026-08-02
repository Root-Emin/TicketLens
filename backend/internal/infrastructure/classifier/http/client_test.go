package httpclassifier

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Root-Emin/TicketLens/internal/domain/triage/port"
)

func TestClientClassifySuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/classify" {
			t.Fatalf("path = %s", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"priority":            "high",
			"priority_confidence": 0.82,
			"category":            "integration",
			"category_confidence": 0.91,
			"model_name":          "test-model",
			"model_version":       "v1",
			"raw":                 map[string]any{"ok": true},
		})
	}))
	defer srv.Close()

	c := New(Config{URL: srv.URL, Timeout: 0, MaxRetries: 0}, nil)
	got, err := c.Classify(context.Background(), port.ClassifyInput{
		Subject: "sync broken",
		Body:    "marketplace integration failing",
	})
	if err != nil {
		t.Fatal(err)
	}
	if got.Category != "integration" || got.Priority != "high" {
		t.Fatalf("got %+v", got)
	}
	if got.ModelName != "test-model" {
		t.Fatalf("model = %s", got.ModelName)
	}
}

func TestClientFallsBackToStub(t *testing.T) {
	failures := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		failures++
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer srv.Close()

	c := New(Config{
		URL:            srv.URL,
		Timeout:        0,
		MaxRetries:     1,
		FallbackToStub: true,
	}, nil)

	got, err := c.Classify(context.Background(), port.ClassifyInput{
		Subject: "How do I create a campaign?",
		Body:    "Looking for documentation or a guide.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if failures < 2 {
		t.Fatalf("expected retries, failures=%d", failures)
	}
	if got.ModelName != "stub" {
		t.Fatalf("expected stub fallback, got %s", got.ModelName)
	}
}

func TestClientNoFallbackReturnsError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "down", http.StatusBadGateway)
	}))
	defer srv.Close()

	c := New(Config{URL: srv.URL, MaxRetries: 0, FallbackToStub: false}, nil)
	_, err := c.Classify(context.Background(), port.ClassifyInput{Subject: "x", Body: "y"})
	if err == nil {
		t.Fatal("expected error")
	}
}
