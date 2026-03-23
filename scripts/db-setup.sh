#!/usr/bin/env bash
# scripts/db-setup.sh
# Creates local PostgreSQL roles and databases for Postulate development.
# Safe to run multiple times — uses IF NOT EXISTS throughout.

set -euo pipefail

DB_USER="postulate_dev"
DB_PASS="postulate_dev"
DB_NAME_DEV="postulate_dev"
DB_NAME_TEST="postulate_test"

echo "→ Creating role: ${DB_USER}"
psql postgres -tc "SELECT 1 FROM pg_roles WHERE rolname='${DB_USER}'" \
  | grep -q 1 \
  || psql postgres -c "CREATE ROLE ${DB_USER} WITH LOGIN PASSWORD '${DB_PASS}';"

echo "→ Creating database: ${DB_NAME_DEV}"
psql postgres -tc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME_DEV}'" \
  | grep -q 1 \
  || psql postgres -c "CREATE DATABASE ${DB_NAME_DEV} OWNER ${DB_USER};"

echo "→ Creating database: ${DB_NAME_TEST}"
psql postgres -tc "SELECT 1 FROM pg_database WHERE datname='${DB_NAME_TEST}'" \
  | grep -q 1 \
  || psql postgres -c "CREATE DATABASE ${DB_NAME_TEST} OWNER ${DB_USER};"

echo "→ Writing api/config.local.yaml"
if [ ! -f api/config.local.yaml ]; then
  cp api/config.example.yaml api/config.local.yaml
  echo "   Created api/config.local.yaml — update values as needed"
else
  echo "   api/config.local.yaml already exists — skipping"
fi

echo ""
echo "✓ Local database setup complete."
echo "  Start PostgreSQL with: make db-start"
echo "  Then start the API with: make run"
