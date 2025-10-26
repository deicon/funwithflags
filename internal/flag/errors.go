package flag

import "errors"

var (
	ErrFlagNotFound = errors.New("flag not found")
	ErrFlagConflict = errors.New("flag update conflict")
	ErrInvalidFlag  = errors.New("invalid feature flag")
)
