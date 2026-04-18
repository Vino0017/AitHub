package skillformat

import (
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
