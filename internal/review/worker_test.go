package review

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/rivertype"
)

// TestReviewJobArgs_Kind tests job kind identifier
func TestReviewJobArgs_Kind(t *testing.T) {
	args := ReviewJobArgs{RevisionID: "test-id"}

	kind := args.Kind()

	if kind != "skill_review" {
		t.Errorf("Expected kind 'skill_review', got '%s'", kind)
	}
}

// TestReviewJobArgs_InsertOpts tests job insert options
func TestReviewJobArgs_InsertOpts(t *testing.T) {
	args := ReviewJobArgs{RevisionID: "test-id"}

	opts := args.InsertOpts()

	if opts.Queue != "review" {
		t.Errorf("Expected queue 'review', got '%s'", opts.Queue)
	}
	if opts.MaxAttempts != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", opts.MaxAttempts)
	}
}

// TestNewReviewWorker tests worker creation
func TestNewReviewWorker(t *testing.T) {
	reviewer := &Reviewer{}

	worker := NewReviewWorker(reviewer)

	if worker == nil {
		t.Fatal("Expected non-nil worker")
	}
	if worker.reviewer != reviewer {
		t.Error("Expected reviewer to be set")
	}
}

// TestReviewWorker_Work_ValidID tests successful job processing
func TestReviewWorker_Work_ValidID(t *testing.T) {
	t.Skip("Requires database setup")

	// Create mock reviewer
	reviewer := &Reviewer{}
	worker := NewReviewWorker(reviewer)

	// Create test job
	revisionID := uuid.New()
	job := &river.Job[ReviewJobArgs]{
		JobRow: &rivertype.JobRow{
			Attempt: 1,
		},
		Args: ReviewJobArgs{
			RevisionID: revisionID.String(),
		},
	}

	// Execute work
	err := worker.Work(context.Background(), job)

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

// TestReviewWorker_Work_InvalidID tests invalid revision ID handling
func TestReviewWorker_Work_InvalidID(t *testing.T) {
	reviewer := &Reviewer{}
	worker := NewReviewWorker(reviewer)

	// Create job with invalid ID
	job := &river.Job[ReviewJobArgs]{
		JobRow: &rivertype.JobRow{
			Attempt: 1,
		},
		Args: ReviewJobArgs{
			RevisionID: "invalid-uuid",
		},
	}

	// Execute work
	err := worker.Work(context.Background(), job)

	// Should return nil (don't retry on invalid ID)
	if err != nil {
		t.Errorf("Expected nil error for invalid ID, got: %v", err)
	}
}

// TestReviewWorker_Work_EmptyID tests empty revision ID handling
func TestReviewWorker_Work_EmptyID(t *testing.T) {
	reviewer := &Reviewer{}
	worker := NewReviewWorker(reviewer)

	// Create job with empty ID
	job := &river.Job[ReviewJobArgs]{
		JobRow: &rivertype.JobRow{
			Attempt: 1,
		},
		Args: ReviewJobArgs{
			RevisionID: "",
		},
	}

	// Execute work
	err := worker.Work(context.Background(), job)

	// Should return nil (don't retry on invalid ID)
	if err != nil {
		t.Errorf("Expected nil error for empty ID, got: %v", err)
	}
}

// TestReviewWorker_Work_MultipleAttempts tests retry attempts
func TestReviewWorker_Work_MultipleAttempts(t *testing.T) {
	t.Skip("Requires database setup")

	reviewer := &Reviewer{}
	worker := NewReviewWorker(reviewer)

	revisionID := uuid.New()

	// Test different attempt numbers
	for attempt := 1; attempt <= 5; attempt++ {
		job := &river.Job[ReviewJobArgs]{
			JobRow: &rivertype.JobRow{
				Attempt: attempt,
			},
			Args: ReviewJobArgs{
				RevisionID: revisionID.String(),
			},
		}

		err := worker.Work(context.Background(), job)
		if err != nil {
			t.Errorf("Attempt %d: expected no error, got: %v", attempt, err)
		}
	}
}
