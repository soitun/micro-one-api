#!/usr/bin/env bash
# Integration test for docker-compose deployment.
# Usage: ./scripts/test-docker-compose.sh
#
# Prerequisites:
#   - Docker and Docker Compose v2 installed
#   - Ports 3000 and 8080 available
#
# This script:
#   1. Starts all services via docker-compose
#   2. Waits for health checks to pass
#   3. Verifies each service endpoint responds
#   4. Tears down the environment

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
COMPOSE_DIR="$PROJECT_ROOT/deployments/docker-compose"
COMPOSE_FILE="$COMPOSE_DIR/docker-compose.yml"
ENV_FILE="$(mktemp "${TMPDIR:-/tmp}/micro-one-api-compose.XXXXXX")"
COMPOSE_PROJECT_NAME="micro-one-api-smoke-$$"
COMPOSE=(docker compose --project-name "$COMPOSE_PROJECT_NAME" --env-file "$ENV_FILE" -f "$COMPOSE_FILE")

cp "$COMPOSE_DIR/.env.example" "$ENV_FILE"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[TEST]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
fail() { echo -e "${RED}[FAIL]${NC} $*"; }

PASSED=0
FAILED=0

check_endpoint() {
    local name="$1"
    local url="$2"
    local max_retries="${3:-30}"
    local retry_interval="${4:-2}"

    for i in $(seq 1 "$max_retries"); do
        if curl -sf --max-time 3 "$url" > /dev/null 2>&1; then
            log "$name: OK ($url)"
            PASSED=$((PASSED + 1))
            return 0
        fi
        if [ "$i" -lt "$max_retries" ]; then
            sleep "$retry_interval"
        fi
    done

    fail "$name: FAILED after $max_retries attempts ($url)"
    FAILED=$((FAILED + 1))
    return 1
}

check_grpc_port() {
    local name="$1"
    local host="$2"
    local port="$3"
    local max_retries="${4:-30}"
    local retry_interval="${5:-2}"

    for i in $(seq 1 "$max_retries"); do
        if nc -z -w 2 "$host" "$port" 2>/dev/null; then
            log "$name port $port: OK"
            PASSED=$((PASSED + 1))
            return 0
        fi
        if [ "$i" -lt "$max_retries" ]; then
            sleep "$retry_interval"
        fi
    done

    fail "$name port $port: FAILED after $max_retries attempts"
    FAILED=$((FAILED + 1))
    return 1
}

check_container_health() {
    local name="$1"
    local container="$2"
    local check_cmd="$3"
    local max_retries="${4:-30}"
    local retry_interval="${5:-2}"

    for i in $(seq 1 "$max_retries"); do
        if docker exec "$container" sh -c "$check_cmd" >/dev/null 2>&1; then
            log "$name: OK"
            PASSED=$((PASSED + 1))
            return 0
        fi
        if [ "$i" -lt "$max_retries" ]; then
            sleep "$retry_interval"
        fi
    done

    fail "$name: FAILED after $max_retries attempts"
    FAILED=$((FAILED + 1))
    return 1
}

check_docker_health() {
    local name="$1"
    local container="$2"
    local max_retries="${3:-30}"
    local retry_interval="${4:-2}"

    for i in $(seq 1 "$max_retries"); do
        local health
        health=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null || echo "unknown")
        if [ "$health" = "healthy" ]; then
            log "$name: OK (healthy)"
            PASSED=$((PASSED + 1))
            return 0
        fi
        if [ "$i" -lt "$max_retries" ]; then
            sleep "$retry_interval"
        fi
    done

    fail "$name: FAILED after $max_retries attempts (last status: $health)"
    FAILED=$((FAILED + 1))
    return 1
}

check_container_running() {
    local container="$1"
    local max_retries="${2:-30}"
    local retry_interval="${3:-2}"

    for i in $(seq 1 "$max_retries"); do
        local state
        state=$(docker inspect --format='{{.State.Status}}' "$container" 2>/dev/null || echo "missing")
        if [ "$state" = "running" ]; then
            log "$container: OK (running)"
            PASSED=$((PASSED + 1))
            return 0
        fi
        if [ "$i" -lt "$max_retries" ]; then
            sleep "$retry_interval"
        fi
    done

    fail "$container: FAILED after $max_retries attempts (last state: $state)"
    FAILED=$((FAILED + 1))
    return 1
}

check_container_completed() {
    local container="$1"
    local status
    local exit_code
    status=$(docker inspect --format='{{.State.Status}}' "$container" 2>/dev/null || echo "missing")
    exit_code=$(docker inspect --format='{{.State.ExitCode}}' "$container" 2>/dev/null || echo "unknown")
    if [ "$status" = "exited" ] && [ "$exit_code" = "0" ]; then
        log "$container: OK (completed successfully)"
        PASSED=$((PASSED + 1))
        return 0
    fi
    fail "$container: expected successful completion, got status=$status exit=$exit_code"
    FAILED=$((FAILED + 1))
    return 1
}

cleanup() {
    log "Tearing down docker-compose..."
    "${COMPOSE[@]}" down -v --remove-orphans 2>/dev/null || true
    rm -f "$ENV_FILE"
}

trap cleanup EXIT

# ── Start ──

log "Starting docker-compose environment..."
"${COMPOSE[@]}" config --quiet
# Keep BuildKit progress visible so a slow or failed build is diagnosable.
"${COMPOSE[@]}" --progress plain build
"${COMPOSE[@]}" up -d

log "Waiting for services to be ready (up to 120s)..."

# ── Infrastructure checks (ports not exposed to host, check container health status) ──

check_docker_health "MySQL" "mysql" 30 3
check_docker_health "Redis" "redis" 30 2
check_container_completed "micro-one-api-migrate"

# ── Backend service checks (ports not exposed to host, verify container running) ──

check_container_running "identity-service" 30 2
check_container_running "channel-service" 30 2
check_container_running "billing-service" 30 2
check_container_running "admin-api" 30 2
check_container_running "config-service" 30 2
check_container_running "log-service" 30 2
check_container_running "monitor-worker" 30 2
check_container_running "notify-worker" 30 2
check_container_running "relay-gateway" 30 2

# Verify every internal HTTP service is reachable over the backend network.
check_container_health "identity-service /healthz" "admin-api" \
    "wget -qO- http://identity-service:8001/healthz" 30 2
check_container_health "channel-service /healthz" "admin-api" \
    "wget -qO- http://channel-service:8002/healthz" 30 2
check_container_health "billing-service /healthz" "admin-api" \
    "wget -qO- http://billing-service:8004/healthz" 30 2
check_container_health "config-service /healthz" "admin-api" \
    "wget -qO- http://config-service:8005/healthz" 30 2
check_container_health "log-service /healthz" "admin-api" \
    "wget -qO- http://log-service:8006/healthz" 30 2
check_container_health "monitor-worker /healthz" "admin-api" \
    "wget -qO- http://monitor-worker:8007/healthz" 30 2
check_container_health "notify-worker /healthz" "admin-api" \
    "wget -qO- http://notify-worker:8008/healthz" 30 2

# The protected log API must accept the shared service token.
check_container_health "log-service authenticated API" "admin-api" \
    "wget -qO- --header='Authorization: Bearer change-me-to-a-long-random-string' http://log-service:8006/v1/logs" 30 2

# ── API endpoint checks ──

check_endpoint "relay-gateway /healthz" "http://127.0.0.1:8080/healthz" 30 2

# Verify /healthz returns {"status":"ok"}
health_resp=$(curl -sf --max-time 3 "http://127.0.0.1:8080/healthz" 2>/dev/null || echo "{}")
if echo "$health_resp" | grep -q '"status"'; then
    log "relay-gateway health response: OK"
    PASSED=$((PASSED + 1))
else
    fail "relay-gateway health response: unexpected body: $health_resp"
    FAILED=$((FAILED + 1))
fi

# Verify /v1/models requires auth (should return 401)
models_status=$(curl -s -o /dev/null -w "%{http_code}" --max-time 3 "http://127.0.0.1:8080/v1/models" 2>/dev/null || echo "000")
if [ "$models_status" = "401" ]; then
    log "/v1/models auth check: OK (401 as expected)"
    PASSED=$((PASSED + 1))
else
    fail "/v1/models auth check: expected 401, got $models_status"
    FAILED=$((FAILED + 1))
fi

# ── Summary ──

echo ""
echo "========================================="
echo -e "  Results: ${GREEN}$PASSED passed${NC}, ${RED}$FAILED failed${NC}"
echo "========================================="

if [ "$FAILED" -gt 0 ]; then
    exit 1
fi
