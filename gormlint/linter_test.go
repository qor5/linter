package gormlint

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestGormRules(t *testing.T) {
	linter := &Linter{settings: Settings{}}
	analyzers, err := linter.BuildAnalyzers()
	if err != nil {
		t.Fatalf("Failed to build analyzers: %v", err)
	}
	if len(analyzers) != 1 {
		t.Fatalf("Expected 1 analyzer, got %d", len(analyzers))
	}
	analysistest.Run(t, analysistest.TestData(), analyzers[0], "testdata/testpkg")
}
