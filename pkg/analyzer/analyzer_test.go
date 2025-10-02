package analyzer_test

import (
	"testing"

	"github.com/ZZTmercari/spannerclosecheck/pkg/analyzer"
	"golang.org/x/tools/go/analysis/analysistest"
)

func Test(t *testing.T) {
	testdata := analysistest.TestData()
	analysistest.Run(t, testdata, analyzer.Analyzer, "a")
}
