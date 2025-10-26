package flag

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu    sync.RWMutex
	flags map[string]FeatureFlag
	now   func() time.Time
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		flags: make(map[string]FeatureFlag),
		now:   func() time.Time { return time.Now().UTC() },
	}
}

func (r *InMemoryRepository) GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	compositeKey := makeCompositeKey(project, stage, key)
	flag, ok := r.flags[compositeKey]
	if !ok {
		return FeatureFlag{}, ErrFlagNotFound
	}

	return cloneFlag(flag), nil
}

func (r *InMemoryRepository) ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	flags := make([]FeatureFlag, 0)
	for _, f := range r.flags {
		if f.Project == project && f.Stage == stage {
			flags = append(flags, cloneFlag(f))
		}
	}

	return flags, nil
}

func (r *InMemoryRepository) UpsertFlag(ctx context.Context, flag FeatureFlag) error {
	if err := validateFlag(flag); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFlag, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	compositeKey := makeCompositeKey(flag.Project, flag.Stage, flag.Key)
	now := r.now()
	existing, exists := r.flags[compositeKey]
	if !exists {
		flag.CreatedAt = now
		flag.UpdatedAt = now
		r.flags[compositeKey] = cloneFlag(flag)
		return nil
	}

	if !flag.UpdatedAt.IsZero() && !flag.UpdatedAt.Equal(existing.UpdatedAt) {
		return ErrFlagConflict
	}

	flag.CreatedAt = existing.CreatedAt
	flag.UpdatedAt = now
	r.flags[compositeKey] = cloneFlag(flag)

	return nil
}

func (r *InMemoryRepository) DeleteFlag(ctx context.Context, project, stage, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	compositeKey := makeCompositeKey(project, stage, key)
	if _, exists := r.flags[compositeKey]; !exists {
		return ErrFlagNotFound
	}

	delete(r.flags, compositeKey)
	return nil
}

func makeCompositeKey(project, stage, key string) string {
	return fmt.Sprintf("%s:%s:%s", project, stage, key)
}

func cloneFlag(flag FeatureFlag) FeatureFlag {
	cpy := flag
	if len(flag.Variations) > 0 {
		cpy.Variations = make([]Variation, len(flag.Variations))
		copy(cpy.Variations, flag.Variations)
	}
	if len(flag.Rules) > 0 {
		cpy.Rules = make([]Rule, len(flag.Rules))
		for i, rule := range flag.Rules {
			cpy.Rules[i] = cloneRule(rule)
		}
	}
	return cpy
}

func validateFlag(flag FeatureFlag) error {
	if flag.Key == "" {
		return fmt.Errorf("key is required")
	}
	if len(flag.Variations) == 0 {
		return fmt.Errorf("at least one variation required")
	}

	keys := make(map[string]struct{}, len(flag.Variations))
	var defaultFound bool
	for _, variation := range flag.Variations {
		if variation.Key == "" {
			return fmt.Errorf("variation key is required")
		}
		if _, exists := keys[variation.Key]; exists {
			return fmt.Errorf("duplicate variation key %q", variation.Key)
		}
		keys[variation.Key] = struct{}{}

		if err := validateVariation(variation); err != nil {
			return err
		}
		if variation.Key == flag.DefaultKey {
			defaultFound = true
		}
	}

	if flag.DefaultKey == "" {
		return fmt.Errorf("default variation key is required")
	}

	if !defaultFound {
		return fmt.Errorf("default variation %q not found", flag.DefaultKey)
	}

	for _, rule := range flag.Rules {
		if err := validateRule(rule, keys); err != nil {
			return err
		}
	}

	return nil
}

func validateVariation(variation Variation) error {
	switch variation.Type {
	case BooleanVariation:
		if _, ok := variation.Value.(bool); !ok {
			return fmt.Errorf("variation %q expects boolean value", variation.Key)
		}
	case StringVariation:
		if _, ok := variation.Value.(string); !ok {
			return fmt.Errorf("variation %q expects string value", variation.Key)
		}
	case NumberVariation:
		if !isNumeric(variation.Value) {
			return fmt.Errorf("variation %q expects numeric value", variation.Key)
		}
	case ObjectVariation:
		if !isObjectType(variation.Value) {
			return fmt.Errorf("variation %q expects object value", variation.Key)
		}
	default:
		return fmt.Errorf("variation %q has unsupported type %q", variation.Key, variation.Type)
	}
	return nil
}

func isNumeric(value any) bool {
	switch value.(type) {
	case int, int8, int16, int32, int64,
		uint, uint8, uint16, uint32, uint64,
		float32, float64:
		return true
	default:
		return false
	}
}

func isObjectType(value any) bool {
	if value == nil {
		return false
	}
	if _, ok := value.([]byte); ok {
		return true
	}

	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Map, reflect.Struct:
		return true
	default:
		return false
	}
}

func cloneRule(rule Rule) Rule {
	cpy := rule
	if len(rule.Conditions) > 0 {
		cpy.Conditions = make([]Condition, len(rule.Conditions))
		copy(cpy.Conditions, rule.Conditions)
	}
	if rule.Rollout != nil {
		rolloutCopy := *rule.Rollout
		if len(rule.Rollout.Buckets) > 0 {
			rolloutCopy.Buckets = make([]RolloutBucket, len(rule.Rollout.Buckets))
			copy(rolloutCopy.Buckets, rule.Rollout.Buckets)
		}
		cpy.Rollout = &rolloutCopy
	}
	return cpy
}

func validateRule(rule Rule, variations map[string]struct{}) error {
	for _, condition := range rule.Conditions {
		if condition.Attribute == "" {
			return fmt.Errorf("rule %q condition missing attribute", rule.ID)
		}
		if err := validateCondition(condition); err != nil {
			return fmt.Errorf("rule %q: %w", rule.ID, err)
		}
	}

	if rule.Rollout != nil {
		if rule.VariationKey != "" {
			return fmt.Errorf("rule %q cannot define both variation key and rollout", rule.ID)
		}
		if err := validateRollout(rule.Rollout, variations); err != nil {
			return fmt.Errorf("rule %q rollout: %w", rule.ID, err)
		}
		return nil
	}

	if rule.VariationKey == "" {
		return fmt.Errorf("rule %q must specify variation key when rollout absent", rule.ID)
	}

	if _, ok := variations[rule.VariationKey]; !ok {
		return fmt.Errorf("rule %q references unknown variation %q", rule.ID, rule.VariationKey)
	}

	return nil
}

func validateCondition(condition Condition) error {
	switch condition.Operator {
	case MatcherExists:
		return nil
	case MatcherEquals, MatcherNotEquals:
		if condition.Value == nil {
			return fmt.Errorf("operator %q requires value", condition.Operator)
		}
		return nil
	case MatcherContains, MatcherStartsWith, MatcherEndsWith:
		if _, ok := condition.Value.(string); !ok {
			return fmt.Errorf("operator %q expects string value", condition.Operator)
		}
		return nil
	case MatcherGreater, MatcherLess:
		if !isNumeric(condition.Value) {
			return fmt.Errorf("operator %q expects numeric value", condition.Operator)
		}
		return nil
	case MatcherIn:
		rv := reflect.ValueOf(condition.Value)
		if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
			return fmt.Errorf("operator %q expects slice or array value", condition.Operator)
		}
		if rv.Len() == 0 {
			return fmt.Errorf("operator %q expects non-empty collection", condition.Operator)
		}
		return nil
	default:
		return fmt.Errorf("unknown operator %q", condition.Operator)
	}
}

func validateRollout(rollout *PercentageRollout, variations map[string]struct{}) error {
	if rollout.Attribute == "" {
		return fmt.Errorf("attribute is required")
	}
	if len(rollout.Buckets) == 0 {
		return fmt.Errorf("at least one rollout bucket required")
	}

	var total float64
	for _, bucket := range rollout.Buckets {
		if bucket.Weight <= 0 {
			return fmt.Errorf("bucket for variation %q must have positive weight", bucket.VariationKey)
		}
		if _, ok := variations[bucket.VariationKey]; !ok {
			return fmt.Errorf("bucket references unknown variation %q", bucket.VariationKey)
		}
		total += bucket.Weight
	}

	if math.Abs(total-100.0) > 0.0001 {
		return fmt.Errorf("rollout weights must total 100, got %f", total)
	}

	return nil
}
