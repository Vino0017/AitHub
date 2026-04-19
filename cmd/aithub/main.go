package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	defaultAPIURL = "https://your-domain.com"
	version       = "3.0.0"
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
			// Load token from env if not provided via flag
			if token == "" {
				token = os.Getenv("SKILLHUB_TOKEN")
			}
			if apiURL == "" {
				apiURL = os.Getenv("SKILLHUB_API")
				if apiURL == "" {
					apiURL = defaultAPIURL
				}
			}
		},
	}

	rootCmd.PersistentFlags().StringVar(&apiURL, "api", "", "API URL (default: $SKILLHUB_API or https://your-domain.com)")
	rootCmd.PersistentFlags().StringVar(&token, "token", "", "Auth token (default: $SKILLHUB_TOKEN)")

	rootCmd.AddCommand(searchCmd())
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(rateCmd())
	rootCmd.AddCommand(submitCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(forkCmd())
	rootCmd.AddCommand(detailsCmd())

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
		jsonOut   bool
	)

	cmd := &cobra.Command{
		Use:   "search <query>",
		Short: "Search for skills",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			query := strings.Join(args, " ")

			url := fmt.Sprintf("%s/v1/skills?q=%s&limit=%d", apiURL, query, limit)
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
				} `json:"skills"`
			}

			if err := json.Unmarshal(body, &result); err != nil {
				return err
			}

			if len(result.Skills) == 0 {
				fmt.Println("No skills found.")
				return nil
			}

			fmt.Printf("Found %d skill(s):\n\n", len(result.Skills))
			for i, skill := range result.Skills {
				fmt.Printf("%d. %s\n", i+1, skill.FullName)
				fmt.Printf("   %s\n", skill.Description)
				fmt.Printf("   ⭐ %.1f | 📦 %d installs | ✅ %.0f%% success\n",
					skill.AvgRating, skill.InstallCount, skill.OutcomeSuccessRate*100)
				if len(skill.Tags) > 0 {
					fmt.Printf("   Tags: %s\n", strings.Join(skill.Tags, ", "))
				}
				fmt.Println()
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&framework, "framework", "", "Filter by framework")
	cmd.Flags().StringVar(&sort, "sort", "rating", "Sort by: rating, installs, new")
	cmd.Flags().StringVar(&osFilter, "os", "", "Filter by OS")
	cmd.Flags().IntVar(&limit, "limit", 10, "Max results")
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
		Short: "Rate a skill (1-10)",
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
			if score < 1 || score > 10 {
				return fmt.Errorf("score must be between 1 and 10")
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
				fmt.Fprintf(os.Stderr, "Run: bash <(curl -fsSL %s/install) --register --github\n", apiURL)
				return fmt.Errorf("authentication required")
			}

			if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
				return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(body))
			}

			fmt.Printf("✓ Rating submitted for %s (score: %d, outcome: %s)\n", args[0], score, outcome)
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
				fmt.Fprintf(os.Stderr, "Run: bash <(curl -fsSL %s/install) --register --github\n", apiURL)
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

			fmt.Printf("Status: %s\n", result.Status)
			if len(result.ReviewFeedback.Issues) > 0 {
				fmt.Println("\nIssues found:")
				for i, issue := range result.ReviewFeedback.Issues {
					fmt.Printf("%d. [%s] %s\n", i+1, issue.Type, issue.Message)
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
				fmt.Fprintf(os.Stderr, "Run: bash <(curl -fsSL %s/install) --register --github\n", apiURL)
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

			fmt.Printf("✓ Skill forked to: %s\n", result.FullName)
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

			fmt.Printf("Skill: %s\n", result["full_name"])
			fmt.Printf("Description: %s\n", result["description"])

			// Handle version field safely
			if version, ok := result["version"].(string); ok && version != "" {
				fmt.Printf("Version: %s\n", version)
			} else {
				fmt.Printf("Version: (not specified)\n")
			}

			fmt.Printf("Framework: %s\n", result["framework"])
			fmt.Printf("Rating: %.1f (%d ratings)\n", result["avg_rating"], int(result["rating_count"].(float64)))
			fmt.Printf("Installs: %d\n", int(result["install_count"].(float64)))
			fmt.Printf("Success Rate: %.0f%%\n", result["outcome_success_rate"].(float64)*100)

			if reqs, ok := result["requirements"].(map[string]interface{}); ok {
				fmt.Println("\nRequirements:")
				if software, ok := reqs["software"].([]interface{}); ok && len(software) > 0 {
					fmt.Println("  Software:")
					for _, s := range software {
						sw := s.(map[string]interface{})
						fmt.Printf("    - %s", sw["name"])
						if opt, ok := sw["optional"].(bool); ok && opt {
							fmt.Print(" (optional)")
						}
						fmt.Println()
					}
				}
				if apis, ok := reqs["apis"].([]interface{}); ok && len(apis) > 0 {
					fmt.Println("  APIs:")
					for _, a := range apis {
						api := a.(map[string]interface{})
						fmt.Printf("    - %s (env: %s)", api["name"], api["env_var"])
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
