package testpkg

import (
	"fmt"
	"strconv"
	"testdata/testpkg/pkg1"

	pkgerrors "github.com/pkg/errors"
)

// Good: Using aliased errors.New - no wrapping needed for error handling libraries
func goodAliasErrorsNew() error {
	return pkgerrors.New("something went wrong")
}

// Good: Using aliased errors.Errorf - no wrapping needed for error handling libraries
func goodAliasErrorsErrorf() error {
	return pkgerrors.Errorf("failed to process item %d", 42)
}

// Bad: Using fmt.Errorf instead of pkg/errors
func badAliasErrorCreation() error {
	return fmt.Errorf("should use pkg/errors") // want "error should use github.com/pkg/errors"
}

// Good: Internal project error with alias - no wrapping needed
func goodAliasInternalError() error {
	return pkg1.SomeFunction()
}

// Bad: Standard library error - should be wrapped
func badStandardLibraryError() error {
	_, err := strconv.Atoi("foo")
	return err // want "error should use github.com/pkg/errors"
}

// Good: Third-party error with alias properly wrapped
func goodAliasThirdPartyErrorWrapped() error {
	_, err := strconv.Atoi("foo")
	if err != nil {
		return pkgerrors.Wrap(err, "failed to do something")
	}
	return nil
}
