package validation

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"strings"

	"github.com/skillhub/api/internal/models"
)

// EnvironmentValidator checks if requirements are met.
type EnvironmentValidator struct{}

func NewEnvironmentValidator() *EnvironmentValidator {
	return &EnvironmentValidator{}
}

// ValidateRequirements checks if the given environment meets skill requirements.
// Returns validation result with missing/incompatible items.
func (v *EnvironmentValidator) ValidateRequirements(ctx context.Context, requirements *models.Requirements, env EnvironmentInfo) ValidationResult {
	result := ValidationResult{
		Compatible:      true,
		MissingTools:    []string{},
		MissingSoftware: []models.SoftwareReq{},
		MissingAPIs:     []models.APIReq{},
		Warnings:        []string{},
	}

	if requirements == nil {
		return result
	}

	// Validate platform (OS/Arch)
	if requirements.Platform != nil {
		if !v.checkPlatform(requirements.Platform, env) {
			result.Compatible = false
			result.Warnings = append(result.Warnings,
				fmt.Sprintf("Platform mismatch: requires %v/%v, got %s/%s",
					requirements.Platform.OS, requirements.Platform.Arch, env.OS, env.Arch))
		}
	}

	// Validate tools
	for _, tool := range requirements.Tools {
		if !contains(env.AvailableTools, tool) {
			result.MissingTools = append(result.MissingTools, tool)
		}
	}

	// Validate software
	for _, sw := range requirements.Software {
		if !sw.Optional && !contains(env.InstalledSoftware, sw.Name) {
			result.MissingSoftware = append(result.MissingSoftware, sw)
		}
	}

	// Validate APIs
	for _, api := range requirements.APIs {
		if !api.Optional && !contains(env.AvailableAPIs, api.Name) {
			result.MissingAPIs = append(result.MissingAPIs, api)
		}
	}

	// Set compatibility based on missing required items
	if len(result.MissingTools) > 0 || len(result.MissingSoftware) > 0 || len(result.MissingAPIs) > 0 {
		result.Compatible = false
	}

	return result
}

func (v *EnvironmentValidator) checkPlatform(platform *models.Platform, env EnvironmentInfo) bool {
	osMatch := len(platform.OS) == 0 || contains(platform.OS, env.OS) || contains(platform.OS, "any")
	archMatch := len(platform.Arch) == 0 || contains(platform.Arch, env.Arch) || contains(platform.Arch, "any")
	return osMatch && archMatch
}

// DetectEnvironment attempts to detect the current environment.
// In production, this would be provided by the AI agent.
func DetectEnvironment() EnvironmentInfo {
	return EnvironmentInfo{
		OS:                runtime.GOOS,
		Arch:              runtime.GOARCH,
		AvailableTools:    []string{}, // Would be populated by agent
		InstalledSoftware: []string{}, // Would be populated by agent
		AvailableAPIs:     []string{}, // Would be populated by agent
	}
}

// EnvironmentInfo describes the AI agent's runtime environment.
type EnvironmentInfo struct {
	OS                string   `json:"os"`
	Arch              string   `json:"arch"`
	AvailableTools    []string `json:"available_tools"`
	InstalledSoftware []string `json:"installed_software"`
	AvailableAPIs     []string `json:"available_apis"`
}

// ValidationResult holds the outcome of requirements validation.
type ValidationResult struct {
	Compatible      bool                  `json:"compatible"`
	MissingTools    []string              `json:"missing_tools,omitempty"`
	MissingSoftware []models.SoftwareReq  `json:"missing_software,omitempty"`
	MissingAPIs     []models.APIReq       `json:"missing_apis,omitempty"`
	Warnings        []string              `json:"warnings,omitempty"`
}

// ToJSON serializes the validation result.
func (vr *ValidationResult) ToJSON() ([]byte, error) {
	return json.Marshal(vr)
}

// GetInstallInstructions returns human-readable install instructions.
func (vr *ValidationResult) GetInstallInstructions() string {
	var instructions []string

	if len(vr.MissingTools) > 0 {
		instructions = append(instructions, fmt.Sprintf("Missing tools: %s", strings.Join(vr.MissingTools, ", ")))
	}

	for _, sw := range vr.MissingSoftware {
		if sw.InstallURL != "" {
			instructions = append(instructions, fmt.Sprintf("Install %s: %s", sw.Name, sw.InstallURL))
		} else {
			instructions = append(instructions, fmt.Sprintf("Install %s (check command: %s)", sw.Name, sw.CheckCommand))
		}
	}

	for _, api := range vr.MissingAPIs {
		if api.ObtainURL != "" {
			instructions = append(instructions, fmt.Sprintf("Get %s API key: %s (set %s)", api.Name, api.ObtainURL, api.EnvVar))
		} else {
			instructions = append(instructions, fmt.Sprintf("Configure %s API (set %s)", api.Name, api.EnvVar))
		}
	}

	if len(instructions) == 0 {
		return "All requirements met."
	}

	return strings.Join(instructions, "\n")
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if strings.EqualFold(s, item) {
			return true
		}
	}
	return false
}
