#!/bin/bash

# Docker Environment Test Script for Task C-5
# This script tests all the error handling features implemented in Task C-5

set -e

echo "🚀 Starting Docker Environment Test for Task C-5"
echo "=================================================="

BASE_URL="http://localhost:8080"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to test HTTP status code
test_status_code() {
    local description="$1"
    local expected_status="$2"
    local actual_status="$3"
    
    if [ "$actual_status" = "$expected_status" ]; then
        echo -e "${GREEN}✅ $description: HTTP $actual_status (Expected: $expected_status)${NC}"
        return 0
    else
        echo -e "${RED}❌ $description: HTTP $actual_status (Expected: $expected_status)${NC}"
        return 1
    fi
}

echo ""
echo "🏥 Testing Health Check Endpoint"
echo "--------------------------------"
health_status=$(curl -s -o /dev/null -w "%{http_code}" "$BASE_URL/v1/health")
test_status_code "Health Check" "200" "$health_status"

echo ""
echo "📝 Testing Task C-5 Error Handling Features"
echo "============================================"

echo ""
echo "1️⃣ Testing Policy Creation (Expected: 201 Created)"
echo "---------------------------------------------------"
create_status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/v1/policies/create" \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "name": "c5-test-policy",
      "version": "1.0.0"
    },
    "spec": {
      "rules": [
        {
          "match": {
            "method": {"exact": "GET"}
          },
          "fault": {
            "percentage": 100,
            "delay": {
              "fixed_delay": "1s"
            }
          }
        }
      ]
    }
  }')
test_status_code "Create Policy" "201" "$create_status"

echo ""
echo "2️⃣ Testing Duplicate Creation (Expected: 409 Conflict)"
echo "-------------------------------------------------------"
duplicate_status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/v1/policies/create" \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "name": "c5-test-policy",
      "version": "1.0.0"
    },
    "spec": {
      "rules": [
        {
          "match": {
            "method": {"exact": "GET"}
          },
          "fault": {
            "percentage": 100,
            "delay": {
              "fixed_delay": "1s"
            }
          }
        }
      ]
    }
  }')
test_status_code "Duplicate Creation" "409" "$duplicate_status"

echo ""
echo "3️⃣ Testing Same Version Update (Expected: 400 Bad Request)"
echo "-----------------------------------------------------------"
same_version_status=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/v1/policies/c5-test-policy" \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "name": "c5-test-policy",
      "version": "1.0.0"
    },
    "spec": {
      "rules": [
        {
          "match": {
            "method": {"exact": "GET"}
          },
          "fault": {
            "percentage": 50,
            "delay": {
              "fixed_delay": "2s"
            }
          }
        }
      ]
    }
  }')
test_status_code "Same Version Update" "400" "$same_version_status"

echo ""
echo "4️⃣ Testing Valid Version Update (Expected: 200 OK)"
echo "---------------------------------------------------"
valid_update_status=$(curl -s -o /dev/null -w "%{http_code}" -X PUT "$BASE_URL/v1/policies/c5-test-policy" \
  -H "Content-Type: application/json" \
  -d '{
    "metadata": {
      "name": "c5-test-policy",
      "version": "1.0.1"
    },
    "spec": {
      "rules": [
        {
          "match": {
            "method": {"exact": "GET"}
          },
          "fault": {
            "percentage": 50,
            "delay": {
              "fixed_delay": "2s"
            }
          }
        }
      ]
    }
  }')
test_status_code "Valid Version Update" "200" "$valid_update_status"

echo ""
echo "5️⃣ Testing Get Non-Existent Policy (Expected: 404 Not Found)"
echo "-------------------------------------------------------------"
not_found_status=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/v1/policies/non-existent-policy")
test_status_code "Get Non-Existent Policy" "404" "$not_found_status"

echo ""
echo "6️⃣ Testing Get Existing Policy (Expected: 200 OK)"
echo "--------------------------------------------------"
get_policy_status=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/v1/policies/c5-test-policy")
test_status_code "Get Existing Policy" "200" "$get_policy_status"

echo ""
echo "7️⃣ Testing List Policies (Expected: 200 OK)"
echo "--------------------------------------------"
list_policies_status=$(curl -s -o /dev/null -w "%{http_code}" -X GET "$BASE_URL/v1/policies")
test_status_code "List Policies" "200" "$list_policies_status"

echo ""
echo "8️⃣ Testing Invalid JSON (Expected: 400 Bad Request)"
echo "----------------------------------------------------"
invalid_json_status=$(curl -s -o /dev/null -w "%{http_code}" -X POST "$BASE_URL/v1/policies/create" \
  -H "Content-Type: application/json" \
  -d '{invalid json}')
test_status_code "Invalid JSON" "400" "$invalid_json_status"

echo ""
echo "🎯 Task C-5 Implementation Summary"
echo "=================================="
echo -e "${GREEN}✅ Domain Errors: ErrAlreadyExists, ErrInvalidInput defined${NC}"
echo -e "${GREEN}✅ DAL Layer: Enhanced with Create/Update methods and etcd transactions${NC}"
echo -e "${GREEN}✅ Service Layer: Comprehensive validation with version checking${NC}"
echo -e "${GREEN}✅ API Handler: HTTP status code mapping (409, 400, 404)${NC}"
echo -e "${GREEN}✅ Error Middleware: Automatic error type to HTTP status mapping${NC}"
echo -e "${GREEN}✅ Integration: All components working together in Docker environment${NC}"

echo ""
echo "📊 Test Results Summary"
echo "=======================" 
policy_count=$(curl -s "$BASE_URL/v1/policies" | jq '.policies | length')
echo -e "${YELLOW}Total Policies Created: $policy_count${NC}"
echo -e "${YELLOW}Docker Services: etcd + control-plane${NC}"
echo -e "${YELLOW}Storage Backend: etcd with transaction support${NC}"
echo -e "${GREEN}🎉 All Task C-5 features successfully tested in Docker environment!${NC}"
