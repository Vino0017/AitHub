package review

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/riverqueue/river"
)

// ReviewJobArgs defines the arguments for the review job.
type ReviewJobArgs struct {
	RevisionID string `json:"revision_id"`
}

// Kind returns the unique job identifier.
func (ReviewJobArgs) Kind() string { return "skill_review" }

// InsertOpts returns default options for the review job (e.g., retry policy).
func (ReviewJobArgs) InsertOpts() river.InsertOpts {
	return river.InsertOpts{
		Queue:      "review",
		MaxAttempts: 5,
	}
}

// ReviewWorker processes review jobs from the River queue.
type ReviewWorker struct {
	river.WorkerDefaults[ReviewJobArgs]
	reviewer *Reviewer
}

// NewReviewWorker creates a ReviewWorker with the given Reviewer.
func NewReviewWorker(reviewer *Reviewer) *ReviewWorker {
	return &ReviewWorker{reviewer: reviewer}
}

// Work executes the two-layer review on a revision.
func (w *ReviewWorker) Work(ctx context.Context, job *river.Job[ReviewJobArgs]) error {
	revID, err := uuid.Parse(job.Args.RevisionID)
	if err != nil {
		log.Printf("review worker: invalid revision ID %q: %v", job.Args.RevisionID, err)
		return nil // Don't retry on invalid ID
	}

	log.Printf("review worker: processing revision %s (attempt %d)", revID, job.Attempt)
	w.reviewer.Review(ctx, revID)
	return nil
}
