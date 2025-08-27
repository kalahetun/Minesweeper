#!/bin/bash

# Test Script for Task O-1: Core Metrics Integration
# This script tests the metrics functionality in the Wasm plugin

set -e

echo "🚀 Starting Task O-1 Metrics Test"
echo "=================================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

BASE_URL="http://localhost:8080"
ENVOY_PROXY_URL="http://localhost:18000"
ENVOY_ADMIN_URL="http://localhost:19000"

echo ""
echo "🏥 Checking System Health"
echo "-------------------------"

# Check control plane health
echo -n "Control Plane Health: "
health_status=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/v1/health")
if [ "$health_status" = "200" ]; then
    echo -e "${GREEN}✅ OK${NC}"
else
    echo -e "${RED}❌ Failed (HTTP $health_status)${NC}"
    exit 1
fi

# Check Envoy admin interface
echo -n "Envoy Admin Interface: "
envoy_admin_status=$(curl -s -o /dev/null -w "%{http_code}" "$ENVOY_ADMIN_URL/stats")
if [ "$envoy_admin_status" = "200" ]; then
    echo -e "${GREEN}✅ OK${NC}"
else
    echo -e "${RED}❌ Failed (HTTP $envoy_admin_status)${NC}"
    exit 1
fi

echo ""
echo "📊 Checking Initial Metrics State"
echo "---------------------------------"

# Get initial metrics from Envoy
echo "Getting initial metrics from Envoy admin interface..."
initial_metrics=$(curl -s "$ENVOY_ADMIN_URL/stats" | grep -E "hfi\.faults\.(aborts_total|delays_total|delay_duration_milliseconds)" || echo "No metrics found")
echo -e "${BLUE}Initial Metrics:${NC}"
echo "$initial_metrics"

echo ""
echo "🔧 Setting up Test Policy"
echo "-------------------------"

echo "Creating policy with abort fault..."
create_status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/v1/policies/create" \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "name": "metrics-test-abort",
      "version": "1.0.0"
    },
    "spec": {
      "rules": [
        {
          "match": {
            "method": {"exact": "GET"},
            "path": {"prefix": "/abort"}
          },
          "fault": {
            "percentage": 100,
            "abort": {
              "httpStatus": 500
            }
          }
        }
      ]
    }
  }')

if [ "$create_status" = "201" ]; then
    echo -e "${GREEN}✅ Abort policy created successfully${NC}"
else
    echo -e "${RED}❌ Failed to create abort policy (HTTP $create_status)${NC}"
fi

echo "Creating policy with delay fault..."
delay_create_status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/v1/policies/create" \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "name": "metrics-test-delay",
      "version": "1.0.0"
    },
    "spec": {
      "rules": [
        {
          "match": {
            "method": {"exact": "GET"},
            "path": {"prefix": "/delay"}
          },
          "fault": {
            "percentage": 100,
            "delay": {
              "fixed_delay": "500ms"
            }
          }
        }
      ]
    }
  }')

if [ "$delay_create_status" = "201" ]; then
    echo -e "${GREEN}✅ Delay policy created successfully${NC}"
else
    echo -e "${RED}❌ Failed to create delay policy (HTTP $delay_create_status)${NC}"
fi

# Wait a moment for policies to propagate
echo "Waiting 5 seconds for policies to propagate..."
sleep 5

echo ""
echo "🎯 Testing Abort Fault and Metrics"
echo "===================================="

echo "Triggering abort faults..."
for i in {1..3}; do
    echo -n "Request $i: "
    abort_status=$(curl -s -o /dev/null -w "%{http_code}" "$ENVOY_PROXY_URL/abort/test$i")
    if [ "$abort_status" = "500" ]; then
        echo -e "${GREEN}✅ Abort triggered (HTTP $abort_status)${NC}"
    else
        echo -e "${YELLOW}⚠️  Unexpected response (HTTP $abort_status)${NC}"
    fi
    sleep 0.5
done

echo ""
echo "🐌 Testing Delay Fault and Metrics"
echo "===================================="

echo "Triggering delay faults..."
for i in {1..3}; do
    echo -n "Request $i: "
    start_time=$(date +%s%N)
    delay_status=$(curl -s -o /dev/null -w "%{http_code}" "$ENVOY_PROXY_URL/delay/test$i")
    end_time=$(date +%s%N)
    duration_ms=$(( (end_time - start_time) / 1000000 ))
    
    if [ "$delay_status" = "200" ] && [ "$duration_ms" -gt 400 ]; then
        echo -e "${GREEN}✅ Delay triggered (HTTP $delay_status, ${duration_ms}ms delay)${NC}"
    else
        echo -e "${YELLOW}⚠️  Unexpected response (HTTP $delay_status, ${duration_ms}ms)${NC}"
    fi
    sleep 0.5
done

echo ""
echo "📈 Checking Updated Metrics"
echo "==========================="

# Wait a moment for metrics to be updated
echo "Waiting 3 seconds for metrics to update..."
sleep 3

echo "Getting updated metrics from Envoy admin interface..."
updated_metrics=$(curl -s "$ENVOY_ADMIN_URL/stats" | grep -E "hfi\.faults\.(aborts_total|delays_total|delay_duration_milliseconds)" || echo "No metrics found")

echo -e "${BLUE}Updated Metrics:${NC}"
echo "$updated_metrics"

echo ""
echo "🔍 Analyzing Metrics Results"
echo "============================"

# Check for abort counter
abort_count=$(echo "$updated_metrics" | grep "hfi.faults.aborts_total" | grep -o '[0-9]\+' | tail -1 || echo "0")
echo "Abort Counter: $abort_count (Expected: 3)"

# Check for delay counter
delay_count=$(echo "$updated_metrics" | grep "hfi.faults.delays_total" | grep -o '[0-9]\+' | tail -1 || echo "0")
echo "Delay Counter: $delay_count (Expected: 3)"

# Check for delay histogram (if implemented)
histogram_metrics=$(echo "$updated_metrics" | grep "hfi.faults.delay_duration_milliseconds" || echo "")
if [ -n "$histogram_metrics" ]; then
    echo -e "${GREEN}✅ Delay duration histogram metrics found${NC}"
    echo "$histogram_metrics"
else
    echo -e "${YELLOW}ℹ️  No delay duration histogram metrics found (optional feature)${NC}"
fi

echo ""
echo "✨ Task O-1 Metrics Test Summary"
echo "==============================="

# Validate results
errors=0

if [ "$abort_count" -ge 3 ]; then
    echo -e "${GREEN}✅ Abort metrics working (Count: $abort_count)${NC}"
else
    echo -e "${RED}❌ Abort metrics not working (Count: $abort_count, Expected: >= 3)${NC}"
    errors=$((errors + 1))
fi

if [ "$delay_count" -ge 3 ]; then
    echo -e "${GREEN}✅ Delay metrics working (Count: $delay_count)${NC}"
else
    echo -e "${RED}❌ Delay metrics not working (Count: $delay_count, Expected: >= 3)${NC}"
    errors=$((errors + 1))
fi

echo ""
echo "🔧 Cleanup"
echo "=========="
echo "Cleaning up test policies..."

curl -s -o /dev/null -X DELETE "$BASE_URL/v1/policies/metrics-test-abort"
curl -s -o /dev/null -X DELETE "$BASE_URL/v1/policies/metrics-test-delay"

echo -e "${GREEN}✅ Cleanup completed${NC}"

echo ""
if [ $errors -eq 0 ]; then
    echo -e "${GREEN}🎉 Task O-1 Metrics Test PASSED!${NC}"
    echo -e "${GREEN}All core metrics are working correctly.${NC}"
    exit 0
else
    echo -e "${RED}❌ Task O-1 Metrics Test FAILED!${NC}"
    echo -e "${RED}$errors metric(s) not working correctly.${NC}"
    exit 1
fi
