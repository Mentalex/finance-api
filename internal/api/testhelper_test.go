package api_test

import (
	"context"
	"database/sql"
	"finance-api/internal/api"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

const testUserID = "a0000000-0000-0000-0000-000000000001"
const otherUserID = "b0000000-0000-0000-0000-000000000002"

type testEnv struct {
	db      *sql.DB
	handler *api.AccountHandler
	router  *chi.Mux
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()
	ctx := context.Background()

	// spin up a real Postgres container
	container, err := tcpostgres.RunContainer(ctx,
		testcontainers.WithImage("postgres:16"),
		tcpostgres.WithDatabase("finance_test"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("password"),
		testcontainers.WithWaitStrategy(
			wait.ForListeningPort("5432/tcp"),
		),
	)
	if err != nil {
		t.Fatalf("failed to start postgres container: %v", err)
	}

	t.Cleanup(func() { container.Terminate(ctx) })

	connStr, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
	}

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		t.Fatalf("failed to open db: %v", err)
	}

	// run migrations
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		t.Fatalf("failed to create migration driver: %v", err)
	}

	m, err := migrate.NewWithDatabaseInstance("file://../../migrations", "postgres", driver)
	if err != nil {
		t.Fatalf("failed to create migrator: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("failed to run migrations: %v", err)
	}

	// insert test users so foreign key constraint is satisfied
	_, err = db.ExecContext(context.Background(),
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`,
		testUserID, "test@example.com", "hashedpassword",
	)
	if err != nil {
		t.Fatalf("failed to insert test user: %v", err)
	}

	_, err = db.ExecContext(context.Background(),
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`,
		otherUserID, "other@example.com", "hashedpassword",
	)
	if err != nil {
		t.Fatalf("failed to insert other test user: %v", err)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	handler := api.NewAccountHandler(db, logger)

	r := chi.NewRouter()
	r.Get("/accounts", handler.List)
	r.Post("/accounts", handler.Create)
	r.Get("/accounts/{id}", handler.Get)
	r.Put("/accounts/{id}", handler.Update)
	r.Delete("/accounts/{id}", handler.Delete)

	return &testEnv{db: db, handler: handler, router: r}
}

// helper to make requests with a fake authenticated user
func (e *testEnv) request(method, path, body string, userID string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	// inject userID directly into context — bypasses JWT for tests
	ctx := context.WithValue(req.Context(), api.UserIDKey, userID)
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	e.router.ServeHTTP(rr, req)
	return rr
}
