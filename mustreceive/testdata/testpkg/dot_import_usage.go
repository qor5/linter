package testpkg

import (
	"context"

	. "github.com/theplant/appkit/logtracing"
)

func TestDotImportUsage() {
	ctx := context.Background()
	ctx, _ = StartSpan(ctx, "test1")
	ctx, span := StartSpan(ctx, "test2")
	_, span2 := StartSpan(ctx, "test3")
	newCtx, _ := StartSpan(ctx, "test4")

	StartSpan(ctx, "test5") // want "github.com/theplant/appkit/logtracing.StartSpan must receive its return value"

	_ = span
	_ = span2
	_ = newCtx
}
