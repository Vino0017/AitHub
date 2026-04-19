package llm

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestNewClient_Anthropic tests Anthropic client creation
func TestNewClient_Anthropic(t *testing.T) {
	// Save original env vars
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	originalModel := os.Getenv("ANTHROPIC_MODEL")
	originalBaseURL := os.Getenv("ANTHROPIC_BASE_URL")
	defer func() {
		os.Setenv("ANTHROPIC_API_KEY", originalKey)
		os.Setenv("ANTHROPIC_MODEL", originalModel)
		os.Setenv("ANTHROPIC_BASE_URL", originalBaseURL)
	}()

	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")
	os.Setenv("ANTHROPIC_MODEL", "claude-opus-4")
	os.Setenv("ANTHROPIC_BASE_URL", "https://custom.anthropic.com")

	client := NewClient()

	if client.provider != "anthropic" {
		t.Errorf("Expected provider 'anthropic', got '%s'", client.provider)
	}
	if client.apiKey != "sk-ant-test-key" {
		t.Errorf("Expected API key 'sk-ant-test-key', got '%s'", client.apiKey)
	}
	if client.model != "claude-opus-4" {
		t.Errorf("Expected model 'claude-opus-4', got '%s'", client.model)
	}
	if client.baseURL != "https://custom.anthropic.com" {
		t.Errorf("Expected baseURL 'https://custom.anthropic.com', got '%s'", client.baseURL)
	}
}

// TestNewClient_AnthropicDefaults tests Anthropic default values
func TestNewClient_AnthropicDefaults(t *testing.T) {
	originalKey := os.Getenv("ANTHROPIC_API_KEY")
	defer os.Setenv("ANTHROPIC_API_KEY", originalKey)

	os.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	os.Unsetenv("ANTHROPIC_MODEL")
	os.Unsetenv("ANTHROPIC_BASE_URL")

	client := NewClient()

	if client.model != "claude-haiku-4-5-20251001" {
		t.Errorf("Expected default model 'claude-haiku-4-5-20251001', got '%s'", client.model)
	}
	if client.baseURL != "https://api.anthropic.com" {
		t.Errorf("Expected default baseURL 'https://api.anthropic.com', got '%s'", client.baseURL)
	}
}

// TestNewClient_OpenAI tests OpenAI client creation
func TestNewClient_OpenAI(t *testing.T) {
	originalKey := os.Getenv("OPENAI_API_KEY")
	originalModel := os.Getenv("OPENAI_MODEL")
	originalBaseURL := os.Getenv("OPENAI_BASE_URL")
	defer func() {
		os.Setenv("OPENAI_API_KEY", originalKey)
		os.Setenv("OPENAI_MODEL", originalModel)
		os.Setenv("OPENAI_BASE_URL", originalBaseURL)
	}()

	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Setenv("OPENAI_API_KEY", "sk-test-key")
	os.Setenv("OPENAI_MODEL", "gpt-4")
	os.Setenv("OPENAI_BASE_URL", "https://custom.openai.com")

	client := NewClient()

	if client.provider != "openai" {
		t.Errorf("Expected provider 'openai', got '%s'", client.provider)
	}
	if client.apiKey != "sk-test-key" {
		t.Errorf("Expected API key 'sk-test-key', got '%s'", client.apiKey)
	}
	if client.model != "gpt-4" {
		t.Errorf("Expected model 'gpt-4', got '%s'", client.model)
	}
}

// TestNewClient_OpenAIDefaults tests OpenAI default values
func TestNewClient_OpenAIDefaults(t *testing.T) {
	originalKey := os.Getenv("OPENAI_API_KEY")
	defer os.Setenv("OPENAI_API_KEY", originalKey)

	os.Unsetenv("ANTHROPIC_API_KEY")
	os.Setenv("OPENAI_API_KEY", "sk-test")
	os.Unsetenv("OPENAI_MODEL")
	os.Unsetenv("OPENAI_BASE_URL")

	client := NewClient()

	if client.model != "gpt-4o-mini" {
		t.Errorf("Expected default model 'gpt-4o-mini', got '%s'", client.model)
	}
	if client.baseURL != "https://api.openai.com" {
		t.Errorf("Expected default baseURL 'https://api.openai.com', got '%s'", client.baseURL)
	}
}

// TestNewClient_LLMPriority tests LLM_ env vars take priority
func TestNewClient_LLMPriority(t *testing.T) {
	originalLLMKey := os.Getenv("LLM_API_KEY")
	originalLLMProvider := os.Getenv("LLM_PROVIDER")
	originalAnthropicKey := os.Getenv("ANTHROPIC_API_KEY")
	defer func() {
		os.Setenv("LLM_API_KEY", originalLLMKey)
		os.Setenv("LLM_PROVIDER", originalLLMProvider)
		os.Setenv("ANTHROPIC_API_KEY", originalAnthropicKey)
	}()

	os.Setenv("LLM_API_KEY", "llm-key")
	os.Setenv("LLM_PROVIDER", "custom")
	os.Setenv("ANTHROPIC_API_KEY", "anthropic-key")

	client := NewClient()

	if client.provider != "custom" {
		t.Errorf("Expected LLM_PROVIDER to take priority, got '%s'", client.provider)
	}
	if client.apiKey != "llm-key" {
		t.Errorf("Expected LLM_API_KEY to take priority, got '%s'", client.apiKey)
	}
}

// TestIsConfigured tests configuration check
func TestIsConfigured(t *testing.T) {
	tests := []struct {
		name       string
		apiKey     string
		configured bool
	}{
		{"With API key", "sk-test-key", true},
		{"Empty API key", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &Client{apiKey: tt.apiKey}
			if client.IsConfigured() != tt.configured {
				t.Errorf("Expected IsConfigured() = %v, got %v", tt.configured, client.IsConfigured())
			}
		})
	}
}

// TestComplete_NotConfigured tests completion without configuration
func TestComplete_NotConfigured(t *testing.T) {
	client := &Client{} // No API key

	ctx := context.Background()
	_, err := client.Complete(ctx, "test prompt")

	if err == nil {
		t.Error("Expected error for unconfigured client")
	}
	if err.Error() != "LLM not configured" {
		t.Errorf("Expected 'LLM not configured', got '%v'", err)
	}
}

// TestCompleteAnthropic_Success tests successful Anthropic completion
func TestCompleteAnthropic_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Header.Get("x-api-key") != "test-key" {
			t.Error("Expected x-api-key header")
		}
		if r.Header.Get("anthropic-version") != "2023-06-01" {
			t.Error("Expected anthropic-version header")
		}

		// Return mock response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content":[{"text":"{\"result\": \"success\"}"}]}`))
	}))
	defer server.Close()

	client := &Client{
		provider: "anthropic",
		apiKey:   "test-key",
		baseURL:  server.URL,
		model:    "claude-test",
		client:   &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	result, err := client.Complete(ctx, "test prompt")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !strings.Contains(result, "success") {
		t.Errorf("Expected result to contain 'success', got '%s'", result)
	}
}

// TestCompleteAnthropic_Error tests Anthropic error handling
func TestCompleteAnthropic_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid_api_key"}`))
	}))
	defer server.Close()

	client := &Client{
		provider: "anthropic",
		apiKey:   "invalid-key",
		baseURL:  server.URL,
		model:    "claude-test",
		client:   &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, "test prompt")

	if err == nil {
		t.Error("Expected error for invalid API key")
	}
	if !strings.Contains(err.Error(), "401") {
		t.Errorf("Expected error to contain '401', got '%v'", err)
	}
}

// TestCompleteOpenAI_Success tests successful OpenAI completion
func TestCompleteOpenAI_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Error("Expected Bearer token in Authorization header")
		}

		// Return mock response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"{\"result\": \"success\"}"}}]}`))
	}))
	defer server.Close()

	client := &Client{
		provider: "openai",
		apiKey:   "test-key",
		baseURL:  server.URL,
		model:    "gpt-test",
		client:   &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	result, err := client.Complete(ctx, "test prompt")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !strings.Contains(result, "success") {
		t.Errorf("Expected result to contain 'success', got '%s'", result)
	}
}

// TestCompleteOpenAI_Error tests OpenAI error handling
func TestCompleteOpenAI_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": {"message": "Invalid API key"}}`))
	}))
	defer server.Close()

	client := &Client{
		provider: "openai",
		apiKey:   "invalid-key",
		baseURL:  server.URL,
		model:    "gpt-test",
		client:   &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, "test prompt")

	if err == nil {
		t.Error("Expected error for invalid API key")
	}
}

// TestExtractJSON tests JSON extraction from text
func TestExtractJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		contains string
	}{
		{
			"Pure JSON",
			`{"result": "success"}`,
			"result",
		},
		{
			"JSON with prefix",
			`Here is the result: {"result": "success"}`,
			"result",
		},
		{
			"JSON with suffix",
			`{"result": "success"} - that's the answer`,
			"result",
		},
		{
			"No JSON",
			`This is plain text`,
			"plain text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSON(tt.input)
			if !strings.Contains(result, tt.contains) {
				t.Errorf("Expected result to contain '%s', got '%s'", tt.contains, result)
			}
		})
	}
}

// TestComplete_ContextCancellation tests context cancellation
func TestComplete_ContextCancellation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"content":[{"text":"success"}]}`))
	}))
	defer server.Close()

	client := &Client{
		provider: "anthropic",
		apiKey:   "test-key",
		baseURL:  server.URL,
		model:    "claude-test",
		client:   &http.Client{Timeout: 5 * time.Second},
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := client.Complete(ctx, "test prompt")

	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

// TestCompleteOpenAI_BaseURLTrimming tests baseURL trailing slash handling
func TestCompleteOpenAI_BaseURLTrimming(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify path doesn't have double slashes
		if strings.Contains(r.URL.Path, "//") {
			t.Error("URL path contains double slashes")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"choices":[{"message":{"content":"success"}}]}`))
	}))
	defer server.Close()

	client := &Client{
		provider: "openai",
		apiKey:   "test-key",
		baseURL:  server.URL + "/", // Trailing slash
		model:    "gpt-test",
		client:   &http.Client{Timeout: 5 * time.Second},
	}

	ctx := context.Background()
	_, err := client.Complete(ctx, "test prompt")

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}
