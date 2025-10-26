package flag

import "errors"

var (
	ErrFlagNotFound       = errors.New("flag not found")
	ErrFlagConflict       = errors.New("flag update conflict")
	ErrInvalidFlag        = errors.New("invalid feature flag")
	ErrRangeOverlap       = errors.New("flag range overlaps with existing range")
	ErrActiveRangeOverlap = errors.New("cannot activate: overlaps with active range")
	ErrInvalidTimeRange   = errors.New("invalid time range")
)
