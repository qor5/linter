package testpkg

import (
	stderr "errors"
	"fmt"
	"strconv"

	"testdata/testpkg/pkg1"

	"github.com/pkg/errors"
)

// Good: Using errors.New for new error creation
func goodNewError() error {
	return errors.New("something went wrong")
}

// Good: Using errors.Errorf for formatted error creation
func goodErrorf() error {
	return errors.Errorf("failed to process item %d", 42)
}

// Bad: Using fmt.Errorf instead of errors.Errorf
func badErrorCreation() error {
	return fmt.Errorf("should use errors.Errorf") // want "error should use github.com/pkg/errors"
}

// Good: Internal project error - no wrapping needed
func goodInternalError() error {
	return pkg1.SomeFunction()
}

func badErrorCreation2() (err error) {
	return fmt.Errorf("should use errors.Errorf") // want "error should use github.com/pkg/errors"
}

func newStdError() error {
	return stderr.New("something went wrong") // want "error should use github.com/pkg/errors"
}

func testFmtErrorf() error {
	return fmt.Errorf("this is an error") // want "error should use github.com/pkg/errors"
}

func testFmtErrorfWithAssignment() error {
	err := fmt.Errorf("this is an error")
	return err // want "error should use github.com/pkg/errors"
}

// Good: Direct return within package
func goodDirectReturnWithinPackage() error {
	return goodInternalError()
}

func goodDirectVarReturnWithinPackage() error {
	err := goodInternalError()
	return err
}

func badSameVarName() error {
	err := goodInternalError()
	if err != nil {
		return err
	}
	err = badErrorCreation()
	if err != nil {
		return err
	}
	return nil
}

func badSameVarName2() error {
	err := goodInternalError()
	if err != nil {
		return err
	}
	_, err = strconv.Atoi("foo")
	if err != nil {
		return err // want "error should use github.com/pkg/errors"
	}
	return nil
}

func badSameVarName3() error {
	err := goodInternalError()
	if err != nil {
		return err
	}
	_, err = strconv.Atoi("foo")
	if err != nil {
		return errors.Wrap(err, "failed to do something")
	}
	return nil
}

func badDirectReturn() (int64, error) {
	return strconv.ParseInt("foo", 10, 64) // want "error should use github.com/pkg/errors"
}
