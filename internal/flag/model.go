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
	Project     string
	Stage       string
	Key         string
	Name        string
	Description string
	Enabled     bool
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
	GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error)
	ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error)
	UpsertFlag(ctx context.Context, flag FeatureFlag) error
	DeleteFlag(ctx context.Context, project, stage, key string) error
}

type Evaluator interface {
	Evaluate(ctx context.Context, flag FeatureFlag, attrs EvaluationContext) (EvaluationResult, error)
}
