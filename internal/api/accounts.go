package api

import (
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"golang.org/x/sync/errgroup"
)

type Account struct {
	ID      string  `json:"id"`
	Name    string  `json:"name"`
	Balance float64 `json:"balance"`
}

type AccountHandler struct {
	db *sql.DB
}

func NewAccountsHandler(db *sql.DB) *AccountHandler {
	return &AccountHandler{db: db}
}

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), "SELECT * FROM accounts")
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	accounts := []Account{}
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Balance); err != nil {
			http.Error(w, "scan error", http.StatusInternalServerError)
			return
		}
		accounts = append(accounts, a)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(accounts)
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Name    string  `json:"name"`
		Balance float64 `json:"balance"`
	}

	// Decode the JSON body into the input struct
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest) // Return a 400 Bad Request
		return
	}

	// Validate the input - name is required
	if input.Name == "" {
		http.Error(w, "name is required", http.StatusUnprocessableEntity) // Return a 422 Unprocessable Entity
		return
	}

	var account Account
	err := h.db.QueryRowContext(
		r.Context(),
		"INSERT INTO accounts (name, balance) VALUES ($1, $2) RETURNING id, name, balance",
		input.Name, input.Balance,
	).Scan(&account.ID, &account.Name, &account.Balance)

	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	// Return the created account
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated) // Return a 201 something was created
	json.NewEncoder(w).Encode(account)
}

func (h *AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "invalid ID format", http.StatusBadRequest)
		return
	}

	result, err := h.db.ExecContext(r.Context(), "DELETE FROM accounts WHERE id = $1", id)
	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		http.Error(w, "ID doesn't exist", http.StatusNotFound) // Return a 404 Not Found if the ID doesn't exist
		return
	}

	w.WriteHeader(http.StatusNoContent) // Return a 204 No Content
}

func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "invalid ID format", http.StatusBadRequest)
		return
	}

	var account Account
	err := h.db.QueryRowContext(
		r.Context(),
		"SELECT * FROM accounts WHERE id = $1", id,
	).Scan(&account.ID, &account.Name, &account.Balance)

	if err == sql.ErrNoRows {
		http.Error(w, "ID doesn't exist", http.StatusNotFound) // Return a 404 Not Found if the ID doesn't exist
		return
	}

	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError) // Return a 500 Internal Server Error for any database issues
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Return a 200 OK
	json.NewEncoder(w).Encode(account)
}

func (h *AccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := uuid.Parse(id); err != nil {
		http.Error(w, "invalid ID format", http.StatusBadRequest)
		return
	}

	var input struct {
		Name    string  `json:"name"`
		Balance float64 `json:"balance"`
	}

	// Decode the JSON body into the input struct
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest) // Return a 400 Bad Request
		return
	}

	// Validate the input - name is required
	if input.Name == "" {
		http.Error(w, "name is required", http.StatusUnprocessableEntity) // Return a 422 Unprocessable Entity
		return
	}

	var account Account
	err := h.db.QueryRowContext(
		r.Context(),
		"UPDATE accounts SET name = $1, balance = $2 WHERE id = $3 RETURNING id, name, balance",
		input.Name, input.Balance, id,
	).Scan(&account.ID, &account.Name, &account.Balance)

	if err == sql.ErrNoRows {
		http.Error(w, "ID doesn't exist", http.StatusNotFound) // Return a 404 Not Found if the ID doesn't exist
		return
	}

	if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError) // Return a 500 Internal Server Error for any database issues
		return
	}

	// Return the updated account
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Return a 200 OK
	json.NewEncoder(w).Encode(account)
}

func (h *AccountHandler) Summary(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var account Account
	var transactions []Transaction

	g, ctx := errgroup.WithContext(r.Context())

	g.Go(func() error {
		return h.db.QueryRowContext(ctx,
			"SELECT id, name, balance FROM accounts WHERE id = $1", id,
		).Scan(&account.ID, &account.Name, &account.Balance)
	})

	g.Go(func() error {
		rows, err := h.db.QueryContext(ctx,
			"SELECT id, account_id, amount, type, description, created_at FROM transactions WHERE account_id = $1", id,
		)
		if err != nil {
			return err
		}

		defer rows.Close()
		for rows.Next() {
			var t Transaction
			rows.Scan(&t.ID, &t.AccountID, &t.Amount, &t.Type, &t.Description, &t.CreatedAt)
			transactions = append(transactions, t)
		}
		return nil
	})

	if err := g.Wait(); err == sql.ErrNoRows {
		http.Error(w, "account not found", http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, "database error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"account":      account,
		"transactions": transactions,
	})
}
