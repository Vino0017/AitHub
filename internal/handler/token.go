package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/crypto"
	"github.com/skillhub/api/internal/helpers"
	"github.com/skillhub/api/internal/middleware"
)

type TokenHandler struct {
	pool *pgxpool.Pool
}

func NewTokenHandler(pool *pgxpool.Pool) *TokenHandler {
	return &TokenHandler{pool: pool}
}

// Create creates a new anonymous token. POST /v1/tokens
func (h *TokenHandler) Create(w http.ResponseWriter, r *http.Request) {
	raw, err := crypto.GenerateToken()
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to generate token", "")
		return
	}

	hash := crypto.HashToken(raw)
	var id uuid.UUID
	err = h.pool.QueryRow(r.Context(),
		`INSERT INTO tokens (token_hash, label) VALUES ($1, $2) RETURNING id`,
		hash, "auto-generated").Scan(&id)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to create token", "")
		return
	}

	helpers.WriteJSON(w, http.StatusCreated, map[string]string{
		"token": raw,
		"id":    id.String(),
	})
}

// List lists all tokens for the current namespace. GET /v1/tokens
func (h *TokenHandler) List(w http.ResponseWriter, r *http.Request) {
	nsID := middleware.GetNamespaceID(r.Context())
	if nsID == nil {
		helpers.WriteError(w, http.StatusForbidden, "namespace_required", "Must be registered to list tokens", "")
		return
	}

	rows, err := h.pool.Query(r.Context(),
		`SELECT id, label, daily_uses, last_used, created_at FROM tokens WHERE namespace_id = $1 ORDER BY created_at DESC`, nsID)
	if err != nil {
		helpers.WriteError(w, http.StatusInternalServerError, "internal", "Failed to list tokens", "")
		return
	}
	defer rows.Close()

	var tokens []map[string]interface{}
	for rows.Next() {
		var id uuid.UUID
		var label string
		var dailyUses int
		var lastUsed, createdAt interface{}
		rows.Scan(&id, &label, &dailyUses, &lastUsed, &createdAt)
		tokens = append(tokens, map[string]interface{}{
			"id": id.String(), "label": label, "daily_uses": dailyUses,
			"last_used": lastUsed, "created_at": createdAt,
		})
	}
	if tokens == nil {
		tokens = []map[string]interface{}{}
	}

	helpers.WriteJSON(w, http.StatusOK, tokens)
}

// Delete revokes a token. DELETE /v1/tokens/{id}
func (h *TokenHandler) Delete(w http.ResponseWriter, r *http.Request) {
	tokenIDStr := chi.URLParam(r, "id")
	tokenID, err := uuid.Parse(tokenIDStr)
	if err != nil {
		helpers.WriteError(w, http.StatusBadRequest, "invalid_id", "Invalid token ID", "")
		return
	}

	nsID := middleware.GetNamespaceID(r.Context())
	tag, err := h.pool.Exec(r.Context(),
		`DELETE FROM tokens WHERE id = $1 AND namespace_id = $2`, tokenID, nsID)
	if err != nil || tag.RowsAffected() == 0 {
		helpers.WriteError(w, http.StatusNotFound, "not_found", "Token not found", "")
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
