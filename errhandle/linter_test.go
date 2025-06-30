package errhandle

import (
	"testing"

	"golang.org/x/tools/go/analysis/analysistest"
)

func TestErrorHandleLinter(t *testing.T) {
	// Create test settings
	settings := Settings{
		ProjectPath: "testdata", // Example project path
		Whitelist: []string{
			"encoding/json", // Whitelist the "encoding/json" package from standard library
			"github.com/qor5/go-que",
			"github.com/qor5/go-bus",
			"github.com/qor5/go-bus/quex",
			"github.com/qor5/confx",
			"github.com/theplant/inject",
		},
	}

	// Create linter instance
	linter := &Linter{settings: settings}

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
