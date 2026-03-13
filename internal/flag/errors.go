package flag

import "errors"

var (
	ErrFlagNotFound = errors.New("flag not found")
	ErrFlagConflict = errors.New("flag conflict (optimistic lock)")
	ErrInvalidFlag  = errors.New("invalid flag")

	ErrRangeNotFound      = errors.New("range not found")
	ErrRangeOverlap       = errors.New("range overlaps with existing range")
	ErrInvalidTimeRange   = errors.New("invalid time range")
	ErrNoPublishedVersion = errors.New("range has no published version")

	ErrVersionNotFound       = errors.New("version not found")
	ErrVersionNotDraft       = errors.New("version is not a draft")
	ErrDraftExists           = errors.New("a draft version already exists")
	ErrCannotDeletePublished = errors.New("cannot delete a published version")
)
