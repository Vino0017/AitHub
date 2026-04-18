package helpers

import (
	"encoding/json"
	"net/http"
)

// WriteJSON writes a JSON response.
func WriteJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// WriteError writes a structured error response.
func WriteError(w http.ResponseWriter, status int, errCode, message, action string) {
	resp := map[string]interface{}{
		"error":   errCode,
		"status":  status,
		"message": message,
	}
	if action != "" {
		resp["action"] = action
	}
	WriteJSON(w, status, resp)
}

// ReadJSON decodes a JSON request body.
func ReadJSON(r *http.Request, v interface{}) error {
	return json.NewDecoder(r.Body).Decode(v)
}
