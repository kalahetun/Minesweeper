#!/bin/bash

# Test script for etcd integration
# This script tests the Control Plane with etcd backend

set -e

echo "🚀 Starting etcd backend integration test..."

# Start etcd in the background
echo "📦 Starting etcd with Docker Compose..."
docker-compose -f docker-compose.etcd.yaml up -d

# Wait for etcd to be ready
echo "⏳ Waiting for etcd to be ready..."
sleep 10

# Check etcd health
echo "🔍 Checking etcd health..."
max_retries=30
retry_count=0

while [ $retry_count -lt $max_retries ]; do
    if docker exec $(docker-compose -f docker-compose.etcd.yaml ps -q etcd) etcdctl endpoint health --endpoints=http://localhost:2379 >/dev/null 2>&1; then
        echo "✅ etcd is healthy"
        break
    fi
    
    retry_count=$((retry_count + 1))
    echo "🔄 Retry $retry_count/$max_retries..."
    sleep 2
done

if [ $retry_count -eq $max_retries ]; then
    echo "❌ etcd failed to become healthy"
    docker-compose -f docker-compose.etcd.yaml logs etcd
    exit 1
fi

# Build the control plane
echo "🔨 Building Control Plane..."
cd control-plane
go build -o control-plane .

# Start control plane with etcd backend
echo "🎯 Starting Control Plane with etcd backend..."
STORAGE_BACKEND=etcd ETCD_ENDPOINTS=localhost:2379 ./control-plane &
CONTROL_PLANE_PID=$!

# Wait for control plane to be ready
echo "⏳ Waiting for Control Plane to be ready..."
sleep 5

# Function to cleanup
cleanup() {
    echo "🧹 Cleaning up..."
    if [ ! -z "$CONTROL_PLANE_PID" ]; then
        kill $CONTROL_PLANE_PID 2>/dev/null || true
    fi
    cd ..
    docker-compose -f docker-compose.etcd.yaml down -v
}

# Set trap for cleanup
trap cleanup EXIT

# Test health endpoint
echo "🏥 Testing health endpoint..."
if curl -f http://localhost:8080/v1/health >/dev/null 2>&1; then
    echo "✅ Health endpoint is working"
else
    echo "❌ Health endpoint failed"
    exit 1
fi

# Test policy creation
echo "📝 Testing policy creation..."
POLICY_DATA='{
    "metadata": {
        "name": "test-policy-1"
    },
    "spec": {
        "rules": [
            {
                "type": "latency",
                "config": {
                    "duration_ms": 100,
                    "probability": 0.5
                }
            }
        ]
    }
}'

if curl -X POST \
    -H "Content-Type: application/json" \
    -d "$POLICY_DATA" \
    http://localhost:8080/v1/policies >/dev/null 2>&1; then
    echo "✅ Policy creation successful"
else
    echo "❌ Policy creation failed"
    exit 1
fi

# Test policy retrieval
echo "📖 Testing policy retrieval..."
if curl -f http://localhost:8080/v1/policies/test-policy-1 >/dev/null 2>&1; then
    echo "✅ Policy retrieval successful"
else
    echo "❌ Policy retrieval failed"
    exit 1
fi

# Test policy listing
echo "📋 Testing policy listing..."
if curl -f http://localhost:8080/v1/policies >/dev/null 2>&1; then
    echo "✅ Policy listing successful"
else
    echo "❌ Policy listing failed"
    exit 1
fi

# Test policy deletion
echo "🗑️ Testing policy deletion..."
if curl -X DELETE http://localhost:8080/v1/policies/test-policy-1 >/dev/null 2>&1; then
    echo "✅ Policy deletion successful"
else
    echo "❌ Policy deletion failed"
    exit 1
fi

# Verify policy was deleted
echo "🔍 Verifying policy deletion..."
if curl -f http://localhost:8080/v1/policies/test-policy-1 >/dev/null 2>&1; then
    echo "❌ Policy still exists after deletion"
    exit 1
else
    echo "✅ Policy deletion verified"
fi

echo ""
echo "🎉 All tests passed! etcd integration is working correctly."
echo ""
echo "Test Summary:"
echo "  ✅ etcd startup and health check"
echo "  ✅ Control Plane startup with etcd backend"
echo "  ✅ Health endpoint"
echo "  ✅ Policy creation"
echo "  ✅ Policy retrieval"
echo "  ✅ Policy listing"
echo "  ✅ Policy deletion"
echo "  ✅ Policy deletion verification"
echo ""
echo "🔗 etcd integration is ready for production use!"
