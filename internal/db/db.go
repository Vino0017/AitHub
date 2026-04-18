package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/pressly/goose/v3"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
)

// Connect creates a connection pool to PostgreSQL.
func Connect(ctx context.Context) (*pgxpool.Pool, error) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		return nil, fmt.Errorf("DATABASE_URL not set")
	}

	config, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("parse db config: %w", err)
	}

	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 30 * time.Minute
	config.MaxConnIdleTime = 5 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return pool, nil
}

// RunMigrations runs goose migrations from the "migrations" directory.
func RunMigrations(ctx context.Context, pool *pgxpool.Pool) error {
	db := stdlib.OpenDB(*pool.Config().ConnConfig.Copy())
	defer db.Close()

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("set dialect: %w", err)
	}

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	log.Println("migrations completed")

	// Run River queue migrations
	riverMigrator, err := rivermigrate.New(riverpgxv5.New(pool), nil)
	if err != nil {
		return fmt.Errorf("river migrator: %w", err)
	}
	_, err = riverMigrator.Migrate(ctx, rivermigrate.DirectionUp, nil)
	if err != nil {
		return fmt.Errorf("river migrations: %w", err)
	}
	log.Println("river queue migrations completed")

	return nil
}

// RunSeed executes the seed SQL file if it exists.
func RunSeed(ctx context.Context, pool *pgxpool.Pool) error {
	seedSQL, err := os.ReadFile("scripts/seed.sql")
	if err != nil {
		if os.IsNotExist(err) {
			log.Println("no seed.sql found, skipping")
			return nil
		}
		return fmt.Errorf("read seed.sql: %w", err)
	}

	_, err = pool.Exec(ctx, string(seedSQL))
	if err != nil {
		return fmt.Errorf("exec seed.sql: %w", err)
	}

	log.Println("seed data loaded")
	return nil
}

// OpenStdDB creates a standard *sql.DB from the pool config.
func OpenStdDB(pool *pgxpool.Pool) *sql.DB {
	return stdlib.OpenDB(*pool.Config().ConnConfig.Copy())
}
