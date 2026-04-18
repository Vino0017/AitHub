package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// Client is a unified LLM client supporting Anthropic and OpenAI-compatible APIs.
type Client struct {
	provider string
	apiKey   string
	baseURL  string
	model    string
	client   *http.Client
}

// NewClient creates a new LLM client from environment variables.
func NewClient() *Client {
	c := &Client{
		client: &http.Client{Timeout: 60 * time.Second},
	}

	// Priority: explicit LLM_ vars > Anthropic > OpenAI
	if key := os.Getenv("LLM_API_KEY"); key != "" {
		c.provider = os.Getenv("LLM_PROVIDER")
		c.apiKey = key
		c.baseURL = os.Getenv("LLM_BASE_URL")
		c.model = os.Getenv("LLM_MODEL")
	} else if key := os.Getenv("ANTHROPIC_API_KEY"); key != "" {
		c.provider = "anthropic"
		c.apiKey = key
		c.baseURL = os.Getenv("ANTHROPIC_BASE_URL")
		if c.baseURL == "" {
			c.baseURL = "https://api.anthropic.com"
		}
		c.model = os.Getenv("ANTHROPIC_MODEL")
		if c.model == "" {
			c.model = "claude-haiku-4-5-20251001"
		}
	} else if key := os.Getenv("OPENAI_API_KEY"); key != "" {
		c.provider = "openai"
		c.apiKey = key
		c.baseURL = os.Getenv("OPENAI_BASE_URL")
		if c.baseURL == "" {
			c.baseURL = "https://api.openai.com"
		}
		c.model = os.Getenv("OPENAI_MODEL")
		if c.model == "" {
			c.model = "gpt-4o-mini"
		}
	}

	return c
}

// IsConfigured returns true if the client has a valid API key.
func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}

// Complete sends a prompt and returns the response text.
func (c *Client) Complete(ctx context.Context, prompt string) (string, error) {
	if !c.IsConfigured() {
		return "", fmt.Errorf("LLM not configured")
	}

	if c.provider == "anthropic" {
		return c.completeAnthropic(ctx, prompt)
	}
	return c.completeOpenAI(ctx, prompt)
}

func (c *Client) completeAnthropic(ctx context.Context, prompt string) (string, error) {
	body, _ := json.Marshal(map[string]interface{}{
		"model":      c.model,
		"max_tokens": 2048,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", c.baseURL+"/v1/messages", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("anthropic request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("anthropic %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	json.Unmarshal(respBody, &result)
	if len(result.Content) == 0 {
		return "", fmt.Errorf("empty anthropic response")
	}

	return extractJSON(result.Content[0].Text), nil
}

func (c *Client) completeOpenAI(ctx context.Context, prompt string) (string, error) {
	baseURL := strings.TrimRight(c.baseURL, "/")
	body, _ := json.Marshal(map[string]interface{}{
		"model":      c.model,
		"max_tokens": 2048,
		"messages":   []map[string]string{{"role": "user", "content": prompt}},
	})

	req, _ := http.NewRequestWithContext(ctx, "POST", baseURL+"/v1/chat/completions", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("openai request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("openai %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	json.Unmarshal(respBody, &result)
	if len(result.Choices) == 0 {
		return "", fmt.Errorf("empty openai response")
	}

	return extractJSON(result.Choices[0].Message.Content), nil
}

// extractJSON tries to find a JSON object in the response text.
func extractJSON(text string) string {
	text = strings.TrimSpace(text)
	// Try to find JSON block
	if idx := strings.Index(text, "{"); idx >= 0 {
		if end := strings.LastIndex(text, "}"); end > idx {
			return text[idx : end+1]
		}
	}
	return text
}
