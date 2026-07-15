#!/usr/bin/env bash
# Validate deploy manifests and local documentation links without starting services.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_DIR="$PROJECT_ROOT/deployments/docker-compose"
K8S_DIR="$PROJECT_ROOT/deployments/k8s"

require_tool() {
    if ! command -v "$1" >/dev/null 2>&1; then
        echo "required tool not found: $1" >&2
        exit 1
    fi
}

for tool in docker go python3 kustomize kubeconform; do
    require_tool "$tool"
done

validate_compose() {
    local env_file="$1"
    shift
    echo "==> docker compose config: $*"
    (
        cd "$COMPOSE_DIR"
        docker compose --env-file "$env_file" "$@" config --quiet
    )
}

validate_compose .env.example -f docker-compose.yml
validate_compose .env.lite.example -f docker-compose.lite.yml
validate_compose .env.postgres.example -f docker-compose.postgres.yml
validate_compose .env.example \
    -f docker-compose.yml \
    -f docker-compose.test.yml \
    -f docker-compose.e2e-ports.yml

rendered_manifest="$(mktemp "${TMPDIR:-/tmp}/micro-one-api-k8s.XXXXXX.yaml")"
trap 'rm -f "$rendered_manifest"' EXIT

echo "==> kustomize build"
kustomize build "$K8S_DIR" > "$rendered_manifest"

echo "==> kubeconform"
kubeconform \
    -strict \
    -summary \
    -kubernetes-version 1.33.0 \
    "$rendered_manifest"

echo "==> Kubernetes Secret/ConfigMap references"
(
    cd "$PROJECT_ROOT"
    go run ./scripts/check-k8s-references.go
)

echo "==> Markdown local links"
"$SCRIPT_DIR/check-markdown-links.py"
