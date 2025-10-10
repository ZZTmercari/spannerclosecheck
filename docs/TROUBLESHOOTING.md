# Troubleshooting Guide

This guide helps you understand and resolve warnings from `spannerclosecheck`.

## Understanding the Warnings

### Warning Message Format

```
filename.go:42:10: ReadOnlyTransaction.Close() must be deferred
filename.go:55:15: RowIterator.Stop() must be deferred
filename.go:63:12: BatchReadOnlyTransaction.Close() must be deferred
```

Each warning indicates:
- **Location**: File, line, and column where the resource is created
- **Resource type**: What Spanner resource needs cleanup
- **Required action**: Must call `Close()` or `Stop()` in a defer statement

## Common Scenarios

### Scenario 1: "But I did call Close()!"

**Your code:**
```go
func example(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    txn.Close()  // ⚠️ Still flagged!
}
```

**Problem:** Close is called but **not deferred**. If code panics before `Close()`, the resource leaks.

**Solution:** Always use `defer`:
```go
func example(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()  // ✅ Guaranteed cleanup
}
```

### Scenario 2: "I'm using t.Cleanup() in tests"

**Your code:**
```go
func TestExample(t *testing.T) {
    client := getTestClient()
    txn := client.ReadOnlyTransaction()  // ⚠️ Flagged
    t.Cleanup(func() { txn.Close() })
}
```

**Problem:** Linter doesn't recognize `t.Cleanup()` as equivalent to `defer`.

**Solution:** Use `nolint` directive:
```go
func TestExample(t *testing.T) {
    client := getTestClient()
    txn := client.ReadOnlyTransaction() //nolint:spannerclosecheck // closed in t.Cleanup()
    t.Cleanup(func() { txn.Close() })
}
```

**Alternative:** Use defer instead:
```go
func TestExample(t *testing.T) {
    client := getTestClient()
    txn := client.ReadOnlyTransaction()
    defer txn.Close()  // ✅ Simpler and clear
}
```

### Scenario 3: "Resource is stored in a struct"

**Your code:**
```go
type Repository struct {
    txn *spanner.ReadOnlyTransaction
}

func NewRepository(client *spanner.Client) *Repository {
    return &Repository{
        txn: client.ReadOnlyTransaction(),  // ⚠️ Flagged
    }
}
```

**Problem:** Resource lifetime extends beyond function scope.

**Solution:** Add cleanup method and use `nolint`:
```go
type Repository struct {
    txn *spanner.ReadOnlyTransaction  //nolint:spannerclosecheck // closed in Repository.Close()
}

func NewRepository(client *spanner.Client) *Repository {
    return &Repository{
        txn: client.ReadOnlyTransaction(), //nolint:spannerclosecheck
    }
}

func (r *Repository) Close() error {
    return r.txn.Close()
}

// Usage:
func handler(client *spanner.Client) {
    repo := NewRepository(client)
    defer repo.Close()  // ✅ Cleanup at appropriate scope
}
```

### Scenario 4: "I'm passing it to a helper function"

**Your code:**
```go
func processWithTransaction(txn *spanner.ReadOnlyTransaction) error {
    defer txn.Close()
    // ... use txn
}

func main(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()  // ⚠️ Flagged
    processWithTransaction(txn)
}
```

**Problem:** Linter can't verify the helper closes it. This is an anti-pattern anyway.

**Solution A (Recommended):** Caller owns and closes:
```go
func processWithTransaction(txn *spanner.ReadOnlyTransaction) error {
    // Use but don't close
    return nil
}

func main(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()  // ✅ Caller maintains ownership
    processWithTransaction(txn)
}
```

**Solution B:** Use `nolint` if helper must close:
```go
func processWithTransaction(txn *spanner.ReadOnlyTransaction) error {
    defer txn.Close()
    return nil
}

func main(client *spanner.Client) {
    txn := client.ReadOnlyTransaction() //nolint:spannerclosecheck // closed in helper
    processWithTransaction(txn)
}
```

### Scenario 5: "Variable is reassigned"

**Your code:**
```go
func example(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    myTxn := txn  // Reassignment
    defer myTxn.Close()  // ⚠️ Linter doesn't track this
}
```

**Problem:** Linter can't follow reassignments (by design).

**Solution A (Best):** Don't reassign:
```go
func example(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()  // ✅ Direct and clear
}
```

**Solution B:** Use better naming:
```go
func example(client *spanner.Client) {
    myTxn := client.ReadOnlyTransaction()  // Name it correctly from the start
    defer myTxn.Close()  // ✅ No reassignment needed
}
```

### Scenario 6: "Function returns the resource"

**Your code:**
```go
func createIterator(client *spanner.Client) *spanner.RowIterator {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()
    return txn.Query(ctx, stmt)  // ⚠️ Iterator might be flagged
}
```

**Problem:** Iterator is returned, but linter may flag it.

**Status:** This is **handled correctly** by the linter as of version 1.x. The linter skips `RowIterator` when it's returned from a function, as the caller becomes responsible.

**If flagged:** Please report as a bug!

### Scenario 7: "Using Client.Single()"

**Your code:**
```go
func quickRead(client *spanner.Client) {
    txn := client.Single()  // ⚠️ Might be flagged?
    iter := txn.Query(ctx, stmt)
    defer iter.Stop()
}
```

**Status:** `Client.Single()` is **correctly handled** by the linter. It auto-releases sessions and doesn't need `defer txn.Close()`.

**Note:** You still need `defer iter.Stop()` for the iterator!

### Scenario 8: "Generated code is flagged"

**Your code:** Generated files like `models.yo.go`

**Status:** The following patterns are **automatically skipped**:
- `*.yo.go` - Generated by xo/yo
- `*.pb.go` - Protocol buffer files
- `*_gen.go` - General generated files
- Paths containing `generated`

**If still flagged:** Use file-level `nolint`:
```go
package models

//nolint:spannerclosecheck // generated code
```

### Scenario 9: "Framework handles cleanup"

**Your code:**
```go
// Using a framework that manages lifecycle
func handler(ctx framework.Context) {
    txn := ctx.GetClient().ReadOnlyTransaction()  // ⚠️ Flagged
    // Framework closes txn automatically at request end
}
```

**Problem:** Framework cleanup isn't visible to static analysis.

**Solution:** Use `nolint` with explanation:
```go
func handler(ctx framework.Context) {
    txn := ctx.GetClient().ReadOnlyTransaction() //nolint:spannerclosecheck // framework closes on request end
    // Use txn
}
```

**Better:** Make cleanup explicit:
```go
func handler(ctx framework.Context) {
    txn := ctx.GetClient().ReadOnlyTransaction()
    defer txn.Close()  // ✅ Explicit is better than implicit
    // Use txn
}
```

## Nolint Best Practices

### DO: Include a reason
```go
txn := client.ReadOnlyTransaction() //nolint:spannerclosecheck // closed in t.Cleanup()
```

### DON'T: Use without explanation
```go
txn := client.ReadOnlyTransaction() //nolint:spannerclosecheck
```

### DO: Use for legitimate edge cases
- Test fixtures with `t.Cleanup()`
- Long-lived struct fields with cleanup methods
- Framework-managed resources

### DON'T: Use to silence valid warnings
- "I'll remember to close it later" ❌
- "It works in testing" ❌
- "The function is short" ❌

## Debugging Tips

### 1. Check if defer is actually present
```bash
# Search for defer in the flagged function
grep -A 10 "func problematicFunc" yourfile.go | grep defer
```

### 2. Verify correct Close/Stop method
- `ReadOnlyTransaction` → `Close()`
- `BatchReadOnlyTransaction` → `Close()`
- `RowIterator` → `Stop()`

### 3. Check defer is on the right variable
```go
txn := client.ReadOnlyTransaction()
otherTxn := getOtherTransaction()
defer otherTxn.Close()  // ⚠️ Wrong variable!
```

### 4. Test with minimal example
Create a small test case:
```go
package main

import "cloud.google.com/go/spanner"

func minimal(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()
}
```

Run the linter:
```bash
spannerclosecheck ./...
```

## Still Stuck?

### Check the version
```bash
spannerclosecheck -version
```

### Enable verbose output (if supported)
```bash
go vet -vettool=$(which spannerclosecheck) -v ./...
```

### Report an issue
If you believe you've found a bug or false positive:

1. Create a minimal reproduction case
2. Check existing issues: https://github.com/ZZTmercari/spannerclosecheck/issues
3. Open a new issue with:
   - Go version
   - spannerclosecheck version
   - Minimal code example
   - Expected vs actual behavior

## Philosophy

Remember: This linter is **intentionally strict** to:
- Prevent resource leaks
- Make ownership clear
- Encourage local reasoning
- Catch bugs early

When the linter flags something, ask:
1. "Is there a better way to structure this code?"
2. "Can I make the resource lifetime clearer?"
3. "Is this a legitimate edge case that needs `nolint`?"

Most of the time, option 1 is the answer. 🎯
