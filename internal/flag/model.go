package flag

import (
	"context"
	"time"
)

// --- Variation types ---

type VariationType string

const (
	BooleanVariation VariationType = "boolean"
	StringVariation  VariationType = "string"
	NumberVariation  VariationType = "number"
	ObjectVariation  VariationType = "object"
)

type Variation struct {
	Key         string        `json:"key"`
	Type        VariationType `json:"type"`
	Value       any           `json:"value"`
	Description string        `json:"description,omitempty"`
}

// --- FeatureFlag: identity only (no temporal or rule data) ---

type FeatureFlag struct {
	ID          int64       `json:"id"`
	Project     string      `json:"project"`
	Stage       string      `json:"stage"`
	Key         string      `json:"key"`
	Name        string      `json:"name"`
	Description string      `json:"description,omitempty"`
	Enabled     bool        `json:"enabled"`
	DefaultKey  string      `json:"defaultKey"`
	Variations  []Variation `json:"variations"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

// --- FlagRange: temporal validity window ---

type FlagRange struct {
	ID        int64      `json:"id"`
	FlagID    int64      `json:"flagId"`
	Active    bool       `json:"active"`
	ValidFrom time.Time  `json:"validFrom"`
	ValidTo   *time.Time `json:"validTo,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// --- RangeVersion: rules with draft/publish lifecycle ---

type VersionStatus string

const (
	VersionStatusDraft     VersionStatus = "draft"
	VersionStatusPublished VersionStatus = "published"
)

type RangeVersion struct {
	ID        int64         `json:"id"`
	RangeID   int64         `json:"rangeId"`
	Version   int           `json:"version"`
	Status    VersionStatus `json:"status"`
	Rules     []Rule        `json:"rules"`
	CreatedAt time.Time     `json:"createdAt"`
	UpdatedAt time.Time     `json:"updatedAt"`
}

// --- Evaluation ---

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
	ReasonNoActiveRange     = "NO_ACTIVE_RANGE"
	ReasonNoPublishedVersion = "NO_PUBLISHED_VERSION"
)

// --- Matcher operators ---

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

// --- Rules and conditions ---

type Condition struct {
	Attribute string          `json:"attribute"`
	Operator  MatcherOperator `json:"operator"`
	Value     any             `json:"value"`
}

type RolloutBucket struct {
	VariationKey string  `json:"variationKey"`
	Weight       float64 `json:"weight"`
}

type PercentageRollout struct {
	Attribute string          `json:"attribute"`
	Seed      string          `json:"seed"`
	Buckets   []RolloutBucket `json:"buckets"`
}

type Rule struct {
	ID           string             `json:"id"`
	Description  string             `json:"description,omitempty"`
	Conditions   []Condition        `json:"conditions,omitempty"`
	VariationKey string             `json:"variationKey,omitempty"`
	Rollout      *PercentageRollout `json:"rollout,omitempty"`
}

// --- Audit action constants ---

const (
	ActionCreate = "CREATE"
	ActionUpdate = "UPDATE"
	ActionDelete = "DELETE"
)

// --- Interfaces ---

type AuditService interface {
	LogAction(ctx context.Context, project, stage, flagKey, action, performedBy string, oldValue, newValue any) error
	GetAuditLogs(ctx context.Context, project, stage, flagKey string, limit int) ([]AuditLog, error)
}

type Repository interface {
	// Flag identity CRUD
	GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error)
	GetFlagByID(ctx context.Context, id int64) (FeatureFlag, error)
	ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error)
	CreateFlag(ctx context.Context, flag *FeatureFlag) error
	UpdateFlag(ctx context.Context, flag *FeatureFlag) error
	DeleteFlag(ctx context.Context, id int64) error

	// Range CRUD
	GetRange(ctx context.Context, id int64) (FlagRange, error)
	ListRanges(ctx context.Context, flagID int64) ([]FlagRange, error)
	CreateRange(ctx context.Context, r *FlagRange) error
	UpdateRange(ctx context.Context, r *FlagRange) error
	DeleteRange(ctx context.Context, id int64) error
	CheckOverlap(ctx context.Context, flagID int64, validFrom time.Time, validTo *time.Time, excludeID int64) (bool, error)

	// Version CRUD
	GetVersion(ctx context.Context, id int64) (RangeVersion, error)
	ListVersions(ctx context.Context, rangeID int64) ([]RangeVersion, error)
	CreateVersion(ctx context.Context, v *RangeVersion) error
	UpdateVersion(ctx context.Context, v *RangeVersion) error
	DeleteVersion(ctx context.Context, id int64) error
}

type Evaluator interface {
	Evaluate(ctx context.Context, flag FeatureFlag, v RangeVersion, attrs EvaluationContext) (EvaluationResult, error)
}
