package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/crypto"
	"github.com/skillhub/api/internal/email"
	"github.com/skillhub/api/internal/helpers"
)

type AuthHandler struct {
	pool   *pgxpool.Pool
	email  *email.Sender
}

func NewAuthHandler(pool *pgxpool.Pool, emailSender *email.Sender) *AuthHandler {
	return &AuthHandler{pool: pool, email: emailSender}
}

// GitHubDeviceStart initiates GitHub OAuth device flow. POST /v1/auth/github
func (h *AuthHandler) GitHubDeviceStart(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	if clientID == "" {
		helpers.WriteError(w, http.StatusServiceUnavailable, "github_not_configured", "GitHub OAuth not configured", "")
		return
	}

	formData := url.Values{
		"client_id": {clientID},
		"scope":     {"read:user"},
	}
	req, _ := http.NewRequest("POST", "https://github.com/login/device/code",
		strings.NewReader(formData.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		helpers.WriteError(w, http.StatusBadGateway, "github_error", "Failed to contact GitHub", "")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		helpers.WriteError(w, http.StatusBadGateway, "github_error",
			fmt.Sprintf("GitHub returned status %d", resp.StatusCode), "")
		return
	}

	var result struct {
		DeviceCode      string `json:"device_code"`
		UserCode        string `json:"user_code"`
		VerificationURI string `json:"verification_uri"`
		ExpiresIn       int    `json:"expires_in"`
		Error           string `json:"error"`
	}
	respBody, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(respBody, &result); err != nil {
		helpers.WriteError(w, http.StatusBadGateway, "github_error", "Invalid response from GitHub", "")
		return
	}
	if result.Error != "" {
		helpers.WriteError(w, http.StatusBadGateway, "github_error", "GitHub error: "+result.Error, "")
		return
	}

	h.pool.Exec(r.Context(),
		`INSERT INTO oauth_device_flows (provider, device_code, user_code, verification_uri, expires_at)
		 VALUES ('github', $1, $2, $3, $4)`,
		result.DeviceCode, result.UserCode, result.VerificationURI,
		time.Now().Add(time.Duration(result.ExpiresIn)*time.Second))

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"device_code": result.DeviceCode, "user_code": result.UserCode,
		"verification_uri": result.VerificationURI, "expires_in": result.ExpiresIn,
		"instruction": fmt.Sprintf("Open %s and enter code: %s", result.VerificationURI, result.UserCode),
	})
}

// GitHubDevicePoll polls for completion. POST /v1/auth/github/poll
func (h *AuthHandler) GitHubDevicePoll(w http.ResponseWriter, r *http.Request) {
	clientID := os.Getenv("GITHUB_CLIENT_ID")
	if clientID == "" {
		helpers.WriteError(w, http.StatusServiceUnavailable, "github_not_configured", "GitHub OAuth not configured", "")
		return
	}

	var req struct {
		DeviceCode string `json:"device_code"`
	}
	if err := helpers.ReadJSON(r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON", "")
		return
	}

	formData := url.Values{
		"client_id":   {clientID},
		"device_code": {req.DeviceCode},
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
	}
	tokenReq, _ := http.NewRequest("POST", "https://github.com/login/oauth/access_token",
		strings.NewReader(formData.Encode()))
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenReq.Header.Set("Accept", "application/json")
	tokenResp, err := http.DefaultClient.Do(tokenReq)
	if err != nil {
		helpers.WriteError(w, http.StatusBadGateway, "github_error", "Failed to poll", "")
		return
	}
	defer tokenResp.Body.Close()

	if tokenResp.StatusCode != http.StatusOK {
		helpers.WriteError(w, http.StatusBadGateway, "github_error",
			fmt.Sprintf("GitHub returned status %d", tokenResp.StatusCode), "")
		return
	}

	var tokenResult struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	respBody, _ := io.ReadAll(tokenResp.Body)
	if err := json.Unmarshal(respBody, &tokenResult); err != nil {
		helpers.WriteError(w, http.StatusBadGateway, "github_error", "Invalid response from GitHub", "")
		return
	}

	if tokenResult.Error != "" {
		helpers.WriteJSON(w, http.StatusOK, map[string]string{"status": "pending", "error": tokenResult.Error})
		return
	}

	// Get GitHub user
	userReq, _ := http.NewRequest("GET", "https://api.github.com/user", nil)
	userReq.Header.Set("Authorization", "Bearer "+tokenResult.AccessToken)
	userResp, err := http.DefaultClient.Do(userReq)
	if err != nil {
		helpers.WriteError(w, http.StatusBadGateway, "github_error", "Failed to get user", "")
		return
	}
	defer userResp.Body.Close()

	if userResp.StatusCode != http.StatusOK {
		helpers.WriteError(w, http.StatusBadGateway, "github_error",
			fmt.Sprintf("GitHub /user returned status %d", userResp.StatusCode), "")
		return
	}

	var ghUser struct {
		Login string `json:"login"`
		ID    int    `json:"id"`
	}
	respBody, _ = io.ReadAll(userResp.Body)
	if err := json.Unmarshal(respBody, &ghUser); err != nil {
		helpers.WriteError(w, http.StatusBadGateway, "github_error", "Invalid user response from GitHub", "")
		return
	}
	if ghUser.Login == "" || ghUser.ID == 0 {
		helpers.WriteError(w, http.StatusBadGateway, "github_error", "GitHub returned empty user data", "")
		return
	}

	ghIDStr := fmt.Sprintf("%d", ghUser.ID)
	var nsID uuid.UUID
	err = h.pool.QueryRow(r.Context(), `SELECT id FROM namespaces WHERE github_id = $1`, ghIDStr).Scan(&nsID)
	if err != nil {
		err = h.pool.QueryRow(r.Context(),
			`INSERT INTO namespaces (name, type, github_id) VALUES ($1, 'personal', $2) RETURNING id`,
			ghUser.Login, ghIDStr).Scan(&nsID)
		if err != nil {
			helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to create namespace", "")
			return
		}
	}

	raw, _ := crypto.GenerateToken()
	hash := crypto.HashToken(raw)
	var tokenID uuid.UUID
	h.pool.QueryRow(r.Context(),
		`INSERT INTO tokens (namespace_id, token_hash, label) VALUES ($1, $2, 'github-oauth') RETURNING id`,
		nsID, hash).Scan(&tokenID)

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status": "complete", "token": raw, "namespace": ghUser.Login, "token_id": tokenID,
	})
}

// EmailSend sends verification code. POST /v1/auth/email/send
func (h *AuthHandler) EmailSend(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email     string `json:"email"`
		Namespace string `json:"namespace"`
	}
	if err := helpers.ReadJSON(r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON", "")
		return
	}

	code := crypto.RandomHex(3) // 6 char hex
	h.pool.Exec(r.Context(),
		`INSERT INTO email_verifications (email, namespace, code, expires_at) VALUES ($1, $2, $3, $4)`,
		req.Email, req.Namespace, code, time.Now().Add(10*time.Minute))

	// Send email if SMTP configured, otherwise dev fallback
	resp := map[string]interface{}{
		"status": "sent", "message": "Verification code sent to " + req.Email,
	}
	if h.email.IsEnabled() {
		if err := h.email.SendVerificationCode(req.Email, code, req.Namespace); err != nil {
			log.Printf("email: failed to send to %s: %v", req.Email, err)
			helpers.WriteError(w, http.StatusInternalServerError, "email_failed",
				"Failed to send verification email. Please try again.", "")
			return
		}
	} else if os.Getenv("SKILLHUB_DEV_MODE") == "true" {
		resp["dev_code"] = code
	}
	helpers.WriteJSON(w, http.StatusOK, resp)
}

// EmailVerify verifies code, creates namespace. POST /v1/auth/email/verify
func (h *AuthHandler) EmailVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Code  string `json:"code"`
	}
	if err := helpers.ReadJSON(r, &req); err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON", "")
		return
	}

	var nsName string
	err := h.pool.QueryRow(r.Context(),
		`UPDATE email_verifications SET used = TRUE
		 WHERE email = $1 AND code = $2 AND used = FALSE AND expires_at > NOW()
		 RETURNING namespace`, req.Email, req.Code).Scan(&nsName)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_code", "Invalid or expired code", "")
		return
	}

	var nsID uuid.UUID
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO namespaces (name, type, email) VALUES ($1, 'personal', $2) RETURNING id`,
		nsName, req.Email).Scan(&nsID)
	if err != nil {
		helpers.WriteError(w, http.StatusConflict, "name_taken", "Namespace name already taken", "")
		return
	}

	raw, _ := crypto.GenerateToken()
	hash := crypto.HashToken(raw)
	var tokenID uuid.UUID
	h.pool.QueryRow(r.Context(),
		`INSERT INTO tokens (namespace_id, token_hash, label) VALUES ($1, $2, 'email-auth') RETURNING id`,
		nsID, hash).Scan(&tokenID)

	helpers.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status": "verified", "token": raw, "namespace": nsName, "token_id": tokenID,
	})
}
