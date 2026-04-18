package credibility

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Analyzer handles rating credibility analysis and anti-cheating.
type Analyzer struct {
	pool *pgxpool.Pool
}

func NewAnalyzer(pool *pgxpool.Pool) *Analyzer {
	return &Analyzer{pool: pool}
}

// CalculateConfidence computes a confidence score (0.0-1.0) for a rating.
// Factors: execution time, error details, context richness, token history.
func (a *Analyzer) CalculateConfidence(ctx context.Context, rating RatingInput) float64 {
	confidence := 1.0

	// Factor 1: Execution time plausibility (too fast = suspicious)
	if rating.ExecutionTimeMs > 0 {
		if rating.ExecutionTimeMs < 100 {
			// < 100ms is suspiciously fast for real AI execution
			confidence *= 0.3
		} else if rating.ExecutionTimeMs < 500 {
			confidence *= 0.6
		} else if rating.ExecutionTimeMs > 300000 {
			// > 5 minutes is suspiciously slow
			confidence *= 0.7
		}
	} else {
		// No execution time provided = less confident
		confidence *= 0.8
	}

	// Factor 2: Context metadata richness
	if rating.ContextMetadata != nil {
		var meta map[string]interface{}
		if err := json.Unmarshal(rating.ContextMetadata, &meta); err == nil {
			// More context = higher confidence
			if len(meta) >= 3 {
				confidence *= 1.1
			} else if len(meta) == 0 {
				confidence *= 0.7
			}
		}
	} else {
		confidence *= 0.8
	}

	// Factor 3: Error details for failures
	if rating.Outcome == "failure" || rating.Outcome == "partial" {
		if rating.ErrorDetails != nil || rating.FailureReason != "" {
			confidence *= 1.05 // Good: provided error context
		} else {
			confidence *= 0.6 // Suspicious: failure without details
		}
	}

	// Factor 4: Token history (check for spam patterns)
	suspiciousFlags := a.CheckTokenHistory(ctx, rating.TokenID, rating.SkillID)
	if len(suspiciousFlags) > 0 {
		confidence *= math.Pow(0.8, float64(len(suspiciousFlags)))
	}

	// Clamp to [0.0, 1.0]
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// CheckTokenHistory detects suspicious rating patterns (exported for handler use).
func (a *Analyzer) CheckTokenHistory(ctx context.Context, tokenID, skillID uuid.UUID) []string {
	flags := []string{}

	// Check 1: Rating velocity (too many ratings in short time)
	var recentCount int
	a.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ratings WHERE token_id = $1 AND created_at > NOW() - INTERVAL '1 hour'`,
		tokenID).Scan(&recentCount)
	if recentCount > 20 {
		flags = append(flags, "high_velocity")
	}

	// Check 2: Score variance (always same score = bot)
	var variance float64
	a.pool.QueryRow(ctx,
		`SELECT VARIANCE(score) FROM ratings WHERE token_id = $1 AND created_at > NOW() - INTERVAL '7 days'`,
		tokenID).Scan(&variance)
	if variance < 0.5 && recentCount > 5 {
		flags = append(flags, "low_variance")
	}

	// Check 3: Same skill rated multiple times in short period
	var sameSkillCount int
	a.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM ratings WHERE token_id = $1 AND skill_id = $2 AND created_at > NOW() - INTERVAL '1 hour'`,
		tokenID, skillID).Scan(&sameSkillCount)
	if sameSkillCount > 3 {
		flags = append(flags, "repeated_skill")
	}

	// Check 4: All ratings are extreme (1 or 10)
	var extremeRatio float64
	a.pool.QueryRow(ctx,
		`SELECT COALESCE(SUM(CASE WHEN score IN (1, 10) THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0), 0)
		 FROM ratings WHERE token_id = $1 AND created_at > NOW() - INTERVAL '7 days'`,
		tokenID).Scan(&extremeRatio)
	if extremeRatio > 0.8 && recentCount > 10 {
		flags = append(flags, "extreme_scores")
	}

	return flags
}

// UpdateRatingPattern updates the rating_patterns table for anomaly tracking.
func (a *Analyzer) UpdateRatingPattern(ctx context.Context, tokenID, skillID uuid.UUID, score int, suspiciousFlags []string) {
	flagsJSON, _ := json.Marshal(suspiciousFlags)

	a.pool.Exec(ctx,
		`INSERT INTO rating_patterns (token_id, skill_id, rating_count, avg_score, last_rating_at, suspicious_flags)
		 VALUES ($1, $2, 1, $3, NOW(), $4)
		 ON CONFLICT (token_id, skill_id) DO UPDATE SET
		   rating_count = rating_patterns.rating_count + 1,
		   avg_score = (rating_patterns.avg_score * rating_patterns.rating_count + $3) / (rating_patterns.rating_count + 1),
		   last_rating_at = NOW(),
		   suspicious_flags = $4,
		   updated_at = NOW()`,
		tokenID, skillID, float64(score), flagsJSON)
}

// CalculateSkillCredibility computes overall credibility score for a skill.
// Based on: rating confidence distribution, pattern anomalies, cross-validation.
func (a *Analyzer) CalculateSkillCredibility(ctx context.Context, skillID uuid.UUID) float64 {
	// Get latest approved revision
	var revisionID uuid.UUID
	err := a.pool.QueryRow(ctx,
		`SELECT id FROM revisions WHERE skill_id = $1 AND review_status = 'approved'
		 ORDER BY created_at DESC LIMIT 1`, skillID).Scan(&revisionID)
	if err != nil {
		return 0.5 // Default credibility
	}

	// Calculate average confidence of all ratings
	var avgConfidence float64
	var ratingCount int
	a.pool.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(AVG(confidence_score), 0.5)
		 FROM ratings r JOIN tokens t ON r.token_id = t.id
		 WHERE r.revision_id = $1 AND t.namespace_id IS NOT NULL`,
		revisionID).Scan(&ratingCount, &avgConfidence)

	if ratingCount == 0 {
		return 0.5
	}

	// Penalty for low sample size
	samplePenalty := 1.0
	if ratingCount < 5 {
		samplePenalty = 0.7
	} else if ratingCount < 10 {
		samplePenalty = 0.85
	}

	// Check for suspicious patterns
	var suspiciousCount int
	a.pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM rating_patterns
		 WHERE skill_id = $1 AND jsonb_array_length(suspicious_flags) > 0`,
		skillID).Scan(&suspiciousCount)

	suspiciousPenalty := 1.0
	if suspiciousCount > 0 {
		suspiciousPenalty = math.Pow(0.9, float64(suspiciousCount))
	}

	credibility := avgConfidence * samplePenalty * suspiciousPenalty

	// Clamp to [0.0, 1.0]
	if credibility > 1.0 {
		credibility = 1.0
	}
	if credibility < 0.0 {
		credibility = 0.0
	}

	return credibility
}

// RatingInput holds the data needed for confidence calculation.
type RatingInput struct {
	TokenID         uuid.UUID
	SkillID         uuid.UUID
	Score           int
	Outcome         string
	ExecutionTimeMs int
	FailureReason   string
	ErrorDetails    json.RawMessage
	ContextMetadata json.RawMessage
}
