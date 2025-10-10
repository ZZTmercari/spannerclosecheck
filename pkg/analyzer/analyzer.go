package analyzer

import (
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/buildssa"
)

const Doc = `check for unclosed Spanner transactions and statements

Checks for improperly closed Cloud Spanner transactions, statements, and row iterators.
Spanner resources should be closed to prevent memory leaks and connection issues.`

// Constants
const (
	methodNameClose  = "Close"
	methodNameStop   = "Stop"
	methodNameSingle = "Single"

	typeNameReadOnlyTransaction      = "ReadOnlyTransaction"
	typeNameBatchReadOnlyTransaction = "BatchReadOnlyTransaction"
	typeNameRowIterator              = "RowIterator"

	pathGoogleSpanner = "cloud.google.com/go/spanner"

	nolintSpanner = "nolint:spannerclosecheck"
	nolintAll     = "nolint:all"
	nolintPrefix  = "nolint"
)

// Analyzer is the main analyzer for spannerclosecheck
// TODO: Flag for Lenient Mode (skip some checks or skip some files)
var Analyzer = &analysis.Analyzer{
	Name:     "spannerclosecheck",
	Doc:      Doc,
	Run:      run,
	Requires: []*analysis.Analyzer{buildssa.Analyzer},
}

func run(pass *analysis.Pass) (interface{}, error) {
	return deferOnlyAnalyzer(pass)
}
