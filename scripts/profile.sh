#!/usr/bin/env bash

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
PROFILE_DIR="./profiles"
BENCHMARK_PATTERN="${BENCHMARK_PATTERN:-BenchmarkTaskProcessing}"
BENCHMARK_TIME="${BENCHMARK_TIME:-10s}"
BENCHMARK_COUNT="${BENCHMARK_COUNT:-3}"

# Create profiles directory
mkdir -p "${PROFILE_DIR}"

echo -e "${BLUE}🔍 Goque Performance Profiling${NC}"
echo "========================================"
echo -e "Benchmark: ${GREEN}${BENCHMARK_PATTERN}${NC}"
echo -e "Duration:  ${GREEN}${BENCHMARK_TIME}${NC}"
echo -e "Runs:      ${GREEN}${BENCHMARK_COUNT}${NC}"
echo -e "Output:    ${GREEN}${PROFILE_DIR}${NC}"
echo "========================================"
echo ""

# Function to run benchmark with specific profile
run_profile() {
    local profile_type=$1
    local profile_flag=$2
    local output_file="${PROFILE_DIR}/${profile_type}.prof"

    echo -e "${YELLOW}Running ${profile_type} profiling...${NC}"

    go test -bench="${BENCHMARK_PATTERN}" \
        -benchtime="${BENCHMARK_TIME}" \
        -count="${BENCHMARK_COUNT}" \
        -"${profile_flag}"="${output_file}" \
        -benchmem \
        ./test 2>&1 | tee "${PROFILE_DIR}/${profile_type}_bench.txt"

    if [ -f "${output_file}" ]; then
        echo -e "${GREEN}✓ ${profile_type} profile saved to ${output_file}${NC}"
    else
        echo -e "${RED}✗ Failed to generate ${profile_type} profile${NC}"
    fi
    echo ""
}

# Function to analyze profile
analyze_profile() {
    local profile_type=$1
    local profile_file="${PROFILE_DIR}/${profile_type}.prof"

    if [ ! -f "${profile_file}" ]; then
        echo -e "${RED}Profile file ${profile_file} not found${NC}"
        return 1
    fi

    echo -e "${BLUE}Analyzing ${profile_type} profile...${NC}"

    # Generate top 20 entries
    echo "Top 20 functions by ${profile_type}:" > "${PROFILE_DIR}/${profile_type}_top.txt"
    go tool pprof -top -nodecount=20 "${profile_file}" >> "${PROFILE_DIR}/${profile_type}_top.txt"

    # Generate list format
    echo "Detailed list:" > "${PROFILE_DIR}/${profile_type}_list.txt"
    go tool pprof -list=. "${profile_file}" >> "${PROFILE_DIR}/${profile_type}_list.txt" 2>/dev/null || true

    echo -e "${GREEN}✓ Analysis saved to ${PROFILE_DIR}/${profile_type}_top.txt${NC}"
}

# Main profiling workflow
main() {
    local mode="${1:-all}"

    case "${mode}" in
        cpu)
            run_profile "cpu" "cpuprofile"
            analyze_profile "cpu"
            ;;
        mem)
            run_profile "mem" "memprofile"
            analyze_profile "mem"
            ;;
        block)
            run_profile "block" "blockprofile"
            analyze_profile "block"
            ;;
        mutex)
            run_profile "mutex" "mutexprofile"
            analyze_profile "mutex"
            ;;
        all)
            echo -e "${BLUE}Running all profiles...${NC}\n"
            run_profile "cpu" "cpuprofile"
            run_profile "mem" "memprofile"
            run_profile "block" "blockprofile"
            run_profile "mutex" "mutexprofile"

            echo -e "${BLUE}Analyzing profiles...${NC}\n"
            analyze_profile "cpu"
            analyze_profile "mem"
            analyze_profile "block"
            analyze_profile "mutex"
            ;;
        analyze)
            echo -e "${BLUE}Analyzing existing profiles...${NC}\n"
            analyze_profile "cpu"
            analyze_profile "mem"
            analyze_profile "block"
            analyze_profile "mutex"
            ;;
        clean)
            echo -e "${YELLOW}Cleaning profile directory...${NC}"
            rm -rf "${PROFILE_DIR}"
            echo -e "${GREEN}✓ Cleaned${NC}"
            exit 0
            ;;
        help|*)
            echo "Usage: $0 [mode]"
            echo ""
            echo "Modes:"
            echo "  all     - Run all profiles (cpu, mem, block, mutex) [default]"
            echo "  cpu     - Run CPU profiling only"
            echo "  mem     - Run memory profiling only"
            echo "  block   - Run block profiling only"
            echo "  mutex   - Run mutex profiling only"
            echo "  analyze - Analyze existing profile files"
            echo "  clean   - Remove profile directory"
            echo "  help    - Show this help message"
            echo ""
            echo "Environment variables:"
            echo "  BENCHMARK_PATTERN - Benchmark pattern to run (default: BenchmarkTaskProcessing)"
            echo "  BENCHMARK_TIME    - Duration per benchmark (default: 10s)"
            echo "  BENCHMARK_COUNT   - Number of runs (default: 3)"
            echo ""
            echo "Examples:"
            echo "  $0 all"
            echo "  $0 cpu"
            echo "  BENCHMARK_PATTERN=BenchmarkWorkerPool $0 all"
            echo "  BENCHMARK_TIME=30s BENCHMARK_COUNT=5 $0 mem"
            exit 0
            ;;
    esac

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}✓ Profiling complete!${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo ""
    echo "Profile files location: ${PROFILE_DIR}"
    echo ""
    echo "To view interactive profile:"
    echo "  go tool pprof -http=:8080 ${PROFILE_DIR}/cpu.prof"
    echo "  go tool pprof -http=:8080 ${PROFILE_DIR}/mem.prof"
    echo ""
    echo "To compare profiles:"
    echo "  go tool pprof -http=:8080 -diff_base=${PROFILE_DIR}/cpu.prof ${PROFILE_DIR}/cpu.prof"
}

# Check if Docker is running (for databases)
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}⚠ Docker is not running. Please start Docker first.${NC}"
    echo -e "Run: ${YELLOW}make docker-up${NC}"
    exit 1
fi

# Run main workflow
main "$@"
