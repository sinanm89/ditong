# Contributing to ditong

Thank you for your interest in contributing to ditong!

## Development Setup

### Prerequisites

- Python 3.11+
- Go 1.21+
- Git

### Clone the Repository

```bash
git clone https://github.com/snn/ditong.git
cd ditong
```

### Python Setup

```bash
cd python
python -m venv .venv
source .venv/bin/activate  # On Windows: .venv\Scripts\activate
pip install -e ".[dev]"
```

### Go Setup

```bash
cd go
go mod download
```

## Code Style

### Python

We follow PEP 8 with these tools:
- **black** for formatting (line length 88)
- **isort** for import sorting
- **flake8** for linting
- **mypy** for type checking

```bash
# Format
make fmt-py

# Lint
make lint-py
```

### Go

We follow the Google Go Style Guide with standard tools:
- **gofmt** for formatting
- **go vet** for static analysis
- **staticcheck** for additional linting

```bash
# Format
make fmt-go

# Lint
make lint-go
```

## Testing

### Running Tests

```bash
# All tests
make test

# Python only
make test-py

# Go only
make test-go
```

### Running with Coverage

```bash
# All with coverage
make coverage

# Python
make coverage-py

# Go
make coverage-go
```

### Writing Tests

- Place Python tests in `python/tests/`
- Place Go tests alongside source files with `_test.go` suffix
- Aim for >80% coverage on new code
- Include both unit tests and integration tests where appropriate

## Pull Request Process

1. **Fork** the repository
2. **Create a branch** for your feature/fix:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Make your changes** with appropriate tests
4. **Ensure all tests pass**:
   ```bash
   make test
   make lint
   ```
5. **Commit your changes** with a clear message:
   ```bash
   git commit -m "feat: add new feature X"
   ```
6. **Push** to your fork
7. **Open a Pull Request** against `main`

## Commit Message Format

We follow conventional commits:

- `feat:` - New feature
- `fix:` - Bug fix
- `docs:` - Documentation only
- `style:` - Formatting, no code change
- `refactor:` - Code change that neither fixes a bug nor adds a feature
- `test:` - Adding or updating tests
- `chore:` - Maintenance tasks

Examples:
```
feat: add German language support to normalizer
fix: handle empty word lists in synthesis builder
docs: update installation instructions
test: add coverage for edge cases in ingest module
```

## Adding New Features

### Adding a New Language

1. Add character mappings to `normalizer.py` / `normalizer.go`
2. Add Hunspell URL to `ingest/hunspell.py` / `ingest.go`
3. Add IPA support to `phonetics/transcribe.py` if available
4. Add tests for the new language
5. Update README with the new language

### Adding a New Ingestor

1. Create a new file in `python/ingest/` (e.g., `urban_dictionary.py`)
2. Inherit from `Ingestor` or `DownloadableIngestor`
3. Implement the `parse()` method
4. Register in `python/ingest/__init__.py`
5. Add corresponding Go implementation
6. Add tests

### Adding a New Transcriber Backend

1. Create a new class inheriting from `Transcriber`
2. Implement the `transcribe()` method
3. Add to the registry in `phonetics/transcribe.py`
4. Add tests

## Project Structure

```
ditong/
├── python/
│   ├── ingest/          # Dictionary ingestion
│   ├── builder/         # Output builders
│   ├── phonetics/       # IPA transcription
│   ├── tests/           # Python tests
│   └── main.py          # CLI entry point
├── go/
│   ├── cmd/             # CLI entry point
│   └── internal/        # Internal packages
├── .github/workflows/   # CI configuration
├── Makefile            # Build commands
└── README.md
```

## Questions?

Feel free to open an issue for:
- Bug reports
- Feature requests
- Questions about the codebase

## License

By contributing, you agree that your contributions will be licensed under the GPL-3.0 License.
