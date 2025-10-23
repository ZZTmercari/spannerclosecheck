# Resource Leak Pattern: `defer` vs Immediate Cleanup

## Overview

This document explains a common resource leak pattern in Go when dealing with long-running operations that recreate resources inside loops. It uses a real example from our Cloud Spanner migration code.

## The Problem: Resource Reassignment in Loops

### Anti-Pattern Example

```go
func (r *dbMigrateRunner) FetchRows(ctx context.Context) {
    rotx := r.paymentdb.ReadOnlyTransaction()
    iter := rotx.Query(ctx, query)
    defer iter.Stop()   // ⚠️ Only cleans up LAST instance
    defer rotx.Close()  // ⚠️ Only cleans up LAST instance

    for {
        row, err := iter.Next()
        if err != nil {
            // Handle Spanner transaction expiry (after 1 hour max staleness)
            if isTransactionExpired(err) {
                // ❌ LEAK: Old rotx and iter are orphaned!
                rotx = r.paymentdb.ReadOnlyTransaction()  // Creates new, abandons old
                iter = rotx.Query(ctx, query)              // Creates new, abandons old
                continue
            }
        }

        processRow(row)
    }
}
```

### Why This Leaks

1. **Initial resources**: `rotx` and `iter` created at function start
2. **Transaction expires**: After 1 hour, Spanner returns staleness error
3. **Reassignment**: New `rotx` and `iter` are created (line 333-334)
4. **Orphaned resources**: OLD `rotx` and OLD `iter` are lost but never closed
5. **`defer` limitation**: `defer` statements only execute when function RETURNS, not when variables are reassigned
6. **Memory leak**: Each recreation leaks the previous transaction and iterator until function exits

### Visual Representation

```
Time 0:     rotx₁, iter₁ created  ← defer will clean these up at function end
            ↓
Time 60min: Transaction expires
            ↓
Time 60min: rotx₂, iter₂ created  ← rotx₁, iter₁ LEAKED (still in memory)
            ↓                         defer will clean rotx₂, iter₂ at function end
Time 120min: Transaction expires
            ↓
Time 120min: rotx₃, iter₃ created ← rotx₂, iter₂ LEAKED (still in memory)
            ↓                         defer will clean rotx₃, iter₃ at function end
...

Function ends: Only rotxₙ, iterₙ are cleaned up
               rotx₁...rotxₙ₋₁ and iter₁...iterₙ₋₁ were LEAKED
```

## The Solution: Immediate Cleanup Before Reassignment

### Correct Pattern

```go
func (r *dbMigrateRunner) FetchRows(ctx context.Context) {
    rotx := r.paymentdb.ReadOnlyTransaction()
    iter := rotx.Query(ctx, query)
    defer iter.Stop()   // ✅ Cleans up final instance
    defer rotx.Close()  // ✅ Cleans up final instance

    for {
        row, err := iter.Next()
        if err != nil {
            // Handle Spanner transaction expiry
            if isTransactionExpired(err) {
                // ✅ CORRECT: Clean up old resources IMMEDIATELY
                iter.Stop()
                rotx.Close()

                // Now create new resources
                rotx = r.paymentdb.ReadOnlyTransaction()
                iter = rotx.Query(ctx, query)
                continue
            }
        }

        processRow(row)
    }
}
```

### Why This Works

1. **Immediate cleanup**: Old resources are closed BEFORE creating new ones
2. **No orphans**: Every resource is explicitly cleaned up
3. **`defer` as safety net**: Still keeps `defer` to clean up the final instance when function returns
4. **No leaks**: Memory is freed immediately when resources are replaced

### Alternative: wrapper function
Alternative for close in loop is wrapper function and defer close in that function
https://stackoverflow.com/questions/45617758/proper-way-to-release-resources-with-defer-in-a-loop

### Visual Representation (Fixed)

```
Time 0:     rotx₁, iter₁ created
            ↓
Time 60min: Transaction expires
            ↓
Time 60min: iter₁.Stop(), rotx₁.Close()  ← Cleaned immediately
            rotx₂, iter₂ created
            ↓
Time 120min: Transaction expires
            ↓
Time 120min: iter₂.Stop(), rotx₂.Close() ← Cleaned immediately
            rotx₃, iter₃ created
            ↓
...

Function ends: rotxₙ, iterₙ cleaned by defer
               No leaks! ✅
```

## Key Principles

### When to Use `defer`

✅ **Use `defer` when:**
- Resource is created once and lives for the entire function scope
- You want guaranteed cleanup even if function panics
- Resource is not reassigned to the same variable

```go
func processFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()  // ✅ GOOD: f is never reassigned

    // ... process file ...
    return nil
}
```

### When to Use Immediate Cleanup

✅ **Use immediate cleanup when:**
- Resource variable is reassigned in a loop
- You need to create multiple instances of the same resource type
- Resource lifetime is shorter than function scope
- You want to free memory immediately

```go
func processMultipleConnections(urls []string) error {
    var conn *Connection
    defer func() {
        if conn != nil {
            conn.Close()  // ✅ Safety net for last instance
        }
    }()

    for _, url := range urls {
        if conn != nil {
            conn.Close()  // ✅ GOOD: Clean up before reassignment
        }

        conn = dial(url)
        // ... use connection ...
    }
    return nil
}
```

### Anti-Pattern: `defer` in Loops

❌ **NEVER do this:**

```go
func processFiles(paths []string) error {
    for _, path := range paths {
        f, err := os.Open(path)
        if err != nil {
            continue
        }
        defer f.Close()  // ❌ BAD: All files stay open until function returns!

        // ... process file ...
    }
    return nil
}
```

**Why it's bad:**
- If you process 1000 files, all 1000 file handles stay open until function ends
- You'll hit OS file descriptor limits
- Memory bloat from keeping all resources alive

**Correct approach:**

```go
func processFiles(paths []string) error {
    for _, path := range paths {
        if err := processSingleFile(path); err != nil {
            return err
        }
    }
    return nil
}

func processSingleFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return err
    }
    defer f.Close()  // ✅ GOOD: Closes when THIS function returns

    // ... process file ...
    return nil
}
```

Or use immediate cleanup:

```go
func processFiles(paths []string) error {
    for _, path := range paths {
        f, err := os.Open(path)
        if err != nil {
            continue
        }

        // Process file
        err = processFile(f)
        f.Close()  // ✅ GOOD: Immediate cleanup

        if err != nil {
            return err
        }
    }
    return nil
}
```

## Real-World Context: Why Transaction Recreation?

### Cloud Spanner's Maximum Staleness Limit

Cloud Spanner read-only transactions have a **maximum staleness limit** (typically 1 hour). For long-running data migrations:

1. **Transaction created** at timestamp T₀
2. **Migration runs** for 90 minutes, iterating through millions of rows
3. **After 60 minutes**: Spanner returns `"exceeded the maximum timestamp staleness"` error
4. **Recovery needed**: Create fresh transaction and continue

### Is Recreation an Anti-Pattern?

⚠️ **Partially** - The recreation is necessary, but the implementation is naive:

**Problems:**
- No checkpointing: Restarts from beginning, potentially reprocessing rows
- No cursor: Can't resume from where it left off
- Duplicate processing: Relies on idempotency in the receiver

**Better alternatives:**
1. **Pagination with cursor**: `SELECT * FROM table WHERE id > @last_id ORDER BY id LIMIT 1000`
2. **Batch processing**: Process in smaller batches that complete within staleness window
3. **Checkpointing**: Store progress and resume from last processed row

## Common Pitfall: Adding `defer` at Reassignment Point

❌ **This doesn't work:**

```go
for {
    // ...
    if needToRecreate {
        rotx = r.paymentdb.ReadOnlyTransaction()
        defer rotx.Close()  // ❌ Won't execute until function returns!

        iter = rotx.Query(ctx, query)
        defer iter.Stop()   // ❌ Won't execute until function returns!

        continue  // Loop continues, defers don't execute
    }
}
```

**Why it fails:**
- `defer` executes when **function returns**, not when you hit the `defer` statement
- Each loop iteration adds MORE deferred calls that stack up
- All resources stay open until function ends
- If loop runs 100 times, you have 100 deferred calls and 99 leaked resources

## Testing for Resource Leaks

### Use Go's Race Detector and Leak Detector

```bash
# Run with race detector
go test -race ./...

# Use leak detector in tests
import "go.uber.org/goleak"

func TestMain(m *testing.M) {
    goleak.VerifyTestMain(m)
}
```

### Monitor Resource Usage

```go
import _ "net/http/pprof"

// In your main:
go func() {
    log.Println(http.ListenAndServe("localhost:6060", nil))
}()

// Then visit: http://localhost:6060/debug/pprof/
```

## Summary Table

| Pattern | When to Use | Risk Level | Example |
|---------|-------------|------------|---------|
| `defer` at function start | Single resource for entire function | ✅ Safe | Opening a file |
| `defer` in loop | Multiple short-lived resources | ❌ Dangerous | Processing 1000 files |
| `defer` before reassignment | Resource variable reassigned | ❌ Ineffective | Transaction recreation |
| Immediate cleanup + `defer` | Resource reassignment in loop | ✅ Safe | Our fixed pattern |
| Immediate cleanup only | Very tight loop, panic OK | ⚠️ Risky | High-performance code |

## Checklist for Resource Management

- [ ] Does my function create resources that need cleanup?
- [ ] Are these resources reassigned to the same variable?
- [ ] Am I using `defer` inside a loop?
- [ ] Do I have immediate cleanup before reassignment?
- [ ] Do I have a `defer` safety net for the final instance?
- [ ] Have I tested for leaks with `goleak` or profiling?

## References

- **Rule CC-2 (MUST)**: Tie goroutine lifetime to a `context.Context`; prevent leaks
- **Rule CC-3 (MUST)**: Protect shared state with `sync.Mutex`/`atomic`; no "probably safe" races
- Cloud Spanner: [Read-only transactions and timestamp bounds](https://cloud.google.com/spanner/docs/reads#read-only_transactions)
- Go Blog: [Defer, Panic, and Recover](https://go.dev/blog/defer-panic-and-recover)

---

**Key Takeaway**: When reassigning resource variables in loops, use **immediate cleanup before reassignment** + **`defer` as safety net for the final instance**. Never rely solely on `defer` for resources that are recreated.
