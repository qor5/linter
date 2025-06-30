package testpkg

import (
	stderrors "errors"
	"github.com/pkg/errors"
	"strconv"
	"testdata/testpkg/pkg1"
)

func goodDeferErrorHandler() (err error) {
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "failed to do something")
		}
	}()
	_, err = strconv.Atoi("foo")
	return err
}

func goodDeferErrorHandler2() error {
	var err error
	defer func() {
		if err != nil {
			err = errors.Wrap(err, "failed to do something")
		}
	}()
	_, err = strconv.Atoi("foo")
	return err
}

// We don't handle this case because it can cause stack overflow.
// Since this pattern is rarely used, we skip checking it.
// func goodDeferErrorHandler3() error {
// 	var err error
// 	defer func() {
// 		if err != nil {
// 			err = err
// 		}
// 	}()
// 	err = pkg1.SomeFunction()
// 	return err
// }

func badDeferErrorHandler() (err error) {
	defer func() {
		if err != nil {
			err = stderrors.New("failed to do something") // want "error in defer should use github.com/pkg/errors"
		}
	}()
	err = pkg1.SomeFunction()
	return err
}
