package db

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func newDB(ctx context.Context, dsn string) (*sql.DB, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// buildDSN builds a connection string with the specified database name
func buildDSN(dbName string) string {
	host := envOrDefault("POP_DB_HOST", "pop-rw.citypost.us")
	port := envOrDefault("POP_DB_PORT", "5432")
	user := envOrDefault("POP_DB_USER", "postgres")
	password := envOrDefault("POP_DB_PASSWORD", "Asterisk@123")

	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbName,
	)
}

// ensurePopDatabase ensures the pop database exists
func ensurePopDatabase(ctx context.Context) error {
	// Connect to default postgres database first
	dsn := buildDSN("postgres")
	db, err := newDB(ctx, dsn)
	if err != nil {
		return fmt.Errorf("failed to connect to default postgres database: %w", err)
	}
	defer db.Close()

	// Check if pop database exists
	var exists int
	err = db.QueryRowContext(ctx, "SELECT 1 FROM pg_database WHERE datname = $1", "pop").Scan(&exists)
	if err == sql.ErrNoRows {
		// Create pop database if it doesn't exist
		if _, err = db.ExecContext(ctx, "CREATE DATABASE pop"); err != nil {
			return fmt.Errorf("failed to create pop database: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("failed to check if pop database exists: %w", err)
	}

	return nil
}

// NewPopDB creates a new connection to the pop database
func NewPopDB(ctx context.Context) (*sql.DB, error) {
	// First ensure the pop database exists
	if err := ensurePopDatabase(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure pop database exists: %w", err)
	}
	
	// Now connect to the pop database
	return newDB(ctx, buildDSN("pop"))
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// RunMigrations ensures the pop database exists and applies all SQL files in the
// given migrations directory in filename order.
func RunMigrations(ctx context.Context, migrationsDir string) error {
	if err := ensurePopDatabase(ctx); err != nil {
		return err
	}

	popDB, err := NewPopDB(ctx)
	if err != nil {
		return err
	}
	defer popDB.Close()

	entries := []fs.DirEntry{}
	if err := filepath.WalkDir(migrationsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		entries = append(entries, d)
		return nil
	}); err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, e := range entries {
		path := filepath.Join(migrationsDir, e.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if _, err := popDB.ExecContext(ctx, string(b)); err != nil {
			return err
		}
	}

	return nil
}

