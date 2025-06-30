package testpkg

import (
	stderr "errors"
	"testdata/testpkg/pkg1"

	"github.com/pkg/errors"
)

type person struct {
	Name   string
	Age    int
	animal *pkg1.Foo
}

func (p *person) validate() error {
	if _, err := p.animal.Validate(); err != nil {
		return err
	}
	b := pkg1.Bar{}
	if _, err := b.Validate(); err != nil {
		return err
	}
	return errors.New("test validate")
}

func (p *person) getName() (string, error) {
	if err := p.validate(); err != nil {
		return "", err
	}
	return p.Name, nil
}

func (p *person) stdValidate() error {
	err := stderr.New("test validate")
	return errors.WithStack(err)
}

func (p *person) declErrAtReturn() (err error) {
	if err := p.declErrAtReturn2(); err != nil {
		return err
	}
	return nil
}

func (p *person) declErrAtReturn2() error {
	return errors.New("foo error")
}
