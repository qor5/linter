package testpkg

import (
	"fmt"

	. "strconv" // dot import for third-party package

	. "github.com/pkg/errors" // dot import for errors package
)

// Good: Using dot imported errors.New
func goodDotImportErrorsNew() error {
	return New("something went wrong")
}

// Good: Using dot imported errors.Errorf
func goodDotImportErrorsErrorf() error {
	return Errorf("failed to process item %d", 42)
}

// Bad: Using fmt.Errorf with dot import context
func badDotImportFmtErrorf() error {
	return fmt.Errorf("should use errors package") // want "error should use github.com/pkg/errors"
}

// Bad: Standard library function from dot import - should be wrapped
func badDotImportStandardLibrary() error {
	_, err := Atoi("foo")
	return err // want "error should use github.com/pkg/errors"
}
