package flag

import (
	"context"
	"time"
)

type VariationType string

const (
	BooleanVariation VariationType = "boolean"
	StringVariation  VariationType = "string"
	NumberVariation  VariationType = "number"
	ObjectVariation  VariationType = "object"
)

type Variation struct {
	Key         string
	Type        VariationType
	Value       any
	Description string
}

type FeatureFlag struct {
	ID          int64
	Project     string
	Stage       string
	Key         string
	Name        string
	Description string
	Enabled     bool
	Active      bool
	ValidFrom   time.Time
	ValidTo     *time.Time
	DefaultKey  string
	Variations  []Variation
	Rules       []Rule
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type EvaluationContext map[string]any

type EvaluationResult struct {
	Variation Variation
	Reason    string
}

const (
	ReasonTargetMatch       = "TARGET_MATCH"
	ReasonPercentageRollout = "PERCENTAGE_ROLLOUT"
	ReasonDefault           = "DEFAULT"
	ReasonDisabled          = "DISABLED"
)

type MatcherOperator string

const (
	MatcherEquals     MatcherOperator = "equals"
	MatcherNotEquals  MatcherOperator = "not_equals"
	MatcherContains   MatcherOperator = "contains"
	MatcherStartsWith MatcherOperator = "starts_with"
	MatcherEndsWith   MatcherOperator = "ends_with"
	MatcherGreater    MatcherOperator = "greater_than"
	MatcherLess       MatcherOperator = "less_than"
	MatcherIn         MatcherOperator = "in"
	MatcherExists     MatcherOperator = "exists"
)

type Condition struct {
	Attribute string
	Operator  MatcherOperator
	Value     any
}

type RolloutBucket struct {
	VariationKey string
	Weight       float64
}

type PercentageRollout struct {
	Attribute string
	Seed      string
	Buckets   []RolloutBucket
}

type Rule struct {
	ID           string
	Description  string
	Conditions   []Condition
	VariationKey string
	Rollout      *PercentageRollout
}

type Repository interface {
	// GetFlag gets the currently active flag valid at the current time
	GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error)

	// GetFlagByID gets a specific flag range by ID
	GetFlagByID(ctx context.Context, id int64) (FeatureFlag, error)

	// GetFlagAt gets the active flag valid at a specific time
	GetFlagAt(ctx context.Context, project, stage, key string, at time.Time) (FeatureFlag, error)

	// GetFlagRanges gets all temporal ranges (active and inactive) for a flag
	GetFlagRanges(ctx context.Context, project, stage, key string) ([]FeatureFlag, error)

	// ListFlags lists all currently active flags
	ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error)

	// UpsertFlag creates or updates a flag range (validates non-overlapping ranges)
	UpsertFlag(ctx context.Context, flag FeatureFlag) error

	// DeleteFlag deletes a specific flag range by ID
	DeleteFlag(ctx context.Context, id int64) error

	// ActivateFlag activates a flag range (checks for overlapping active ranges)
	ActivateFlag(ctx context.Context, id int64) error

	// DeactivateFlag deactivates a flag range
	DeactivateFlag(ctx context.Context, id int64) error

	// CheckOverlap checks if a time range overlaps with any existing ranges
	CheckOverlap(ctx context.Context, project, stage, key string, validFrom time.Time, validTo *time.Time, excludeID int64) (bool, error)
}

type Evaluator interface {
	Evaluate(ctx context.Context, flag FeatureFlag, attrs EvaluationContext) (EvaluationResult, error)
}
