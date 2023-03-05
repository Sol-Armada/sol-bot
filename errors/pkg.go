package errors

import e "errors"

var (
	ErrMissingName     = e.New("Missing Name")
	ErrMissingStart    = e.New("Missing Start")
	ErrMissingDuration = e.New("Missing Duration")

	ErrStartWrongFormat = e.New("Start is in wrong format")
)
