package main

import (
	"fmt"
	"os"

	"github.com/ZZTmercari/spannerclosecheck/pkg/analyzer"
	"golang.org/x/tools/go/analysis/singlechecker"
)

func main() {
	// Check for version flag before singlechecker takes over
	for _, arg := range os.Args {
		if arg == "-version" || arg == "--version" {
			fmt.Printf("spannerclosecheck %s\n", Version)
			fmt.Printf("Build date: %s\n", BuildDate)
			fmt.Printf("Git commit: %s\n", GitCommit)
			os.Exit(0)
		}
	}

	singlechecker.Main(analyzer.Analyzer)
}
