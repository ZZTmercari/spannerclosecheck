package analyzer

import (
	"go/token"
	"go/types"
	"strings"

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
			break
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
	if fn == nil {
		return
	}

	// Skip generated files (e.g., .yo.go files)
	if isGeneratedFile(pass, fn.Pos()) {
		return
	}

	// Check all instructions for Spanner resource allocations
	for _, block := range fn.Blocks {
		for _, instr := range block.Instrs {
			// Check if this instruction produces a Spanner type value
			if val, ok := instr.(ssa.Value); ok {
				typeName := getSpannerType(val.Type(), spannerTypes)
				if typeName != "" {
					// Skip ReadOnlyTransaction from Single() - it auto-releases
					if typeName == "ReadOnlyTransaction" && isFromSingle(val) {
						continue
					}

					// Skip RowIterator that's returned from a function - caller is responsible
					if typeName == "RowIterator" && isReturnedFromFunction(fn, val) {
						continue
					}

					// Found a Spanner resource - check if it has a deferred Close/Stop
					if !hasDeferredClose(val) {
						// Get the position - for Extract, use the tuple call's position
						pos := val.Pos()
						if extract, ok := val.(*ssa.Extract); ok {
							if extract.Tuple != nil {
								pos = extract.Tuple.Pos()
							}
						}

						// Check for nolint directive
						if !hasNolintDirective(pass, pos) {
							pass.Reportf(pos, "%s.Close() must be deferred", typeName)
						}
					}
				}
			}
		}
	}
}

// hasDeferredClose checks if a value has a deferred Close() or Stop() method call
func hasDeferredClose(val ssa.Value) bool {
	if val.Referrers() == nil {
		return false
	}

	for _, ref := range *val.Referrers() {
		// Check if the reference is in a defer instruction
		if _, ok := ref.(*ssa.Defer); ok {
			// This value is used directly in a defer
			return true
		}

		// Check if the reference is a method call (Close/Stop) in a defer
		if call, ok := ref.(*ssa.Call); ok {
			if call.Common().Method != nil {
				methodName := call.Common().Method.Name()
				if methodName == "Close" || methodName == "Stop" {
					// Check if this call is in a defer by looking at its referrers
					if call.Referrers() != nil {
						for _, callRef := range *call.Referrers() {
							if _, ok := callRef.(*ssa.Defer); ok {
								return true
							}
						}
					}
				}
			}
		}
	}

	return false
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

// isFromSingle checks if a value comes from a Client.Single() call
func isFromSingle(val ssa.Value) bool {
	// Direct call check
	if call, ok := val.(*ssa.Call); ok {
		// Check method call (for interface-based calls)
		if call.Common().Method != nil {
			methodName := call.Common().Method.Name()
			if methodName == "Single" {
				return true
			}
		}
		// Check function value call (for concrete type calls)
		if call.Common().Value != nil {
			if call.Common().Value.Name() == "Single" {
				return true
			}
		}
	}
	return false
}

// isReturnedFromFunction checks if a value is returned from the function
func isReturnedFromFunction(fn *ssa.Function, val ssa.Value) bool {
	if val.Referrers() == nil {
		return false
	}

	for _, ref := range *val.Referrers() {
		// Check if the value is used in a Return instruction
		if ret, ok := ref.(*ssa.Return); ok {
			// Check if val is one of the return values
			for _, result := range ret.Results {
				if result == val {
					return true
				}
			}
		}
	}

	return false
}

// isGeneratedFile checks if a position is in a generated file
func isGeneratedFile(pass *analysis.Pass, pos token.Pos) bool {
	file := pass.Fset.File(pos)
	if file == nil {
		return false
	}

	filename := file.Name()

	// Check for common generated file patterns
	if strings.HasSuffix(filename, ".yo.go") {
		return true
	}
	if strings.HasSuffix(filename, ".pb.go") {
		return true
	}
	if strings.HasSuffix(filename, "_gen.go") {
		return true
	}
	if strings.Contains(filename, "generated") {
		return true
	}

	// Check for file-level nolint directive
	if hasFileLevelNolint(pass, pos) {
		return true
	}

	return false
}

// hasFileLevelNolint checks if there's a file-level nolint directive
func hasFileLevelNolint(pass *analysis.Pass, pos token.Pos) bool {
	file := pass.Fset.File(pos)
	if file == nil {
		return false
	}

	// Look through all comment groups in the file
	for _, f := range pass.Files {
		// Check if this is the right file
		if pass.Fset.File(f.Pos()) != file {
			continue
		}

		// Check all comment groups
		for _, cg := range f.Comments {
			// Only check comments near the top of the file (before line 10)
			commentLine := file.Line(cg.Pos())
			if commentLine > 10 {
				break
			}

			for _, c := range cg.List {
				text := c.Text
				// Check for nolint directives
				if strings.Contains(text, "nolint:spannerclosecheck") ||
					strings.Contains(text, "nolint:all") {
					return true
				}
			}
		}
	}

	return false
}

// hasNolintDirective checks if there's a nolint comment for this position
func hasNolintDirective(pass *analysis.Pass, pos token.Pos) bool {
	// Get the file and position
	file := pass.Fset.File(pos)
	if file == nil {
		return false
	}

	// Find the line containing this position
	line := file.Line(pos)

	// Look through all comment groups in the file
	for _, f := range pass.Files {
		// Check if this is the right file
		if pass.Fset.File(f.Pos()) != file {
			continue
		}

		// Check all comment groups
		for _, cg := range f.Comments {
			commentLine := file.Line(cg.Pos())

			// Check if comment is on the same line or the line before
			if commentLine == line || commentLine == line-1 {
				for _, c := range cg.List {
					text := c.Text
					// Check for nolint directives
					// Supports: //nolint:spannerclosecheck, //nolint:all, //nolint
					if strings.Contains(text, "nolint:spannerclosecheck") ||
						strings.Contains(text, "nolint:all") ||
						(strings.Contains(text, "nolint") && !strings.Contains(text, ":")) {
						return true
					}
				}
			}
		}
	}

	return false
}
