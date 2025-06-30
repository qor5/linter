package testpkg

import (
	"context"

	"github.com/theplant/appkit/logtracing"
)

func TestFunction() {
	ctx := context.Background()

	// These should NOT trigger warnings (correct usage)
	ctx, _ = logtracing.StartSpan(ctx, "test1")     // variable reassignment
	ctx, span := logtracing.StartSpan(ctx, "test2") // new variable declaration
	_, span2 := logtracing.StartSpan(ctx, "test3")  // new variable declaration
	newCtx, _ := logtracing.StartSpan(ctx, "test4") // new variable declaration

	// These SHOULD trigger warnings (incorrect usage)
	logtracing.StartSpan(ctx, "test5") // want "github.com/theplant/appkit/logtracing.StartSpan must receive its return value"

	// Use variables to avoid unused variable warnings
	_ = span
	_ = span2
	_ = newCtx
}
