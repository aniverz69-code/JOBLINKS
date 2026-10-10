#!/usr/bin/env bash
# Starts the whole freehire stack inside a GitHub Codespace, seeds a few companies the
# first time, crawls them, and indexes the jobs. Safe to re-run: every step checks first.
set -euo pipefail
cd "$(dirname "$0")/.."

# The public URL Codespaces gives port 8090. The web app needs it as its own origin or
# logins and form posts are rejected as cross-site.
if [ -n "${CODESPACE_NAME:-}" ] && [ -n "${GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN:-}" ]; then
  ORIGIN="https://${CODESPACE_NAME}-8090.${GITHUB_CODESPACES_PORT_FORWARDING_DOMAIN}"
else
  ORIGIN="http://localhost:8090"
fi
touch .env
grep -v -E '^(WEB_ORIGIN|FRONTEND_ORIGIN)=' .env > .env.tmp || true
printf 'WEB_ORIGIN=%s\nFRONTEND_ORIGIN=%s\n' "$ORIGIN" "$ORIGIN" >> .env.tmp
mv .env.tmp .env

echo ">> Starting containers (first build takes 10-20 minutes)..."
docker compose up -d --build

echo ">> Waiting for the API..."
for _ in $(seq 1 120); do
  if curl -fsS http://localhost:8080/health >/dev/null 2>&1; then break; fi
  sleep 5
done

psql_q() { docker compose exec -T db psql -U hire -d hire -tAc "$1"; }

if [ "$(psql_q 'SELECT count(*) FROM boards' | tr -d '[:space:]')" = "0" ]; then
  echo ">> Seeding companies..."
  # Edit this list freely: ('greenhouse', '<board slug>', '<display name>')
  psql_q "INSERT INTO boards (provider, board, company, status, activated_at) VALUES
    ('greenhouse','stripe','Stripe','active',now()),
    ('greenhouse','airbnb','Airbnb','active',now()),
    ('greenhouse','cloudflare','Cloudflare','active',now()),
    ('greenhouse','discord','Discord','active',now()),
    ('greenhouse','reddit','Reddit','active',now()),
    ('greenhouse','databricks','Databricks','active',now())
    ON CONFLICT DO NOTHING" >/dev/null
fi

if [ "$(psql_q 'SELECT count(*) FROM jobs' | tr -d '[:space:]')" = "0" ]; then
  echo ">> Crawling jobs..."
  docker compose exec -T app /app/ingest greenhouse || echo "ingest reported errors (some boards may be unreachable)"
  echo ">> Building the search index..."
  docker compose exec -T app /app/reindex || echo "reindex failed - run it again"
fi

echo
echo "==============================================="
echo " freehire is up:  ${ORIGIN}/jobs"
echo " In the Ports tab, set port 8090 to Public to open it on your phone."
echo "==============================================="
