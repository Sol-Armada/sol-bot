package projects

import (
	"errors"
)

var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrTaskStatusNotFound = errors.New("task status not found")
	ErrTaskNotFound       = errors.New("task not found")
)

var projectStore projectBackend

func Setup() error {
	return setupProjectBackend()
}
