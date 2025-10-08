# Test Files for Spanner Close Check Analyzer

This directory contains test cases organized by resource type and feature.

## File Structure

### Resource-Specific Tests

- **`readonly_transaction_test.go`** - Tests for `ReadOnlyTransaction` resource management
  - Proper defer usage
  - Missing defer detection
  - Non-deferred Close() calls

- **`batch_transaction_test.go`** - Tests for `BatchReadOnlyTransaction` resource management
  - Tuple return handling (resource, error)
  - Defer requirement validation

- **`row_iterator_test.go`** - Tests for `RowIterator` resource management
  - Query() result handling
  - Read() result handling
  - Stop() defer requirement

- **`readwrite_transaction_test.go`** - Tests for `ReadWriteTransaction`
  - Validates that ReadWriteTransaction does NOT require defer (managed by client)

### Feature Tests

- **`nolint_test.go`** - Tests for nolint directive support
  - `//nolint:spannerclosecheck` - Analyzer-specific suppression
  - `//nolint:all` - All-linter suppression
  - `//nolint` - Generic suppression
  - Same-line and line-before placement

## Test Naming Convention

Functions are prefixed with:
- `good*` - Cases that should NOT trigger warnings
- `bad*` - Cases that SHOULD trigger warnings

## Running Tests

```bash
# Run all tests
make test

# Run with verbose output
make test-verbose

# Run with coverage
make test-coverage
```

## Expected Warnings

Test functions marked with `// want "pattern"` comments indicate expected analyzer warnings.
The pattern uses regex format to match the diagnostic message.

Example:
```go
func badReadOnlyTransactionNoDefer(client *spanner.Client) {
    txn := client.ReadOnlyTransaction() // want "ReadOnlyTransaction\\.Close\\(\\) must be deferred"
    _ = txn
}
```
