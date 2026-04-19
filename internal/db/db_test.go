package db

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestConnect_Success tests successful database connection
func TestConnect_Success(t *testing.T) {
	// Set test database URL
	originalURL := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", originalURL)

	testURL := "postgres://skillhub:skillhub_dev@localhost:5432/skillhub_test?sslmode=disable"
	os.Setenv("DATABASE_URL", testURL)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pool, err := Connect(ctx)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test DB: %v", err)
	}
	defer pool.Close()

	// Verify connection
	if err := pool.Ping(ctx); err != nil {
		t.Errorf("Failed to ping database: %v", err)
	}

	// Verify pool configuration
	config := pool.Config()
	if config.MaxConns != 25 {
		t.Errorf("Expected MaxConns 25, got %d", config.MaxConns)
	}
	if config.MinConns != 5 {
		t.Errorf("Expected MinConns 5, got %d", config.MinConns)
	}
	if config.MaxConnLifetime != 30*time.Minute {
		t.Errorf("Expected MaxConnLifetime 30m, got %v", config.MaxConnLifetime)
	}
	if config.MaxConnIdleTime != 5*time.Minute {
		t.Errorf("Expected MaxConnIdleTime 5m, got %v", config.MaxConnIdleTime)
	}
}

// TestConnect_MissingDatabaseURL tests missing DATABASE_URL
func TestConnect_MissingDatabaseURL(t *testing.T) {
	originalURL := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", originalURL)

	os.Unsetenv("DATABASE_URL")

	ctx := context.Background()
	_, err := Connect(ctx)

	if err == nil {
		t.Error("Expected error for missing DATABASE_URL")
	}
	if err.Error() != "DATABASE_URL not set" {
		t.Errorf("Expected 'DATABASE_URL not set', got '%v'", err)
	}
}

// TestConnect_InvalidDatabaseURL tests invalid DATABASE_URL
func TestConnect_InvalidDatabaseURL(t *testing.T) {
	originalURL := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", originalURL)

	os.Setenv("DATABASE_URL", "invalid://url")

	ctx := context.Background()
	_, err := Connect(ctx)

	if err == nil {
		t.Error("Expected error for invalid DATABASE_URL")
	}
}

// TestConnect_UnreachableDatabase tests connection to unreachable database
func TestConnect_UnreachableDatabase(t *testing.T) {
	originalURL := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", originalURL)

	// Use a non-existent host
	os.Setenv("DATABASE_URL", "postgres://user:pass@nonexistent-host-12345:5432/db")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	_, err := Connect(ctx)

	if err == nil {
		t.Error("Expected error for unreachable database")
	}
}

// TestOpenStdDB tests opening standard database connection
func TestOpenStdDB(t *testing.T) {
	originalURL := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", originalURL)

	testURL := "postgres://skillhub:skillhub_dev@localhost:5432/skillhub_test?sslmode=disable"
	os.Setenv("DATABASE_URL", testURL)

	ctx := context.Background()
	pool, err := Connect(ctx)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test DB: %v", err)
	}
	defer pool.Close()

	stdDB := OpenStdDB(pool)
	if stdDB == nil {
		t.Error("Expected non-nil *sql.DB")
	}
	defer stdDB.Close()

	// Verify standard DB works
	if err := stdDB.Ping(); err != nil {
		t.Errorf("Failed to ping standard DB: %v", err)
	}
}

// TestRunMigrations tests migration execution
func TestRunMigrations(t *testing.T) {
	t.Skip("Requires migrations directory and test database setup")

	originalURL := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", originalURL)

	testURL := "postgres://skillhub:skillhub_dev@localhost:5432/skillhub_test?sslmode=disable"
	os.Setenv("DATABASE_URL", testURL)

	ctx := context.Background()
	pool, err := Connect(ctx)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test DB: %v", err)
	}
	defer pool.Close()

	// Run migrations
	err = RunMigrations(ctx, pool)
	if err != nil {
		t.Errorf("Failed to run migrations: %v", err)
	}

	// Verify migrations ran by checking for a table
	var exists bool
	err = pool.QueryRow(ctx, "SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_name = 'namespaces')").Scan(&exists)
	if err != nil {
		t.Errorf("Failed to check for namespaces table: %v", err)
	}
	if !exists {
		t.Error("Expected namespaces table to exist after migrations")
	}
}

// TestRunSeed_NoSeedFile tests seed with missing file
func TestRunSeed_NoSeedFile(t *testing.T) {
	originalURL := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", originalURL)

	testURL := "postgres://skillhub:skillhub_dev@localhost:5432/skillhub_test?sslmode=disable"
	os.Setenv("DATABASE_URL", testURL)

	ctx := context.Background()
	pool, err := Connect(ctx)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test DB: %v", err)
	}
	defer pool.Close()

	// Change to a directory without seed.sql
	originalDir, _ := os.Getwd()
	defer os.Chdir(originalDir)
	os.Chdir("/tmp")

	// Should not error if seed.sql doesn't exist
	err = RunSeed(ctx, pool)
	if err != nil {
		t.Errorf("Expected no error for missing seed.sql, got: %v", err)
	}
}

// TestConnect_ContextCancellation tests context cancellation
func TestConnect_ContextCancellation(t *testing.T) {
	originalURL := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", originalURL)

	testURL := "postgres://skillhub:skillhub_dev@localhost:5432/skillhub_test?sslmode=disable"
	os.Setenv("DATABASE_URL", testURL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := Connect(ctx)
	if err == nil {
		t.Error("Expected error for cancelled context")
	}
}

// TestConnect_PoolConfiguration tests pool configuration values
func TestConnect_PoolConfiguration(t *testing.T) {
	originalURL := os.Getenv("DATABASE_URL")
	defer os.Setenv("DATABASE_URL", originalURL)

	testURL := "postgres://skillhub:skillhub_dev@localhost:5432/skillhub_test?sslmode=disable"
	os.Setenv("DATABASE_URL", testURL)

	ctx := context.Background()
	pool, err := Connect(ctx)
	if err != nil {
		t.Skipf("Skipping test: cannot connect to test DB: %v", err)
	}
	defer pool.Close()

	config := pool.Config()

	tests := []struct {
		name     string
		actual   interface{}
		expected interface{}
	}{
		{"MaxConns", config.MaxConns, int32(25)},
		{"MinConns", config.MinConns, int32(5)},
		{"MaxConnLifetime", config.MaxConnLifetime, 30 * time.Minute},
		{"MaxConnIdleTime", config.MaxConnIdleTime, 5 * time.Minute},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.actual != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.actual)
			}
		})
	}
}
