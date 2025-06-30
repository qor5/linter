package testpkg

import (
	"context"

	log "github.com/theplant/appkit/logtracing"
)

func TestAliasUsage() {
	ctx := context.Background()
	ctx, _ = log.StartSpan(ctx, "test1")
	ctx, span := log.StartSpan(ctx, "test2")
	_, span2 := log.StartSpan(ctx, "test3")
	newCtx, _ := log.StartSpan(ctx, "test4")

	log.StartSpan(ctx, "test5") // want "github.com/theplant/appkit/logtracing.StartSpan must receive its return value"

	_ = span
	_ = span2
	_ = newCtx
}
