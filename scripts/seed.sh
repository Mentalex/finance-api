#!/bin/bash

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "🌱 Seeding database..."

# ── Register & login ───────────────────────────────────────────
echo "\n→ Registering user..."
curl -s -X POST $BASE_URL/register \
  -H "Content-Type: application/json" \
  -d '{"email": "ada@example.com", "password": "secret123"}' | jq

echo "\n→ Logging in..."
TOKEN=$(curl -s -X POST $BASE_URL/login \
  -H "Content-Type: application/json" \
  -d '{"email": "ada@example.com", "password": "secret123"}' | jq -r '.token')

if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
  echo "❌ Login failed — stopping."
  exit 1
fi

echo "✅ Token obtained"

# ── Create accounts ────────────────────────────────────────────
echo "\n→ Creating accounts..."
CHECKING_ID=$(curl -s -X POST $BASE_URL/accounts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "Checking", "balance": 5000}' | jq -r '.id')
echo "Checking account: $CHECKING_ID"

SAVINGS_ID=$(curl -s -X POST $BASE_URL/accounts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "Savings", "balance": 12000}' | jq -r '.id')
echo "Savings account: $SAVINGS_ID"

INVESTMENTS_ID=$(curl -s -X POST $BASE_URL/accounts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"name": "Investments", "balance": 30000}' | jq -r '.id')
echo "Investments account: $INVESTMENTS_ID"

# ── Checking transactions ──────────────────────────────────────
echo "\n→ Adding transactions to Checking..."
curl -s -X POST $BASE_URL/accounts/$CHECKING_ID/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 3000, "type": "deposit", "description": "Salary"}' | jq
curl -s -X POST $BASE_URL/accounts/$CHECKING_ID/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 120, "type": "withdrawal", "description": "Electricity bill"}' | jq
curl -s -X POST $BASE_URL/accounts/$CHECKING_ID/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 85, "type": "withdrawal", "description": "Groceries"}' | jq
curl -s -X POST $BASE_URL/accounts/$CHECKING_ID/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 500, "type": "withdrawal", "description": "Rent"}' | jq

# ── Savings transactions ───────────────────────────────────────
echo "\n→ Adding transactions to Savings..."
curl -s -X POST $BASE_URL/accounts/$SAVINGS_ID/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 1000, "type": "deposit", "description": "Monthly transfer"}' | jq
curl -s -X POST $BASE_URL/accounts/$SAVINGS_ID/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 200, "type": "withdrawal", "description": "Emergency fund"}' | jq

  # ── Investments transactions ───────────────────────────────────
echo "\n→ Adding transactions to Investments..."
curl -s -X POST $BASE_URL/accounts/$INVESTMENTS_ID/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 5000, "type": "deposit", "description": "ETF purchase"}' | jq
curl -s -X POST $BASE_URL/accounts/$INVESTMENTS_ID/transactions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer $TOKEN" \
  -d '{"amount": 1500, "type": "deposit", "description": "Dividend reinvestment"}' | jq

echo "\n✅ Done! Database seeded."
