package api

import (
	"database/sql"
	"encoding/json"
	"log/slog"
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
	db     *sql.DB
	logger *slog.Logger
}

func NewAccountsHandler(db *sql.DB, logger *slog.Logger) *AccountHandler {
	return &AccountHandler{db: db, logger: logger}
}

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	rows, err := h.db.QueryContext(r.Context(), "SELECT * FROM accounts")
	if err != nil {
		h.logger.Error("failed to query accounts", "error", err)
		errInternal(w)
		return
	}
	defer rows.Close()

	accounts := []Account{}
	for rows.Next() {
		var a Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Balance); err != nil {
			h.logger.Error("failed to scan account", "error", err)
			errInternal(w)
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
		errBadRequest(w, "invalid JSON")
		return
	}

	// Validate the input - name is required
	if input.Name == "" {
		errUnprocessable(w, "name is required")
		return
	}

	var account Account
	err := h.db.QueryRowContext(
		r.Context(),
		"INSERT INTO accounts (name, balance) VALUES ($1, $2) RETURNING id, name, balance",
		input.Name, input.Balance,
	).Scan(&account.ID, &account.Name, &account.Balance)

	if err != nil {
		h.logger.Error("failed to create account", "error", err)
		errInternal(w)
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
		errBadRequest(w, "invalid ID format")
		return
	}

	result, err := h.db.ExecContext(r.Context(), "DELETE FROM accounts WHERE id = $1", id)
	if err != nil {
		h.logger.Error("failed to delete account", "error", err)
		errInternal(w)
		return
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil || rowsAffected == 0 {
		errNotFound(w, "ID doesn't exist")
		return
	}

	w.WriteHeader(http.StatusNoContent) // Return a 204 No Content
}

func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := uuid.Parse(id); err != nil {
		errBadRequest(w, "invalid ID format")
		return
	}

	var account Account
	err := h.db.QueryRowContext(
		r.Context(),
		"SELECT * FROM accounts WHERE id = $1", id,
	).Scan(&account.ID, &account.Name, &account.Balance)

	if err == sql.ErrNoRows {
		errNotFound(w, "ID doesn't exist")
		return
	}

	if err != nil {
		h.logger.Error("failed to query account", "error", err)
		errInternal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK) // Return a 200 OK
	json.NewEncoder(w).Encode(account)
}

func (h *AccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	if _, err := uuid.Parse(id); err != nil {
		errBadRequest(w, "invalid ID format")
		return
	}

	var input struct {
		Name    string  `json:"name"`
		Balance float64 `json:"balance"`
	}

	// Decode the JSON body into the input struct
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errBadRequest(w, "invalid JSON")
		return
	}

	// Validate the input - name is required
	if input.Name == "" {
		errUnprocessable(w, "name is required")
		return
	}

	var account Account
	err := h.db.QueryRowContext(
		r.Context(),
		"UPDATE accounts SET name = $1, balance = $2 WHERE id = $3 RETURNING id, name, balance",
		input.Name, input.Balance, id,
	).Scan(&account.ID, &account.Name, &account.Balance)

	if err == sql.ErrNoRows {
		errNotFound(w, "ID doesn't exist")
		return
	}

	if err != nil {
		h.logger.Error("failed to update account", "error", err)
		errInternal(w)
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
		errNotFound(w, "account not found")
		return
	} else if err != nil {
		h.logger.Error("failed to query account summary", "error", err)
		errInternal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"account":      account,
		"transactions": transactions,
	})
}
