package useany

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestModernTypeLinter(t *testing.T) {
	// Create linter instance
	linter := &Linter{settings: Settings{}}

	// Build analyzers
	analyzers, err := linter.BuildAnalyzers()
	if err != nil {
		t.Fatalf("Failed to build analyzers: %v", err)
	}

	if len(analyzers) != 1 {
		t.Fatalf("Expected 1 analyzer, got %d", len(analyzers))
	}

	// Run the test using analysistest with go.mod support
	analysistest.Run(t, analysistest.TestData(), analyzers[0], "testdata/testpkg")
}
