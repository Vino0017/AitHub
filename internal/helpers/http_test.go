package helpers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestReadJSON_ValidJSON tests reading valid JSON
func TestReadJSON_ValidJSON(t *testing.T) {
	type TestStruct struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	jsonData := `{"name":"test","value":42}`
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(jsonData))
	req.Header.Set("Content-Type", "application/json")

	var result TestStruct
	err := ReadJSON(req, &result)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if result.Name != "test" {
		t.Errorf("Expected name 'test', got '%s'", result.Name)
	}
	if result.Value != 42 {
		t.Errorf("Expected value 42, got %d", result.Value)
	}
}

// TestReadJSON_InvalidJSON tests reading invalid JSON
func TestReadJSON_InvalidJSON(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}

	jsonData := `{"name":"test",}` // Invalid JSON (trailing comma)
	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(jsonData))
	req.Header.Set("Content-Type", "application/json")

	var result TestStruct
	err := ReadJSON(req, &result)

	if err == nil {
		t.Error("Expected error for invalid JSON")
	}
}

// TestReadJSON_EmptyBody tests reading empty body
func TestReadJSON_EmptyBody(t *testing.T) {
	type TestStruct struct {
		Name string `json:"name"`
	}

	req := httptest.NewRequest("POST", "/test", bytes.NewBufferString(""))
	req.Header.Set("Content-Type", "application/json")

	var result TestStruct
	err := ReadJSON(req, &result)

	if err == nil {
		t.Error("Expected error for empty body")
	}
}

// TestWriteJSON_ValidData tests writing valid JSON response
func TestWriteJSON_ValidData(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]interface{}{
		"status":  "success",
		"message": "test message",
		"count":   42,
	}

	WriteJSON(w, http.StatusOK, data)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	if err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	if result["status"] != "success" {
		t.Errorf("Expected status 'success', got '%v'", result["status"])
	}
	if result["message"] != "test message" {
		t.Errorf("Expected message 'test message', got '%v'", result["message"])
	}
	if result["count"] != float64(42) { // JSON numbers are float64
		t.Errorf("Expected count 42, got '%v'", result["count"])
	}
}

// TestWriteJSON_NilData tests writing nil data
func TestWriteJSON_NilData(t *testing.T) {
	w := httptest.NewRecorder()

	WriteJSON(w, http.StatusOK, nil)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()
	if body != "null\n" {
		t.Errorf("Expected 'null\\n', got '%s'", body)
	}
}

// TestWriteError_BasicError tests writing basic error response
func TestWriteError_BasicError(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusBadRequest, "invalid_input", "Invalid input provided", "")

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400, got %d", w.Code)
	}

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	if err != nil {
		t.Fatalf("Failed to parse error JSON: %v", err)
	}

	if result["error"] != "invalid_input" {
		t.Errorf("Expected error 'invalid_input', got '%v'", result["error"])
	}
	if result["message"] != "Invalid input provided" {
		t.Errorf("Expected message 'Invalid input provided', got '%v'", result["message"])
	}
}

// TestWriteError_WithAction tests writing error with action
func TestWriteError_WithAction(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusUnauthorized, "missing_token", "Token required", "Please provide a valid token")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	if err != nil {
		t.Fatalf("Failed to parse error JSON: %v", err)
	}

	if result["action"] != "Please provide a valid token" {
		t.Errorf("Expected action 'Please provide a valid token', got '%v'", result["action"])
	}
}

// TestWriteError_EmptyAction tests writing error with empty action
func TestWriteError_EmptyAction(t *testing.T) {
	w := httptest.NewRecorder()

	WriteError(w, http.StatusNotFound, "not_found", "Resource not found", "")

	var result map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &result)
	if err != nil {
		t.Fatalf("Failed to parse error JSON: %v", err)
	}

	// Action should not be present if empty
	if _, exists := result["action"]; exists {
		t.Error("Expected no 'action' field for empty action")
	}
}

// TestWriteJSON_DifferentStatusCodes tests various HTTP status codes
func TestWriteJSON_DifferentStatusCodes(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
	}{
		{"OK", http.StatusOK},
		{"Created", http.StatusCreated},
		{"Accepted", http.StatusAccepted},
		{"No Content", http.StatusNoContent},
		{"Bad Request", http.StatusBadRequest},
		{"Unauthorized", http.StatusUnauthorized},
		{"Forbidden", http.StatusForbidden},
		{"Not Found", http.StatusNotFound},
		{"Internal Server Error", http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			WriteJSON(w, tt.statusCode, map[string]string{"status": "test"})

			if w.Code != tt.statusCode {
				t.Errorf("Expected status %d, got %d", tt.statusCode, w.Code)
			}
		})
	}
}

// TestReadJSON_LargePayload tests reading large JSON payload
func TestReadJSON_LargePayload(t *testing.T) {
	type TestStruct struct {
		Data []string `json:"data"`
	}

	// Create a large array
	largeArray := make([]string, 1000)
	for i := range largeArray {
		largeArray[i] = "item"
	}

	data := TestStruct{Data: largeArray}
	jsonData, _ := json.Marshal(data)

	req := httptest.NewRequest("POST", "/test", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	var result TestStruct
	err := ReadJSON(req, &result)

	if err != nil {
		t.Errorf("Expected no error for large payload, got %v", err)
	}
	if len(result.Data) != 1000 {
		t.Errorf("Expected 1000 items, got %d", len(result.Data))
	}
}

// TestWriteJSON_SpecialCharacters tests writing JSON with special characters
func TestWriteJSON_SpecialCharacters(t *testing.T) {
	w := httptest.NewRecorder()

	data := map[string]string{
		"message": "Test with \"quotes\" and \n newlines",
		"emoji":   "🚀 🎉",
	}

	WriteJSON(w, http.StatusOK, data)

	var result map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &result)
	if err != nil {
		t.Fatalf("Failed to parse JSON with special characters: %v", err)
	}

	if result["message"] != data["message"] {
		t.Errorf("Special characters not preserved in message")
	}
	if result["emoji"] != data["emoji"] {
		t.Errorf("Emoji not preserved")
	}
}
