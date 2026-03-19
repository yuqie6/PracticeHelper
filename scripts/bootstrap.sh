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

pnpm install
(
  cd sidecar
  uv sync
)
(
  cd server
  GOCACHE=/tmp/go-build go test ./...
)
(
  cd sidecar
  uv run pytest -q
)
(
  cd web
  pnpm build
)

echo "bootstrap complete"
