package main

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/joho/godotenv"
)

type ClawHubSearchResult struct {
	Results []ClawHubSkill `json:"results"`
}

type ClawHubSkill struct {
	Slug        string `json:"slug"`
	DisplayName string `json:"displayName"`
	Summary     string `json:"summary"`
}

type ImportStats struct {
	mu        sync.Mutex
	Imported  int
	Skipped   int
	Failed    int
	Processed int
}

func (s *ImportStats) Inc(field string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	switch field {
	case "imported":
		s.Imported++
	case "skipped":
		s.Skipped++
	case "failed":
		s.Failed++
	}
	s.Processed++
}

func main() {
	_ = godotenv.Load()

	aithubURL := os.Getenv("DOMAIN")
	if aithubURL == "" {
		aithubURL = "http://localhost:8080"
	}

	adminToken := os.Getenv("ADMIN_TOKEN")
	if adminToken == "" {
		log.Fatal("ADMIN_TOKEN not set")
	}

	// Search queries for diverse skills
	queries := []string{
		"python", "javascript", "typescript", "react", "vue", "angular",
		"api", "rest", "graphql", "database", "sql", "mongodb",
		"testing", "pytest", "jest", "security", "auth", "oauth",
		"devops", "ci-cd", "docker", "kubernetes", "terraform",
		"git", "github", "gitlab", "web", "frontend", "backend",
		"mobile", "ios", "android", "ai", "ml", "data",
		"aws", "azure", "gcp", "monitoring", "logging", "metrics",
	}

	stats := &ImportStats{}
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // Max 5 concurrent imports

	log.Printf("=== ClawHub Batch Importer (Go) ===")
	log.Printf("Target: %s", aithubURL)
	log.Printf("Queries: %d", len(queries))
	log.Println()

	for _, query := range queries {
		log.Printf("━━━ Searching: %s ━━━", query)

		skills, err := fetchSkills(query, 20)
		if err != nil {
			log.Printf("Failed to fetch skills for %s: %v", query, err)
			continue
		}

		log.Printf("Found %d skills", len(skills))

		for _, skill := range skills {
			wg.Add(1)
			semaphore <- struct{}{} // Acquire

			go func(s ClawHubSkill, q string) {
				defer wg.Done()
				defer func() { <-semaphore }() // Release

				if err := importSkill(aithubURL, adminToken, s, q, stats); err != nil {
					log.Printf("  [%d] ✗ %s: %v", stats.Processed, s.DisplayName, err)
					stats.Inc("failed")
				}
			}(skill, query)

			time.Sleep(100 * time.Millisecond) // Rate limit
		}
	}

	wg.Wait()

	log.Println()
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("=== Batch Import Complete ===")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("Imported: %d", stats.Imported)
	log.Printf("Skipped: %d", stats.Skipped)
	log.Printf("Failed: %d", stats.Failed)
	log.Printf("Total Processed: %d", stats.Processed)
}

func fetchSkills(query string, limit int) ([]ClawHubSkill, error) {
	url := fmt.Sprintf("https://clawhub.ai/api/v1/search?q=%s&limit=%d", query, limit)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result ClawHubSearchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Results, nil
}

func importSkill(baseURL, token string, skill ClawHubSkill, query string, stats *ImportStats) error {
	// Download skill ZIP
	zipData, err := downloadSkill(skill.Slug)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	// Extract SKILL.md
	content, err := extractSkillMD(zipData)
	if err != nil {
		return fmt.Errorf("extract failed: %w", err)
	}

	// Add required fields
	content = addRequiredFields(content)

	// Clean name
	cleanName := cleanSkillName(skill.Slug)

	// Submit to AitHub
	payload := map[string]interface{}{
		"namespace":   "clawhub",
		"name":        cleanName,
		"description": skill.Summary,
		"content":     content,
		"tags":        fmt.Sprintf("openclaw,%s", query),
		"framework":   "openclaw",
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", baseURL+"/v1/skills", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("submit failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 200 || resp.StatusCode == 201 {
		log.Printf("  [%d] ✓ %s", stats.Processed+1, skill.DisplayName)
		stats.Inc("imported")
		return nil
	}

	if strings.Contains(string(respBody), "already exists") {
		log.Printf("  [%d] ⊘ %s (exists)", stats.Processed+1, skill.DisplayName)
		stats.Inc("skipped")
		return nil
	}

	return fmt.Errorf("HTTP %d: %s", resp.StatusCode, respBody)
}

func downloadSkill(slug string) ([]byte, error) {
	url := fmt.Sprintf("https://clawhub.ai/api/v1/download?slug=%s", slug)
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

func extractSkillMD(zipData []byte) (string, error) {
	reader, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return "", err
	}

	for _, file := range reader.File {
		if file.Name == "SKILL.md" {
			rc, err := file.Open()
			if err != nil {
				return "", err
			}
			defer rc.Close()

			content, err := io.ReadAll(rc)
			if err != nil {
				return "", err
			}

			return string(content), nil
		}
	}

	return "", fmt.Errorf("SKILL.md not found in ZIP")
}

func addRequiredFields(content string) string {
	// Check if version exists
	if !strings.Contains(content, "version:") {
		// Insert after description
		re := regexp.MustCompile(`(?m)(^description:.*$)`)
		content = re.ReplaceAllString(content, "$1\nversion: 1.0.0\nframework: openclaw")
	}

	// Check if tags exist
	if !strings.Contains(content, "tags:") {
		re := regexp.MustCompile(`(?m)(^framework:.*$)`)
		content = re.ReplaceAllString(content, "$1\ntags: [openclaw, imported]")
	}

	return content
}

func cleanSkillName(slug string) string {
	// Convert to lowercase
	name := strings.ToLower(slug)
	// Replace spaces and special chars with hyphens
	re := regexp.MustCompile(`[^a-z0-9-]+`)
	name = re.ReplaceAllString(name, "-")
	// Remove leading/trailing hyphens
	name = strings.Trim(name, "-")
	return name
}
