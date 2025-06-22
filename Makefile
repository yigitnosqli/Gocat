# 🚀 GoCat Makefile
# Modern netcat alternative written in Go

# 🎨 Colors for output
RED := \033[31m
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
MAGENTA := \033[35m
CYAN := \033[36m
WHITE := \033[37m
RESET := \033[0m
BOLD := \033[1m

# 📋 Project information
PROJECT_NAME := gocat
PROJECT_DESC := Modern netcat alternative written in Go
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")
BUILT_BY := $(shell whoami)

# 🏗️ Build configuration
GO_VERSION := 1.21
BINARY_NAME := gocat
BINARY_PATH := ./$(BINARY_NAME)
MAIN_PACKAGE := .
BUILD_DIR := build
DIST_DIR := dist
COVERAGE_DIR := coverage

# 🎯 Build flags
LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.buildTime=$(BUILD_TIME) \
	-X main.gitCommit=$(GIT_COMMIT) \
	-X main.gitBranch=$(GIT_BRANCH) \
	-X main.builtBy=$(BUILT_BY)

BUILD_FLAGS := -ldflags="$(LDFLAGS)" -trimpath
BUILD_FLAGS_DEBUG := -ldflags="$(LDFLAGS)" -gcflags="-N -l"

# 🌐 Platform targets
PLATFORMS := \
	linux/amd64 \
	linux/arm64 \
	linux/arm \
	darwin/amd64 \
	darwin/arm64 \
	windows/amd64 \
	windows/arm64 \
	freebsd/amd64

# 🔧 Tools
GOLINT := golangci-lint
GOSEC := gosec
GOVULNCHECK := govulncheck

# 📁 Directories
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

$(DIST_DIR):
	@mkdir -p $(DIST_DIR)

$(COVERAGE_DIR):
	@mkdir -p $(COVERAGE_DIR)

# 🎯 Default target
.PHONY: all
all: clean build test ## Build and test the project

# 📖 Help target
.PHONY: help
help: ## Show this help message
	@echo "$(BOLD)$(CYAN)🚀 $(PROJECT_NAME) - $(PROJECT_DESC)$(RESET)"
	@echo "$(BOLD)$(BLUE)Version: $(VERSION)$(RESET)"
	@echo ""
	@echo "$(BOLD)$(YELLOW)Available targets:$(RESET)"
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  $(CYAN)%-20s$(RESET) %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "$(BOLD)$(YELLOW)Platform targets:$(RESET)"
	@echo "  $(CYAN)build-linux$(RESET)        Build for Linux (amd64)"
	@echo "  $(CYAN)build-darwin$(RESET)       Build for macOS (amd64)"
	@echo "  $(CYAN)build-windows$(RESET)      Build for Windows (amd64)"
	@echo "  $(CYAN)build-all$(RESET)          Build for all platforms"
	@echo ""
	@echo "$(BOLD)$(YELLOW)Docker targets:$(RESET)"
	@echo "  $(CYAN)docker-build$(RESET)       Build Docker image"
	@echo "  $(CYAN)docker-run$(RESET)         Run Docker container"
	@echo "  $(CYAN)docker-push$(RESET)        Push Docker image"

# 🏗️ Build targets
.PHONY: build
build: $(BUILD_DIR) ## Build the binary
	@echo "$(BOLD)$(BLUE)🏗️  Building $(PROJECT_NAME)...$(RESET)"
	@echo "$(YELLOW)Version: $(VERSION)$(RESET)"
	@echo "$(YELLOW)Commit:  $(GIT_COMMIT)$(RESET)"
	@echo "$(YELLOW)Branch:  $(GIT_BRANCH)$(RESET)"
	@echo "$(YELLOW)Time:    $(BUILD_TIME)$(RESET)"
	CGO_ENABLED=0 go build $(BUILD_FLAGS) -o $(BINARY_PATH) $(MAIN_PACKAGE)
	@echo "$(BOLD)$(GREEN)✅ Build completed: $(BINARY_PATH)$(RESET)"

.PHONY: build-debug
build-debug: $(BUILD_DIR) ## Build with debug information
	@echo "$(BOLD)$(BLUE)🐛 Building $(PROJECT_NAME) with debug info...$(RESET)"
	CGO_ENABLED=0 go build $(BUILD_FLAGS_DEBUG) -o $(BINARY_PATH)-debug $(MAIN_PACKAGE)
	@echo "$(BOLD)$(GREEN)✅ Debug build completed: $(BINARY_PATH)-debug$(RESET)"

.PHONY: build-race
build-race: $(BUILD_DIR) ## Build with race detector
	@echo "$(BOLD)$(BLUE)🏃 Building $(PROJECT_NAME) with race detector...$(RESET)"
	CGO_ENABLED=1 go build $(BUILD_FLAGS) -race -o $(BINARY_PATH)-race $(MAIN_PACKAGE)
	@echo "$(BOLD)$(GREEN)✅ Race build completed: $(BINARY_PATH)-race$(RESET)"

# 🌐 Platform-specific builds
.PHONY: build-linux
build-linux: $(DIST_DIR) ## Build for Linux
	@echo "$(BOLD)$(BLUE)🐧 Building for Linux...$(RESET)"
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PACKAGE)
	@echo "$(BOLD)$(GREEN)✅ Linux build completed$(RESET)"

.PHONY: build-darwin
build-darwin: $(DIST_DIR) ## Build for macOS
	@echo "$(BOLD)$(BLUE)🍎 Building for macOS...$(RESET)"
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PACKAGE)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PACKAGE)
	@echo "$(BOLD)$(GREEN)✅ macOS builds completed$(RESET)"

.PHONY: build-windows
build-windows: $(DIST_DIR) ## Build for Windows
	@echo "$(BOLD)$(BLUE)🪟 Building for Windows...$(RESET)"
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PACKAGE)
	CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-windows-arm64.exe $(MAIN_PACKAGE)
	@echo "$(BOLD)$(GREEN)✅ Windows builds completed$(RESET)"

.PHONY: build-freebsd
build-freebsd: $(DIST_DIR) ## Build for FreeBSD
	@echo "$(BOLD)$(BLUE)👹 Building for FreeBSD...$(RESET)"
	CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build $(BUILD_FLAGS) -o $(DIST_DIR)/$(BINARY_NAME)-freebsd-amd64 $(MAIN_PACKAGE)
	@echo "$(BOLD)$(GREEN)✅ FreeBSD build completed$(RESET)"

.PHONY: build-all
build-all: $(DIST_DIR) ## Build for all platforms
	@echo "$(BOLD)$(BLUE)🌐 Building for all platforms...$(RESET)"
	@for platform in $(PLATFORMS); do \
		echo "$(YELLOW)Building for $$platform...$(RESET)"; \
		GOOS=$$(echo $$platform | cut -d'/' -f1); \
		GOARCH=$$(echo $$platform | cut -d'/' -f2); \
		EXT=""; \
		if [ "$$GOOS" = "windows" ]; then EXT=".exe"; fi; \
		CGO_ENABLED=0 GOOS=$$GOOS GOARCH=$$GOARCH go build $(BUILD_FLAGS) \
			-o $(DIST_DIR)/$(BINARY_NAME)-$$GOOS-$$GOARCH$$EXT $(MAIN_PACKAGE); \
	done
	@echo "$(BOLD)$(GREEN)✅ All platform builds completed$(RESET)"

# 🧪 Test targets
.PHONY: test
test: ## Run tests
	@echo "$(BOLD)$(BLUE)🧪 Running tests...$(RESET)"
	go test -v -race ./...
	@echo "$(BOLD)$(GREEN)✅ Tests completed$(RESET)"

.PHONY: test-short
test-short: ## Run short tests
	@echo "$(BOLD)$(BLUE)⚡ Running short tests...$(RESET)"
	go test -short -v ./...
	@echo "$(BOLD)$(GREEN)✅ Short tests completed$(RESET)"

.PHONY: test-coverage
test-coverage: $(COVERAGE_DIR) ## Run tests with coverage
	@echo "$(BOLD)$(BLUE)📊 Running tests with coverage...$(RESET)"
	go test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out -covermode=atomic ./...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "$(BOLD)$(GREEN)✅ Coverage report generated: $(COVERAGE_DIR)/coverage.html$(RESET)"

.PHONY: test-bench
test-bench: ## Run benchmarks
	@echo "$(BOLD)$(BLUE)🏃 Running benchmarks...$(RESET)"
	go test -bench=. -benchmem ./...
	@echo "$(BOLD)$(GREEN)✅ Benchmarks completed$(RESET)"

# 🔍 Code quality targets
.PHONY: lint
lint: ## Run linter
	@echo "$(BOLD)$(BLUE)🔍 Running linter...$(RESET)"
	@if command -v $(GOLINT) >/dev/null 2>&1; then \
		$(GOLINT) run ./...; \
	else \
		echo "$(YELLOW)⚠️  golangci-lint not found, installing...$(RESET)"; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
		$(GOLINT) run ./...; \
	fi
	@echo "$(BOLD)$(GREEN)✅ Linting completed$(RESET)"

.PHONY: fmt
fmt: ## Format code
	@echo "$(BOLD)$(BLUE)🎨 Formatting code...$(RESET)"
	go fmt ./...
	@echo "$(BOLD)$(GREEN)✅ Code formatted$(RESET)"

.PHONY: vet
vet: ## Run go vet
	@echo "$(BOLD)$(BLUE)🔍 Running go vet...$(RESET)"
	go vet ./...
	@echo "$(BOLD)$(GREEN)✅ Vet completed$(RESET)"

.PHONY: security
security: ## Run security scan
	@echo "$(BOLD)$(BLUE)🔒 Running security scan...$(RESET)"
	@if command -v $(GOSEC) >/dev/null 2>&1; then \
		$(GOSEC) ./...; \
	else \
		echo "$(YELLOW)⚠️  gosec not found, installing...$(RESET)"; \
		go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest; \
		$(GOSEC) ./...; \
	fi
	@echo "$(BOLD)$(GREEN)✅ Security scan completed$(RESET)"

.PHONY: vuln
vuln: ## Run vulnerability check
	@echo "$(BOLD)$(BLUE)🛡️  Running vulnerability check...$(RESET)"
	@if command -v $(GOVULNCHECK) >/dev/null 2>&1; then \
		$(GOVULNCHECK) ./...; \
	else \
		echo "$(YELLOW)⚠️  govulncheck not found, installing...$(RESET)"; \
		go install golang.org/x/vuln/cmd/govulncheck@latest; \
		$(GOVULNCHECK) ./...; \
	fi
	@echo "$(BOLD)$(GREEN)✅ Vulnerability check completed$(RESET)"

.PHONY: check
check: fmt vet lint security vuln ## Run all code quality checks
	@echo "$(BOLD)$(GREEN)✅ All checks completed$(RESET)"

# 📦 Dependency targets
.PHONY: deps
deps: ## Download dependencies
	@echo "$(BOLD)$(BLUE)📦 Downloading dependencies...$(RESET)"
	go mod download
	go mod verify
	@echo "$(BOLD)$(GREEN)✅ Dependencies downloaded$(RESET)"

.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "$(BOLD)$(BLUE)🔄 Updating dependencies...$(RESET)"
	go get -u ./...
	go mod tidy
	@echo "$(BOLD)$(GREEN)✅ Dependencies updated$(RESET)"

.PHONY: deps-clean
deps-clean: ## Clean module cache
	@echo "$(BOLD)$(BLUE)🧹 Cleaning module cache...$(RESET)"
	go clean -modcache
	@echo "$(BOLD)$(GREEN)✅ Module cache cleaned$(RESET)"

# 🐳 Docker targets
.PHONY: docker-build
docker-build: ## Build Docker image
	@echo "$(BOLD)$(BLUE)🐳 Building Docker image...$(RESET)"
	docker build \
		--build-arg VERSION=$(VERSION) \
		--build-arg BUILD_TIME=$(BUILD_TIME) \
		--build-arg GIT_COMMIT=$(GIT_COMMIT) \
		-t $(PROJECT_NAME):$(VERSION) \
		-t $(PROJECT_NAME):latest .
	@echo "$(BOLD)$(GREEN)✅ Docker image built$(RESET)"

.PHONY: docker-run
docker-run: ## Run Docker container
	@echo "$(BOLD)$(BLUE)🐳 Running Docker container...$(RESET)"
	docker run --rm -it $(PROJECT_NAME):latest

.PHONY: docker-push
docker-push: ## Push Docker image
	@echo "$(BOLD)$(BLUE)🐳 Pushing Docker image...$(RESET)"
	docker push $(PROJECT_NAME):$(VERSION)
	docker push $(PROJECT_NAME):latest
	@echo "$(BOLD)$(GREEN)✅ Docker image pushed$(RESET)"

.PHONY: docker-compose-up
docker-compose-up: ## Start services with docker-compose
	@echo "$(BOLD)$(BLUE)🐳 Starting services with docker-compose...$(RESET)"
	docker-compose up -d
	@echo "$(BOLD)$(GREEN)✅ Services started$(RESET)"

.PHONY: docker-compose-down
docker-compose-down: ## Stop services with docker-compose
	@echo "$(BOLD)$(BLUE)🐳 Stopping services with docker-compose...$(RESET)"
	docker-compose down
	@echo "$(BOLD)$(GREEN)✅ Services stopped$(RESET)"

# 📦 Package targets
.PHONY: package
package: build ## Create distribution packages
	@echo "$(BOLD)$(BLUE)📦 Creating packages...$(RESET)"
	./pkg/build-packages.sh $(shell echo $(VERSION) | sed 's/^v//')
	@echo "$(BOLD)$(GREEN)✅ Packages created$(RESET)"

.PHONY: homebrew
homebrew: ## Test Homebrew formula
	@echo "$(BOLD)$(BLUE)🍺 Testing Homebrew formula...$(RESET)"
	@if [ -f "Formula/gocat.rb" ]; then \
		brew install --build-from-source Formula/gocat.rb; \
	else \
		echo "$(RED)❌ Homebrew formula not found$(RESET)"; \
		exit 1; \
	fi
	@echo "$(BOLD)$(GREEN)✅ Homebrew formula tested$(RESET)"

.PHONY: release
release: ## Create a new release
	@echo "$(BOLD)$(BLUE)🚀 Creating release...$(RESET)"
	./scripts/create-release.sh
	@echo "$(BOLD)$(GREEN)✅ Release created$(RESET)"

.PHONY: package-rpm
package-rpm: build ## Create RPM package
	@echo "$(BOLD)$(BLUE)📦 Creating RPM package...$(RESET)"
	./build-rpm.sh $(shell echo $(VERSION) | sed 's/^v//') noarch
	@echo "$(BOLD)$(GREEN)✅ RPM package created$(RESET)"

.PHONY: package-deb
package-deb: build ## Create Debian package (requires dpkg-deb)
	@echo "$(BOLD)$(BLUE)📦 Creating Debian package...$(RESET)"
	@if command -v dpkg-deb >/dev/null 2>&1; then \
		cd pkg/debian && ./build-deb.sh $(shell echo $(VERSION) | sed 's/^v//') amd64; \
	else \
		echo "$(RED)❌ dpkg-deb not found. Install dpkg-dev package.$(RESET)"; \
		exit 1; \
	fi
	@echo "$(BOLD)$(GREEN)✅ Debian package created$(RESET)"

# 📋 Install targets
.PHONY: install
install: build ## Install binary to system
	@echo "$(BOLD)$(BLUE)📋 Installing $(PROJECT_NAME)...$(RESET)"
	sudo cp $(BINARY_PATH) /usr/local/bin/$(BINARY_NAME)
	sudo chmod +x /usr/local/bin/$(BINARY_NAME)
	@echo "$(BOLD)$(GREEN)✅ $(PROJECT_NAME) installed to /usr/local/bin/$(RESET)"

.PHONY: uninstall
uninstall: ## Uninstall binary from system
	@echo "$(BOLD)$(BLUE)🗑️  Uninstalling $(PROJECT_NAME)...$(RESET)"
	sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "$(BOLD)$(GREEN)✅ $(PROJECT_NAME) uninstalled$(RESET)"

# 🧹 Clean targets
.PHONY: clean
clean: ## Clean build artifacts
	@echo "$(BOLD)$(BLUE)🧹 Cleaning build artifacts...$(RESET)"
	rm -rf $(BUILD_DIR) $(DIST_DIR) $(COVERAGE_DIR)
	rm -f $(BINARY_NAME) $(BINARY_NAME)-debug $(BINARY_NAME)-race
	go clean -cache -testcache
	@echo "$(BOLD)$(GREEN)✅ Clean completed$(RESET)"

.PHONY: clean-all
clean-all: clean deps-clean ## Clean everything including module cache
	@echo "$(BOLD)$(GREEN)✅ Everything cleaned$(RESET)"

# 🚀 Release targets
.PHONY: release
release: clean check test build-all ## Prepare release
	@echo "$(BOLD)$(BLUE)🚀 Preparing release $(VERSION)...$(RESET)"
	@echo "$(BOLD)$(GREEN)✅ Release $(VERSION) prepared$(RESET)"

.PHONY: tag
tag: ## Create and push git tag
	@echo "$(BOLD)$(BLUE)🏷️  Creating tag $(VERSION)...$(RESET)"
	git tag -a $(VERSION) -m "Release $(VERSION)"
	git push origin $(VERSION)
	@echo "$(BOLD)$(GREEN)✅ Tag $(VERSION) created and pushed$(RESET)"

# 📊 Info targets
.PHONY: info
info: ## Show project information
	@echo "$(BOLD)$(CYAN)📊 Project Information$(RESET)"
	@echo "$(YELLOW)Name:$(RESET)        $(PROJECT_NAME)"
	@echo "$(YELLOW)Description:$(RESET) $(PROJECT_DESC)"
	@echo "$(YELLOW)Version:$(RESET)     $(VERSION)"
	@echo "$(YELLOW)Commit:$(RESET)      $(GIT_COMMIT)"
	@echo "$(YELLOW)Branch:$(RESET)      $(GIT_BRANCH)"
	@echo "$(YELLOW)Build Time:$(RESET)  $(BUILD_TIME)"
	@echo "$(YELLOW)Built By:$(RESET)    $(BUILT_BY)"
	@echo "$(YELLOW)Go Version:$(RESET)  $(shell go version)"

.PHONY: version
version: ## Show version
	@echo "$(VERSION)"

# 🎯 Development targets
.PHONY: dev
dev: build ## Build and run in development mode
	@echo "$(BOLD)$(BLUE)🔧 Running in development mode...$(RESET)"
	./$(BINARY_NAME) --help

.PHONY: watch
watch: ## Watch for changes and rebuild
	@echo "$(BOLD)$(BLUE)👀 Watching for changes...$(RESET)"
	@if command -v fswatch >/dev/null 2>&1; then \
		fswatch -o . | xargs -n1 -I{} make build; \
	else \
		echo "$(RED)❌ fswatch not found. Install with: brew install fswatch$(RESET)"; \
	fi

# 📝 Documentation targets
.PHONY: docs
docs: ## Generate documentation
	@echo "$(BOLD)$(BLUE)📝 Generating documentation...$(RESET)"
	go doc -all > docs/API.md
	@echo "$(BOLD)$(GREEN)✅ Documentation generated$(RESET)"

.PHONY: serve-docs
serve-docs: ## Serve documentation
	@echo "$(BOLD)$(BLUE)📖 Serving documentation...$(RESET)"
	@if command -v godoc >/dev/null 2>&1; then \
		godoc -http=:6060; \
	else \
		echo "$(YELLOW)⚠️  godoc not found, installing...$(RESET)"; \
		go install golang.org/x/tools/cmd/godoc@latest; \
		godoc -http=:6060; \
	fi

# 🎨 Fun targets
.PHONY: logo
logo: ## Show project logo
	@echo "$(BOLD)$(CYAN)"
	@echo "  ██████╗  ██████╗  ██████╗ █████╗ ████████╗"
	@echo " ██╔════╝ ██╔═══██╗██╔════╝██╔══██╗╚══██╔══╝"
	@echo " ██║  ███╗██║   ██║██║     ███████║   ██║   "
	@echo " ██║   ██║██║   ██║██║     ██╔══██║   ██║   "
	@echo " ╚██████╔╝╚██████╔╝╚██████╗██║  ██║   ██║   "
	@echo "  ╚═════╝  ╚═════╝  ╚═════╝╚═╝  ╚═╝   ╚═╝   "
	@echo "$(RESET)"
	@echo "$(BOLD)$(YELLOW)Modern netcat alternative written in Go$(RESET)"
	@echo ""

# 🎯 Aliases
.PHONY: b
b: build ## Alias for build

.PHONY: t
t: test ## Alias for test

.PHONY: c
c: clean ## Alias for clean

.PHONY: r
r: dev ## Alias for dev (run)

# 📋 Make configuration
.DEFAULT_GOAL := help
.SILENT: