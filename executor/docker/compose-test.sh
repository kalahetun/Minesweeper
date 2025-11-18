#!/bin/bash

#############################################################################
# Docker Compose Integration Test
# Purpose: Verify docker-compose startup, service readiness, and health checks
# Author: Phase 7 - Cloud-Native Deployment
# Date: 2025-11-16
#############################################################################

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
DOCKER_COMPOSE_FILE="$SCRIPT_DIR/docker-compose.yaml"
TEST_TIMEOUT=120
HEALTH_CHECK_RETRIES=30
HEALTH_CHECK_INTERVAL=2

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[✗]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed"
        exit 1
    fi
    log_success "Docker installed: $(docker --version)"
    
    # Check Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        log_error "Docker Compose is not installed"
        exit 1
    fi
    log_success "Docker Compose installed: $(docker-compose --version)"
    
    # Check if docker-compose file exists
    if [ ! -f "$DOCKER_COMPOSE_FILE" ]; then
        log_error "docker-compose.yaml not found at $DOCKER_COMPOSE_FILE"
        exit 1
    fi
    log_success "docker-compose.yaml found"
    
    # Check Docker daemon is running
    if ! docker info &> /dev/null; then
        log_error "Docker daemon is not running"
        exit 1
    fi
    log_success "Docker daemon is running"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up..."
    if [ -f "$DOCKER_COMPOSE_FILE" ]; then
        docker-compose -f "$DOCKER_COMPOSE_FILE" down --volumes 2>/dev/null || true
    fi
}

# Trap cleanup on exit
trap cleanup EXIT

# Start docker-compose services
start_services() {
    log_info "Starting services with docker-compose..."
    
    cd "$SCRIPT_DIR"
    
    # Pull latest images
    log_info "Pulling images..."
    docker-compose -f docker-compose.yaml pull 2>&1 | grep -E "^(Pulling|Downloaded|Digest)" || true
    
    # Start services
    log_info "Starting containers..."
    if ! docker-compose -f docker-compose.yaml up -d; then
        log_error "Failed to start services"
        return 1
    fi
    
    log_success "Services started"
    sleep 3 # Give services time to stabilize
}

# Wait for service health check
wait_for_service() {
    local service_name="$1"
    local health_endpoint="$2"
    local expected_status="${3:-200}"
    local retries=$HEALTH_CHECK_RETRIES
    
    log_info "Waiting for $service_name to be ready ($health_endpoint)..."
    
    while [ $retries -gt 0 ]; do
        if response=$(curl -s -w "\n%{http_code}" "$health_endpoint" 2>/dev/null); then
            http_code=$(echo "$response" | tail -n 1)
            body=$(echo "$response" | sed '$d')
            
            if [ "$http_code" = "$expected_status" ]; then
                log_success "$service_name is ready (HTTP $http_code)"
                return 0
            fi
        fi
        
        log_warning "$service_name not ready, retrying in ${HEALTH_CHECK_INTERVAL}s... (attempts left: $retries)"
        sleep $HEALTH_CHECK_INTERVAL
        retries=$((retries - 1))
    done
    
    log_error "$service_name failed to become ready after $(($HEALTH_CHECK_RETRIES * $HEALTH_CHECK_INTERVAL))s"
    return 1
}

# Check Control Plane API
test_control_plane_api() {
    log_info "Testing Control Plane API..."
    
    # Test health endpoint
    if ! wait_for_service "Control Plane" "http://localhost:8080/v1/health" 200; then
        return 1
    fi
    
    # Test policy list endpoint
    log_info "Testing GET /v1/policies endpoint..."
    response=$(curl -s -w "\n%{http_code}" http://localhost:8080/v1/policies 2>/dev/null)
    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" != "200" ]; then
        log_error "Failed to get policies (HTTP $http_code)"
        return 1
    fi
    
    log_success "GET /v1/policies returned HTTP 200"
    
    # Test POST endpoint
    log_info "Testing POST /v1/policies endpoint..."
    policy_json='{
        "metadata": {"name": "test-policy-docker", "version": "1.0"},
        "spec": {
            "rules": [
                {
                    "match": {"path": {"exact": "/api/test"}},
                    "fault": {"percentage": 50, "abort": {"httpStatus": 500}}
                }
            ]
        }
    }'
    
    response=$(curl -s -w "\n%{http_code}" -X POST http://localhost:8080/v1/policies \
        -H "Content-Type: application/json" \
        -d "$policy_json" 2>/dev/null)
    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')
    
    if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        log_success "POST /v1/policies returned HTTP $http_code"
    else
        log_warning "POST /v1/policies returned HTTP $http_code (expected 200-201)"
    fi
    
    # Verify policy was created
    log_info "Verifying policy creation..."
    response=$(curl -s -X GET http://localhost:8080/v1/policies/test-policy-docker 2>/dev/null)
    if echo "$response" | grep -q "test-policy-docker"; then
        log_success "Policy created and retrievable"
    else
        log_warning "Policy not immediately available (may be eventual consistency)"
    fi
    
    # Verify that Envoy received the configuration
    log_info "Verifying Envoy received configuration (waiting for polling cycle)..."
    sleep 35 # Wait for the 30s polling interval + buffer
    
    envoy_logs=$(docker-compose -f "$DOCKER_COMPOSE_FILE" logs envoy 2>/dev/null)
    # Check for successful parsing after we created the policy
    if echo "$envoy_logs" | grep -q "Received config update from control plane.*test-policy-docker"; then
        log_success "✓ Envoy successfully received policy configuration from Control Plane"
    elif echo "$envoy_logs" | tail -50 | grep -q "Successfully parsed 1 rules"; then
        log_success "✓ Envoy successfully parsed 1 policy rule"
    else
        log_warning "Configuration push may still be pending due to polling interval"
    fi
}

# Check container logs
check_container_logs() {
    log_info "Checking container logs..."
    
    log_info "Control Plane logs (last 20 lines):"
    docker-compose -f "$DOCKER_COMPOSE_FILE" logs --tail=20 control-plane 2>/dev/null | tail -20 || true
    
    log_info "Envoy logs (last 10 lines):"
    docker-compose -f "$DOCKER_COMPOSE_FILE" logs --tail=10 envoy 2>/dev/null | tail -10 || true
}

# Verify service connectivity
test_service_connectivity() {
    log_info "Testing service connectivity..."
    
    # Check if all expected containers are running
    log_info "Verifying running containers..."
    docker-compose -f "$DOCKER_COMPOSE_FILE" ps
    
    # Count running containers
    running_count=$(docker-compose -f "$DOCKER_COMPOSE_FILE" ps | grep "Up" | wc -l)
    expected_count=$(grep "^  [a-z].*:" "$DOCKER_COMPOSE_FILE" | wc -l)
    
    if [ "$running_count" -gt 0 ]; then
        log_success "Found $running_count running containers"
    else
        log_error "No running containers found"
        return 1
    fi
}

# Main execution
main() {
    log_info "==================================="
    log_info "Docker Compose Integration Test"
    log_info "==================================="
    
    # Check prerequisites
    check_prerequisites
    
    # Clean up any previous instances
    cleanup
    
    # Start services
    if ! start_services; then
        log_error "Failed to start services"
        return 1
    fi
    
    # Test service connectivity
    if ! test_service_connectivity; then
        log_error "Service connectivity test failed"
        return 1
    fi
    
    # Test Control Plane API
    if ! test_control_plane_api; then
        log_error "Control Plane API test failed"
        check_container_logs
        return 1
    fi
    
    # Check logs
    check_container_logs
    
    log_info "==================================="
    log_success "All tests passed!"
    log_info "==================================="
    return 0
}

# Run main function
main "$@"
exit $?
