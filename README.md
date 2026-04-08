# Personal Finance API

A REST API for managing personal finance accounts and transactions. Built with Go, PostgreSQL, and Docker. Secured with JWT authentication.

## Tech stack

- **Go** — backend language
- **chi** — HTTP router
- **PostgreSQL** — database
- **Docker** — containerization
- **JWT** — authentication
- **golang-migrate** — database migrations
- **Railway** — deployment

---

## Running locally

### Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Docker](https://www.docker.com/)
- [jq](https://stedolan.github.io/jq/) (for the seed script)

### Setup

**1. Clone the repository:**
```bash
git clone https://github.com/yourname/finance-api.git
cd finance-api
```

**2. Create your `.env` file:**
```bash
touch .env
```

`.env` should contain:
```
JWT_SECRET=your-local-dev-secret
```

**3. Start the app and database:**
```bash
docker compose up --build
```

Migrations run automatically on startup via `golang-migrate`.

**4. Seed the database (optional):**
```bash
./scripts/seed.sh
```

The API is now running at `http://localhost:8080`.

### Full reset

To wipe everything and start fresh:

```bash
docker compose down -v
docker compose up --build -d
./scripts/seed.sh
```

---

## Environment variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | Postgres connection string |
| `JWT_SECRET` | Secret key for signing JWT tokens |

---

## Authentication

All routes except `/register` and `/login` require a JWT token in the `Authorization` header:

```
Authorization: Bearer <token>
```

Obtain a token by logging in via `POST /login`.

---

## Endpoints

### Auth

#### `POST /register`

Register a new user.

**Request:**
```json
{
  "email": "ada@example.com",
  "password": "secret123"
}
```

**Responses:**

`201 Created` — user registered successfully, no body.

`409 Conflict` — email already in use.
```json
{"error": "email already in use", "code": "conflict"}
```

`422 Unprocessable Entity` — missing fields.
```json
{"error": "email and password are required", "code": "unprocessable"}
```

---

#### `POST /login`

Login and receive a JWT token.

**Request:**
```json
{
  "email": "ada@example.com",
  "password": "secret123"
}
```

**Responses:**

`200 OK`
```json
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

`401 Unauthorized` — wrong email or password.
```json
{"error": "invalid credentials", "code": "unauthorized"}
```

---

### Accounts

All account endpoints are scoped to the authenticated user — they only return or modify accounts belonging to the user making the request.

#### `GET /accounts`

List all accounts belonging to the authenticated user.

**Responses:**

`200 OK`
```json
[
  {
    "id": "a3f1c2d4-1234-5678-abcd-ef0123456789",
    "name": "Checking",
    "balance": 4295
  },
  {
    "id": "b4e2d3f5-2345-6789-bcde-f01234567890",
    "name": "Savings",
    "balance": 12800.50
  }
]
```

---

#### `POST /accounts`

Create a new account.

**Request:**
```json
{
  "name": "Investments",
  "balance": 30000
}
```

**Responses:**

`201 Created`
```json
{
  "id": "c5f3e4g6-3456-7890-cdef-012345678901",
  "name": "Investments",
  "balance": 30000
}
```

`422 Unprocessable Entity` — missing name.
```json
{"error": "name is required", "code": "unprocessable"}
```

---

#### `GET /accounts/{id}`

Get a single account by ID.

**Responses:**

`200 OK`
```json
{
  "id": "a3f1c2d4-1234-5678-abcd-ef0123456789",
  "name": "Checking",
  "balance": 4295
}
```

`404 Not Found` — account doesn't exist.

`400 Bad Request` — invalid UUID format.

---

#### `PUT /accounts/{id}`

Update an account's name and balance.

**Request:**
```json
{
  "name": "Main Checking",
  "balance": 5000
}
```

**Responses:**

`200 OK`
```json
{
  "id": "a3f1c2d4-1234-5678-abcd-ef0123456789",
  "name": "Main Checking",
  "balance": 5000
}
```

`404 Not Found` — account doesn't exist.

`400 Bad Request` — invalid UUID format.

---

#### `DELETE /accounts/{id}`

Delete an account and all its transactions.

**Responses:**

`204 No Content` — deleted successfully, no body.

`404 Not Found` — account doesn't exist.

`400 Bad Request` — invalid UUID format.

---

#### `GET /accounts/{id}/summary`

Get an account and all its transactions in a single request. Both queries run in parallel.

**Responses:**

`200 OK`
```json
{
  "account": {
    "id": "a3f1c2d4-1234-5678-abcd-ef0123456789",
    "name": "Checking",
    "balance": 4295
  },
  "transactions": [
    {
      "id": "d6g4f5h7-4567-8901-defa-123456789012",
      "account_id": "a3f1c2d4-1234-5678-abcd-ef0123456789",
      "amount": 3000,
      "type": "deposit",
      "description": "Salary",
      "created_at": "2024-01-15T10:30:00Z"
    }
  ]
}
```

`404 Not Found` — account doesn't exist.

---

### Transactions

#### `GET /accounts/{id}/transactions`

List all transactions for an account, ordered by most recent first.

**Responses:**

`200 OK`
```json
[
  {
    "id": "d6g4f5h7-4567-8901-defa-123456789012",
    "account_id": "a3f1c2d4-1234-5678-abcd-ef0123456789",
    "amount": 3000,
    "type": "deposit",
    "description": "Salary",
    "created_at": "2024-01-15T10:30:00Z"
  },
  {
    "id": "e7h5g6i8-5678-9012-efab-234567890123",
    "account_id": "a3f1c2d4-1234-5678-abcd-ef0123456789",
    "amount": 85,
    "type": "withdrawal",
    "description": "Groceries",
    "created_at": "2024-01-14T08:15:00Z"
  }
]
```

---

#### `POST /accounts/{id}/transactions`

Create a transaction. Automatically updates the account balance.

**Request:**
```json
{
  "amount": 500,
  "type": "deposit",
  "description": "Freelance payment"
}
```

| Field | Required | Description |
|---|---|---|
| `amount` | Yes | Must be greater than 0 |
| `type` | Yes | `deposit` or `withdrawal` |
| `description` | No | Optional note |

**Responses:**

`201 Created`
```json
{
  "id": "f8i6h7j9-6789-0123-fabc-345678901234",
  "account_id": "a3f1c2d4-1234-5678-abcd-ef0123456789",
  "amount": 500,
  "type": "deposit",
  "description": "Freelance payment",
  "created_at": "2024-01-16T14:00:00Z"
}
```

`422 Unprocessable Entity` — insufficient funds (withdrawal would make balance negative).
```json
{"error": "insufficient funds", "code": "unprocessable"}
```

`422 Unprocessable Entity` — invalid type or amount.
```json
{"error": "type must be 'deposit' or 'withdrawal'", "code": "unprocessable"}
```

`404 Not Found` — account doesn't exist.

`400 Bad Request` — invalid UUID format.

---

#### `GET /accounts/{id}/transactions/{txID}`

Get a single transaction.

**Responses:**

`200 OK`
```json
{
  "id": "f8i6h7j9-6789-0123-fabc-345678901234",
  "account_id": "a3f1c2d4-1234-5678-abcd-ef0123456789",
  "amount": 500,
  "type": "deposit",
  "description": "Freelance payment",
  "created_at": "2024-01-16T14:00:00Z"
}
```

`404 Not Found` — transaction doesn't exist under this account.

`400 Bad Request` — invalid UUID format.

---

## Example curl commands

```bash
# register
curl -X POST http://localhost:8080/register \
  -H "Content-Type: application/json" \
  -d '{"email": "ada@example.com", "password": "secret123"}'

# login — save the token
TOKEN=$(curl -s -X POST http://localhost:8080/login \
  -H "Content-Type: application/json" \
  -d '{"email": "ada@example.com", "password": "secret123"}' | jq -r '.token')

# create an account
curl -X POST http://localhost:8080/accounts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "Checking", "balance": 5000}'

# list accounts
curl http://localhost:8080/accounts \
  -H "Authorization: Bearer $TOKEN"

# create a deposit
curl -X POST http://localhost:8080/accounts/{id}/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 1000, "type": "deposit", "description": "Salary"}'

# create a withdrawal
curl -X POST http://localhost:8080/accounts/{id}/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 50, "type": "withdrawal", "description": "Coffee"}'
```

---

## Database schema

```sql
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE accounts (
    id      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name    TEXT NOT NULL,
    balance NUMERIC(12, 2) NOT NULL DEFAULT 0,
    CONSTRAINT balance_non_negative CHECK (balance >= 0)
);

CREATE TABLE transactions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id  UUID NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    amount      NUMERIC(12, 2) NOT NULL,
    type        TEXT NOT NULL CHECK (type IN ('deposit', 'withdrawal')),
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

Migrations are managed with [golang-migrate](https://github.com/golang-migrate/migrate) and run automatically on startup. Migration files live in `migrations/` as numbered up/down pairs (e.g. `000001_init.up.sql` / `000001_init.down.sql`).

---

## Project structure

```
finance-api/
├── main.go                  # entry point, router setup, auto-migrations
├── Dockerfile
├── docker-compose.yml
├── .env                     # local secrets — never committed
├── .env.example             # template for .env
├── .gitignore
├── go.mod
├── go.sum
├── migrations/
│   ├── 000001_init.up.sql                        # initial schema
│   ├── 000001_init.down.sql
│   ├── 000002_add_user_id_to_accounts.up.sql     # user-scoped accounts
│   └── 000002_add_user_id_to_accounts.down.sql
├── scripts/
│   └── seed.sh              # populates the database with mock data
└── internal/
    └── api/
        ├── accounts.go      # account handlers
        ├── transactions.go  # transaction handlers
        ├── auth.go          # register, login handlers
        ├── middleware.go    # JWT auth middleware
        └── errors.go        # structured JSON error helpers
```