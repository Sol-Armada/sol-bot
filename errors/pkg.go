package errors

import e "errors"

var (
	ErrMissingName     = e.New("Missing Name")
	ErrMissingStart    = e.New("Missing Start")
	ErrMissingDuration = e.New("Missing Duration")
	ErrMissingId       = e.New("Missing Id")

	ErrStartWrongFormat = e.New("Start is in wrong format")
)
