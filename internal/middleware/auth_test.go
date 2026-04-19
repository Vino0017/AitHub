package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/crypto"
)

// TestAuth_ValidToken tests successful authentication
func TestAuth_ValidToken(t *testing.T) {
	t.Skip("Requires database setup")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	// Create test namespace and token
	nsID := uuid.New()
	tokenID := uuid.New()
	token, _ := crypto.GenerateToken()
	tokenHash := crypto.HashToken(token)

	_, err := pool.Exec(context.Background(),
		`INSERT INTO namespaces (id, name, type, email) VALUES ($1, $2, 'personal', $3)`,
		nsID, "test-user", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO tokens (id, namespace_id, token_hash, label) VALUES ($1, $2, $3, $4)`,
		tokenID, nsID, tokenHash, "test-token")
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	// Create test handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify context values
		if GetTokenID(r.Context()) == nil {
			t.Error("Expected token ID in context")
		}
		if GetNamespaceID(r.Context()) == nil {
			t.Error("Expected namespace ID in context")
		}
		if IsAnonymous(r.Context()) {
			t.Error("Expected non-anonymous token")
		}
		w.WriteHeader(http.StatusOK)
	})

	// Wrap with auth middleware
	authHandler := Auth(pool)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	authHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d: %s", w.Code, w.Body.String())
	}
}

// TestAuth_MissingToken tests missing authorization header
func TestAuth_MissingToken(t *testing.T) {
	t.Skip("Requires database setup")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	authHandler := Auth(pool)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	authHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

// TestAuth_InvalidToken tests invalid token
func TestAuth_InvalidToken(t *testing.T) {
	t.Skip("Requires database setup")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	authHandler := Auth(pool)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-12345")
	w := httptest.NewRecorder()

	authHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

// TestAuth_BannedNamespace tests banned namespace
func TestAuth_BannedNamespace(t *testing.T) {
	t.Skip("Requires database setup")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	// Create banned namespace
	nsID := uuid.New()
	tokenID := uuid.New()
	token, _ := crypto.GenerateToken()
	tokenHash := crypto.HashToken(token)

	_, err := pool.Exec(context.Background(),
		`INSERT INTO namespaces (id, name, type, email, banned) VALUES ($1, $2, 'personal', $3, true)`,
		nsID, "banned-user", "banned@example.com")
	if err != nil {
		t.Fatalf("Failed to create namespace: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO tokens (id, namespace_id, token_hash, label) VALUES ($1, $2, $3, $4)`,
		tokenID, nsID, tokenHash, "test-token")
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	authHandler := Auth(pool)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	authHandler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", w.Code)
	}
}

// TestAuth_AnonymousToken tests anonymous token (no namespace)
func TestAuth_AnonymousToken(t *testing.T) {
	t.Skip("Requires database setup")

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	// Create anonymous token (no namespace_id)
	tokenID := uuid.New()
	token, _ := crypto.GenerateToken()
	tokenHash := crypto.HashToken(token)

	_, err := pool.Exec(context.Background(),
		`INSERT INTO tokens (id, namespace_id, token_hash, label) VALUES ($1, NULL, $2, $3)`,
		tokenID, tokenHash, "anonymous-token")
	if err != nil {
		t.Fatalf("Failed to create token: %v", err)
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !IsAnonymous(r.Context()) {
			t.Error("Expected anonymous token")
		}
		if GetNamespaceID(r.Context()) != nil {
			t.Error("Expected no namespace ID for anonymous token")
		}
		w.WriteHeader(http.StatusOK)
	})

	authHandler := Auth(pool)(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()

	authHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// TestRequireNamespace_WithNamespace tests namespace requirement with valid namespace
func TestRequireNamespace_WithNamespace(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	requireNsHandler := RequireNamespace(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), CtxIsAnonymous, false)
	ctx = context.WithValue(ctx, CtxNamespaceID, uuid.New())
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	requireNsHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// TestRequireNamespace_Anonymous tests namespace requirement with anonymous token
func TestRequireNamespace_Anonymous(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	requireNsHandler := RequireNamespace(handler)

	req := httptest.NewRequest("GET", "/test", nil)
	ctx := context.WithValue(req.Context(), CtxIsAnonymous, true)
	req = req.WithContext(ctx)

	w := httptest.NewRecorder()
	requireNsHandler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected 403, got %d", w.Code)
	}
}

// TestAdminAuth_ValidToken tests admin authentication with valid token
func TestAdminAuth_ValidToken(t *testing.T) {
	adminToken := "admin-secret-token-12345"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	adminHandler := AdminAuth(adminToken)(handler)

	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	w := httptest.NewRecorder()

	adminHandler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// TestAdminAuth_InvalidToken tests admin authentication with invalid token
func TestAdminAuth_InvalidToken(t *testing.T) {
	adminToken := "admin-secret-token-12345"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("Handler should not be called")
	})

	adminHandler := AdminAuth(adminToken)(handler)

	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	w := httptest.NewRecorder()

	adminHandler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

// --- Test Helpers ---

func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	dbURL := "postgres://skillhub:skillhub_dev@localhost:5432/skillhub_test?sslmode=disable"

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test DB: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("Skipping test: cannot ping test DB: %v", err)
	}

	cleanup := func() {
		pool.Exec(context.Background(), "TRUNCATE namespaces, tokens CASCADE")
		pool.Close()
	}

	return pool, cleanup
}
