package testpkg

import (
	"github.com/pkg/errors"
	"strconv"
)

// Case 1: Error handling after variable reassignment
func errorReassignment() error {
	_, err := strconv.Atoi("foo")
	newErr := err // Variable reassignment
	return newErr // want "error should use github.com/pkg/errors"
}

// Case 3: Error handling in nested function calls
func nestedFunctionCalls() error {
	return functionA()
}

func functionA() error {
	return functionB()
}

func functionB() error {
	_, err := strconv.Atoi("foo")
	return err // want "error should use github.com/pkg/errors"
}

// Case 4: Error handling in conditional statements
func conditionalErrorReturn(condition bool) error {
	_, err1 := strconv.Atoi("foo")
	_, err2 := strconv.Atoi("bar")

	if condition {
		return err1 // want "error should use github.com/pkg/errors"
	} else {
		return err2 // want "error should use github.com/pkg/errors"
	}
}

func conditionalErrorReturnB(condition bool) error {
	err1 := errors.New("foo")
	err2 := errors.New("bar")

	if condition {
		return err1
	} else {
		return err2
	}
}

// Case 5: Error type assertion and conversion
func errorTypeAssertion() error {
	_, err := strconv.Atoi("foo")
	var myErr error = err // Type conversion
	return myErr          // want "error should use github.com/pkg/errors"
}

// Case 6: Error handling in switch statements
func switchErrorHandling() error {
	_, err := strconv.Atoi("foo")

	switch {
	case err != nil:
		return err // want "error should use github.com/pkg/errors"
	default:
		return nil
	}
}

// Case 7: Error handling in anonymous functions
func anonymousFunctionError() error {
	_, err := strconv.Atoi("foo")

	f := func() error {
		return err // want "error should use github.com/pkg/errors"
	}

	return f()
}
