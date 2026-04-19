package skillformat

import (
	"fmt"
	"strings"
	"testing"
)

const validSkill = `---
name: code-review
version: 1.0.0
framework: claude-code
tags: [security, review]
description: "Reviews code for security vulnerabilities."
triggers: ["review code"]
estimated_tokens: 1500
---

# code-review

You review code for security issues.`

func TestParse_Valid(t *testing.T) {
	fm, body, err := Parse(validSkill)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if fm.Name != "code-review" {
		t.Errorf("Expected name 'code-review', got '%s'", fm.Name)
	}
	if fm.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", fm.Version)
	}
	if fm.Framework != "claude-code" {
		t.Errorf("Expected framework 'claude-code', got '%s'", fm.Framework)
	}
	if len(fm.Tags) != 2 {
		t.Errorf("Expected 2 tags, got %d", len(fm.Tags))
	}
	if fm.EstimatedTokens != 1500 {
		t.Errorf("Expected 1500 tokens, got %d", fm.EstimatedTokens)
	}
	if body == "" {
		t.Error("Expected non-empty body")
	}
}

func TestParse_MissingFrontmatter(t *testing.T) {
	_, _, err := Parse("# Just a markdown file")
	if err == nil {
		t.Fatal("Expected error for missing frontmatter")
	}
}

func TestParse_InvalidYAML(t *testing.T) {
	_, _, err := Parse("---\n[invalid yaml\n---\n# body")
	if err == nil {
		t.Fatal("Expected error for invalid YAML")
	}
}

func TestValidate_MissingName(t *testing.T) {
	fm, _, _ := Parse(`---
version: 1.0.0
framework: gstack
tags: [test]
description: "test"
---
# test`)
	err := Validate(fm)
	if err == nil {
		t.Fatal("Expected error for missing name")
	}
}

func TestValidate_InvalidNameFormat(t *testing.T) {
	fm, _, _ := Parse(`---
name: InvalidName_123
version: 1.0.0
framework: gstack
tags: [test]
description: "test"
---
# test`)
	err := Validate(fm)
	if err == nil {
		t.Fatal("Expected error for invalid name format")
	}
}

func TestValidate_InvalidVersion(t *testing.T) {
	fm, _, _ := Parse(`---
name: valid-name
version: v1
framework: gstack
tags: [test]
description: "test"
---
# test`)
	err := Validate(fm)
	if err == nil {
		t.Fatal("Expected error for non-semver version")
	}
}

func TestValidate_InvalidSchema(t *testing.T) {
	fm, _, _ := Parse(`---
name: valid-name
version: 1.0.0
schema: invalid-schema
framework: gstack
tags: [test]
description: "test"
---
# test`)
	err := Validate(fm)
	if err == nil {
		t.Fatal("Expected error for invalid schema")
	}
}

func TestValidate_MCPSchemaAccepted(t *testing.T) {
	fm, _, _ := Parse(`---
name: valid-name
version: 1.0.0
schema: mcp-tool
framework: gstack
tags: [test]
description: "test"
---
# test`)
	err := Validate(fm)
	if err != nil {
		t.Fatalf("Expected mcp-tool schema to be accepted, got: %v", err)
	}
}

func TestValidate_MissingTags(t *testing.T) {
	fm, _, _ := Parse(`---
name: valid-name
version: 1.0.0
framework: gstack
tags: []
description: "test"
---
# test`)
	err := Validate(fm)
	if err == nil {
		t.Fatal("Expected error for empty tags")
	}
}

func TestValidate_Full(t *testing.T) {
	fm, _, err := Parse(validSkill)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	err = Validate(fm)
	if err != nil {
		t.Fatalf("Validation error: %v", err)
	}
}

// TestParse_MissingClosingDelimiter tests missing closing ---
func TestParse_MissingClosingDelimiter(t *testing.T) {
	_, _, err := Parse("---\nname: test\nversion: 1.0.0")
	if err == nil {
		t.Fatal("Expected error for missing closing ---")
	}
	if !strings.Contains(err.Error(), "missing closing ---") {
		t.Errorf("Expected 'missing closing ---' error, got: %v", err)
	}
}

// TestParse_EmptyContent tests empty content
func TestParse_EmptyContent(t *testing.T) {
	_, _, err := Parse("")
	if err == nil {
		t.Fatal("Expected error for empty content")
	}
}

// TestParse_WhitespaceHandling tests whitespace trimming
func TestParse_WhitespaceHandling(t *testing.T) {
	content := `
---
name: test-skill
version: 1.0.0
framework: gstack
tags: [test]
description: "test"
---

# Test Skill

Content here.
  `
	fm, body, err := Parse(content)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if fm.Name != "test-skill" {
		t.Errorf("Expected name 'test-skill', got '%s'", fm.Name)
	}
	if !strings.Contains(body, "# Test Skill") {
		t.Error("Expected body to contain header")
	}
}

// TestValidate_MissingVersion tests missing version field
func TestValidate_MissingVersion(t *testing.T) {
	fm, _, _ := Parse(`---
name: valid-name
framework: gstack
tags: [test]
description: "test"
---
# test`)
	err := Validate(fm)
	if err == nil {
		t.Fatal("Expected error for missing version")
	}
	if !strings.Contains(err.Error(), "missing required field: version") {
		t.Errorf("Expected 'missing required field: version', got: %v", err)
	}
}

// TestValidate_MissingFramework tests missing framework field
func TestValidate_MissingFramework(t *testing.T) {
	fm, _, _ := Parse(`---
name: valid-name
version: 1.0.0
tags: [test]
description: "test"
---
# test`)
	err := Validate(fm)
	if err == nil {
		t.Fatal("Expected error for missing framework")
	}
	if !strings.Contains(err.Error(), "missing required field: framework") {
		t.Errorf("Expected 'missing required field: framework', got: %v", err)
	}
}

// TestValidate_MissingDescription tests missing description field
func TestValidate_MissingDescription(t *testing.T) {
	fm, _, _ := Parse(`---
name: valid-name
version: 1.0.0
framework: gstack
tags: [test]
---
# test`)
	err := Validate(fm)
	if err == nil {
		t.Fatal("Expected error for missing description")
	}
	if !strings.Contains(err.Error(), "missing required field: description") {
		t.Errorf("Expected 'missing required field: description', got: %v", err)
	}
}

// TestValidate_NameEdgeCases tests name validation edge cases
func TestValidate_NameEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		skillName string
		shouldErr bool
	}{
		{"Valid short name (3 chars)", "abc", false},
		{"Valid long name (100 chars)", "a" + strings.Repeat("b", 98) + "c", false},
		{"Too short (2 chars)", "ab", true},
		{"Too short (1 char)", "a", true},
		{"Too long (101 chars)", strings.Repeat("a", 101), true},
		{"Starts with hyphen", "-invalid", true},
		{"Ends with hyphen", "invalid-", true},
		{"Contains uppercase", "Invalid-Name", true},
		{"Contains underscore", "invalid_name", true},
		{"Contains space", "invalid name", true},
		{"Valid with numbers", "skill-123", false},
		{"Starts with number", "1skill", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, _, _ := Parse(fmt.Sprintf(`---
name: %s
version: 1.0.0
framework: gstack
tags: [test]
description: "test"
---
# test`, tt.skillName))
			err := Validate(fm)
			if tt.shouldErr && err == nil {
				t.Errorf("Expected error for name '%s'", tt.skillName)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error for name '%s', got: %v", tt.skillName, err)
			}
		})
	}
}

// TestValidate_VersionEdgeCases tests version validation edge cases
func TestValidate_VersionEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		shouldErr bool
	}{
		{"Valid 0.0.0", "0.0.0", false},
		{"Valid 1.2.3", "1.2.3", false},
		{"Valid 10.20.30", "10.20.30", false},
		{"Invalid v1.0.0", "v1.0.0", true},
		{"Invalid 1.0", "1.0", true},
		{"Invalid 1.0.0.0", "1.0.0.0", true},
		{"Invalid 1.x.0", "1.x.0", true},
		{"Invalid empty", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm, _, _ := Parse(fmt.Sprintf(`---
name: valid-name
version: %s
framework: gstack
tags: [test]
description: "test"
---
# test`, tt.version))
			err := Validate(fm)
			if tt.shouldErr && err == nil {
				t.Errorf("Expected error for version '%s'", tt.version)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error for version '%s', got: %v", tt.version, err)
			}
		})
	}
}

// TestValidate_SchemaEdgeCases tests schema validation edge cases
func TestValidate_SchemaEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		schema    string
		shouldErr bool
	}{
		{"Empty schema (allowed)", "", false},
		{"Valid skill-md", "skill-md", false},
		{"Valid mcp-tool", "mcp-tool", false},
		{"Invalid custom", "custom-schema", true},
		{"Invalid random", "random", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schemaField := ""
			if tt.schema != "" {
				schemaField = fmt.Sprintf("\nschema: %s", tt.schema)
			}
			fm, _, _ := Parse(fmt.Sprintf(`---
name: valid-name
version: 1.0.0
framework: gstack
tags: [test]
description: "test"%s
---
# test`, schemaField))
			err := Validate(fm)
			if tt.shouldErr && err == nil {
				t.Errorf("Expected error for schema '%s'", tt.schema)
			}
			if !tt.shouldErr && err != nil {
				t.Errorf("Expected no error for schema '%s', got: %v", tt.schema, err)
			}
		})
	}
}
