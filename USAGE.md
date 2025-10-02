# Usage Guide

This guide explains how to integrate `spannerclosecheck` into your Go projects and CI/CD pipelines.

## Quick Start

###1. Install

```bash
go install github.com/ZZTmercari/spannerclosecheck@latest
```

### 2. Run on Your Project

```bash
cd /path/to/your/spanner/project
spannerclosecheck ./...
```

## Integration Methods

### Method 1: golangci-lint Custom Linter (Recommended)

This is the easiest way to integrate with golangci-lint.

1. **Install spannerclosecheck:**
   ```bash
   go install github.com/ZZTmercari/spannerclosecheck@latest
   ```

2. **Add to `.golangci.yml`:**
   ```yaml
   linters-settings:
     custom:
       spannerclosecheck:
         path: spannerclosecheck  # Or full path: $(which spannerclosecheck)
         description: Checks for unclosed Spanner resources
         original-url: github.com/ZZTmercari/spannerclosecheck

   linters:
     enable:
       - spannerclosecheck
   ```

3. **Run golangci-lint:**
   ```bash
   golangci-lint run
   ```

### Method 2: Import as Go Module

Use this if you're building a custom linter or want to programmatically use the analyzer.

```go
package main

import (
    "github.com/ZZTmercari/spannerclosecheck/pkg/analyzer"
    "golang.org/x/tools/go/analysis/multichecker"
)

func main() {
    multichecker.Main(
        analyzer.Analyzer,
        // ... other analyzers
    )
}
```

### Method 3: go vet Tool

```bash
go vet -vettool=$(which spannerclosecheck) ./...
```

### Method 4: GitHub Actions

Add to your `.github/workflows/ci.yml`:

```yaml
name: Lint

on: [push, pull_request]

jobs:
  spannercheck:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Install spannerclosecheck
        run: go install github.com/ZZTmercari/spannerclosecheck@latest

      - name: Run spannerclosecheck
        run: spannerclosecheck ./...
```

## CI/CD Integration

### GitLab CI

```yaml
lint:spanner:
  image: golang:1.23
  script:
    - go install github.com/ZZTmercari/spannerclosecheck@latest
    - spannerclosecheck ./...
```

### CircleCI

```yaml
version: 2.1
jobs:
  lint:
    docker:
      - image: cimg/go:1.23
    steps:
      - checkout
      - run: go install github.com/ZZTmercari/spannerclosecheck@latest
      - run: spannerclosecheck ./...
```

### Jenkins

```groovy
stage('Spanner Check') {
    steps {
        sh 'go install github.com/ZZTmercari/spannerclosecheck@latest'
        sh 'spannerclosecheck ./...
    }
}
```

## Examples of Issues Detected

### Issue: Unclosed ReadOnlyTransaction

```go
// ❌ Bad
func getUser(ctx context.Context, client *spanner.Client, userID string) (*User, error) {
    txn := client.ReadOnlyTransaction()
    // Missing: defer txn.Close()

    iter := txn.Read(ctx, "Users", spanner.Key{userID}, []string{"name", "email"})
    defer iter.Stop()

    // ... process data
}
```

```go
// ✅ Good
func getUser(ctx context.Context, client *spanner.Client, userID string) (*User, error) {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()  // Properly deferred

    iter := txn.Read(ctx, "Users", spanner.Key{userID}, []string{"name", "email"})
    defer iter.Stop()

    // ... process data
}
```

### Issue: Unclosed RowIterator

```go
// ❌ Bad
func listUsers(ctx context.Context, txn *spanner.ReadOnlyTransaction) ([]*User, error) {
    iter := txn.Query(ctx, spanner.Statement{SQL: "SELECT * FROM Users"})
    // Missing: defer iter.Stop()

    var users []*User
    // ... process results
    return users, nil
}
```

```go
// ✅ Good
func listUsers(ctx context.Context, txn *spanner.ReadOnlyTransaction) ([]*User, error) {
    iter := txn.Query(ctx, spanner.Statement{SQL: "SELECT * FROM Users"})
    defer iter.Stop()  // Properly deferred

    var users []*User
    // ... process results
    return users, nil
}
```

### Issue: Close() Not Deferred

```go
// ❌ Bad - Close is called but not deferred
func processData(ctx context.Context, client *spanner.Client) error {
    txn := client.ReadOnlyTransaction()
    // ... do work
    txn.Close()  // Not deferred - could leak if panic occurs
    return nil
}
```

```go
// ✅ Good
func processData(ctx context.Context, client *spanner.Client) error {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()  // Deferred - will always run
    // ... do work
    return nil
}
```

## Configuration

Currently, `spannerclosecheck` only supports defer-only mode, which requires that all `Close()` and `Stop()` calls are deferred.

Future versions may support:
- `closed` mode: Only requires Close() to be called (not necessarily deferred)
- Custom type configuration
- Exclusion patterns

## Troubleshooting

### False Positives

If you encounter false positives, you can:

1. **Add a comment to suppress:**
   ```go
   txn := client.ReadOnlyTransaction() //nolint:spannerclosecheck
   ```

2. **Configure golangci-lint to exclude specific files:**
   ```yaml
   issues:
     exclude-rules:
       - path: integration_test\.go
         linters:
           - spannerclosecheck
   ```

### Performance

For large codebases, you can:

1. **Run on specific packages:**
   ```bash
   spannerclosecheck ./internal/database/...
   ```

2. **Use build tags:**
   ```bash
   spannerclosecheck -tags=integration ./...
   ```

## Support

- **Issues**: https://github.com/ZZTmercari/spannerclosecheck/issues
- **Discussions**: https://github.com/ZZTmercari/spannerclosecheck/discussions

## Best Practices

1. **Always defer Close()**: Use `defer` immediately after acquiring a resource
2. **Check errors**: Don't ignore errors from Close() in production code
3. **Run in CI**: Integrate into your CI pipeline to catch issues early
4. **Use with golangci-lint**: Combine with other linters for comprehensive code quality

```go
// Best practice example
func queryUsers(ctx context.Context, client *spanner.Client) error {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()  // Always defer immediately

    stmt := spanner.Statement{SQL: "SELECT * FROM Users"}
    iter := txn.Query(ctx, stmt)
    defer iter.Stop()  // Always defer immediately

    // Process results
    err := iter.Do(func(row *spanner.Row) error {
        // ...
        return nil
    })

    return err
}
```
