package eval

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/skillhub/api/internal/review"
)

// ReviewEvalCase represents a single test case for AI review
type ReviewEvalCase struct {
	Name     string `json:"name"`
	Content  string `json:"content"`
	Expected string `json:"expected"` // "approve" | "reject" | "revision_requested"
	Reason   string `json:"reason"`   // Why this is the expected outcome
	Category string `json:"category"` // "malicious" | "benign" | "edge_case"
}

// ReviewEvalSuite runs evaluation tests for the AI review system
type ReviewEvalSuite struct {
	Cases    []ReviewEvalCase
	Reviewer *review.Reviewer
}

// LoadEvalCases loads test cases from JSON file
func LoadEvalCases(filepath string) ([]ReviewEvalCase, error) {
	data, err := os.ReadFile(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to read eval cases: %w", err)
	}

	var cases []ReviewEvalCase
	if err := json.Unmarshal(data, &cases); err != nil {
		return nil, fmt.Errorf("failed to parse eval cases: %w", err)
	}

	return cases, nil
}

// RunEval executes all evaluation cases and returns results
func (s *ReviewEvalSuite) RunEval(ctx context.Context) (*EvalResults, error) {
	results := &EvalResults{
		Total:   len(s.Cases),
		Passed:  0,
		Failed:  0,
		Details: make([]EvalDetail, 0, len(s.Cases)),
	}

	for _, tc := range s.Cases {
		detail := s.runSingleCase(ctx, tc)
		results.Details = append(results.Details, detail)

		if detail.Passed {
			results.Passed++
		} else {
			results.Failed++
		}
	}

	results.Accuracy = float64(results.Passed) / float64(results.Total)
	return results, nil
}

func (s *ReviewEvalSuite) runSingleCase(ctx context.Context, tc ReviewEvalCase) EvalDetail {
	// Run regex scan first
	regexIssues := review.RegexScan(tc.Content)
	securityIssues := review.SecurityScan(tc.Content)

	var actual string
	if len(regexIssues) > 0 || len(securityIssues) > 0 {
		actual = "revision_requested"
	} else {
		// In real implementation, would call LLM reviewer here
		// For now, assume approve if no regex issues
		actual = "approve"
	}

	passed := actual == tc.Expected

	return EvalDetail{
		Name:     tc.Name,
		Category: tc.Category,
		Expected: tc.Expected,
		Actual:   actual,
		Passed:   passed,
		Reason:   tc.Reason,
	}
}

// EvalResults contains the results of running an eval suite
type EvalResults struct {
	Total    int          `json:"total"`
	Passed   int          `json:"passed"`
	Failed   int          `json:"failed"`
	Accuracy float64      `json:"accuracy"`
	Details  []EvalDetail `json:"details"`
}

// EvalDetail contains the result of a single eval case
type EvalDetail struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Expected string `json:"expected"`
	Actual   string `json:"actual"`
	Passed   bool   `json:"passed"`
	Reason   string `json:"reason"`
}

// PrintResults prints eval results in a human-readable format
func (r *EvalResults) PrintResults() {
	fmt.Printf("\n=== Review Eval Results ===\n")
	fmt.Printf("Total: %d | Passed: %d | Failed: %d | Accuracy: %.2f%%\n\n",
		r.Total, r.Passed, r.Failed, r.Accuracy*100)

	if r.Failed > 0 {
		fmt.Println("Failed Cases:")
		for _, d := range r.Details {
			if !d.Passed {
				fmt.Printf("  ❌ %s (%s)\n", d.Name, d.Category)
				fmt.Printf("     Expected: %s | Actual: %s\n", d.Expected, d.Actual)
				fmt.Printf("     Reason: %s\n\n", d.Reason)
			}
		}
	}

	// Print accuracy by category
	categoryStats := make(map[string]struct{ passed, total int })
	for _, d := range r.Details {
		stats := categoryStats[d.Category]
		stats.total++
		if d.Passed {
			stats.passed++
		}
		categoryStats[d.Category] = stats
	}

	fmt.Println("Accuracy by Category:")
	for cat, stats := range categoryStats {
		acc := float64(stats.passed) / float64(stats.total) * 100
		fmt.Printf("  %s: %.2f%% (%d/%d)\n", cat, acc, stats.passed, stats.total)
	}
}
