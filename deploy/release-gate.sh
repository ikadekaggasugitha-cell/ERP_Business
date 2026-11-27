#!/usr/bin/env bash
# Release gate: fails while a rendered overlay still has mutable tags or unset domains.
# Usage: bash deploy/release-gate.sh [staging|prod]   (default: prod)
set -euo pipefail

env="${1:-prod}"
cd "$(dirname "$0")/.."

if ! rendered="$(kubectl kustomize "deploy/k8s/overlays/${env}")"; then
  echo "FAIL: kustomize render failed for ${env}" >&2
  exit 1
fi

fail=0
check() {
  local pattern="$1" message="$2" hits
  hits="$(printf '%s\n' "$rendered" | grep -nE "$pattern" || true)"
  if [ -n "$hits" ]; then
    echo "FAIL: ${message}" >&2
    printf '%s\n' "$hits" | head -5 >&2
    fail=1
  fi
}

check 'image: .*:latest' 'mutable :latest image tag'
check '0\.0\.0-unset' 'image tag not set by CI (kustomize edit set image)'
check 'REPLACE_ME_DOMAIN' 'domain placeholder not replaced'
check 'erp\.example' 'legacy erp.example placeholder'
check 'sslmode=disable' 'database TLS disabled'

if [ "$fail" -ne 0 ]; then
  exit 1
fi
echo "RELEASE_GATE_PASS (${env})"
