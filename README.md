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

## Best Practices & Design Philosophy

This linter enforces **strict resource management** by requiring resources to be closed in the same scope where they are created. This design choice is intentional and helps prevent common resource leak patterns.

### ✅ Recommended Patterns

**Pattern 1: Immediate defer after creation**
```go
func recommended(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()  // ✅ Clear ownership and cleanup

    iter := txn.Query(ctx, stmt)
    defer iter.Stop()  // ✅ Iterator also deferred
}
```

**Pattern 2: Early return with defer**
```go
func withEarlyReturn(client *spanner.Client) error {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()  // ✅ Cleanup happens on all return paths

    if err := validate(); err != nil {
        return err  // txn.Close() still called
    }
    return txn.Query(ctx, stmt)
}
```

**Pattern 3: Single() for quick reads (no defer needed)**
```go
func quickRead(client *spanner.Client) error {
    // ✅ Single() auto-releases, but iterator still needs Stop()
    iter := client.Single().Query(ctx, stmt)
    defer iter.Stop()
    return processResults(iter)
}
```

### ⚠️ Patterns That Will Be Flagged

These patterns are intentionally flagged to encourage better practices:

**Anti-pattern 1: Variable reassignment**
```go
func unnecessaryReassignment(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    myTxn := txn  // ⚠️ Flagged: Adds unnecessary indirection
    defer myTxn.Close()
}
```
**Why flagged:** Reassignment makes ownership unclear. Use the original variable name or choose a better name initially.

**Anti-pattern 2: Passing to helper function for cleanup**
```go
func closeHelper(txn *spanner.ReadOnlyTransaction) {
    defer txn.Close()
}

func delegatingCleanup(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()  // ⚠️ Flagged
    closeHelper(txn)  // Ownership transfer unclear
}
```
**Why flagged:** Violates locality principle. The caller can't tell if the helper closes the resource. **Better approach:**
```go
// ✅ Caller owns and closes
func helper(txn *spanner.ReadOnlyTransaction) error {
    return txn.Query(ctx, stmt)  // Use but don't close
}

func caller(client *spanner.Client) error {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()  // ✅ Caller maintains ownership
    return helper(txn)
}
```

**Anti-pattern 3: Struct storage (legitimate but requires nolint)**
```go
type Handler struct {
    txn *spanner.ReadOnlyTransaction  // ⚠️ Flagged
}

func newHandler(client *spanner.Client) *Handler {
    return &Handler{
        txn: client.ReadOnlyTransaction(),  // ⚠️ Flagged
    }
}
```
**Why flagged:** Resource lifetime extends beyond function scope. While this is sometimes necessary (e.g., HTTP handlers, test fixtures), it requires careful management. Use `nolint` and ensure proper cleanup:
```go
type Handler struct {
    txn *spanner.ReadOnlyTransaction  //nolint:spannerclosecheck // closed in Handler.Close()
}

func (h *Handler) Close() error {
    return h.txn.Close()  // ✅ Explicit cleanup method
}

func processRequest(client *spanner.Client) {
    h := &Handler{
        txn: client.ReadOnlyTransaction(), //nolint:spannerclosecheck
    }
    defer h.Close()  // ✅ Still deferred at appropriate scope
}
```

## Suppressing Warnings

Use `nolint` directives when you have a legitimate reason to deviate from the standard pattern.

### When to Use Nolint

- **Test fixtures** using `t.Cleanup()` instead of defer
- **Long-lived resources** in structs with explicit cleanup methods
- **Complex lifecycle management** patterns
- **Framework integration** where cleanup is handled by the framework

### Inline nolint

Add a comment on the same line or the line before the flagged code:

```go
func example(client *spanner.Client) {
    txn := client.ReadOnlyTransaction() //nolint:spannerclosecheck // closed in t.Cleanup()
    t.Cleanup(func() { txn.Close() })
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

## Troubleshooting

Having issues with false positives or unexpected warnings? Check out our comprehensive [Troubleshooting Guide](docs/TROUBLESHOOTING.md) which covers:
- Common scenarios and solutions
- When to use `nolint` directives
- Understanding warning messages
- Debugging tips and best practices

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
│   ├── analyzer.go      # Main analyzer definition and constants
│   ├── defer_only.go    # Defer-only mode implementation (main logic)
│   ├── error.go         # Unified error messages and resource types
│   ├── analyzer_test.go # Tests
│   └── testdata/        # Test fixtures
├── docs/                # Documentation
│   ├── TROUBLESHOOTING.md  # Common issues and solutions
│   └── ssa_examples.md     # SSA internals and examples
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
