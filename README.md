# spannerclosecheck

A Go linter that checks for unclosed Cloud Spanner resources to prevent memory leaks and connection issues.

## Overview

`spannerclosecheck` is a static analysis tool that ensures Cloud Spanner transactions, statements, and row iterators are properly closed. Inspired by [sqlclosecheck](https://github.com/ryanrolds/sqlclosecheck), this tool helps prevent resource leaks in applications using Google Cloud Spanner.

## Features

- Detects unclosed `ReadOnlyTransaction`
- Detects unclosed `BatchReadOnlyTransaction`
- Detects unclosed `RowIterator`
- Requires `Close()` or `Stop()` calls to be deferred

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

## Checked Resources

The following Cloud Spanner types are checked:

- `*spanner.ReadOnlyTransaction` - Must call `Close()`
- `*spanner.BatchReadOnlyTransaction` - Must call `Close()`
- `*spanner.RowIterator` - Must call `Stop()`

Note: `ReadWriteTransaction` is typically managed by the client when using `client.ReadWriteTransaction()` and does not require manual closing.

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

Areas that need work:
- Improving SSA traversal to catch all allocation patterns
- Adding support for more Spanner types
- Performance optimizations
- Additional test cases

## License

MIT License

## Acknowledgments

This project is inspired by [sqlclosecheck](https://github.com/ryanrolds/sqlclosecheck) by Ryan Olds.
