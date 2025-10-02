package analyzer

import (
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
	"golang.org/x/tools/go/ssa"
)

func deferOnlyAnalyzer(pass *analysis.Pass) (interface{}, error) {
	pssa := pass.ResultOf[buildssa.Analyzer].(*buildssa.SSA)

	// Map to store Spanner types
	spannerTypes := make(map[*types.Named]string)

	// Find Spanner package and register types
	for _, pkg := range pssa.Pkg.Prog.AllPackages() {
		if pkg.Pkg.Path() == "cloud.google.com/go/spanner" {
			registerType(pkg, "ReadOnlyTransaction", spannerTypes)
			registerType(pkg, "BatchReadOnlyTransaction", spannerTypes)
			registerType(pkg, "RowIterator", spannerTypes)
		}
	}

	if len(spannerTypes) == 0 {
		return nil, nil
	}

	// Check each function
	for _, fn := range pssa.SrcFuncs {
		checkFunc(pass, fn, spannerTypes)
	}

	return nil, nil
}

func registerType(pkg *ssa.Package, name string, spannerTypes map[*types.Named]string) {
	obj := pkg.Pkg.Scope().Lookup(name)
	if obj != nil {
		if named, ok := obj.Type().(*types.Named); ok {
			spannerTypes[named] = name
		}
	}
}

func checkFunc(pass *analysis.Pass, fn *ssa.Function, spannerTypes map[*types.Named]string) {
	// First pass: collect all variables with deferred Close/Stop calls
	deferredVars := make(map[ssa.Value]bool)

	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if deferInstr, ok := instr.(*ssa.Defer); ok {
				call := deferInstr.Call
				if call.Method != nil {
					methodName := call.Method.Name()
					if methodName == "Close" || methodName == "Stop" {
						// Mark the value being closed as deferred
						if call.Value != nil {
							deferredVars[call.Value] = true
							// If it's a load, mark the address too
							if unop, ok := call.Value.(*ssa.UnOp); ok {
								deferredVars[unop.X] = true
							}
						}
					}
				}
			}
		}
	}

	// Second pass: find all Store instructions storing Spanner types
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			if store, ok := instr.(*ssa.Store); ok {
				// Check what's being stored
				typeName := ""

				// Case 1: Direct call result
				if call, ok := store.Val.(*ssa.Call); ok {
					typeName = getSpannerType(call.Type(), spannerTypes)
				}

				// Case 2: Extract from tuple (for funcs returning (val, error))
				if extract, ok := store.Val.(*ssa.Extract); ok {
					typeName = getSpannerType(extract.Type(), spannerTypes)
				}

				// Case 3: Direct Spanner value
				if typeName == "" {
					typeName = getSpannerType(store.Val.Type(), spannerTypes)
				}

				if typeName != "" && !deferredVars[store.Addr] {
					pass.Reportf(store.Pos(), "%s.Close() must be deferred", typeName)
				}
			}
		}
	}
}

func getSpannerType(t types.Type, spannerTypes map[*types.Named]string) string {
	// Strip pointer
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}

	// Check if it's a named Spanner type
	if named, ok := t.(*types.Named); ok {
		if name, ok := spannerTypes[named]; ok {
			return name
		}
	}

	return ""
}
