package pkg1

import "github.com/pkg/errors"

func SomeFunction() error {
	return errors.New("some error")
}
