package project

import "errors"

var (
	ErrProjectNotFound    = errors.New("project not found")
	ErrProjectExists      = errors.New("project already exists")
	ErrStageNotFound      = errors.New("stage not found")
	ErrStageExists        = errors.New("stage already exists")
	ErrProjectHasFlags    = errors.New("project has associated flags")
	ErrStageHasFlags      = errors.New("stage has associated flags")
	ErrInvalidProjectKey  = errors.New("project key is required")
	ErrInvalidProjectName = errors.New("project name is required")
	ErrInvalidStageKey    = errors.New("stage key is required")
	ErrInvalidStageName   = errors.New("stage name is required")
	ErrProjectKeyMismatch = errors.New("stage project key mismatch")
)
