package postgres

import "testing"

func TestExtractGooseUp(t *testing.T) {
	src := `-- +goose Up
CREATE TABLE foo (id INT);
-- +goose Down
DROP TABLE foo;
`
	got := extractGooseUp(src)
	if got != "CREATE TABLE foo (id INT);" {
		t.Fatalf("got %q", got)
	}
}

func TestExtractGooseUp_WholeFileWhenUnmarked(t *testing.T) {
	src := "CREATE TABLE bar (id INT);"
	if extractGooseUp(src) != src {
		t.Fatal("expected whole file")
	}
}
