package analyzer

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
)

const Doc = `check for unclosed Spanner transactions and statements

Checks for improperly closed Cloud Spanner transactions, statements, and row iterators.
Spanner resources should be closed to prevent memory leaks and connection issues.`

// Analyzer is the main analyzer for spannerclosecheck
var Analyzer = &analysis.Analyzer{
	Name:     "spannerclosecheck",
	Doc:      Doc,
	Run:      run,
	Requires: []*analysis.Analyzer{buildssa.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	return deferOnlyAnalyzer(pass)
}
