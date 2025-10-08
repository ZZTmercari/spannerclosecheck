# spannerclosecheck

[![CI](https://github.com/ZZTmercari/spannerclosecheck/actions/workflows/ci.yml/badge.svg)](https://github.com/ZZTmercari/spannerclosecheck/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/ZZTmercari/spannerclosecheck)](https://goreportcard.com/report/github.com/ZZTmercari/spannerclosecheck)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Reference](https://pkg.go.dev/badge/github.com/ZZTmercari/spannerclosecheck.svg)](https://pkg.go.dev/github.com/ZZTmercari/spannerclosecheck)

A Go linter that checks for unclosed Cloud Spanner resources to prevent memory leaks and connection issues.

## Overview

`spannerclosecheck` is a static analysis tool that ensures Cloud Spanner transactions, statements, and row iterators are properly closed. Inspired by [sqlclosecheck](https://github.com/ryanrolds/sqlclosecheck), this tool helps prevent resource leaks in applications using Google Cloud Spanner.

## Features

- ✅ Detects unclosed `ReadOnlyTransaction` (from `ReadOnlyTransaction()`)
- ✅ Detects unclosed `BatchReadOnlyTransaction`
- ✅ Detects unclosed `RowIterator`
- ✅ Requires `Close()` or `Stop()` calls to be deferred
- ✅ Supports inline and file-level nolint directives
- ✅ Automatically skips generated files (`.yo.go`, `.pb.go`, `_gen.go`)
- ✅ Excludes `ReadWriteTransaction` (managed by client)
- ✅ Excludes `Single()` transactions (auto-releases sessions)

## Installation

```bash
go install github.com/ZZTmercari/spannerclosecheck@latest
```

Or build from source:

```bash
git clone https://github.com/ZZTmercari/spannerclosecheck.git
cd spannerclosecheck
make build
make install
```

## Usage

Run as a standalone tool:

```bash
spannerclosecheck ./...
```

Or use with `go vet`:

```bash
go vet -vettool=$(which spannerclosecheck) ./...
```

## Examples

### Bad: Not closing transaction

```go
func bad(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    // Missing: defer txn.Close()
    // This will be flagged by spannerclosecheck
}
```

### Good: Properly deferred close

```go
func good(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()
    // Properly deferred close
}
```

### Bad: Close not deferred

```go
func bad(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    txn.Close() // Not deferred - will be flagged
}
```

### Bad: Unclosed RowIterator

```go
func bad(client *spanner.Client) {
    ctx := context.Background()
    txn := client.ReadOnlyTransaction()
    defer txn.Close()

    iter := txn.Query(ctx, spanner.Statement{})
    // Missing: defer iter.Stop()
}
```

### Good: Using Single() (no close needed)

```go
func good(client *spanner.Client) {
    ctx := context.Background()
    // Single() auto-releases - no defer needed for the transaction
    iter := client.Single().Query(ctx, spanner.Statement{})
    defer iter.Stop() // Still need to stop the iterator
}
```

### Good: Inline Single() usage

```go
func good(client *spanner.Client) {
    // Common pattern: Single() used inline for quick reads
    row, err := models.FindUser(ctx, client.Single(), userID)
    // No cleanup needed - Single() auto-releases
}
```

## Checked Resources

The following Cloud Spanner types are checked:

| Type | Method | Required Action | Example |
|------|--------|-----------------|---------|
| `*spanner.ReadOnlyTransaction` | `ReadOnlyTransaction()` | Must defer `Close()` | `txn := client.ReadOnlyTransaction(); defer txn.Close()` |
| `*spanner.BatchReadOnlyTransaction` | `BatchReadOnlyTransaction()` | Must defer `Close()` | `txn, _ := client.BatchReadOnlyTransaction(...); defer txn.Close()` |
| `*spanner.RowIterator` | `Query()`, `Read()`, etc. | Must defer `Stop()` | `iter := txn.Query(...); defer iter.Stop()` |

### Not Checked (Auto-Managed)

| Type | Method | Reason |
|------|--------|--------|
| `*spanner.ReadOnlyTransaction` | `Single()` | Auto-releases sessions after use |
| `*spanner.ReadWriteTransaction` | `ReadWriteTransaction()` | Managed by client callback |
| `*spanner.Client` | `NewClient()` | Long-lived, application-level resource |

**Note:** `Client.Single()` returns a `ReadOnlyTransaction` that automatically releases its session after use, so it does not need to be closed.

## Suppressing Warnings

### Inline nolint

Add a comment on the same line or the line before the flagged code:

```go
func example(client *spanner.Client) {
    txn := client.ReadOnlyTransaction() //nolint:spannerclosecheck
    // Use txn with t.Cleanup or other cleanup mechanism
}
```

### File-level nolint

Add a comment near the top of the file (within the first 10 lines):

```go
package mypackage

//nolint:spannerclosecheck // test file uses t.Cleanup() instead of defer

import (...)
```

### Supported nolint formats

- `//nolint:spannerclosecheck` - Disables only spannerclosecheck
- `//nolint:all` - Disables all linters
- `//nolint` - Generic nolint (inline only)

### Automatically Excluded Files

The analyzer automatically skips these file patterns:
- `*.yo.go` - Generated by [xo/yo](https://github.com/xo/xo)
- `*.pb.go` - Protocol buffer generated files
- `*_gen.go` - General generated files
- Files with `generated` in the path

## Integration with golangci-lint

To use `spannerclosecheck` in your project with golangci-lint:

### Option 1: Custom Linter Plugin (Recommended for golangci-lint v1.52.1+)

Add to your `.golangci.yml`:

```yaml
linters-settings:
  custom:
    spannerclosecheck:
      path: /path/to/spannerclosecheck
      description: Checks for unclosed Spanner resources
      original-url: github.com/ZZTmercari/spannerclosecheck
```

### Option 2: Direct Integration

If you want to include this linter in your own golangci-lint fork or custom linter, import the analyzer:

```go
import "github.com/ZZTmercari/spannerclosecheck/pkg/analyzer"

// Use analyzer.Analyzer in your linter configuration
```

### Option 3: Standalone with go vet

```bash
go install github.com/ZZTmercari/spannerclosecheck@latest
go vet -vettool=$(which spannerclosecheck) ./...
```

## Development

### Running Tests

```bash
make test
```

**Note**: The analyzer tests are currently under development. The core structure is in place but SSA traversal logic needs refinement to properly detect all patterns.

### Building

```bash
make build
```

### Project Structure

```
spannerclosecheck/
├── pkg/analyzer/        # Core analyzer logic
│   ├── analyzer.go      # Main analyzer definition
│   ├── defer_only.go    # Defer-only mode implementation
│   ├── analyzer_test.go # Tests
│   └── testdata/        # Test fixtures
├── main.go              # CLI entry point
├── Makefile             # Build automation
└── README.md            # Documentation
```

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

### Workflow

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes and add tests
4. Run tests (`make test`) and linting (`make lint`)
5. Commit your changes (`git commit -m 'Add amazing feature'`)
6. Push to your branch (`git push origin feature/amazing-feature`)
7. Open a Pull Request

All pull requests require review approval before merging. Code owners will be automatically assigned for review.

### Areas that need work

- Improving SSA traversal to catch all allocation patterns
- Adding support for more Spanner types
- Performance optimizations
- Additional test cases

## License

MIT License

## Acknowledgments

This project is inspired by [sqlclosecheck](https://github.com/ryanrolds/sqlclosecheck) by Ryan Olds.
