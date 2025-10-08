# Contributing to spannerclosecheck

Thank you for your interest in contributing! This document provides guidelines for contributing to the project.

## Getting Started

### Prerequisites

- Go 1.19 or later
- Make

### Setup

1. Clone the repository:
   ```bash
   git clone https://github.com/ZZTmercari/spannerclosecheck.git
   cd spannerclosecheck
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Build the project:
   ```bash
   make build
   ```

4. Run tests:
   ```bash
   make test
   ```

## Development Workflow

### Running Tests

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage
make test-coverage
```

### Building

```bash
# Build the binary
make build

# Clean build artifacts
make clean
```

### Linting

```bash
# Run linter
make lint
```

## Project Structure

```
spannerclosecheck/
├── .github/workflows/    # CI/CD configuration
├── pkg/analyzer/         # Core analyzer implementation
│   ├── analyzer.go       # Main analyzer setup
│   ├── defer_only.go     # Detection logic
│   └── testdata/         # Test cases
│       └── src/a/        # Test files organized by feature
├── example/              # Usage examples
├── main.go               # CLI entry point
├── Makefile              # Build automation
└── README.md             # Project documentation
```

## Adding New Features

### 1. Detection Logic

- Core detection logic is in `pkg/analyzer/defer_only.go`
- Follow the existing patterns for SSA value inspection
- Add appropriate error messages

### 2. Test Cases

Add tests in `pkg/analyzer/testdata/src/a/`:
- Create a new file or add to existing category file
- Use `good*` prefix for valid cases (should not warn)
- Use `bad*` prefix for invalid cases (should warn)
- Add `// want "pattern"` comments for expected warnings

Example:
```go
func badNoDefer(client *spanner.Client) {
    txn := client.ReadOnlyTransaction() // want "ReadOnlyTransaction\\.Close\\(\\) must be deferred"
    _ = txn
}
```

### 3. Documentation

- Update README.md if adding user-facing features
- Update USAGE.md for new configuration options
- Add comments to exported functions and types

## Code Style

- Follow standard Go formatting (`gofmt`)
- Use meaningful variable and function names
- Add comments for complex logic
- Keep functions focused and small

## Commit Messages

Use clear, descriptive commit messages:
- Start with a verb (Add, Fix, Update, Remove, etc.)
- Keep the first line under 72 characters
- Add details in the body if needed

Examples:
```
Add support for Client.Close() detection

Fix false positives for method chaining

Update test cases for nolint directives
```

## Pull Request Process

1. Create a new branch for your feature/fix
2. Make your changes with clear commits
3. Add or update tests as needed
4. Ensure all tests pass: `make test`
5. Run linter: `make lint`
6. Create a pull request with a clear description

## Testing

### Test Organization

Tests are organized by resource type and feature:
- `readonly_transaction_test.go` - ReadOnlyTransaction tests
- `batch_transaction_test.go` - BatchReadOnlyTransaction tests
- `row_iterator_test.go` - RowIterator tests
- `readwrite_transaction_test.go` - ReadWriteTransaction tests
- `nolint_test.go` - Nolint directive tests

### Writing Tests

1. **Good cases** (should NOT produce warnings):
   ```go
   func goodDefer(client *spanner.Client) {
       txn := client.ReadOnlyTransaction()
       defer txn.Close()
   }
   ```

2. **Bad cases** (should produce warnings):
   ```go
   func badNoDefer(client *spanner.Client) {
       txn := client.ReadOnlyTransaction() // want "ReadOnlyTransaction\\.Close\\(\\) must be deferred"
       _ = txn
   }
   ```

## Questions?

If you have questions or need help, please:
- Open an issue for bugs or feature requests
- Check existing issues and documentation first
- Provide clear, minimal examples when reporting bugs

## License

By contributing, you agree that your contributions will be licensed under the project's MIT License.
