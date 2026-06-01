// Package testhelper provides utilities for integration tests that require a
// real PostgreSQL database.
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

// SetupPostgres starts a PostgreSQL 16 container, applies schema.sql, and
// returns a *pgxpool.Pool connected to it together with a teardown function.
//
// Call teardown (typically via defer) to stop and remove the container.
func SetupPostgres(ctx context.Context) (pool *pgxpool.Pool, teardown func()) {
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

	// Apply schema
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
