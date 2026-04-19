package handler

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/skillhub/api/internal/middleware"
)

// setupTestDB creates a test database connection
func setupTestDB(t *testing.T) (*pgxpool.Pool, func()) {
	t.Helper()

	dbURL := "postgres://skillhub:skillhub_dev@localhost:5432/skillhub_test?sslmode=disable"

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("Skipping integration test: cannot connect to test DB: %v", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		t.Skipf("Skipping integration test: cannot ping test DB: %v", err)
	}

	cleanup := func() {
		pool.Exec(context.Background(), "TRUNCATE namespaces, tokens, skills, revisions, ratings CASCADE")
		pool.Close()
	}

	return pool, cleanup
}

// setupTestNamespace creates a test namespace and token
func setupTestNamespace(t *testing.T, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID) {
	t.Helper()

	nsID := uuid.New()
	tokenID := uuid.New()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO namespaces (id, name, type, email) VALUES ($1, $2, 'personal', $3)`,
		nsID, "test-user", "test@example.com")
	if err != nil {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO tokens (id, namespace_id, token_hash, label) VALUES ($1, $2, $3, $4)`,
		tokenID, nsID, "test-hash", "test-token")
	if err != nil {
		t.Fatalf("Failed to create test token: %v", err)
	}

	return nsID, tokenID
}

// setupTestSkill creates a test skill with an approved revision
func setupTestSkill(t *testing.T, pool *pgxpool.Pool, nsID uuid.UUID) (uuid.UUID, uuid.UUID) {
	t.Helper()

	skillID := uuid.New()
	revisionID := uuid.New()

	_, err := pool.Exec(context.Background(),
		`INSERT INTO skills (id, namespace_id, name, description, tags, framework, visibility, latest_version)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		skillID, nsID, "test-skill", "Test skill", []string{"test"}, "gstack", "public", "1.0.0")
	if err != nil {
		t.Fatalf("Failed to create test skill: %v", err)
	}

	_, err = pool.Exec(context.Background(),
		`INSERT INTO revisions (id, skill_id, version, content, review_status)
		 VALUES ($1, $2, $3, $4, $5)`,
		revisionID, skillID, "1.0.0", "test content", "approved")
	if err != nil {
		t.Fatalf("Failed to create test revision: %v", err)
	}

	return skillID, revisionID
}

// contextWithAuth creates a context with authentication values
func contextWithAuth(ctx context.Context, nsID, tokenID uuid.UUID) context.Context {
	ctx = context.WithValue(ctx, middleware.CtxNamespaceID, nsID)
	ctx = context.WithValue(ctx, middleware.CtxTokenID, tokenID)
	ctx = context.WithValue(ctx, middleware.CtxIsAnonymous, false)
	return ctx
}
