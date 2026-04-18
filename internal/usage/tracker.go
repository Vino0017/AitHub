package usage

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Tracker handles skill usage logging and statistics.
type Tracker struct {
	pool *pgxpool.Pool
}

func NewTracker(pool *pgxpool.Pool) *Tracker {
	return &Tracker{pool: pool}
}

// LogUsage records a skill usage event.
func (t *Tracker) LogUsage(ctx context.Context, skillID, tokenID uuid.UUID, action string) {
	t.pool.Exec(ctx,
		`INSERT INTO skill_usage_logs (skill_id, token_id, action) VALUES ($1, $2, $3)`,
		skillID, tokenID, action)

	// Update last_used_at
	t.pool.Exec(ctx,
		`UPDATE skills SET last_used_at = NOW() WHERE id = $1`, skillID)
}

// RefreshUsageStats recalculates DAU, MAU, retention rate, and zombie detection.
func (t *Tracker) RefreshUsageStats(ctx context.Context) error {
	// Calculate DAU (unique tokens in last 24h)
	_, err := t.pool.Exec(ctx,
		`UPDATE skills s SET dau = sub.dau
		 FROM (
		     SELECT skill_id, COUNT(DISTINCT token_id) AS dau
		     FROM skill_usage_logs
		     WHERE created_at > NOW() - INTERVAL '24 hours'
		     GROUP BY skill_id
		 ) sub
		 WHERE s.id = sub.skill_id`)
	if err != nil {
		return err
	}

	// Calculate MAU (unique tokens in last 30 days)
	_, err = t.pool.Exec(ctx,
		`UPDATE skills s SET mau = sub.mau
		 FROM (
		     SELECT skill_id, COUNT(DISTINCT token_id) AS mau
		     FROM skill_usage_logs
		     WHERE created_at > NOW() - INTERVAL '30 days'
		     GROUP BY skill_id
		 ) sub
		 WHERE s.id = sub.skill_id`)
	if err != nil {
		return err
	}

	// Calculate retention rate (users who used it 7+ days ago and again in last 7 days)
	_, err = t.pool.Exec(ctx,
		`UPDATE skills s SET retention_rate = sub.retention
		 FROM (
		     SELECT skill_id,
		            COALESCE(
		                COUNT(DISTINCT CASE WHEN recent.token_id IS NOT NULL THEN old.token_id END)::float /
		                NULLIF(COUNT(DISTINCT old.token_id), 0),
		                0
		            ) AS retention
		     FROM (
		         SELECT DISTINCT skill_id, token_id
		         FROM skill_usage_logs
		         WHERE created_at BETWEEN NOW() - INTERVAL '14 days' AND NOW() - INTERVAL '7 days'
		     ) old
		     LEFT JOIN (
		         SELECT DISTINCT skill_id, token_id
		         FROM skill_usage_logs
		         WHERE created_at > NOW() - INTERVAL '7 days'
		     ) recent ON old.skill_id = recent.skill_id AND old.token_id = recent.token_id
		     GROUP BY skill_id
		 ) sub
		 WHERE s.id = sub.skill_id`)
	if err != nil {
		return err
	}

	// Detect zombie skills (no usage in 30 days, or MAU < 3)
	_, err = t.pool.Exec(ctx,
		`UPDATE skills SET is_zombie = (
		     last_used_at IS NULL OR
		     last_used_at < NOW() - INTERVAL '30 days' OR
		     mau < 3
		 ) WHERE status = 'active'`)
	if err != nil {
		return err
	}

	return nil
}

// GetUsageStats returns usage statistics for a skill.
func (t *Tracker) GetUsageStats(ctx context.Context, skillID uuid.UUID) (UsageStats, error) {
	var stats UsageStats
	err := t.pool.QueryRow(ctx,
		`SELECT dau, mau, retention_rate, last_used_at, is_zombie
		 FROM skills WHERE id = $1`,
		skillID).Scan(&stats.DAU, &stats.MAU, &stats.RetentionRate, &stats.LastUsedAt, &stats.IsZombie)
	if err != nil {
		return stats, err
	}

	// Get usage trend (last 7 days)
	rows, err := t.pool.Query(ctx,
		`SELECT DATE(created_at) AS day, COUNT(DISTINCT token_id) AS users
		 FROM skill_usage_logs
		 WHERE skill_id = $1 AND created_at > NOW() - INTERVAL '7 days'
		 GROUP BY day
		 ORDER BY day DESC`, skillID)
	if err != nil {
		return stats, err
	}
	defer rows.Close()

	stats.DailyTrend = []DailyUsage{}
	for rows.Next() {
		var day time.Time
		var users int
		rows.Scan(&day, &users)
		stats.DailyTrend = append(stats.DailyTrend, DailyUsage{
			Date:  day,
			Users: users,
		})
	}

	return stats, nil
}

// UsageStats holds usage statistics for a skill.
type UsageStats struct {
	DAU           int           `json:"dau"`
	MAU           int           `json:"mau"`
	RetentionRate float64       `json:"retention_rate"`
	LastUsedAt    *time.Time    `json:"last_used_at,omitempty"`
	IsZombie      bool          `json:"is_zombie"`
	DailyTrend    []DailyUsage  `json:"daily_trend"`
}

type DailyUsage struct {
	Date  time.Time `json:"date"`
	Users int       `json:"users"`
}
