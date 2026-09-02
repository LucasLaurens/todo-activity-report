package domain_test

import (
	"sort"
	"testing"

	"github.com/LucasLaurens/todo-activity-report/internal/domain"
)

func TestClassifyStatusOnTheStatusesTheBoardUses(t *testing.T) {
	t.Parallel()
	// Arrange — 15 status names observed on a real board over a month.
	cases := map[string]domain.StatusKind{
		"Bloqué":                   domain.StatusBlocked,
		"Blocked":                  domain.StatusBlocked,
		"En attente retour client": domain.StatusBlocked,
		"Deployed / Completed":     domain.StatusDone,
		"Waiting for release":      domain.StatusDone,
		"En production":            domain.StatusDone,
		"Fermée":                   domain.StatusDone,
		"Annulé":                   domain.StatusAbandoned,
		"Obsolète":                 domain.StatusAbandoned,
		"Cancelled":                domain.StatusAbandoned,
		"In progress dev":          domain.StatusInFlight,
		"Code Review":              domain.StatusInFlight,
		"To test":                  domain.StatusInFlight,
		"Waiting for feat env":     domain.StatusInFlight,
		"À faire":                  domain.StatusNotStarted,
		"Backlog":                  domain.StatusNotStarted,
		"À spécifier":              domain.StatusNotStarted,
		"A QUALIFIER":              domain.StatusNotStarted,
	}

	for status, expected := range cases {
		// Act
		kind := domain.ClassifyStatus(status)

		// Assert
		if kind != expected {
			t.Fatalf("ClassifyStatus(%q) = %v, want %v", status, kind, expected)
		}
	}
}

func TestAttentionRankReadsBlockersFirst(t *testing.T) {
	t.Parallel()
	// Arrange
	kinds := []domain.StatusKind{
		domain.StatusAbandoned,
		domain.StatusDone,
		domain.StatusNotStarted,
		domain.StatusInFlight,
		domain.StatusBlocked,
	}

	// Act
	sort.Slice(kinds, func(i, j int) bool { return kinds[i].AttentionRank() < kinds[j].AttentionRank() })

	// Assert
	expected := []domain.StatusKind{
		domain.StatusBlocked,
		domain.StatusInFlight,
		domain.StatusDone,
		domain.StatusNotStarted,
		domain.StatusAbandoned,
	}
	for i, kind := range expected {
		if kinds[i] != kind {
			t.Fatalf("rank %d = %v, want %v (got %v)", i, kinds[i], kind, kinds)
		}
	}
}

func TestStatusKindTreatsABlockingCommentAsBlocked(t *testing.T) {
	t.Parallel()
	// Arrange — the status says the work is moving, the thread says otherwise.
	task := domain.Task{
		Status:   "In progress dev",
		Comments: []domain.Comment{{Author: "Ada", Body: "On est bloqué par l'environnement de recette."}},
	}

	// Act
	kind := task.StatusKind()

	// Assert
	if kind != domain.StatusBlocked {
		t.Fatalf("expected a blocking comment to outrank the status, got %v", kind)
	}
}

func TestStatusKindKeepsTheStatusWithoutABlockingComment(t *testing.T) {
	t.Parallel()
	// Arrange
	task := domain.Task{
		Status:   "In progress dev",
		Comments: []domain.Comment{{Author: "Ada", Body: "Revue faite, je pousse la correction."}},
	}

	// Act & Assert
	if task.StatusKind() != domain.StatusInFlight {
		t.Fatalf("expected the status to stand, got %v", task.StatusKind())
	}
	if task.HasBlockingComment() {
		t.Fatal("expected no blocking comment")
	}
}
