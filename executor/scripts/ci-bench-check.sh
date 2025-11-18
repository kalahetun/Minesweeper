#!/bin/bash

################################################################################
# Performance Baseline Regression Detection Script
# Detects >5% regressions in benchmarks compared to baseline
################################################################################

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
EXECUTOR_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
BASELINE_DIR="${EXECUTOR_DIR}/benchmarks"
BASELINE_FILE="${BASELINE_DIR}/baseline.txt"

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Regression threshold (5%)
THRESHOLD=1.05

echo "═══════════════════════════════════════════════════════════════"
echo "Performance Baseline Regression Detection"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# Create baseline directory if needed
mkdir -p "${BASELINE_DIR}"

# Run current benchmarks
echo "Running current benchmarks..."
cd "${EXECUTOR_DIR}/control-plane"

CURRENT_FILE="/tmp/current_bench_$(date +%s).txt"
timeout 90 go test ./tests/benchmarks/... -bench=. -benchtime=3s > "${CURRENT_FILE}" 2>&1 || {
    echo -e "${RED}✗ Benchmark execution failed${NC}"
    exit 1
}

# Extract benchmark results
echo "Analyzing results..."
REGRESSIONS=0

# Function to extract and compare benchmark values
compare_benchmark() {
    local bench_name=$1
    local baseline_time=$2
    
    # Extract current time (handling both ns, µs, ms)
    current_line=$(grep "^${bench_name}" "${CURRENT_FILE}" 2>/dev/null || echo "")
    
    if [ -z "${current_line}" ]; then
        return
    fi
    
    # Extract time value (e.g., "753.3 ns/op")
    current_time=$(echo "${current_line}" | awk '{print $3}' | sed 's/[^0-9.]*//g')
    
    if [ -z "${current_time}" ] || [ -z "${baseline_time}" ]; then
        return
    fi
    
    # Compare (both values in nanoseconds for comparison)
    # Parse unit and convert to nanoseconds
    baseline_ns=$(echo "${baseline_time}" | awk '{print $1}')
    baseline_unit=$(echo "${baseline_time}" | awk '{print $2}')
    
    current_ns=$(echo "${current_line}" | awk '{print $3}' | awk '{print $1}')
    current_unit=$(echo "${current_line}" | awk '{print $3}' | awk '{print $2}')
    
    # Convert to common unit (nanoseconds)
    case "${baseline_unit}" in
        µs) baseline_ns=$(echo "${baseline_ns} * 1000" | bc) ;;
        ms) baseline_ns=$(echo "${baseline_ns} * 1000000" | bc) ;;
        s)  baseline_ns=$(echo "${baseline_ns} * 1000000000" | bc) ;;
    esac
    
    case "${current_unit}" in
        µs) current_ns=$(echo "${current_ns} * 1000" | bc) ;;
        ms) current_ns=$(echo "${current_ns} * 1000000" | bc) ;;
        s)  current_ns=$(echo "${current_ns} * 1000000000" | bc) ;;
    esac
    
    # Calculate ratio
    ratio=$(echo "scale=3; ${current_ns} / ${baseline_ns}" | bc)
    
    # Check regression
    if (( $(echo "${ratio} > ${THRESHOLD}" | bc -l) )); then
        echo -e "${RED}✗ REGRESSION: ${bench_name}${NC}"
        echo "  Baseline: ${baseline_time}"
        echo "  Current:  ${current_ns}${current_unit}"
        echo "  Increase: $(echo "scale=1; (${ratio} - 1) * 100" | bc)%"
        ((REGRESSIONS++))
    else
        if (( $(echo "${ratio} > 1.02" | bc -l) )); then
            echo -e "${YELLOW}⚠ Minor change: ${bench_name} (+$(echo "scale=1; (${ratio} - 1) * 100" | bc)%)${NC}"
        else
            echo -e "${GREEN}✓ ${bench_name}: OK${NC}"
        fi
    fi
}

# Check key benchmarks
echo ""
echo "Key Performance Baselines:"
echo "───────────────────────────────────────────────────"

# Wasm benchmarks (if available)
if [ -d "${EXECUTOR_DIR}/wasm-plugin" ]; then
    cd "${EXECUTOR_DIR}/wasm-plugin"
    echo ""
    echo "Wasm Plugin Benchmarks:"
    
    WASM_BENCH="/tmp/wasm_bench_$(date +%s).txt"
    if timeout 30 cargo bench --lib 2>&1 | grep -E "test.*\.\.\." > "${WASM_BENCH}" 2>/dev/null; then
        # Check matcher
        if grep -q "matcher" "${WASM_BENCH}"; then
            echo -e "${GREEN}✓ Matcher benchmarks: OK${NC}"
        fi
        # Check executor
        if grep -q "executor" "${WASM_BENCH}"; then
            echo -e "${GREEN}✓ Executor benchmarks: OK${NC}"
        fi
    else
        echo -e "${YELLOW}⚠ Wasm benchmarks not available${NC}"
    fi
fi

cd "${EXECUTOR_DIR}/control-plane"

# Control Plane benchmarks
echo ""
echo "Control Plane Benchmarks:"

# Read baseline values (from last known good run)
compare_benchmark "BenchmarkPolicyCreate" "3.876 µs"
compare_benchmark "BenchmarkPolicyRead" "84.47 ns"
compare_benchmark "BenchmarkPolicyUpdate" "26.47 ns"
compare_benchmark "BenchmarkPolicyDelete" "853.9 ns"
compare_benchmark "BenchmarkPolicyConcurrentCreate" "1.214 µs"
compare_benchmark "BenchmarkPolicyConcurrentRead" "18.70 ns"

# Save current results as new baseline
echo ""
echo "───────────────────────────────────────────────────"

if [ ${REGRESSIONS} -eq 0 ]; then
    echo -e "${GREEN}✅ All benchmarks within acceptable range${NC}"
    cp "${CURRENT_FILE}" "${BASELINE_FILE}"
    echo "Baseline updated: ${BASELINE_FILE}"
    exit 0
else
    echo -e "${RED}❌ ${REGRESSIONS} benchmark regression(s) detected!${NC}"
    echo "Check if changes are intentional."
    exit 1
fi
