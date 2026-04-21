package embedding

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

// Client handles embedding generation via Jina API.
type Client struct {
	apiKey string
	model  string
	dims   int
	client *http.Client
}

// NewClient creates a new embedding client from environment variables.
func NewClient() *Client {
	model := os.Getenv("JINA_MODEL")
	if model == "" {
		model = "jina-embeddings-v4"
	}
	return &Client{
		apiKey: os.Getenv("JINA_API_KEY"),
		model:  model,
		dims:   1024,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

// IsConfigured returns true if the client has a valid API key.
func (c *Client) IsConfigured() bool {
	return c.apiKey != ""
}

// Embed generates an embedding vector for the given text.
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	vectors, err := c.EmbedBatch(ctx, []string{text})
	if err != nil {
		return nil, err
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	return vectors[0], nil
}

// EmbedBatch generates embeddings for multiple texts.
func (c *Client) EmbedBatch(ctx context.Context, texts []string) ([][]float32, error) {
	if !c.IsConfigured() {
		return nil, fmt.Errorf("Jina embedding not configured (JINA_API_KEY missing)")
	}

	body, _ := json.Marshal(map[string]interface{}{
		"model":      c.model,
		"input":      texts,
		"task":       "text-matching",
		"dimensions": c.dims,
	})

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.jina.ai/v1/embeddings",
		bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("jina request: %w", err)
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("jina %d: %s", resp.StatusCode, string(respBody))
	}

	var result struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	vectors := make([][]float32, len(result.Data))
	for i, d := range result.Data {
		vectors[i] = d.Embedding
	}
	return vectors, nil
}

// SkillEmbeddingText builds the text to embed for a skill.
// Includes name, description, tags, and framework for comprehensive matching.
func SkillEmbeddingText(name, description string, tags []string, framework string) string {
	parts := []string{name}
	if description != "" {
		parts = append(parts, description)
	}
	if len(tags) > 0 {
		parts = append(parts, "tags: "+strings.Join(tags, ", "))
	}
	if framework != "" {
		parts = append(parts, "framework: "+framework)
	}
	return strings.Join(parts, " | ")
}
