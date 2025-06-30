#!/bin/bash

# Seeding Management Utility
# Helps manage content seeding operations

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m'

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SEEDER_BINARY="$SCRIPT_DIR/dist/arch-search-seeder"
LOG_DIR="$SCRIPT_DIR/logs"

# Create logs directory
mkdir -p $LOG_DIR

print_help() {
    echo "Arch Search Content Seeding Utility"
    echo "===================================="
    echo ""
    echo "Usage: $0 [COMMAND] [OPTIONS]"
    echo ""
    echo "Commands:"
    echo "  test              Test seeder in dry-run mode"
    echo "  seed              Run full content seeding"
    echo "  seed-sample       Seed a small sample (3 pages)"
    echo "  status            Check seeding status"
    echo "  clean             Clean up logs and temporary files"
    echo "  build             Build the seeder binary"
    echo ""
    echo "Options:"
    echo "  --dry-run         Don't upload to Alchemyst (test mode)"
    echo "  --verbose         Enable verbose logging"
    echo "  --limit N         Process only N pages"
    echo "  --concurrent N    Use N concurrent workers (default: 2)"
    echo "  --delay DURATION  Delay between requests (default: 2s)"
    echo ""
    echo "Examples:"
    echo "  $0 test                              # Test seeder without uploading"
    echo "  $0 seed-sample                       # Seed 3 pages for testing"
    echo "  $0 seed --limit 10                   # Seed 10 pages"
    echo "  $0 seed --concurrent 1 --delay 5s    # Slower, more polite seeding"
    echo ""
}

log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if Go is installed
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed"
        exit 1
    fi
    
    # Check environment variables
    if [ -z "$ALCHEMYST_API_KEY" ]; then
        log_warning "ALCHEMYST_API_KEY not set - only dry-run mode will work"
    fi
    
    # Check if seeder binary exists
    if [ ! -f "$SEEDER_BINARY" ]; then
        log_warning "Seeder binary not found, building..."
        build_seeder
    fi
    
    log_success "Prerequisites check completed"
}

build_seeder() {
    log_info "Building seeder binary..."
    
    cd "$SCRIPT_DIR"
    
    if go build -o "$SEEDER_BINARY" cmd/seed/main.go; then
        log_success "Seeder built successfully"
    else
        log_error "Failed to build seeder"
        exit 1
    fi
}

run_seeder() {
    local args=("$@")
    local timestamp=$(date +%Y%m%d_%H%M%S)
    local log_file="$LOG_DIR/seeder_$timestamp.log"
    
    log_info "Running seeder with args: ${args[*]}"
    log_info "Logs will be written to: $log_file"
    
    # Run seeder and capture output
    if "$SEEDER_BINARY" "${args[@]}" 2>&1 | tee "$log_file"; then
        log_success "Seeding completed successfully"
        log_info "Log file: $log_file"
    else
        log_error "Seeding failed"
        log_info "Check log file for details: $log_file"
        exit 1
    fi
}

check_status() {
    log_info "Checking seeding status..."
    
    # Check recent log files
    recent_logs=$(find "$LOG_DIR" -name "seeder_*.log" -mtime -1 2>/dev/null | sort -r | head -5)
    
    if [ -z "$recent_logs" ]; then
        log_info "No recent seeding activity found"
        return
    fi
    
    log_info "Recent seeding activities:"
    for log_file in $recent_logs; do
        local basename=$(basename "$log_file")
        local timestamp=$(echo "$basename" | sed 's/seeder_\(.*\)\.log/\1/')
        local formatted_time=$(date -d "${timestamp:0:8} ${timestamp:9:2}:${timestamp:11:2}:${timestamp:13:2}" 2>/dev/null || echo "$timestamp")
        
        if grep -q "completed successfully" "$log_file"; then
            log_success "$formatted_time - Completed successfully"
        elif grep -q "failed\|error" "$log_file"; then
            log_error "$formatted_time - Failed with errors"
        else
            log_warning "$formatted_time - Status unknown"
        fi
        
        # Show summary stats
        local pages_processed=$(grep -c "Processing page" "$log_file" 2>/dev/null || echo "0")
        local errors_found=$(grep -c "ERROR" "$log_file" 2>/dev/null || echo "0")
        echo "    Pages: $pages_processed, Errors: $errors_found"
    done
}

clean_logs() {
    log_info "Cleaning up logs and temporary files..."
    
    # Remove logs older than 7 days
    find "$LOG_DIR" -name "*.log" -mtime +7 -delete 2>/dev/null || true
    
    # Remove temporary files
    rm -f "$SCRIPT_DIR"/perf_test_*.log
    rm -f "$SCRIPT_DIR"/concurrent_test.sh
    
    log_success "Cleanup completed"
}

test_seeder() {
    log_info "Testing seeder in dry-run mode..."
    
    local args=(--dry-run --verbose --limit 3)
    args+=("$@")
    
    run_seeder "${args[@]}"
}

seed_sample() {
    log_info "Seeding sample content (3 pages)..."
    
    if [ -z "$ALCHEMYST_API_KEY" ]; then
        log_error "ALCHEMYST_API_KEY required for seeding"
        exit 1
    fi
    
    local args=(--verbose --limit 3)
    args+=("$@")
    
    run_seeder "${args[@]}"
}

seed_full() {
    log_info "Running full content seeding..."
    
    if [ -z "$ALCHEMYST_API_KEY" ]; then
        log_error "ALCHEMYST_API_KEY required for seeding"
        exit 1
    fi
    
    local args=(--verbose)
    args+=("$@")
    
    # Confirm before running full seed
    read -p "This will seed all configured wiki pages. Continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        log_info "Seeding cancelled"
        exit 0
    fi
    
    run_seeder "${args[@]}"
}

# Main execution
main() {
    local command="$1"
    shift || true
    
    case "$command" in
        "test")
            check_prerequisites
            test_seeder "$@"
            ;;
        "seed")
            check_prerequisites
            seed_full "$@"
            ;;
        "seed-sample")
            check_prerequisites
            seed_sample "$@"
            ;;
        "status")
            check_status
            ;;
        "clean")
            clean_logs
            ;;
        "build")
            build_seeder
            ;;
        "help"|"--help"|"-h"|"")
            print_help
            ;;
        *)
            log_error "Unknown command: $command"
            print_help
            exit 1
            ;;
    esac
}

main "$@"