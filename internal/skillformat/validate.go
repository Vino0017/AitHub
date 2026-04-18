package skillformat

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/skillhub/api/internal/models"
	"gopkg.in/yaml.v3"
)

var namePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{1,98}[a-z0-9]$`)
var semverPattern = regexp.MustCompile(`^\d+\.\d+\.\d+$`)

// Parse extracts YAML frontmatter from a SKILL.md content string.
func Parse(content string) (*models.SkillFrontmatter, string, error) {
	content = strings.TrimSpace(content)
	if !strings.HasPrefix(content, "---") {
		return nil, "", fmt.Errorf("content must start with YAML frontmatter (---)")
	}

	parts := strings.SplitN(content[3:], "---", 2)
	if len(parts) < 2 {
		return nil, "", fmt.Errorf("missing closing --- for YAML frontmatter")
	}

	fm := &models.SkillFrontmatter{}
	if err := yaml.Unmarshal([]byte(parts[0]), fm); err != nil {
		return nil, "", fmt.Errorf("invalid YAML frontmatter: %w", err)
	}

	body := strings.TrimSpace(parts[1])
	return fm, body, nil
}

// Validate checks required fields in the frontmatter.
func Validate(fm *models.SkillFrontmatter) error {
	if fm.Name == "" {
		return fmt.Errorf("missing required field: name")
	}
	if !namePattern.MatchString(fm.Name) {
		return fmt.Errorf("name must be 3-100 chars, lowercase, alphanumeric with hyphens (kebab-case)")
	}
	if fm.Version == "" {
		return fmt.Errorf("missing required field: version")
	}
	if !semverPattern.MatchString(fm.Version) {
		return fmt.Errorf("version must be semver format (e.g. 1.0.0)")
	}
	if fm.Framework == "" {
		return fmt.Errorf("missing required field: framework")
	}
	if fm.Description == "" {
		return fmt.Errorf("missing required field: description")
	}
	if len(fm.Tags) == 0 {
		return fmt.Errorf("missing required field: tags (at least one tag)")
	}

	if fm.Schema != "" && fm.Schema != "skill-md" && fm.Schema != "mcp-tool" {
		return fmt.Errorf("schema must be 'skill-md' or 'mcp-tool'")
	}

	return nil
}
