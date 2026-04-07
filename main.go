package main

import (
	"database/sql"
	"finance-api/internal/api"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
	godotenv.Load()

	// Set up structured JSON logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		slog.Error("failed to open database", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	runMigrations(db)
	slog.Info("connected to database")

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	authHandler := api.NewAuthHandler(db, logger)
	accountHandler := api.NewAccountHandler(db, logger)
	transactionHandler := api.NewTransactionHandler(db, logger)

	// Auth routes
	r.Post("/register", authHandler.Register)
	r.Post("/login", authHandler.Login)

	r.Group(func(r chi.Router) {
		r.Use(api.AuthMiddleware)

		// Account routes
		r.Get("/accounts", accountHandler.List)
		r.Post("/accounts", accountHandler.Create)
		r.Delete("/accounts/{id}", accountHandler.Delete)
		r.Get("/accounts/{id}", accountHandler.Get)
		r.Put("/accounts/{id}", accountHandler.Update)
		r.Get("/accounts/{id}/summary", accountHandler.Summary)

		// Transaction routes
		r.Get("/accounts/{id}/transactions", transactionHandler.List)
		r.Post("/accounts/{id}/transactions", transactionHandler.Create)
		r.Get("/accounts/{id}/transactions/{txID}", transactionHandler.Get)
	})

	slog.Info("server running", "port", 8080)
	http.ListenAndServe(":8080", r)
}

func runMigrations(db *sql.DB) {
	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		slog.Error("failed to create migration driver", "error", err)
		os.Exit(1)
	}

	m, err := migrate.NewWithDatabaseInstance(
		"file:///app/migrations",
		"postgres",
		driver,
	)
	if err != nil {
		slog.Error("failed to create migrator", "error", err)
		os.Exit(1)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	slog.Info("migrations applied")
}
