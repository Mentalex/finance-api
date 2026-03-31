package main

import (
	"database/sql"
	"finance-api/internal/api"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load()
	// fmt.Println("DB URL:", os.Getenv("DATABASE_URL"))

	db, err := sql.Open("pgx", os.Getenv("DATABASE_URL"))
	if err != nil {
		log.Fatal("Failed to open database: ", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("Failed to connect to database: ", err)
	}
	fmt.Println("Successfully connected to the database!")

	r := chi.NewRouter()

	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})

	authHandler := api.NewAuthHandler(db)
	r.Post("/register", authHandler.Register)
	r.Post("/login", authHandler.Login)

	accountHandler := api.NewAccountsHandler(db)
	transactionHandler := api.NewTransactionHandler(db)

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

	fmt.Print("Server is running on 8080...\n")
	http.ListenAndServe(":8080", r)
}
