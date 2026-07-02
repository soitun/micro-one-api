#!/bin/bash
# Quick deploy script for billing-service and admin-api updates
# This script handles cross-platform build and deployment to production

set -e

SERVER="root@43.133.65.212"
SERVER_DIR="/opt/micro-one-api"
COMPOSE_DIR="${SERVER_DIR}/docker-compose"

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }
log_step() { echo -e "${BLUE}[STEP]${NC} $1"; }

# Get project root
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="${SCRIPT_DIR}/.."

log_info "======================================"
log_info "  Micro-One-API Production Deploy"
log_info "======================================"
log_info "Services: billing-service, admin-api"
log_info "Server: ${SERVER}"
log_info "Project root: ${PROJECT_ROOT}"
echo ""

# Check prerequisites
log_step "Checking prerequisites..."
if ! docker buildx version &>/dev/null; then
    log_error "docker buildx not available"
    exit 1
fi

if ! ssh -o ConnectTimeout=5 ${SERVER} "echo 'Connected'" &>/dev/null; then
    log_error "Cannot connect to server ${SERVER}"
    exit 1
fi
log_info "Prerequisites OK"
echo ""

# Function to build and deploy a service
deploy_service() {
    local service=$1
    local image_name="docker-compose-${service}:latest"
    local temp_file="/tmp/${service}-image.tar"

    log_step "========================================"
    log_step "Building ${service} (linux/amd64)..."
    log_step "========================================"

    # Build image (cross-platform)
    (cd ${PROJECT_ROOT} && docker buildx build \
        --platform linux/amd64 \
        --load \
        --progress=plain \
        -f deployments/docker/Dockerfile \
        --build-arg SERVICE_NAME=${service} \
        -t ${image_name} \
        .)

    # Get image size
    local size=$(docker inspect ${image_name} --format='{{.Size}}' | awk '{printf "%.2f MB", $1/1024/1024}')
    log_info "Built image size: ${size}"

    # Save image
    log_info "Saving ${service} image..."
    docker save ${image_name} -o ${temp_file}

    # Transfer to server
    log_info "Uploading to server..."
    scp ${temp_file} ${SERVER}:/tmp/

    # Load and deploy on server
    log_info "Deploying on server..."
    ssh ${SERVER} bash << EOF
        set -e

        echo "Loading image..."
        docker load -i /tmp/${service}-image.tar

        echo "Updating container via docker compose..."
        cd ${COMPOSE_DIR}

        # Restart service with docker compose
        docker compose up -d --no-deps ${service}

        # Cleanup
        rm -f /tmp/${service}-image.tar

        echo "Deployed ${service}!"
EOF

    # Cleanup local temp file
    rm -f ${temp_file}

    log_info "${service} deployed successfully!"
    echo ""
}

# Deploy services
deploy_service "billing-service"
deploy_service "admin-api"

# Apply database migrations
log_step "Applying database migrations..."
ssh ${SERVER} bash << 'EOF'
    # Check if there are new migration files not yet applied
    MIGRATION_DIR="/opt/micro-one-api/migrations"
    LATEST_FILE=$(ls -t ${MIGRATION_DIR}/*.sql 2>/dev/null | head -1)

    if [ -n "$LATEST_FILE" ]; then
        echo "Latest migration: $(basename ${LATEST_FILE})"
        echo "Please verify migrations are applied correctly"
    fi
EOF
echo ""

# Check status
log_step "Checking service status..."
ssh ${SERVER} bash << 'EOF'
    echo "======================================"
    echo "  Container Status"
    echo "======================================"
    docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' | grep -E 'NAME|billing-service|admin-api|relay-gateway'

    echo ""
    echo "======================================"
    echo "  Recent Logs (billing-service)"
    echo "======================================"
    docker logs --tail 20 billing-service 2>&1 | tail -20
EOF

echo ""
log_info "========================================"
log_info "Deployment completed!"
log_info "========================================"
