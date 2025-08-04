# 🔒 Iago - Container Image Builder for CoreOS
# Enhanced justfile with comprehensive development, testing, and deployment commands
# Converted from Makefile with additional utilities and better ergonomics

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🔧 Configuration Variables
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Default domain for machine initialization
DOMAIN := env_var_or_default("DOMAIN", "spouterinn.org")
# Default editor for configuration files
EDITOR := env_var_or_default("EDITOR", "nano")

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 📋 Default Target - Show Help
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Show available commands (default target)
default: help

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 📖 Help & Information
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Display comprehensive help with all available commands
help:
    @echo "🔒 Iago - Container Image Builder for CoreOS"
    @echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    @echo ""
    @echo "🛠️  Core Development:"
    @echo "     build                       🔨 Build iago binary"
    @echo "     test                        🧪 Run tests"
    @echo "     lint                        🔍 Run linting"
    @echo "     fmt                         📝 Format code"
    @echo "     vet                         🔍 Run go vet static analysis"
    @echo "     check                       ✅ Run fmt + vet + lint pipeline"
    @echo ""
    @echo "🧪 Testing & Quality:"
    @echo "     test-coverage               📊 Run tests with coverage"
    @echo "     test-race                   🏃 Run tests with race detection"
    @echo "     pre-commit                  ✨ Full validation pipeline"
    @echo ""
    @echo "🚀 Development Workflows:"
    @echo "     dev                         🚀 Build and validate"
    @echo "     dev-full                    🏗️  Full dev workflow"
    @echo "     watch                       👀 Watch files and rebuild"
    @echo "     debug                       🐛 Build with debug flags"
    @echo "     profile                     📈 Build with profiling"
    @echo ""
    @echo "📦 Container Operations:"
    @echo "     build-all-containers        🏭 Build all containers"
    @echo "     build-container-signed      ✍️  Build with signing"
    @echo "     deploy-machine NAME=[name]  🚢 Complete workflow"
    @echo "     logs NAME=[name]            📋 View container logs"
    @echo "     status NAME=[name]          📊 Check deployment status"
    @echo "     cleanup-containers          🧹 Remove unused containers"
    @echo ""
    @echo "📦 Dependencies & Maintenance:"
    @echo "     mod-tidy                    🧹 Clean up dependencies"
    @echo "     mod-check                   🔍 Check for outdated deps"
    @echo "     deps-update                 ⬆️  Update dependencies"
    @echo "     deps                        📋 List all dependencies"
    @echo ""
    @echo "🏗️  Build & Release:"
    @echo "     snapshot                    📸 Generate dev snapshot"
    @echo "     release                     🚀 Release using goreleaser"
    @echo "     size                        📏 Show binary size"
    @echo ""
    @echo "🔧 Utilities:"
    @echo "     validate                    ✅ Validate configuration"
    @echo "     clean                       🧹 Clean generated files"
    @echo "     version                     ℹ️  Show version info"
    @echo "     iago [args]              🔒 Run iago with args"
    @echo ""
    @echo "💡 Use 'just [command]' to run any command"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🔨 Core Build Commands
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Build iago binary for current platform
build:
    @echo "🔨 Building iago binary..."
    go build -o bin/iago ./cmd/iago

# Build with debug flags for development
debug:
    @echo "🐛 Building iago with debug flags..."
    go build -gcflags="all=-N -l" -o bin/iago-debug ./cmd/iago

# Build with profiling enabled
profile:
    @echo "📈 Building iago with profiling..."
    go build -tags profile -o bin/iago-profile ./cmd/iago

# Install iago to local system
install:
    @echo "📦 Installing iago to system..."
    go install ./cmd/iago

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🧪 Testing Commands
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Run all tests with verbose output
test:
    @echo "🧪 Running tests..."
    go test -v ./...

# Run tests with coverage reporting
test-coverage:
    @echo "📊 Running tests with coverage..."
    go test -v -coverprofile=coverage.out ./...
    go tool cover -html=coverage.out -o coverage.html
    @echo "Coverage report generated: coverage.html"

# Run tests with race detection
test-race:
    @echo "🏃 Running tests with race detection..."
    go test -v -race ./...

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🔍 Code Quality Commands
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Format code using go fmt
fmt:
    @echo "📝 Formatting code..."
    go fmt ./...

# Run go vet static analysis
vet:
    @echo "🔍 Running go vet..."
    go vet ./...

# Run golangci-lint
lint:
    @echo "🔍 Running linting..."
    golangci-lint run

# Run comprehensive code quality checks
check: fmt vet lint
    @echo "✅ All code quality checks completed"

# Full pre-commit validation pipeline
pre-commit: fmt vet lint test
    @echo "✨ Pre-commit validation completed successfully"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🚀 Development Workflows
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Quick development build and validate
dev: build validate
    @echo "🚀 Development build completed"

# Comprehensive development workflow
dev-full: fmt lint test build validate build-ignitions
    @echo "🏗️  Full development workflow completed"

# Watch for file changes and rebuild (requires watchexec)
watch:
    @echo "👀 Watching for changes..."
    watchexec -e go -r -- just build

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 📦 Dependency Management
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Clean up Go module dependencies
mod-tidy:
    @echo "🧹 Cleaning up dependencies..."
    go mod tidy

# Check for outdated dependencies
mod-check:
    @echo "🔍 Checking for outdated dependencies..."
    go list -u -m all

# Update dependencies to latest versions
deps-update:
    @echo "⬆️  Updating dependencies..."
    go get -u ./...
    go mod tidy

# List all project dependencies
deps:
    @echo "📋 Project dependencies:"
    go list -m all

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🔧 Configuration & Validation
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Validate iago configuration
validate: build
    @echo "✅ Validating configuration..."
    -./bin/iago validate

# Build all ignition files
build-ignitions: build
    @echo "🔥 Building ignition files..."
    -./bin/iago build

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 📦 Container Operations
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Build all containers
build-all-containers: build
    @echo "🏭 Building all containers..."
    -./bin/iago build-containers

# Build containers with signing
build-container-signed: build
    @echo "✍️  Building containers with signing..."
    -./bin/iago build-containers --sign

# Deploy a complete machine workflow
deploy-machine NAME: build
    @echo "🚢 Deploying machine: {{NAME}}"
    -./bin/iago deploy --name {{NAME}} --domain {{DOMAIN}}

# View container logs for a specific machine
logs NAME:
    @echo "📋 Viewing logs for: {{NAME}}"
    -docker logs iago-{{NAME}} -f

# Check deployment status for a machine
status NAME:
    @echo "📊 Checking status for: {{NAME}}"
    -docker ps | grep iago-{{NAME}}

# Show container structure for debugging
show-container NAME: build
    @echo "📁 Container structure for: {{NAME}}"
    -./bin/iago show-container --name {{NAME}}

# Show machine configuration
show-machine NAME: build
    @echo "🖥️  Machine configuration for: {{NAME}}"
    -./bin/iago show-machine --name {{NAME}}

# Clean up unused containers
cleanup-containers:
    @echo "🧹 Cleaning up unused containers..."
    -docker container prune -f
    -docker image prune -f

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🏗️  Build & Release
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Generate development snapshot with goreleaser
snapshot:
    @echo "📸 Generating development snapshot..."
    goreleaser release --snapshot --clean

# Release using goreleaser (requires git tag)
release:
    @echo "🚀 Creating release..."
    goreleaser release --clean

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🧹 Cleanup & Utilities
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Clean generated files and build artifacts
clean:
    @echo "🧹 Cleaning generated files..."
    rm -rf bin/ coverage.out coverage.html

# Show binary size after build
size: build
    @echo "📏 Binary size:"
    ls -lh bin/iago | awk '{print $5, $9}'

# Show version and git information
version:
    @echo "ℹ️  Version information:"
    @echo "Git commit: $(git rev-parse --short HEAD)"
    @echo "Git branch: $(git branch --show-current)"
    @echo "Git status: $(git status --porcelain | wc -l | xargs echo) uncommitted changes"
    @echo "Go version: $(go version)"

# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# 🔒 Iago Execution
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# Run iago with arguments (ensures binary is built first)
iago *ARGS: build
    @echo "🔒 Running iago with args: {{ARGS}}"
    ./bin/iago {{ARGS}}
