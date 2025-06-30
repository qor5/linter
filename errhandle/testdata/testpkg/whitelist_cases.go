package testpkg

import (
	"encoding/json"
	"github.com/qor5/go-que"
	"github.com/qor5/go-que/pg"
)

func goodWhitelistJson() error {
	obj := map[string]any{
		"name": "test",
	}
	_, err := json.Marshal(obj)
	if err != nil {
		return err // this should be ignored
	}
	return nil
}

func goodWhitelistGoQue() error {
	_, err := que.NewWorker(que.WorkerOptions{})
	if err != nil {
		return err // this should be ignored
	}
	return nil
}

// pg is a subpackage of github.com/qor5/go-que, so it will also be whitelisted
func goodWhitelistGoQue2() error {
	_, err := pg.New(nil)
	if err != nil {
		return err // this should be ignored
	}
	return nil
}
