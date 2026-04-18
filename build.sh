#!/bin/bash
# dotgo Build Script
# Simple build script for dotgo CLI

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Variables
BINARY_NAME="dotgo"
BUILD_DIR="build"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Build flags
LDFLAGS="-ldflags -X main.Version=${VERSION} -X main.Commit=${COMMIT} -X main.BuildTime=${BUILD_TIME}"

# Function to print colored output
print_status() {
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

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Main build function
build() {
    print_status "Building dotgo version ${VERSION}..."
    
    # Check if Go is installed
    if ! command_exists go; then
        print_error "Go is not installed or not in PATH"
        exit 1
    fi

    # Clean previous builds
    if [ -f "${BINARY_NAME}" ]; then
        print_status "Cleaning previous build..."
        rm -f "${BINARY_NAME}"
    fi

    # Download dependencies
    print_status "Downloading dependencies..."
    go mod download
    go mod tidy

    # Build the binary
    print_status "Building binary..."
    if go build ${LDFLAGS} -o "${BINARY_NAME}" .; then
        print_success "Build completed successfully!"
        echo "  Binary: ${BINARY_NAME}"
        echo "  Version: ${VERSION}"
        echo "  Commit: ${COMMIT}"
        echo "  Build Time: ${BUILD_TIME}"
    else
        print_error "Build failed!"
        exit 1
    fi
}

# Function to run tests
test() {
    print_status "Running tests..."
    if go test -v ./...; then
        print_success "All tests passed!"
    else
        print_error "Tests failed!"
        exit 1
    fi
}

# Function to install the binary
install() {
    if [ ! -f "${BINARY_NAME}" ]; then
        print_status "Binary not found, building first..."
        build
    fi

    GOBIN=$(go env GOPATH)/bin
    print_status "Installing ${BINARY_NAME} to ${GOBIN}..."
    
    if cp "${BINARY_NAME}" "${GOBIN}/"; then
        print_success "${BINARY_NAME} installed successfully!"
        echo "  Location: ${GOBIN}/${BINARY_NAME}"
        echo "  Run with: ${BINARY_NAME} --help"
    else
        print_error "Installation failed!"
        exit 1
    fi
}

# Function to show help
help() {
    echo "dotgo Build Script"
    echo "=================="
    echo ""
    echo "Usage: ./build.sh [command]"
    echo ""
    echo "Commands:"
    echo "  build     Build the binary (default)"
    echo "  test      Run tests"
    echo "  install   Build and install to GOPATH/bin"
    echo "  clean     Clean build artifacts"
    echo "  help      Show this help message"
    echo ""
    echo "Environment:"
    echo "  VERSION:    ${VERSION}"
    echo "  COMMIT:     ${COMMIT}"
    echo "  BUILD_TIME: ${BUILD_TIME}"
}

# Function to clean build artifacts
clean() {
    print_status "Cleaning build artifacts..."
    rm -f "${BINARY_NAME}"
    rm -rf "${BUILD_DIR}"
    go clean
    print_success "Clean completed!"
}

# Main script logic
case "${1:-build}" in
    build)
        build
        ;;
    test)
        test
        ;;
    install)
        install
        ;;
    clean)
        clean
        ;;
    help|--help|-h)
        help
        ;;
    *)
        print_error "Unknown command: $1"
        echo ""
        help
        exit 1
        ;;
esac