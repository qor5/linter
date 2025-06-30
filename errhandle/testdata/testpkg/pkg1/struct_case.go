package pkg1

import "github.com/pkg/errors"

type Foo struct{}

func (p *Foo) Validate() (int, error) {
	return 0, errors.New("test validate")
}

func (p *Foo) getName() error {
	num, err := p.Validate()
	if err != nil {
		return err
	}
	_ = num
	return nil
}

type Bar struct{}

func (b Bar) Validate() (int, error) {
	return 0, errors.New("test validate")
}
