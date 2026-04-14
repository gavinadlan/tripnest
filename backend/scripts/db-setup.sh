#!/usr/bin/env bash
set -euo pipefail

POSTGRES_CONTAINER="${POSTGRES_CONTAINER:-tripnest-postgres}"
POSTGRES_USER="${POSTGRES_USER:-postgres}"

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

psql_exec() {
  docker exec "${POSTGRES_CONTAINER}" psql -v ON_ERROR_STOP=1 -U "${POSTGRES_USER}" "$@"
}

ensure_db() {
  local db_name="$1"
  local exists
  exists="$(psql_exec -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='${db_name}';" | tr -d '[:space:]')"
  if [[ "${exists}" != "1" ]]; then
    echo "Creating database: ${db_name}"
    psql_exec -d postgres -c "CREATE DATABASE ${db_name};"
  else
    echo "Database already exists: ${db_name}"
  fi
}

ensure_migration_table() {
  local db_name="$1"
  psql_exec -d "${db_name}" -c "CREATE TABLE IF NOT EXISTS schema_migrations (version bigint PRIMARY KEY, dirty boolean NOT NULL DEFAULT false);"
}

apply_service_migrations() {
  local db_name="$1"
  local migration_dir="$2"

  ensure_migration_table "${db_name}"

  while IFS= read -r migration; do
    local base
    local version
    local current_version
    base="$(basename "${migration}")"
    version="$((10#${base%%_*}))"

    current_version="$(psql_exec -d "${db_name}" -tAc "SELECT COALESCE(MAX(version), 0) FROM schema_migrations;" | tr -d '[:space:]')"
    if [[ -z "${current_version}" ]]; then
      current_version=0
    fi
    if (( current_version >= version )); then
      echo "[${db_name}] migration already applied: ${base}"
      continue
    fi

    echo "[${db_name}] applying migration: ${base}"
    docker exec -i "${POSTGRES_CONTAINER}" psql -v ON_ERROR_STOP=1 -U "${POSTGRES_USER}" -d "${db_name}" < "${migration}"
    psql_exec -d "${db_name}" -c "INSERT INTO schema_migrations(version, dirty) VALUES (${version}, false);"
  done < <(ls -1 "${migration_dir}"/*.up.sql | sort)
}

echo "Ensuring required databases exist..."
ensure_db "tripnest_booking"
ensure_db "tripnest_payments"
ensure_db "tripnest_inventory"

echo "Applying booking-service migrations..."
apply_service_migrations "tripnest_booking" "${ROOT_DIR}/booking-service/migrations"

echo "Applying payment-service migrations..."
apply_service_migrations "tripnest_payments" "${ROOT_DIR}/payment-service/migrations"

echo "Applying inventory-service migrations..."
apply_service_migrations "tripnest_inventory" "${ROOT_DIR}/inventory-service/migrations"

echo "Database setup complete."
