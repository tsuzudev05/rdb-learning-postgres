// Package testhelper provides utilities for integration tests that require a
// real PostgreSQL database.
//
// # Connection strategy
//
// If the DATABASE_URL environment variable is set, SetupPostgres connects to
// that existing PostgreSQL instance (DevContainer mode).  Otherwise it starts
// a fresh PostgreSQL 16 container via testcontainers-go (CI / standalone mode).
//
// # Usage
//
// In your test file, call SetupPostgres in TestMain:
//
//	func TestMain(m *testing.M) {
//	    pool, teardown := testhelper.SetupPostgres(context.Background())
//	    defer teardown()
//	    testhelper.Pool = pool
//	    os.Exit(m.Run())
//	}
//
// Call TruncateAll between tests to reset state:
//
//	func TestXxx(t *testing.T) {
//	    t.Cleanup(func() { testhelper.TruncateAll(t, testhelper.Pool) })
//	    ...
//	}
package testhelper

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// Pool is the shared *pgxpool.Pool used by integration tests.
// Set it in TestMain after calling SetupPostgres.
var Pool *pgxpool.Pool

// schemaPath returns the absolute path to schema.sql relative to this file.
// Works regardless of where `go test` is invoked from.
func schemaPath() string {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("testhelper: cannot determine source file path")
	}
	// thisFile: .../05_DDD統合/go/internal/testhelper/postgres.go
	// schema.sql: .../05_DDD統合/schema.sql
	return filepath.Join(filepath.Dir(thisFile), "..", "..", "..", "schema.sql")
}

// SetupPostgres returns a *pgxpool.Pool connected to PostgreSQL and a teardown
// function.  The connection strategy depends on the DATABASE_URL env var:
//
//   - DATABASE_URL set   → connects to the existing instance (DevContainer)
//   - DATABASE_URL unset → starts a PostgreSQL 16 container via testcontainers-go
//
// Call teardown (typically via defer) to release resources.
func SetupPostgres(ctx context.Context) (pool *pgxpool.Pool, teardown func()) {
	if connStr := os.Getenv("DATABASE_URL"); connStr != "" {
		return setupWithExistingDB(ctx, connStr)
	}
	return setupWithTestcontainers(ctx)
}

// setupWithExistingDB connects to a running PostgreSQL instance, creates a
// dedicated "okr_test" database (idempotent), wipes it clean, and applies
// schema.sql.  Using a separate database keeps the main application DB intact.
func setupWithExistingDB(ctx context.Context, connStr string) (pool *pgxpool.Pool, teardown func()) {
	fmt.Println("[testhelper] Using existing PostgreSQL (DATABASE_URL)")

	// Step 1: create okr_test if it doesn't exist yet.
	// CREATE DATABASE cannot run inside a transaction, so we use a short-lived
	// single connection via the original connStr.
	adminPool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		panic(fmt.Sprintf("testhelper: admin connect failed: %v", err))
	}
	// Ignore "already exists" (42P04); any other error is a real failure.
	if _, err := adminPool.Exec(ctx, `CREATE DATABASE okr_test`); err != nil {
		if !isPgErrorCode(err, "42P04") {
			adminPool.Close()
			panic(fmt.Sprintf("testhelper: CREATE DATABASE okr_test failed: %v", err))
		}
	}
	adminPool.Close()

	// Step 2: connect to the isolated test database.
	cfg, err := pgxpool.ParseConfig(connStr)
	if err != nil {
		panic(fmt.Sprintf("testhelper: parse config failed: %v", err))
	}
	cfg.ConnConfig.Database = "okr_test"
	pool, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		panic(fmt.Sprintf("testhelper: connect to okr_test failed: %v", err))
	}

	// Step 3: wipe any previous test state, then apply a fresh schema.
	if _, err := pool.Exec(ctx, `DROP SCHEMA public CASCADE; CREATE SCHEMA public;`); err != nil {
		pool.Close()
		panic(fmt.Sprintf("testhelper: schema reset failed: %v", err))
	}

	schemaSQL, err := os.ReadFile(schemaPath())
	if err != nil {
		pool.Close()
		panic(fmt.Sprintf("testhelper: failed to read schema.sql: %v", err))
	}

	if _, err := pool.Exec(ctx, string(schemaSQL)); err != nil {
		pool.Close()
		panic(fmt.Sprintf("testhelper: failed to apply schema.sql: %v", err))
	}

	teardown = func() { pool.Close() }
	return pool, teardown
}

// isPgErrorCode returns true when err is a *pgconn.PgError with the given code.
func isPgErrorCode(err error, code string) bool {
	var pge interface{ SQLState() string }
	if errors.As(err, &pge) {
		return pge.SQLState() == code
	}
	return false
}

// setupWithTestcontainers starts a PostgreSQL 16 container and applies
// schema.sql.  Used in CI / standalone environments without a running DB.
func setupWithTestcontainers(ctx context.Context) (pool *pgxpool.Pool, teardown func()) {
	fmt.Println("[testhelper] Starting PostgreSQL container via testcontainers-go")

	schemaSQL, err := os.ReadFile(schemaPath())
	if err != nil {
		panic(fmt.Sprintf("testhelper: failed to read schema.sql: %v", err))
	}

	pgc, err := postgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16-alpine"),
		postgres.WithDatabase("okr_test"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(60*time.Second),
		),
	)
	if err != nil {
		panic(fmt.Sprintf("testhelper: failed to start postgres container: %v", err))
	}

	connStr, err := pgc.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		_ = pgc.Terminate(ctx)
		panic(fmt.Sprintf("testhelper: failed to get connection string: %v", err))
	}

	pool, err = pgxpool.New(ctx, connStr)
	if err != nil {
		_ = pgc.Terminate(ctx)
		panic(fmt.Sprintf("testhelper: failed to create pgxpool: %v", err))
	}

	if _, err := pool.Exec(ctx, string(schemaSQL)); err != nil {
		pool.Close()
		_ = pgc.Terminate(ctx)
		panic(fmt.Sprintf("testhelper: failed to apply schema.sql: %v", err))
	}

	teardown = func() {
		pool.Close()
		if err := pgc.Terminate(ctx); err != nil {
			fmt.Fprintf(os.Stderr, "testhelper: failed to terminate container: %v\n", err)
		}
	}
	return pool, teardown
}

// TruncateAll removes all rows from every data table in dependency order.
// Call this in t.Cleanup to isolate test cases.
func TruncateAll(t *testing.T, pool *pgxpool.Pool) {
	t.Helper()
	const q = `
		TRUNCATE
			kr_progress_logs,
			key_results,
			objectives,
			periods,
			team_members,
			teams,
			users
		RESTART IDENTITY CASCADE`
	if _, err := pool.Exec(context.Background(), q); err != nil {
		t.Fatalf("testhelper.TruncateAll: %v", err)
	}
}
