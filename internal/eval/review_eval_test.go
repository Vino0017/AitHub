// +build eval

package eval

import (
	"context"
	"testing"
)

// TestReviewEval runs the evaluation suite
func TestReviewEval(t *testing.T) {
	cases, err := LoadEvalCases("testdata/review_cases.json")
	if err != nil {
		t.Fatalf("Failed to load eval cases: %v", err)
	}

	if len(cases) == 0 {
		t.Fatal("No eval cases loaded")
	}

	suite := &ReviewEvalSuite{
		Cases:    cases,
		Reviewer: nil, // No actual LLM reviewer for now
	}

	results, err := suite.RunEval(context.Background())
	if err != nil {
		t.Fatalf("Eval failed: %v", err)
	}

	results.PrintResults()

	// For now, we only test regex scanner (no LLM)
	// So we expect some failures on edge cases
	const minAccuracy = 0.70 // 70% accuracy with regex only
	if results.Accuracy < minAccuracy {
		t.Errorf("Accuracy %.2f%% is below threshold %.2f%%",
			results.Accuracy*100, minAccuracy*100)
	}

	t.Logf("Eval completed: %d/%d passed (%.2f%%)",
		results.Passed, results.Total, results.Accuracy*100)
}
