package testpkg

import (
	"context"

	"github.com/theplant/appkit/logtracing"
)

func BadUsageExamples() {
	ctx := context.Background()

	// This should trigger a warning
	logtracing.StartSpan(ctx, "bad1") // want "github.com/theplant/appkit/logtracing.StartSpan must receive its return value"

	// This should also trigger a warning
	logtracing.StartSpan(ctx, "bad2") // want "github.com/theplant/appkit/logtracing.StartSpan must receive its return value"
}

func GoodUsageExamples() {
	ctx := context.Background()

	// These should NOT trigger warnings
	ctx, _ = logtracing.StartSpan(ctx, "good1")
	_, span := logtracing.StartSpan(ctx, "good2")

	_ = span // avoid unused variable warning
}
