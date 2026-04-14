package api_test

import (
	"encoding/json"
	"finance-api/internal/api"
	"net/http"
	"testing"
)

func TestCreateAccount(t *testing.T) {
	env := setupTestEnv(t)

	t.Run("valid account", func(t *testing.T) {
		rr := env.request("POST", "/accounts",
			`{"name": "Checking", "balance": 1000}`, testUserID)

		if rr.Code != http.StatusCreated {
			t.Errorf("expected 201, got %d", rr.Code)
		}

		var account api.Account
		json.NewDecoder(rr.Body).Decode(&account)

		if account.Name != "Checking" {
			t.Errorf("expected name 'Checking', got '%s'", account.Name)
		}
		if account.Balance != 1000 {
			t.Errorf("expected balance 1000, got %f", account.Balance)
		}
		if account.ID == "" {
			t.Error("expected account ID to be set")
		}
	})

	t.Run("missing name", func(t *testing.T) {
		rr := env.request("POST", "/accounts",
			`{"balance": 1000}`, testUserID)

		if rr.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rr.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		rr := env.request("POST", "/accounts",
			`not json`, testUserID)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})
}

func TestGetAccount(t *testing.T) {
	env := setupTestEnv(t)

	// create an account to test with
	rr := env.request("POST", "/accounts",
		`{"name": "Savings", "balance": 5000}`, testUserID)
	var created api.Account
	json.NewDecoder(rr.Body).Decode(&created)

	t.Run("existing account", func(t *testing.T) {
		rr := env.request("GET", "/accounts/"+created.ID, "", testUserID)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}

		var account api.Account
		json.NewDecoder(rr.Body).Decode(&account)
		if account.ID != created.ID {
			t.Errorf("expected ID %s, got %s", created.ID, account.ID)
		}
	})

	t.Run("non-existent account", func(t *testing.T) {
		rr := env.request("GET", "/accounts/00000000-0000-0000-0000-000000000000", "", testUserID)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		rr := env.request("GET", "/accounts/not-a-uuid", "", testUserID)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("another user's account", func(t *testing.T) {
		rr := env.request("GET", "/accounts/"+created.ID, "", otherUserID)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d — other users should not see this account", rr.Code)
		}
	})
}

func TestListAccounts(t *testing.T) {
	t.Run("empty list", func(t *testing.T) {
		env := setupTestEnv(t) // own isolated database
		rr := env.request("GET", "/accounts", "", testUserID)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}

		var accounts []api.Account
		json.NewDecoder(rr.Body).Decode(&accounts)
		if len(accounts) != 0 {
			t.Errorf("expected 0 accounts, got %d", len(accounts))
		}
	})

	t.Run("returns only user's accounts", func(t *testing.T) {
		env := setupTestEnv(t) // own isolated database
		env.request("POST", "/accounts", `{"name": "Checking", "balance": 100}`, testUserID)
		env.request("POST", "/accounts", `{"name": "Savings", "balance": 500}`, testUserID)
		env.request("POST", "/accounts", `{"name": "Other User", "balance": 999}`, otherUserID)

		rr := env.request("GET", "/accounts", "", testUserID)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}

		var accounts []api.Account
		json.NewDecoder(rr.Body).Decode(&accounts)
		if len(accounts) != 2 {
			t.Errorf("expected 2 accounts, got %d", len(accounts))
		}
		for _, a := range accounts {
			if a.Name == "Other User" {
				t.Error("should not return another user's account")
			}
		}
	})
}

func TestUpdateAccount(t *testing.T) {
	env := setupTestEnv(t)

	// create an account to test with
	rr := env.request("POST", "/accounts",
		`{"name": "Original", "balance": 100}`, testUserID)
	var created api.Account
	json.NewDecoder(rr.Body).Decode(&created)

	t.Run("valid update", func(t *testing.T) {
		rr := env.request("PUT", "/accounts/"+created.ID,
			`{"name": "Updated", "balance": 200}`, testUserID)

		if rr.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rr.Code)
		}

		var account api.Account
		json.NewDecoder(rr.Body).Decode(&account)
		if account.Name != "Updated" {
			t.Errorf("expected name 'Updated', got '%s'", account.Name)
		}
		if account.Balance != 200 {
			t.Errorf("expected balance 200, got %f", account.Balance)
		}
		if account.ID != created.ID {
			t.Errorf("expected ID %s, got %s", created.ID, account.ID)
		}
	})

	t.Run("missing name", func(t *testing.T) {
		rr := env.request("PUT", "/accounts/"+created.ID,
			`{"balance": 50}`, testUserID)

		if rr.Code != http.StatusUnprocessableEntity {
			t.Errorf("expected 422, got %d", rr.Code)
		}
	})

	t.Run("invalid JSON", func(t *testing.T) {
		rr := env.request("PUT", "/accounts/"+created.ID,
			`not json`, testUserID)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		rr := env.request("PUT", "/accounts/not-a-uuid",
			`{"name": "X", "balance": 0}`, testUserID)

		if rr.Code != http.StatusBadRequest {
			t.Errorf("expected 400, got %d", rr.Code)
		}
	})

	t.Run("non-existent account", func(t *testing.T) {
		rr := env.request("PUT", "/accounts/00000000-0000-0000-0000-000000000000",
			`{"name": "Ghost", "balance": 0}`, testUserID)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})

	t.Run("another user's account", func(t *testing.T) {
		rr := env.request("PUT", "/accounts/"+created.ID,
			`{"name": "Hijack", "balance": 0}`, otherUserID)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d — other users should not update this account", rr.Code)
		}
	})
}

func TestDeleteAccount(t *testing.T) {
	env := setupTestEnv(t)

	t.Run("existing account", func(t *testing.T) {
		rr := env.request("POST", "/accounts",
			`{"name": "To Delete", "balance": 0}`, testUserID)
		var created api.Account
		json.NewDecoder(rr.Body).Decode(&created)

		rr = env.request("DELETE", "/accounts/"+created.ID, "", testUserID)
		if rr.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d", rr.Code)
		}

		// verify it's gone
		rr = env.request("GET", "/accounts/"+created.ID, "", testUserID)
		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404 after delete, got %d", rr.Code)
		}
	})

	t.Run("non-existent account", func(t *testing.T) {
		rr := env.request("DELETE", "/accounts/00000000-0000-0000-0000-000000000000", "", testUserID)

		if rr.Code != http.StatusNotFound {
			t.Errorf("expected 404, got %d", rr.Code)
		}
	})
}
