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
	Key         string
	Name        string
	Description string
	Enabled     bool
	DefaultKey  string
	Variations  []Variation
	Rules       []TargetingRule
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type EvaluationContext map[string]any

type EvaluationResult struct {
	Variation Variation
	Reason    string
}

type Repository interface {
	GetFlag(ctx context.Context, key string) (FeatureFlag, error)
	ListFlags(ctx context.Context) ([]FeatureFlag, error)
	UpsertFlag(ctx context.Context, flag FeatureFlag) error
	DeleteFlag(ctx context.Context, key string) error
}

type Evaluator interface {
	Evaluate(ctx context.Context, flag FeatureFlag, attrs EvaluationContext) (EvaluationResult, error)
}

type TargetingRule interface {
	Evaluate(ctx context.Context, attrs EvaluationContext) (EvaluationResult, bool, error)
}
