#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
DATA_DIR="$ROOT/data/qdrant"
CONTAINER_NAME="practicehelper-qdrant"
IMAGE="qdrant/qdrant:v1.13.4"

command -v docker >/dev/null 2>&1 || {
  echo "docker is required to run local Qdrant" >&2
  exit 1
}

mkdir -p "$DATA_DIR"

if docker inspect "$CONTAINER_NAME" >/dev/null 2>&1; then
  if [ "$(docker inspect -f '{{.State.Running}}' "$CONTAINER_NAME")" = "true" ]; then
    echo "Qdrant is already running at http://127.0.0.1:6333"
    exit 0
  fi

  docker start "$CONTAINER_NAME" >/dev/null
  echo "Qdrant started at http://127.0.0.1:6333"
  exit 0
fi

docker run -d \
  --name "$CONTAINER_NAME" \
  -p 6333:6333 \
  -v "$DATA_DIR:/qdrant/storage" \
  "$IMAGE" >/dev/null

echo "Qdrant started at http://127.0.0.1:6333"
