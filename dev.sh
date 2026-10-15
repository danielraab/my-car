#!/usr/bin/env bash
# Starts the local development stack: the compose "dev" profile (db, pgadmin,
# mailpit), the Go backend on the host and the Vite dev server.
# Ctrl+C stops backend and frontend; the containers keep running
# (`docker compose --profile dev down` to stop them).
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

missing=()
for cmd in go node pnpm; do
  command -v "$cmd" >/dev/null 2>&1 || missing+=("$cmd")
done
if (( ${#missing[@]} > 0 )); then
  echo "Missing required tools: ${missing[*]}" >&2
  exit 1
fi

if [[ ! -f "$ROOT/backend/.env" ]]; then
  echo "backend/.env not found, copying backend/.env.example"
  cp "$ROOT/backend/.env.example" "$ROOT/backend/.env"
fi

docker compose -f "$ROOT/docker-compose.yml" --profile dev up -d --wait

pids=()
cleanup() {
  trap - INT TERM EXIT
  for pid in "${pids[@]}"; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup INT TERM EXIT

(
  cd "$ROOT/backend"
  set -a
  # shellcheck source=/dev/null
  source .env
  set +a
  exec go run .
) &
pids+=($!)

(
  cd "$ROOT/frontend"
  exec pnpm dev
) &
pids+=($!)

# Exit (and stop the other process) as soon as either one ends.
wait -n
