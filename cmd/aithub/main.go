package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

const (
	defaultAPIURL = "https://aithub.space"
	version       = "4.1.0"
)

var (
	apiURL string
	token  string
)

func main() {
	rootCmd := &cobra.Command{
		Use:     "aithub",
		Short:   "SkillHub CLI - AI skill registry client",
		Version: version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			// Load config file
			homeDir, _ := os.UserHomeDir()
			configFile := filepath.Join(homeDir, ".aithub", "config.json")
			fileConfig := make(map[string]string)
			if data, err := os.ReadFile(configFile); err == nil {
				json.Unmarshal(data, &fileConfig)
			}

			// Token priority: --token flag > $SKILLHUB_TOKEN > config.json
			if token == "" {
				token = os.Getenv("SKILLHUB_TOKEN")
			}
			if token == "" {
				token = fileConfig["token"]
			}

			// API URL priority: --api flag > $SKILLHUB_API > config.json > default
			if apiURL == "" {
				apiURL = os.Getenv("SKILLHUB_API")
			}
			if apiURL == "" {
				apiURL = fileConfig["api"]
			}
			if apiURL == "" {
				apiURL = defaultAPIURL
			}
		},
	}

	rootCmd.PersistentFlags().StringVar(&apiURL, "api", "", "API URL (default: $SKILLHUB_API or https://aithub.space)")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Auth token (default: $SKILLHUB_TOKEN)")

	rootCmd.AddCommand(searchCmd())
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(rateCmd())
	rootCmd.AddCommand(submitCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(forkCmd())
	rootCmd.AddCommand(detailsCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(diffCmd())
	rootCmd.AddCommand(registerCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func searchCmd() *cobra.Command {
	var (
		framework string
		sort      string
		osFilter  string
		limit     int
		offset    int
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for skills",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			url := fmt.Sprintf("%s/v1/skills?q=%s&limit=%d&offset=%d", apiURL, url.QueryEscape(query), limit, offset)
			if framework != "" {
				url += "&framework=" + framework
			}
			if sort != "" {
				url += "&sort=" + sort
			}
			if osFilter != "" {
				url += "&os=" + osFilter
			}

			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			}

			if jsonOut {
				fmt.Println(string(body))
				return nil
			}

			var result struct {
				Skills []struct {
					FullName           string   `json:"full_name"`
					Description        string   `json:"description"`
					AvgRating          float64  `json:"avg_rating"`
					InstallCount       int      `json:"install_count"`
					OutcomeSuccessRate float64  `json:"outcome_success_rate"`
					Tags               []string `json:"tags"`
					RelevanceScore     float64  `json:"relevance_score"`
				} `json:"skills"`
				Total      int    `json:"total"`
				Limit      int    `json:"limit"`
				Offset     int    `json:"offset"`
				SearchMode string `json:"search_mode"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return err
			}

			if len(result.Skills) == 0 {
				fmt.Println("No skills found.")
				return nil
			}

			fmt.Printf("Found %d skill(s) (showing %d-%d of %d):\n",
				len(result.Skills),
				result.Offset+1,
				result.Offset+len(result.Skills),
				result.Total)
			if result.SearchMode != "" {
				fmt.Printf("Search mode: %s\n", result.SearchMode)
			}
			fmt.Println()

			// Bug #8: query for highlighting
			queryLower := strings.ToLower(query)
			queryWords := strings.Fields(queryLower)

			for i, skill := range result.Skills {
				fmt.Printf("%d. %s\n", i+1, skill.FullName)
				// Bug #3: consistent truncation at 80 chars
				desc := skill.Description
				if len(desc) > 80 {
					desc = desc[:77] + "..."
				}
				// Bug #8: highlight matching keywords
				if len(queryWords) > 0 {
					for _, word := range queryWords {
						if word != "" {
							desc = highlightWord(desc, word)
						}
					}
				}
				fmt.Printf("   %s\n", desc)
				statsLine := fmt.Sprintf("   ⭐ %.1f | 📦 %d installs | ✅ %.0f%% success",
					skill.AvgRating, skill.InstallCount, skill.OutcomeSuccessRate*100)
				if skill.RelevanceScore > 0 {
					statsLine += fmt.Sprintf(" | 🎯 %.0f%% match", skill.RelevanceScore*100)
				}
				fmt.Println(statsLine)
				if len(skill.Tags) > 0 {
					fmt.Printf("   Tags: %s\n", strings.Join(skill.Tags, ", "))
				}
				fmt.Println()
			}

			// Show pagination hint
			if result.Offset+len(result.Skills) < result.Total {
				nextOffset := result.Offset + result.Limit
				fmt.Printf("💡 To see more results, use: --offset %d\n", nextOffset)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&framework, "framework", "", "Filter by framework")
	cmd.Flags().StringVar(&sort, "sort", "rating", "Sort by: rating, installs, new")
	cmd.Flags().StringVar(&osFilter, "os", "", "Filter by OS")
	cmd.Flags().IntVar(&limit, "limit", 50, "Max results (default 50, max 100)")
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset for pagination")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output raw JSON")

	return cmd
}

func installCmd() *cobra.Command {
	var (
		output     string
		jsonOut    bool
		autoDeploy bool
	)

	cmd := &cobra.Command{
		Use:   "install <namespace/name>",
		Short: "Install a skill (get content)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid format, use: namespace/name")
			}

			url := fmt.Sprintf("%s/v1/skills/%s/%s/content", apiURL, parts[0], parts[1])
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode == http.StatusNotFound {
				// Bug #1: better error for missing content
				fmt.Fprintf(os.Stderr, "Error: Skill content not available.\n\n")
				fmt.Fprintf(os.Stderr, "Possible reasons:\n")
				fmt.Fprintf(os.Stderr, "  - Skill has no approved revision yet (may be pending review)\n")
				fmt.Fprintf(os.Stderr, "  - Skill namespace/name is incorrect\n\n")
				fmt.Fprintf(os.Stderr, "Check status: aithub status %s/%s\n", parts[0], parts[1])
				return fmt.Errorf("skill content not found")
			}
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			}

			if jsonOut {
				fmt.Println(string(body))
				return nil
			}

			var result struct {
				Content   string `json:"content"`
				Version   string `json:"version"`
				Framework string `json:"framework"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return err
			}

			// Auto-deploy logic
			if autoDeploy {
				deployed := false
				homeDir, _ := os.UserHomeDir()

				// Framework directory mapping
				frameworkDirs := map[string]string{
					"gstack":      homeDir + "/.gstack/skills",
					"openclaw":    homeDir + "/.openclaw/skills",
					"hermes":      homeDir + "/.hermes/skills",
					"claude-code": homeDir + "/.claude/skills",
					"cursor":      homeDir + "/.cursor/skills",
					"windsurf":    homeDir + "/.windsurf/skills",
				}

				// Try to detect installed frameworks
				for fw, dir := range frameworkDirs {
					parentDir := strings.TrimSuffix(dir, "/skills")
					if _, err := os.Stat(parentDir); err == nil {
						// Framework detected, deploy here
						skillDir := fmt.Sprintf("%s/%s", dir, parts[0])
						if err := os.MkdirAll(skillDir, 0755); err != nil {
							continue
						}
						skillFile := fmt.Sprintf("%s/SKILL.md", skillDir)
						if err := os.WriteFile(skillFile, []byte(result.Content), 0644); err != nil {
							continue
						}
						fmt.Printf("✓ Deployed to %s: %s (version %s)\n", fw, skillFile, result.Version)
						deployed = true
					}
				}

				if !deployed {
					return fmt.Errorf("no AI framework detected. Install one of: Claude Code, Cursor, Windsurf, GStack, OpenClaw, Hermes")
				}
				return nil
			}

			// Manual output
			if output != "" {
				if err := os.WriteFile(output, []byte(result.Content), 0644); err != nil {
					return fmt.Errorf("failed to write file: %w", err)
				}
				fmt.Printf("✓ Skill saved to %s (version %s)\n", output, result.Version)
			} else {
				fmt.Println(result.Content)
			}

			return nil
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path")
	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output raw JSON")
	cmd.Flags().BoolVar(&autoDeploy, "deploy", false, "Auto-deploy to detected AI frameworks")

	return cmd
}

func rateCmd() *cobra.Command {
	var (
		outcome        string
		taskType       string
		model          string
		tokens         int
		failureReason  string
	)

	cmd := &cobra.Command{
		Use:   "rate <namespace/name> <score>",
		Short: "Rate a skill (1-5 stars)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid format, use: namespace/name")
			}

			var score int
			if _, err := fmt.Sscanf(args[1], "%d", &score); err != nil {
				return fmt.Errorf("invalid score: %s", args[1])
			}
			if score < 1 || score > 5 {
				return fmt.Errorf("score must be between 1 and 5")
			}

			payload := map[string]interface{}{
				"score":   score,
				"outcome": outcome,
			}
			if taskType != "" {
				payload["task_type"] = taskType
			}
			if model != "" {
				payload["model_used"] = model
			}
			if tokens > 0 {
				payload["tokens_consumed"] = tokens
			}
			if failureReason != "" {
				payload["failure_reason"] = failureReason
			}

			jsonData, err := json.Marshal(payload)
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/v1/skills/%s/%s/ratings", apiURL, parts[0], parts[1])
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				fmt.Fprintf(os.Stderr, "Error: Authentication required\n\n")
				fmt.Fprintf(os.Stderr, "To rate skills, you need a registered account.\n")
				fmt.Fprintf(os.Stderr, "Run: npx @aithub/cli --register --github\n")
				return fmt.Errorf("authentication required")
			}

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			}

			stars := strings.Repeat("⭐", score)
			fmt.Printf("✓ Rating submitted for %s (%s, outcome: %s)\n", args[0], stars, outcome)
			return nil
		},
	}

	cmd.Flags().StringVar(&outcome, "outcome", "success", "Outcome: success, partial, failure")
	cmd.Flags().StringVar(&taskType, "task-type", "", "Task type description")
	cmd.Flags().StringVar(&model, "model", "", "Model used")
	cmd.Flags().IntVar(&tokens, "tokens", 0, "Tokens consumed")
	cmd.Flags().StringVar(&failureReason, "failure-reason", "", "Reason for failure (if outcome=failure)")

	return cmd
}

func submitCmd() *cobra.Command {
	var (
		visibility string
	)

	cmd := &cobra.Command{
		Use:   "submit <file>",
		Short: "Submit a new skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			content, err := os.ReadFile(args[0])
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}

			payload := map[string]interface{}{
				"content":    string(content),
				"visibility": visibility,
			}

			jsonData, err := json.Marshal(payload)
			if err != nil {
				return err
			}

			url := fmt.Sprintf("%s/v1/skills", apiURL)
			req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
			if err != nil {
				return err
			}
			req.Header.Set("Content-Type", "application/json")
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				fmt.Fprintf(os.Stderr, "Error: Authentication required\n\n")
				fmt.Fprintf(os.Stderr, "To submit skills, you need a registered account.\n")
				fmt.Fprintf(os.Stderr, "Run: npx @aithub/cli --register --github\n")
				return fmt.Errorf("authentication required")
			}

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			}

			var result struct {
				FullName string `json:"full_name"`
				Status   string `json:"status"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				fmt.Println(string(body))
				return nil
			}

			fmt.Printf("✓ Skill submitted: %s\n", result.FullName)
			fmt.Printf("  Status: %s\n", result.Status)
			if result.Status == "pending" {
				fmt.Println("  Your skill is under review. Check status with: aithub status " + result.FullName)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&visibility, "visibility", "public", "Visibility: public, private")

	return cmd
}

func statusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status <namespace/name>",
		Short: "Check skill review status",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid format, use: namespace/name")
			}

			url := fmt.Sprintf("%s/v1/skills/%s/%s/status", apiURL, parts[0], parts[1])
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			}

			var result struct {
				Status         string `json:"status"`
				Version        string `json:"version"`
				ReviewFeedback struct {
					Issues []struct {
						Type    string `json:"type"`
						Message string `json:"message"`
					} `json:"issues"`
				} `json:"review_feedback"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return err
			}

			// Bug #4: Better status messages
			statusIcons := map[string]string{"approved": "✅", "pending": "⏳", "rejected": "❌"}
			icon := statusIcons[result.Status]
			if icon == "" {
				icon = "❓"
			}
			fmt.Printf("%s Status: %s\n", icon, result.Status)
			if result.Version != "" {
				fmt.Printf("   Version: %s\n", result.Version)
			}

			switch result.Status {
			case "approved":
				fmt.Println("\n   Your skill is live and installable!")
				fmt.Printf("   Install: aithub install %s\n", args[0])
			case "pending":
				fmt.Println("\n   Your skill is being reviewed by the AitHub team.")
				fmt.Println("   Most reviews complete within a few minutes.")
			case "rejected":
				fmt.Println("\n   Your skill was not approved. See issues below.")
				fmt.Println("   Fix the issues and resubmit with: aithub submit <file>")
			}

			if len(result.ReviewFeedback.Issues) > 0 {
				fmt.Println("\n   Issues found:")
				for i, issue := range result.ReviewFeedback.Issues {
					fmt.Printf("   %d. [%s] %s\n", i+1, issue.Type, issue.Message)
				}
			}

			return nil
		},
	}

	return cmd
}

func forkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "fork <namespace/name>",
		Short: "Fork a skill to your namespace",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid format, use: namespace/name")
			}

			url := fmt.Sprintf("%s/v1/skills/%s/%s/fork", apiURL, parts[0], parts[1])
			req, err := http.NewRequest("POST", url, nil)
			if err != nil {
				return err
			}
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
				fmt.Fprintf(os.Stderr, "Error: Authentication required\n\n")
				fmt.Fprintf(os.Stderr, "To fork skills, you need a registered account.\n")
				fmt.Fprintf(os.Stderr, "Run: npx @aithub/cli --register --github\n")
				return fmt.Errorf("authentication required")
			}

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			}

			var result struct {
				FullName string `json:"full_name"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				fmt.Println(string(body))
				return nil
			}

			// Bug #7: Add next-step hints after fork
			fmt.Printf("✓ Skill forked to: %s\n\n", result.FullName)
			fmt.Println("Next steps:")
			fmt.Printf("  📖 View:    aithub details %s\n", result.FullName)
			fmt.Printf("  📦 Install: aithub install %s\n", result.FullName)
			fmt.Printf("  ✏️  Edit:    aithub install %s -o skill.md && edit skill.md && aithub submit skill.md\n", result.FullName)
			return nil
		},
	}

	return cmd
}

func detailsCmd() *cobra.Command {
	var jsonOut bool

	cmd := &cobra.Command{
		Use:   "details <namespace/name>",
		Short: "Get skill details (metadata, requirements)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			parts := strings.Split(args[0], "/")
			if len(parts) != 2 {
				return fmt.Errorf("invalid format, use: namespace/name")
			}

			url := fmt.Sprintf("%s/v1/skills/%s/%s", apiURL, parts[0], parts[1])
			req, err := http.NewRequest("GET", url, nil)
			if err != nil {
				return err
			}
			if token != "" {
				req.Header.Set("Authorization", "Bearer "+token)
			}

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("request failed: %w", err)
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return err
			}

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			}

			if jsonOut {
				fmt.Println(string(body))
				return nil
			}

			var result map[string]interface{}
			if err := json.Unmarshal(body, &result); err != nil {
				return err
			}

			// Bug #6: Show all key metadata
			fmt.Printf("╔══════════════════════════════════════════╗\n")
			fmt.Printf("║  %s\n", result["full_name"])
			fmt.Printf("╚══════════════════════════════════════════╝\n\n")
			fmt.Printf("  Description: %s\n", result["description"])
			fmt.Printf("  Author:      %s\n", result["namespace"])

			// Version
			if lv, ok := result["latest_version"].(string); ok && lv != "" {
				fmt.Printf("  Version:     %s\n", lv)
			}
			fmt.Printf("  Framework:   %s\n", result["framework"])
			fmt.Printf("  Visibility:  %s\n", result["visibility"])

			// Tags
			if tags, ok := result["tags"].([]interface{}); ok && len(tags) > 0 {
				tagStrs := make([]string, len(tags))
				for i, t := range tags {
					tagStrs[i] = fmt.Sprintf("%v", t)
				}
				fmt.Printf("  Tags:        %s\n", strings.Join(tagStrs, ", "))
			}

			// Created at
			if ca, ok := result["created_at"].(string); ok && ca != "" {
				fmt.Printf("  Created:     %s\n", ca)
			}
			if ua, ok := result["updated_at"].(string); ok && ua != "" {
				fmt.Printf("  Updated:     %s\n", ua)
			}

			fmt.Println()
			fmt.Printf("  ⭐ Rating:   %.1f/5 (%d ratings)\n", result["avg_rating"], int(result["rating_count"].(float64)))
			fmt.Printf("  📦 Installs: %d\n", int(result["install_count"].(float64)))
			fmt.Printf("  ✅ Success:  %.0f%%\n", result["outcome_success_rate"].(float64)*100)
			if fc, ok := result["fork_count"].(float64); ok && fc > 0 {
				fmt.Printf("  🍴 Forks:    %d\n", int(fc))
			}

			// Forked from
			if ff, ok := result["forked_from"]; ok && ff != nil {
				fmt.Printf("  Forked from: %v\n", ff)
			}

			// Requirements
			if reqs, ok := result["requirements"].(map[string]interface{}); ok {
				fmt.Println("\n  Requirements:")
				if software, ok := reqs["software"].([]interface{}); ok && len(software) > 0 {
					fmt.Println("    Software:")
					for _, s := range software {
						sw := s.(map[string]interface{})
						fmt.Printf("      - %s", sw["name"])
						if opt, ok := sw["optional"].(bool); ok && opt {
							fmt.Print(" (optional)")
						}
						fmt.Println()
					}
				}
				if apis, ok := reqs["apis"].([]interface{}); ok && len(apis) > 0 {
					fmt.Println("    APIs:")
					for _, a := range apis {
						api := a.(map[string]interface{})
						fmt.Printf("      - %s (env: %s)", api["name"], api["env_var"])
						if opt, ok := api["optional"].(bool); ok && opt {
							fmt.Print(" (optional)")
						}
						fmt.Println()
					}
				}
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&jsonOut, "json", false, "Output raw JSON")

	return cmd
}

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage configuration",
	}

	setCmd := &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a config value (api, token)",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}

			configDir := homeDir + "/.aithub"
			if err := os.MkdirAll(configDir, 0755); err != nil {
				return fmt.Errorf("failed to create config directory: %w", err)
			}

			configFile := configDir + "/config.json"
			config := make(map[string]string)

			// Load existing config
			if data, err := os.ReadFile(configFile); err == nil {
				json.Unmarshal(data, &config)
			}

			// Update config
			config[key] = value

			// Save config
			data, err := json.MarshalIndent(config, "", "  ")
			if err != nil {
				return err
			}

			if err := os.WriteFile(configFile, data, 0644); err != nil {
				return fmt.Errorf("failed to write config: %w", err)
			}

			fmt.Printf("✓ Config updated: %s = %s\n", key, value)
			fmt.Printf("  Saved to: %s\n", configFile)
			return nil
		},
	}

	getCmd := &cobra.Command{
		Use:   "get <key>",
		Short: "Get a config value",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]

			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}

			configFile := homeDir + "/.aithub/config.json"
			data, err := os.ReadFile(configFile)
			if err != nil {
				return fmt.Errorf("config file not found: %s", configFile)
			}

			config := make(map[string]string)
			if err := json.Unmarshal(data, &config); err != nil {
				return err
			}

			if value, ok := config[key]; ok {
				fmt.Println(value)
			} else {
				return fmt.Errorf("key not found: %s", key)
			}

			return nil
		},
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all config values",
		RunE: func(cmd *cobra.Command, args []string) error {
			homeDir, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("failed to get home directory: %w", err)
			}

			configFile := homeDir + "/.aithub/config.json"
			data, err := os.ReadFile(configFile)
			if err != nil {
				fmt.Println("No config file found. Use 'aithub config set' to create one.")
				return nil
			}

			config := make(map[string]string)
			if err := json.Unmarshal(data, &config); err != nil {
				return err
			}

			if len(config) == 0 {
				fmt.Println("No config values set.")
				return nil
			}

			fmt.Println("Current configuration:")
			for key, value := range config {
				// Mask sensitive values
				if key == "token" && len(value) > 8 {
					value = value[:8] + "..." + value[len(value)-4:]
				}
				fmt.Printf("  %s = %s\n", key, value)
			}

			return nil
		},
	}

	cmd.AddCommand(setCmd)
	cmd.AddCommand(getCmd)
	cmd.AddCommand(listCmd)

	return cmd
}

func diffCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "diff <namespace/name@v1> <namespace/name@v2>",
		Short: "Show differences between two skill versions",
		Long: `Compare two versions of a skill and show the changelog.

Examples:
  aithub diff anthropics/pdf@1.0.0 anthropics/pdf@1.1.0
  aithub diff anthropics/pdf@1.0.0 anthropics/pdf  (compare with latest)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			// Parse first version
			parts1 := strings.Split(args[0], "@")
			skillPath1 := parts1[0]
			version1 := ""
			if len(parts1) > 1 {
				version1 = parts1[1]
			}

			// Parse second version
			parts2 := strings.Split(args[1], "@")
			skillPath2 := parts2[0]
			version2 := ""
			if len(parts2) > 1 {
				version2 = parts2[1]
			}

			// Validate same skill
			if skillPath1 != skillPath2 {
				return fmt.Errorf("cannot compare different skills: %s vs %s", skillPath1, skillPath2)
			}

			skillParts := strings.Split(skillPath1, "/")
			if len(skillParts) != 2 {
				return fmt.Errorf("invalid format, use: namespace/name@version")
			}

			// Fetch both versions
			content1, err := fetchSkillContent(skillParts[0], skillParts[1], version1)
			if err != nil {
				return fmt.Errorf("failed to fetch version %s: %w", version1, err)
			}

			content2, err := fetchSkillContent(skillParts[0], skillParts[1], version2)
			if err != nil {
				return fmt.Errorf("failed to fetch version %s: %w", version2, err)
			}

			// Show diff
			fmt.Printf("Comparing %s@%s vs %s@%s\n\n", skillPath1, version1, skillPath2, version2)

			if content1 == content2 {
				fmt.Println("✓ No differences found.")
				return nil
			}

			// Simple line-by-line diff
			lines1 := strings.Split(content1, "\n")
			lines2 := strings.Split(content2, "\n")

			fmt.Println("Changes:")
			fmt.Println("--------")

			maxLen := len(lines1)
			if len(lines2) > maxLen {
				maxLen = len(lines2)
			}

			changes := 0
			for i := 0; i < maxLen; i++ {
				line1 := ""
				line2 := ""

				if i < len(lines1) {
					line1 = lines1[i]
				}
				if i < len(lines2) {
					line2 = lines2[i]
				}

				if line1 != line2 {
					changes++
					if line1 != "" && line2 == "" {
						fmt.Printf("- %s\n", line1)
					} else if line1 == "" && line2 != "" {
						fmt.Printf("+ %s\n", line2)
					} else {
						fmt.Printf("- %s\n", line1)
						fmt.Printf("+ %s\n", line2)
					}
				}
			}

			fmt.Printf("\n%d line(s) changed\n", changes)

			return nil
		},
	}

	return cmd
}

func fetchSkillContent(namespace, name, version string) (string, error) {
	url := fmt.Sprintf("%s/v1/skills/%s/%s/content", apiURL, namespace, name)
	if version != "" {
		url += "?version=" + version
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
	}

	var result struct {
		Content string `json:"content"`
		Version string `json:"version"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}

	return result.Content, nil
}

func registerCmd() *cobra.Command {
	var github bool

	cmd := &cobra.Command{
		Use:   "register",
		Short: "Register an account (needed for rate/submit/fork)",
		Long: `Register with AitHub to unlock: rating skills, submitting skills, and forking.

Search, install, and details work without registration.

Examples:
  aithub register --github`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !github {
				fmt.Println("Please specify a registration method:")
				fmt.Println("  aithub register --github    Register with GitHub")
				fmt.Println("")
				fmt.Println("Search/install/details work without registration.")
				return nil
			}

			fmt.Println("→ Starting GitHub registration...")
			fmt.Printf("  API: %s\n", apiURL)
			fmt.Println("")

			// Step 1: Start device flow
			url := fmt.Sprintf("%s/v1/auth/github", apiURL)
			fmt.Printf("  → POST %s\n", url)
			req, err := http.NewRequest("POST", url, nil)
			if err != nil {
				return fmt.Errorf("failed to create request: %w", err)
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return fmt.Errorf("failed to contact API: %w\n\nCheck your internet connection and that %s is reachable.", err, apiURL)
			}
			defer resp.Body.Close()

			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("registration failed (HTTP %d): %s\n  URL: %s\n\nIf this persists, report to admin@aithub.space", resp.StatusCode, string(body), url)
			}

			var deviceFlow struct {
				DeviceCode      string `json:"device_code"`
				UserCode        string `json:"user_code"`
				VerificationURI string `json:"verification_uri"`
				ExpiresIn       int    `json:"expires_in"`
				Instruction     string `json:"instruction"`
			}

			if err := json.Unmarshal(body, &deviceFlow); err != nil {
				return fmt.Errorf("invalid response from API: %w", err)
			}

			fmt.Println("╔══════════════════════════════════════════╗")
			fmt.Println("║        GitHub Device Authorization       ║")
			fmt.Println("╠══════════════════════════════════════════╣")
			fmt.Printf("║  1. Open: %-30s ║\n", deviceFlow.VerificationURI)
			fmt.Printf("║  2. Enter code: %-24s ║\n", deviceFlow.UserCode)
			fmt.Println("║  3. Authorize AitHub                     ║")
			fmt.Println("╚══════════════════════════════════════════╝")
			fmt.Println("")
			fmt.Println("Waiting for authorization...")

			// Step 2: Poll for completion
			pollURL := fmt.Sprintf("%s/v1/auth/github/poll", apiURL)
			pollPayload, _ := json.Marshal(map[string]string{
				"device_code": deviceFlow.DeviceCode,
			})

			deadline := time.Now().Add(time.Duration(deviceFlow.ExpiresIn) * time.Second)
			interval := 6 * time.Second // GitHub default is 5s, use 6 for safety
			pollCount := 0

			for time.Now().Before(deadline) {
				time.Sleep(interval)
				pollCount++

				pollReq, _ := http.NewRequest("POST", pollURL, bytes.NewBuffer(pollPayload))
				pollReq.Header.Set("Content-Type", "application/json")

				pollResp, err := http.DefaultClient.Do(pollReq)
				if err != nil {
					fmt.Printf("\r⏳ Waiting for authorization... (%ds) [network error, retrying]", pollCount*int(interval.Seconds()))
					continue
				}

				pollBody, _ := io.ReadAll(pollResp.Body)
				pollResp.Body.Close()

				if pollResp.StatusCode != http.StatusOK {
					fmt.Printf("\r⏳ Waiting for authorization... (%ds) [server error %d]", pollCount*int(interval.Seconds()), pollResp.StatusCode)
					continue
				}

				var pollResult struct {
					Status    string `json:"status"`
					Token     string `json:"token"`
					Namespace string `json:"namespace"`
					Error     string `json:"error"`
				}

				if err := json.Unmarshal(pollBody, &pollResult); err != nil {
					fmt.Printf("\r⏳ Waiting for authorization... (%ds)", pollCount*int(interval.Seconds()))
					continue
				}

				// Handle slow_down from GitHub (via server)
				if pollResult.Error == "slow_down" {
					interval = interval + 5*time.Second // GitHub requires +5s on slow_down
					fmt.Printf("\r⏳ Waiting for authorization... (%ds) [rate limited, slowing]", pollCount*int(interval.Seconds()))
					continue
				}

				if pollResult.Status == "pending" {
					fmt.Printf("\r⏳ Waiting for authorization... (%ds)", pollCount*int(interval.Seconds()))
					continue
				}

				if pollResult.Status == "complete" && pollResult.Token != "" {
					fmt.Println("")
					fmt.Println("")

					// Save to config
					homeDir, _ := os.UserHomeDir()
					configDir := filepath.Join(homeDir, ".aithub")
					os.MkdirAll(configDir, 0755)
					configFile := filepath.Join(configDir, "config.json")

					config := make(map[string]string)
					if data, err := os.ReadFile(configFile); err == nil {
						json.Unmarshal(data, &config)
					}
					config["token"] = pollResult.Token
					config["namespace"] = pollResult.Namespace
					config["api"] = apiURL

					data, _ := json.MarshalIndent(config, "", "  ")
					os.WriteFile(configFile, data, 0644)

					// Also set environment variable hint
					fmt.Println("╔══════════════════════════════════════════╗")
					fmt.Println("║        ✓ Registration Complete!          ║")
					fmt.Println("╠══════════════════════════════════════════╣")
					fmt.Printf("║  Namespace: %-28s ║\n", pollResult.Namespace)
					fmt.Printf("║  Token: %-32s ║\n", pollResult.Token)
					fmt.Printf("║  Config: %-31s ║\n", configFile)
					fmt.Println("╚══════════════════════════════════════════╝")
					fmt.Println("")
					fmt.Println("You can now:")
					fmt.Println("  aithub rate <namespace/name> <score>    Rate a skill")
					fmt.Println("  aithub submit SKILL.md                  Submit a skill")
					fmt.Println("  aithub fork <namespace/name>            Fork a skill")
					return nil
				}
			}

			fmt.Println("")
			return fmt.Errorf("authorization timed out. Please try again: aithub register --github")
		},
	}

	cmd.Flags().BoolVar(&github, "github", false, "Register with GitHub")

	return cmd
}

// highlightWord wraps matching words with ANSI bold
func highlightWord(text, word string) string {
	lower := strings.ToLower(text)
	idx := strings.Index(lower, word)
	if idx == -1 {
		return text
	}
	return text[:idx] + "\033[1;33m" + text[idx:idx+len(word)] + "\033[0m" + text[idx+len(word):]
}
