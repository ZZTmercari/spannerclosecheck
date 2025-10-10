# Understanding SSA (Static Single Assignment) in Go

## What is SSA?

SSA (Static Single Assignment) is an intermediate representation (IR) where:
- Each variable is assigned exactly once
- Variables are split into versions (e.g., `x`, `x₁`, `x₂`)
- Makes data flow analysis easier

## Basic SSA Structure

### Example 1: Simple Assignment
```go
func example() {
    x := 5
    y := x + 3
}
```

**SSA Representation:**
```
b0:
  t0 = 5              // *ssa.Const
  t1 = local x        // *ssa.Alloc (allocate local variable)
  *t1 = t0            // *ssa.Store (store 5 into x)
  t2 = *t1            // *ssa.UnOp (load from x)
  t3 = t2 + 3         // *ssa.BinOp
  t4 = local y        // *ssa.Alloc
  *t4 = t3            // *ssa.Store
  return
```

Key SSA instruction types:
- `*ssa.Alloc` - Allocates memory for a variable
- `*ssa.Store` - Stores a value into memory
- `*ssa.UnOp` - Unary operation (including loads: `*ptr`)
- `*ssa.Call` - Function/method call
- `*ssa.Defer` - Defer statement

## Tracking Spanner Resources

### Scenario 1: Direct Usage (Current Implementation Handles This ✓)

```go
func directUsage(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()
}
```

**SSA:**
```
b0:
  t0 = local client           // *ssa.Alloc
  t1 = *t0                    // *ssa.UnOp (load client)
  t2 = t1.ReadOnlyTransaction() // *ssa.Call - THIS IS WHAT WE DETECT
  t3 = local txn              // *ssa.Alloc
  *t3 = t2                    // *ssa.Store
  t4 = *t3                    // *ssa.UnOp (load for defer)
  t5 = defer t4.Close()       // *ssa.Defer
  return
```

**How current code tracks this:**
1. Detects `t2` (the Call) creates a ReadOnlyTransaction
2. Checks `t2.Referrers()` - finds it's stored in `*t3 = t2`
3. Checks `t4` (UnOp load) - finds it's used in Defer

### Scenario 2: Variable Reassignment (CURRENT GAP ⚠️)

```go
func reassignment(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    myTxn := txn  // Reassignment
    defer myTxn.Close()
}
```

**SSA:**
```
b0:
  t0 = *client
  t1 = t0.ReadOnlyTransaction() // *ssa.Call - WE DETECT THIS
  t2 = local txn
  *t2 = t1                      // *ssa.Store (txn = t1)
  t3 = local myTxn
  t4 = *t2                      // *ssa.UnOp (load txn)
  *t3 = t4                      // *ssa.Store (myTxn = txn) - PROBLEM: We lose track here!
  t5 = *t3                      // *ssa.UnOp (load myTxn)
  t6 = defer t5.Close()         // *ssa.Defer
  return
```

**Why it fails:**
- `t1.Referrers()` only includes `*t2 = t1`
- The defer uses `t5`, not `t1`
- We need to follow: `t1` → `*t2` → `t4` → `*t3` → `t5` → defer

### Scenario 3: Struct Storage (CURRENT GAP ⚠️)

```go
type Handler struct {
    txn *spanner.ReadOnlyTransaction
}

func structStorage(client *spanner.Client) {
    h := &Handler{
        txn: client.ReadOnlyTransaction(),
    }
    defer h.txn.Close()
}
```

**SSA:**
```
b0:
  t0 = *client
  t1 = t0.ReadOnlyTransaction()    // *ssa.Call - WE DETECT THIS
  t2 = new Handler                 // *ssa.Alloc (heap allocation)
  t3 = &t2.txn                     // *ssa.FieldAddr (address of field)
  *t3 = t1                         // *ssa.Store (store into field) - PROBLEM: We lose track!
  t4 = local h
  *t4 = t2                         // *ssa.Store
  t5 = *t4                         // *ssa.UnOp (load h)
  t6 = &t5.txn                     // *ssa.FieldAddr
  t7 = *t6                         // *ssa.UnOp (load field)
  t8 = defer t7.Close()            // *ssa.Defer
  return
```

**Why it fails:**
- `t1.Referrers()` includes `*t3 = t1` (Store into FieldAddr)
- But we don't follow through `FieldAddr` operations
- Need to track: `t1` → field store → field load → defer

### Scenario 4: Passed to Function (CURRENT GAP ⚠️)

```go
func closeHelper(txn *spanner.ReadOnlyTransaction) {
    defer txn.Close()
}

func passedToFunction(client *spanner.Client) {
    txn := client.ReadOnlyTransaction()
    closeHelper(txn)
}
```

**SSA:**
```
// In passedToFunction:
b0:
  t0 = *client
  t1 = t0.ReadOnlyTransaction()  // *ssa.Call - WE DETECT THIS
  t2 = local txn
  *t2 = t1                       // *ssa.Store
  t3 = *t2                       // *ssa.UnOp
  t4 = closeHelper(t3)           // *ssa.Call - PROBLEM: We don't check if callee defers it
  return

// In closeHelper (different function):
b0:
  t0 = parameter 0 (txn)         // *ssa.Parameter
  t1 = defer t0.Close()          // *ssa.Defer
  return
```

**Why it fails:**
- Defer is in a different function
- Need inter-procedural analysis to follow into `closeHelper`
- This is much harder - requires analyzing all callees

### Scenario 5: Return Value (HANDLED ✓)

```go
func returnIterator(client *spanner.Client) *spanner.RowIterator {
    txn := client.ReadOnlyTransaction()
    defer txn.Close()
    return txn.Query(ctx, stmt)
}
```

**SSA:**
```
b0:
  t0 = *client
  t1 = t0.ReadOnlyTransaction()
  t2 = defer t1.Close()
  t3 = t1.Query(...)             // *ssa.Call (creates RowIterator)
  return t3                      // *ssa.Return - WE CHECK THIS
```

**Current code handles this:**
- `isReturnedFromFunction()` checks if value is in Return instruction
- Skips the check because caller is responsible

## How to Fix the Gaps

### Fix 1: Follow Store/Load Chains

```go
// Enhanced hasDeferredClose that follows stores/loads
func hasDeferredClose(val ssa.Value) bool {
    visited := make(map[ssa.Value]bool)
    return hasDeferredCloseRecursive(val, visited, 0)
}

func hasDeferredCloseRecursive(val ssa.Value, visited map[ssa.Value]bool, depth int) bool {
    // Prevent infinite loops
    if depth > 10 || visited[val] {
        return false
    }
    visited[val] = true

    if val.Referrers() == nil {
        return false
    }

    for _, ref := range *val.Referrers() {
        switch r := ref.(type) {
        case *ssa.Defer:
            // Direct defer
            return true

        case *ssa.Store:
            // Value is stored somewhere: *addr = val
            // Now we need to find loads from that address
            addr := r.Addr
            if addr.Referrers() != nil {
                for _, addrRef := range *addr.Referrers() {
                    if load, ok := addrRef.(*ssa.UnOp); ok && load.Op == token.MUL {
                        // This is a load: *addr
                        if hasDeferredCloseRecursive(load, visited, depth+1) {
                            return true
                        }
                    }
                }
            }

        case *ssa.Call:
            // Method call on the value
            if r.Common().Method != nil {
                methodName := r.Common().Method.Name()
                if methodName == methodNameClose || methodName == methodNameStop {
                    // Check if the call itself is deferred
                    if r.Referrers() != nil {
                        for _, callRef := range *r.Referrers() {
                            if _, ok := callRef.(*ssa.Defer); ok {
                                return true
                            }
                        }
                    }
                }
            }

        case *ssa.FieldAddr:
            // Stored in a struct field
            // This gets complex - field address is used for store
            // We'd need to track field loads too
            // For now, might want to skip this or handle specially
        }
    }

    return false
}
```

### Fix 2: Handle Struct Fields

```go
func isStoredInStructField(val ssa.Value) bool {
    if val.Referrers() == nil {
        return false
    }

    for _, ref := range *val.Referrers() {
        // Check if stored via FieldAddr
        if store, ok := ref.(*ssa.Store); ok {
            if _, isFieldAddr := store.Addr.(*ssa.FieldAddr); isFieldAddr {
                return true
            }
        }
    }
    return false
}

func findFieldDeferClose(val ssa.Value, fieldStore *ssa.Store) bool {
    // fieldStore.Addr is a *ssa.FieldAddr
    fieldAddr := fieldStore.Addr.(*ssa.FieldAddr)

    // Find the struct that contains this field
    structVal := fieldAddr.X

    // Look for loads from this struct's field that lead to defer
    // This requires tracking through the entire function...
    // Very complex!
    return false
}
```

### Fix 3: Inter-procedural Analysis (Advanced)

```go
// This is HARD and expensive
func isClosedInCallee(val ssa.Value, fn *ssa.Function) bool {
    // Would need to:
    // 1. Find all Call instructions that use val as argument
    // 2. Find the actual function being called
    // 3. Analyze that function to see if it defers Close on the parameter
    // 4. Handle function pointers, interfaces (very hard)

    // Most static analyzers skip this level of analysis
    // Better to use nolint comments for these cases
    return false
}
```

## Practical Recommendations

1. **Implement Store/Load tracking** (Medium difficulty, high value)
   - Handles variable reassignment
   - Follow through `*ssa.Store` and `*ssa.UnOp` chains

2. **Struct field tracking** (Hard, medium value)
   - More complex due to aliasing
   - May have false positives/negatives

3. **Inter-procedural** (Very hard, low value)
   - Expensive and complex
   - Better to rely on nolint comments
   - Could add a heuristic: if passed to function named `close*` or `cleanup*`, assume it's handled

4. **Add escape analysis awareness** (Medium difficulty)
   - If resource escapes to heap and is stored long-term, might not need immediate defer
   - Could reduce false positives in some patterns

## Debugging SSA

To see SSA output for any Go code:

```bash
# Install ssadump
go install golang.org/x/tools/cmd/ssadump@latest

# View SSA for a file
ssadump -build=F /path/to/file.go
```

This shows you the exact SSA instructions for your code!
