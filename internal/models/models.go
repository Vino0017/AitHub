package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Namespace represents a user or organization.
type Namespace struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	Type      string     `json:"type"` // personal | org
	GitHubID  *string    `json:"github_id,omitempty"`
	GoogleID  *string    `json:"google_id,omitempty"`
	Email     *string    `json:"email,omitempty"`
	Banned    bool       `json:"banned"`
	CreatedAt time.Time  `json:"created_at"`
}

// OrgMember represents a membership in an organization.
type OrgMember struct {
	OrgID    uuid.UUID `json:"org_id"`
	MemberID uuid.UUID `json:"member_id"`
	Role     string    `json:"role"` // owner | member
	JoinedAt time.Time `json:"joined_at"`
}

// Token represents an API token.
type Token struct {
	ID          uuid.UUID  `json:"id"`
	NamespaceID *uuid.UUID `json:"namespace_id,omitempty"`
	TokenHash   string     `json:"-"`
	Label       string     `json:"label"`
	DailyUses   int        `json:"daily_uses"`
	LastUsed    *time.Time `json:"last_used,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// TokenWithNamespace is a Token joined with its Namespace info.
type TokenWithNamespace struct {
	Token
	NamespaceName   *string `json:"namespace_name,omitempty"`
	NamespaceType   *string `json:"namespace_type,omitempty"`
	NamespaceBanned *bool   `json:"namespace_banned,omitempty"`
}

// Skill represents a skill repository.
type Skill struct {
	ID                 uuid.UUID  `json:"id"`
	NamespaceID        uuid.UUID  `json:"namespace_id"`
	NamespaceName      string     `json:"namespace"`
	Name               string     `json:"name"`
	FullName           string     `json:"full_name"`
	Description        string     `json:"description"`
	Tags               []string   `json:"tags"`
	Framework          string     `json:"framework"`
	Visibility         string     `json:"visibility"` // public | private | org
	ForkedFrom         *uuid.UUID `json:"forked_from,omitempty"`
	InstallCount       int        `json:"install_count"`
	AvgRating          float64    `json:"avg_rating"`
	RatingCount        int        `json:"rating_count"`
	OutcomeSuccessRate float64    `json:"outcome_success_rate"`
	CredibilityScore   float64    `json:"credibility_score"`
	LatestVersion      string     `json:"latest_version"`
	ForkCount          int        `json:"fork_count"`
	Status             string     `json:"status"` // active | yanked | removed
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
}

// Revision represents a version of a skill.
type Revision struct {
	ID               uuid.UUID        `json:"id"`
	SkillID          uuid.UUID        `json:"skill_id"`
	Version          string           `json:"version"`
	Content          string           `json:"content"`
	ChangeSummary    string           `json:"change_summary"`
	AuthorTokenID    *uuid.UUID       `json:"author_token_id,omitempty"`
	ReviewStatus     string           `json:"review_status"` // pending | approved | revision_requested | rejected
	ReviewFeedback   json.RawMessage  `json:"review_feedback,omitempty"`
	ReviewResult     json.RawMessage  `json:"review_result,omitempty"`
	ReviewRetryCount int              `json:"review_retry_count"`
	BreakingChange   bool             `json:"breaking_change"`
	MigrationGuide   *string          `json:"migration_guide,omitempty"`
	SchemaType       string           `json:"schema_type"` // skill-md | mcp-tool
	Triggers         []string         `json:"triggers"`
	CompatibleModels []string         `json:"compatible_models"`
	EstimatedTokens  int              `json:"estimated_tokens"`
	Requirements     json.RawMessage  `json:"requirements,omitempty"`
	Platform         json.RawMessage  `json:"platform,omitempty"`
	CreatedAt        time.Time        `json:"created_at"`
}

// Rating represents a usage feedback entry.
type Rating struct {
	ID              uuid.UUID       `json:"id"`
	SkillID         uuid.UUID       `json:"skill_id"`
	RevisionID      uuid.UUID       `json:"revision_id"`
	TokenID         uuid.UUID       `json:"token_id"`
	Score           int             `json:"score"`
	Outcome         string          `json:"outcome"` // success | partial | failure
	TaskType        string          `json:"task_type"`
	ModelUsed       string          `json:"model_used"`
	TokensConsumed  int             `json:"tokens_consumed"`
	FailureReason   string          `json:"failure_reason,omitempty"`
	ConfidenceScore float64         `json:"confidence_score"`
	ExecutionTimeMs int             `json:"execution_time_ms,omitempty"`
	ErrorDetails    json.RawMessage `json:"error_details,omitempty"`
	ContextMetadata json.RawMessage `json:"context_metadata,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

// Platform describes OS and architecture compatibility.
type Platform struct {
	OS   []string `json:"os,omitempty" yaml:"os,omitempty"`
	Arch []string `json:"arch,omitempty" yaml:"arch,omitempty"`
}

// Requirements describes skill dependencies.
type Requirements struct {
	Tools    []string            `json:"tools,omitempty" yaml:"tools,omitempty"`
	Platform *Platform           `json:"platform,omitempty" yaml:"platform,omitempty"`
	Software []SoftwareReq       `json:"software,omitempty" yaml:"software,omitempty"`
	APIs     []APIReq            `json:"apis,omitempty" yaml:"apis,omitempty"`
}

type SoftwareReq struct {
	Name         string `json:"name" yaml:"name"`
	CheckCommand string `json:"check_command,omitempty" yaml:"check_command,omitempty"`
	InstallURL   string `json:"install_url,omitempty" yaml:"install_url,omitempty"`
	Optional     bool   `json:"optional" yaml:"optional"`
}

type APIReq struct {
	Name      string `json:"name" yaml:"name"`
	EnvVar    string `json:"env_var,omitempty" yaml:"env_var,omitempty"`
	ObtainURL string `json:"obtain_url,omitempty" yaml:"obtain_url,omitempty"`
	Purpose   string `json:"purpose,omitempty" yaml:"purpose,omitempty"`
	Optional  bool   `json:"optional" yaml:"optional"`
}

// SkillFrontmatter represents the YAML frontmatter in a SKILL.md file.
type SkillFrontmatter struct {
	Name             string       `yaml:"name"`
	Version          string       `yaml:"version"`
	Schema           string       `yaml:"schema,omitempty"`
	Framework        string       `yaml:"framework"`
	Tags             []string     `yaml:"tags"`
	Description      string       `yaml:"description"`
	Triggers         []string     `yaml:"triggers,omitempty"`
	CompatibleModels []string     `yaml:"compatible_models,omitempty"`
	EstimatedTokens  int          `yaml:"estimated_tokens,omitempty"`
	Requirements     *Requirements `yaml:"requirements,omitempty"`
}

// ReviewFeedback is the structured feedback from AI review.
type ReviewFeedback struct {
	Issues     []ReviewIssue `json:"issues"`
	Suggestion string        `json:"suggestion,omitempty"`
}

type ReviewIssue struct {
	Type   string `json:"type"`   // privacy | format | security | quality
	Line   int    `json:"line,omitempty"`
	Field  string `json:"field,omitempty"`
	Detail string `json:"detail"`
}

// --- API Request/Response types ---

type CreateTokenResponse struct {
	Token string `json:"token"`
	ID    string `json:"id"`
}

type SubmitSkillRequest struct {
	Namespace  string `json:"namespace"`
	Name       string `json:"name"`
	Content    string `json:"content"`
	Visibility string `json:"visibility,omitempty"`
}

type SubmitRatingRequest struct {
	Score          int    `json:"score"`
	Outcome        string `json:"outcome"`
	TaskType       string `json:"task_type,omitempty"`
	ModelUsed      string `json:"model_used,omitempty"`
	TokensConsumed int    `json:"tokens_consumed,omitempty"`
	FailureReason  string `json:"failure_reason,omitempty"`
}

type SubmitRevisionRequest struct {
	Version       string `json:"version"`
	Content       string `json:"content"`
	ChangeSummary string `json:"change_summary,omitempty"`
}

type SkillSearchParams struct {
	Query      string
	Framework  string
	Tag        string
	Visibility string
	OS         string
	Arch       string
	Sort       string
	Limit      int
	Offset     int
}

type ErrorResponse struct {
	Error   string `json:"error"`
	Status  int    `json:"status"`
	Detail  string `json:"detail,omitempty"`
	Action  string `json:"action,omitempty"`
}

type SkillStatusResponse struct {
	Status         string          `json:"status"`
	ReviewFeedback json.RawMessage `json:"review_feedback,omitempty"`
	Version        string          `json:"version"`
}
