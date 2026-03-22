#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT"

command -v pnpm >/dev/null 2>&1 || { echo "pnpm is required" >&2; exit 1; }
command -v uv >/dev/null 2>&1 || { echo "uv is required" >&2; exit 1; }
command -v go >/dev/null 2>&1 || { echo "go is required" >&2; exit 1; }

if [ ! -f .env ]; then
  cp .env.example .env
fi

set -a
# shellcheck disable=SC1091
. "./.env"
set +a

pnpm install
uv sync --project sidecar
(
  cd server
  GOCACHE=/tmp/go-build go test -tags sqlite_fts5 ./...
)
uv run --project sidecar pytest -q
(
  cd web
  pnpm build
)

echo "bootstrap complete"
