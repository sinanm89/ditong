.PHONY: all test test-py test-go coverage coverage-py coverage-go lint lint-py lint-go fmt fmt-py fmt-go build clean help

# Default target
all: test

# =============================================================================
# Testing
# =============================================================================

test: test-py test-go

test-py:
	@echo "Running Python tests..."
	cd python && python -m pytest tests/ -v

test-go:
	@echo "Running Go tests..."
	cd go && go test -v ./...

# =============================================================================
# Coverage
# =============================================================================

coverage: coverage-py coverage-go

coverage-py:
	@echo "Running Python tests with coverage..."
	cd python && python -m pytest tests/ --cov=. --cov-report=term-missing --cov-report=xml:coverage.xml

coverage-go:
	@echo "Running Go tests with coverage..."
	cd go && go test -v -coverprofile=coverage.out ./...
	cd go && go tool cover -html=coverage.out -o coverage.html

# =============================================================================
# Linting
# =============================================================================

lint: lint-py lint-go

lint-py:
	@echo "Linting Python..."
	cd python && python -m flake8 . --count --show-source --statistics
	cd python && python -m mypy . --ignore-missing-imports

lint-go:
	@echo "Linting Go..."
	cd go && go vet ./...
	@command -v staticcheck >/dev/null 2>&1 && cd go && staticcheck ./... || echo "staticcheck not installed, skipping"

# =============================================================================
# Formatting
# =============================================================================

fmt: fmt-py fmt-go

fmt-py:
	@echo "Formatting Python..."
	cd python && python -m black .
	cd python && python -m isort .

fmt-go:
	@echo "Formatting Go..."
	cd go && gofmt -s -w .

# =============================================================================
# Building
# =============================================================================

build: build-py build-go

build-py:
	@echo "Building Python package..."
	cd python && pip install -e .

build-go:
	@echo "Building Go binary..."
	cd go && go build -o ../bin/ditong ./cmd

# =============================================================================
# Running
# =============================================================================

run-py:
	@echo "Running Python CLI..."
	cd python && python main.py $(ARGS)

run-go:
	@echo "Running Go CLI..."
	cd go && go run ./cmd $(ARGS)

# =============================================================================
# Cleaning
# =============================================================================

clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -rf python/__pycache__ python/**/__pycache__
	rm -rf python/.pytest_cache
	rm -rf python/coverage.xml python/.coverage
	rm -rf python/*.egg-info
	rm -rf go/coverage.out go/coverage.html
	rm -rf sources/
	rm -rf dicts/

clean-cache:
	@echo "Cleaning downloaded dictionaries..."
	rm -rf sources/

clean-output:
	@echo "Cleaning generated dictionaries..."
	rm -rf dicts/

# =============================================================================
# Development
# =============================================================================

install-dev:
	@echo "Installing development dependencies..."
	cd python && pip install -e ".[dev]"
	cd go && go mod download

# =============================================================================
# Help
# =============================================================================

help:
	@echo "ditong Makefile"
	@echo ""
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  test          Run all tests"
	@echo "  test-py       Run Python tests"
	@echo "  test-go       Run Go tests"
	@echo "  coverage      Run tests with coverage"
	@echo "  coverage-py   Run Python tests with coverage"
	@echo "  coverage-go   Run Go tests with coverage"
	@echo "  lint          Run all linters"
	@echo "  lint-py       Run Python linters"
	@echo "  lint-go       Run Go linters"
	@echo "  fmt           Format all code"
	@echo "  fmt-py        Format Python code"
	@echo "  fmt-go        Format Go code"
	@echo "  build         Build all"
	@echo "  build-py      Build Python package"
	@echo "  build-go      Build Go binary"
	@echo "  run-py        Run Python CLI (use ARGS=...)"
	@echo "  run-go        Run Go CLI (use ARGS=...)"
	@echo "  clean         Clean all build artifacts"
	@echo "  clean-cache   Clean downloaded dictionaries"
	@echo "  clean-output  Clean generated dictionaries"
	@echo "  install-dev   Install development dependencies"
	@echo "  help          Show this help"
