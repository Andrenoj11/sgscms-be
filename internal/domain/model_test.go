package domain

import (
	"github.com/google/uuid"
	"testing"
	"time"
)

func TestScheduledNewsRequiresFutureDate(t *testing.T) {
	id := uuid.New()
	n := News{Title: "x", Slug: "x", Content: "x", CategoryID: &id, CreatedBy: uuid.New(), Status: NewsScheduled}
	past := time.Now().Add(-time.Hour)
	n.ScheduledAt = &past
	if n.ValidateForPublication(time.Now()) == nil {
		t.Fatal("past schedule accepted")
	}
}
func TestActiveTeamRequiredFields(t *testing.T) {
	if (TeamMember{Name: "A", Slug: "a"}).ValidateForActivation() == nil {
		t.Fatal("incomplete profile accepted")
	}
}
