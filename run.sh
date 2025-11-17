#!/bin/bash

# vaultctl - Build and Run Script
# This script helps build and run the vaultctl password manager

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BINARY_NAME="vaultctl"
BINARY_PATH="${SCRIPT_DIR}/${BINARY_NAME}"

# Function to print colored messages
print_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Function to check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."
    
    local missing=0
    
    if ! command_exists go; then
        print_error "Go is not installed or not in PATH"
        print_info "Please install Go 1.23 or later from https://go.dev"
        missing=1
    else
        GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
        print_success "Go is installed (version: $GO_VERSION)"
    fi
    
    if ! command_exists aws; then
        print_warning "AWS CLI is not installed or not in PATH"
        print_info "DynamoDB sync will not work without AWS CLI"
        print_info "Install from: https://aws.amazon.com/cli/"
    else
        print_success "AWS CLI is installed"
    fi
    
    if [ $missing -eq 1 ]; then
        print_error "Missing required prerequisites. Please install them and try again."
        exit 1
    fi
}

# Function to build the application
build_application() {
    print_info "Building ${BINARY_NAME}..."
    
    cd "$SCRIPT_DIR"
    
    # Download dependencies
    print_info "Downloading dependencies..."
    if ! go mod download; then
        print_error "Failed to download dependencies"
        exit 1
    fi
    
    # Tidy modules
    print_info "Tidying modules..."
    go mod tidy
    
    # Build the application
    print_info "Compiling..."
    if go build -o "$BINARY_NAME" .; then
        print_success "${BINARY_NAME} built successfully!"
        print_info "Binary location: ${BINARY_PATH}"
        
        # Make it executable
        chmod +x "$BINARY_PATH"
        
        return 0
    else
        print_error "Build failed"
        exit 1
    fi
}

# Function to check if binary exists
binary_exists() {
    [ -f "$BINARY_PATH" ] && [ -x "$BINARY_PATH" ]
}

# Function to run the application
run_application() {
    if ! binary_exists; then
        print_warning "Binary not found. Building first..."
        build_application
    fi
    
    print_info "Running ${BINARY_NAME}..."
    echo ""
    
    # Pass all arguments to the binary
    "$BINARY_PATH" "$@"
}

# Function to show usage
show_usage() {
    cat << EOF
${GREEN}vaultctl Build and Run Script${NC}

Usage: ./run.sh [command] [options]

Commands:
  build              Build the application
  run [args...]      Run the application (passes all args to vaultctl)
  build-and-run      Build and then run the application
  clean              Remove build artifacts
  help               Show this help message

Examples:
  ./run.sh build
  ./run.sh run init
  ./run.sh run add --name github --username user@example.com
  ./run.sh build-and-run list
  ./run.sh clean

If no command is specified, the script will:
  1. Check prerequisites
  2. Build the application if needed
  3. Show vaultctl help

EOF
}

# Function to clean build artifacts
clean_build() {
    print_info "Cleaning build artifacts..."
    
    if [ -f "$BINARY_PATH" ]; then
        rm -f "$BINARY_PATH"
        print_success "Removed ${BINARY_NAME}"
    else
        print_info "No binary to clean"
    fi
    
    # Optionally clean Go cache (commented out by default)
    # print_info "Cleaning Go cache..."
    # go clean -cache -modcache -i -r
}

# Main script logic
main() {
    case "${1:-}" in
        build)
            check_prerequisites
            build_application
            ;;
        run)
            shift  # Remove 'run' from arguments
            run_application "$@"
            ;;
        build-and-run|buildrun)
            check_prerequisites
            build_application
            shift  # Remove command from arguments
            run_application "$@"
            ;;
        clean)
            clean_build
            ;;
        help|--help|-h)
            show_usage
            ;;
        "")
            # No command specified - check, build if needed, show help
            check_prerequisites
            if ! binary_exists; then
                print_info "Binary not found. Building..."
                build_application
            fi
            echo ""
            print_info "Showing vaultctl help:"
            echo ""
            "$BINARY_PATH" --help
            ;;
        *)
            # Unknown command - try to run it as vaultctl command
            if binary_exists; then
                run_application "$@"
            else
                print_warning "Binary not found. Building first..."
                check_prerequisites
                build_application
                run_application "$@"
            fi
            ;;
    esac
}

# Run main function
main "$@"

