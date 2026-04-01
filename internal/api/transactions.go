package api

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

type Transaction struct {
	ID          string    `json:"id"`
	AccountID   string    `json:"account_id"`
	Amount      float64   `json:"amount"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

type TransactionHandler struct {
	db     *sql.DB
	logger *slog.Logger
}

func NewTransactionHandler(db *sql.DB, logger *slog.Logger) *TransactionHandler {
	return &TransactionHandler{db: db, logger: logger}
}

func (h *TransactionHandler) List(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	_, err := uuid.Parse(accountID)
	if accountID == "" || err != nil {
		errBadRequest(w, "account ID is required and must be a valid UUID")
		return
	}

	rows, err := h.db.QueryContext(r.Context(), "SELECT * FROM transactions WHERE account_id = $1 ORDER BY created_at DESC", accountID)
	if err != nil {
		h.logger.Error("failed to query transactions", "error", err)
		errInternal(w)
		return
	}
	defer rows.Close()

	txns := []Transaction{}
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.AccountID, &t.Amount, &t.Type, &t.Description, &t.CreatedAt); err != nil {
			h.logger.Error("failed to scan transaction", "error", err)
			errInternal(w)
			return
		}
		txns = append(txns, t)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(txns)
}

func (h *TransactionHandler) Get(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	txID := chi.URLParam(r, "txID")

	if accountID == "" || txID == "" {
		errBadRequest(w, "account ID and Transaction ID are required")
		return
	}

	if _, err := uuid.Parse(accountID); err != nil {
		errBadRequest(w, "account ID must be a valid UUID")
		return
	}

	if _, err := uuid.Parse(txID); err != nil {
		errBadRequest(w, "transaction ID must be a valid UUID")
		return
	}

	var tx Transaction
	err := h.db.QueryRowContext(
		r.Context(),
		"SELECT * FROM transactions WHERE id = $1 AND account_id = $2", txID, accountID,
	).Scan(&tx.ID, &tx.AccountID, &tx.Amount, &tx.Type, &tx.Description, &tx.CreatedAt)
	if err == sql.ErrNoRows {
		errNotFound(w, "transaction not found")
		return
	}
	if err != nil {
		h.logger.Error("failed to query transaction", "error", err)
		errInternal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(tx)
}

func (h *TransactionHandler) Create(w http.ResponseWriter, r *http.Request) {
	accountID := chi.URLParam(r, "id")
	if _, err := uuid.Parse(accountID); err != nil {
		errBadRequest(w, "account ID must be a valid UUID")
		return
	}

	var input struct {
		Amount      float64 `json:"amount"`
		Type        string  `json:"type"`
		Description string  `json:"description"`
	}

	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		errBadRequest(w, "invalid request body")
		return
	}

	if input.Amount <= 0 {
		errUnprocessable(w, "amount must be greater than 0")
		return
	}

	if input.Type != "deposit" && input.Type != "withdrawal" {
		errUnprocessable(w, "type must be 'deposit' or 'withdrawal'")
		return
	}

	dbTx, err := h.db.BeginTx(r.Context(), nil)
	if err != nil {
		h.logger.Error("failed to begin transaction", "error", err)
		errInternal(w)
		return
	}
	defer dbTx.Rollback()

	var transaction Transaction
	err = dbTx.QueryRowContext(
		r.Context(),
		"INSERT INTO transactions (account_id, amount, type, description) VALUES ($1, $2, $3, $4) RETURNING id, account_id, amount, type, description, created_at",
		accountID, input.Amount, input.Type, input.Description,
	).Scan(&transaction.ID, &transaction.AccountID, &transaction.Amount, &transaction.Type, &transaction.Description, &transaction.CreatedAt)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23503" {
			errNotFound(w, "account not found")
			return
		}
		h.logger.Error("failed to create transaction", "error", err)
		errInternal(w)
		return
	}

	balanceChange := input.Amount
	if input.Type == "withdrawal" {
		balanceChange = -input.Amount
	}

	_, err = dbTx.ExecContext(
		r.Context(),
		"UPDATE accounts SET balance = balance + $1 WHERE id = $2",
		balanceChange, accountID,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23514" {
			errUnprocessable(w, "insufficient funds")
			return
		}
		h.logger.Error("failed to update account balance", "error", err)
		errInternal(w)
		return
	}

	if err := dbTx.Commit(); err != nil {
		h.logger.Error("failed to commit transaction", "error", err)
		errInternal(w)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(transaction)
}
