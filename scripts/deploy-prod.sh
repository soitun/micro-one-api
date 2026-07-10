#!/bin/bash
# Deploy micro-one-api to production server
# Usage: ./scripts/deploy-prod.sh [services...]
# Examples:
#   ./scripts/deploy-prod.sh admin-api billing-service
#   ./scripts/deploy-prod.sh (deploys all services)

set -e

# Configuration
SERVER="root@43.133.65.212"
SERVER_DIR="/opt/micro-one-api"
LOCAL_REGISTRY="localhost:5000"
COMPOSE_FILE="${SERVER_DIR}/docker-compose.yml"
SERVICES=("$@")

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_step() {
    echo -e "${BLUE}[STEP]${NC} $1"
}

# Get all available services from docker-compose
# Map service name to build path
service_path() {
    case "$1" in
        relay-gateway)      echo "./app/relay/interface/cmd/relay-gateway" ;;
        admin-api)          echo "./app/admin/admin/cmd/admin-api" ;;
        identity-service)   echo "./app/identity/service/cmd/identity-service" ;;
        channel-service)    echo "./app/channel/service/cmd/channel-service" ;;
        billing-service)    echo "./app/billing/service/cmd/billing-service" ;;
        config-service)     echo "./app/config/service/cmd/config-service" ;;
        log-service)        echo "./app/log/service/cmd/log-service" ;;
        monitor-worker)     echo "./app/monitor/job/cmd/monitor-worker" ;;
        notify-worker)      echo "./app/notify/job/cmd/notify-worker" ;;
        *)                  echo "" ;;
    esac
}

get_all_services() {
    echo "identity-service channel-service billing-service admin-api config-service log-service monitor-worker notify-worker relay-gateway"
}

# Check if service exists
validate_services() {
    local all_services=$(get_all_services)
    local invalid=()

    if [ ${#SERVICES[@]} -eq 0 ]; then
        return
    fi

    for svc in "${SERVICES[@]}"; do
        if [[ ! " $all_services " =~ " $svc " ]]; then
            invalid+=("$svc")
        fi
    done

    if [ ${#invalid[@]} -gt 0 ]; then
        log_error "Invalid services: ${invalid[*]}"
        log_info "Available services: $all_services"
        exit 1
    fi
}

# Build image for a service (cross-platform)
build_image() {
    local service=$1
    local image_name="micro-one-api/${service}:latest"

    log_step "Building ${service} (linux/amd64)..."

    local script_dir="$(cd "$(dirname "$0")" && pwd)"
    local project_dir="${script_dir}/.."

    (cd ${project_dir} && docker buildx build \
        --platform linux/amd64 \
        --load \
        -f deployments/docker/Dockerfile \
        --build-arg SERVICE_NAME=${service} --build-arg SERVICE_PATH=$(service_path ${service}) \
        -t ${image_name} \
        .)

    log_info "Built ${service} successfully"
}

# Save and transfer image to server
transfer_image() {
    local service=$1
    local image_name="micro-one-api/${service}:latest"
    local temp_file="/tmp/${service}-image.tar.gz"

    log_step "Transferring ${service} image to server..."

    # Save image to tar.gz
    log_info "Saving image..."
    docker save ${image_name} | gzip > ${temp_file}

    # Get file size
    local size=$(du -h ${temp_file} | cut -f1)
    log_info "Image size: ${size}"

    # Transfer to server
    log_info "Uploading to server..."
    scp ${temp_file} ${SERVER}:/tmp/

    # Load image on server
    log_info "Loading image on server..."
    ssh ${SERVER} "docker load < /tmp/${service}-image.tar.gz"

    # Cleanup
    rm -f ${temp_file}
    ssh ${SERVER} "rm -f /tmp/${service}-image.tar.gz"

    log_info "Transferred ${service} successfully"
}

# Update specific service using docker-compose
update_service() {
    local service=$1
    local image_name="micro-one-api/${service}:latest"
    local container_name=${service}

    log_step "Updating ${service} on server..."

    ssh ${SERVER} bash << EOF
        set -e

        # Check if using docker-compose or standalone containers
        if [ -f "${COMPOSE_FILE}" ]; then
            echo "Using docker-compose to update ${service}..."

            cd ${SERVER_DIR}

            # Pull the image (already loaded, but this updates the reference)
            docker tag ${image_name} ${image_name}

            # Restart the service
            docker-compose up -d --no-deps ${service}

            echo "${service} updated successfully"
        else
            echo "Updating standalone container ${service}..."

            # Stop and remove existing container
            if docker ps -a --format '{{{{.Names}}}}' | grep -q '^${container_name}\$'; then
                docker stop ${container_name} || true
                docker rm ${container_name} || true
            fi

            # Start new container with the new image
            docker run -d \\
                --name ${container_name} \\
                --restart unless-stopped \\
                --network backend \\
                ${image_name}

            echo "${service} updated successfully"
        fi
EOF
}

# Check prerequisites
check_prerequisites() {
    log_step "Checking prerequisites..."

    # Check docker
    if ! command -v docker &>/dev/null; then
        log_error "docker not found"
        exit 1
    fi

    # Check docker buildx
    if ! docker buildx version &>/dev/null; then
        log_error "docker buildx not found"
        exit 1
    fi

    # Check ssh connectivity
    if ! ssh -o ConnectTimeout=5 ${SERVER} "echo 'Connected'" &>/dev/null; then
        log_error "Cannot connect to server ${SERVER}"
        exit 1
    fi

    log_info "All prerequisites satisfied"
}

# Run database migrations if needed
run_migrations() {
    log_step "Checking database migrations..."

    # Check if there are new migration files
    local new_migrations=$(ssh ${SERVER} "ls ${SERVER_DIR}/migrations/*.sql 2>/dev/null | wc -l" || echo "0")

    if [ "$new_migrations" -gt 0 ]; then
        log_warn "New migration files detected. Please run migrations manually if needed."
        log_info "Migration files location: ${SERVER_DIR}/migrations/"
    fi
}

# Main deployment flow
main() {
    echo "======================================"
    echo "  Micro-One-API Production Deploy"
    echo "======================================"
    echo ""

    # If no services specified, deploy all
    if [ ${#SERVICES[@]} -eq 0 ]; then
        log_warn "No services specified, deploying all core services..."
        SERVICES=("billing-service" "admin-api")
        log_info "Default services: ${SERVICES[*]}"
        log_info "To deploy all services, run: $0 $(get_all_services)"
    fi

    validate_services

    log_info "Deployment configuration:"
    log_info "  Server: ${SERVER}"
    log_info "  Services: ${SERVICES[*]}"
    echo ""

    check_prerequisites
    echo ""

    # Build and transfer each service
    for service in "${SERVICES[@]}"; do
        echo ""
        build_image ${service}
        transfer_image ${service}
        update_service ${service}
    done

    echo ""
    run_migrations

    echo ""
    log_info "========================================"
    log_info "Deployment completed successfully!"
    log_info "========================================"
    log_info "Deployed services: ${SERVICES[*]}"
    echo ""

    # Show running containers on server
    log_info "Checking service status..."
    ssh ${SERVER} "docker ps --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}' | grep -E 'NAME|${SERVICES[0]}|${SERVICES[1]}"
}

# Run main
main "$@"
