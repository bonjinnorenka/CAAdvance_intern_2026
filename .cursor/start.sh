#!/usr/bin/env bash
# Start dockerd for this Cloud Agent boot. Safe to rerun.
set -euo pipefail

if docker info >/dev/null 2>&1; then
  exit 0
fi

sudo service docker start

for _ in $(seq 1 30); do
  if docker info >/dev/null 2>&1; then
    exit 0
  fi
  sleep 1
done

echo "Docker daemon failed to become ready" >&2
exit 1
