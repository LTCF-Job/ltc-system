#!/usr/bin/env bash
# 部署正式 API 前的護欄：正式與 Demo 資料庫的 migration 版本或 schema 有漂移就中止部署。
# 用法：DATABASE_URL=<正式> DEMO_DATABASE_URL=<Demo> ./compare-demo-prod-schema.sh（需要 docker）

set -euo pipefail

if [[ -z "${DATABASE_URL:-}" || -z "${DEMO_DATABASE_URL:-}" ]]; then
  echo "error: 必須同時設定 DATABASE_URL 與 DEMO_DATABASE_URL" >&2
  exit 1
fi

PSQL_IMAGE="postgres:16"

run_psql() {
  local url="$1"
  shift
  docker run --rm "$PSQL_IMAGE" psql "$url" -X -q "$@"
}

echo "== 比對 schema_migrations 版本 =="
PROD_VERSIONS=$(run_psql "$DATABASE_URL" -t -c "SELECT version FROM schema_migrations ORDER BY version" | tr -d ' ')
DEMO_VERSIONS=$(run_psql "$DEMO_DATABASE_URL" -t -c "SELECT version FROM schema_migrations ORDER BY version" | tr -d ' ')

if [[ "$PROD_VERSIONS" != "$DEMO_VERSIONS" ]]; then
  echo "error: schema_migrations 版本不一致" >&2
  echo "--- 正式 ---" >&2
  echo "$PROD_VERSIONS" >&2
  echo "--- Demo ---" >&2
  echo "$DEMO_VERSIONS" >&2
  exit 1
fi
echo "版本一致：$(echo "$PROD_VERSIONS" | wc -l) 個 migration"

echo "== 比對業務 schema（排除 auth.*，兩邊本來就不同）=="
normalize_schema() {
  local url="$1"
  docker run --rm "$PSQL_IMAGE" pg_dump "$url" \
    --schema-only --no-owner --no-privileges --no-tablespaces \
    --schema=public \
    | grep -v -E '^-- (Dumped|Started|Completed|PostgreSQL database dump)' \
    | grep -v -E '^SET (statement_timeout|lock_timeout|idle_in_transaction|client_encoding|standard_conforming|check_function_bodies|xmloption|client_min_messages|row_security)' \
    | grep -v -E '^\\(un)?restrict '
}

PROD_SCHEMA=$(normalize_schema "$DATABASE_URL")
DEMO_SCHEMA=$(normalize_schema "$DEMO_DATABASE_URL")

if [[ "$PROD_SCHEMA" != "$DEMO_SCHEMA" ]]; then
  echo "error: public schema（table／constraint／index）不一致" >&2
  diff <(echo "$PROD_SCHEMA") <(echo "$DEMO_SCHEMA") >&2 || true
  exit 1
fi

echo "schema 一致，允許部署正式 API。"
