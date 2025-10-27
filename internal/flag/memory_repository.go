package flag

import (
	"context"
	"fmt"
	"math"
	"reflect"
	"sort"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu     sync.RWMutex
	flags  map[int64]FeatureFlag // flags by ID
	nextID int64
	now    func() time.Time
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		flags:  make(map[int64]FeatureFlag),
		nextID: 1,
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// GetFlag gets the currently active flag valid at the current time
func (r *InMemoryRepository) GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error) {
	return r.GetFlagAt(ctx, project, stage, key, r.now())
}

// GetFlagByID gets a specific flag range by ID
func (r *InMemoryRepository) GetFlagByID(ctx context.Context, id int64) (FeatureFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	flag, ok := r.flags[id]
	if !ok {
		return FeatureFlag{}, ErrFlagNotFound
	}

	return cloneFlag(flag), nil
}

// GetFlagAt gets the active flag valid at a specific time
func (r *InMemoryRepository) GetFlagAt(ctx context.Context, project, stage, key string, at time.Time) (FeatureFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var found *FeatureFlag
	for _, flag := range r.flags {
		if flag.Project == project && flag.Stage == stage && flag.Key == key && flag.Active {
			if IsValidInRange(at, flag.ValidFrom, flag.ValidTo) {
				if found == nil || flag.ValidFrom.After(found.ValidFrom) {
					flagCopy := flag
					found = &flagCopy
				}
			}
		}
	}

	if found == nil {
		return FeatureFlag{}, ErrFlagNotFound
	}

	return cloneFlag(*found), nil
}

// GetFlagRanges gets all temporal ranges (active and inactive) for a flag
func (r *InMemoryRepository) GetFlagRanges(ctx context.Context, project, stage, key string) ([]FeatureFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	flags := make([]FeatureFlag, 0)
	for _, flag := range r.flags {
		if flag.Project == project && flag.Stage == stage && flag.Key == key {
			flags = append(flags, cloneFlag(flag))
		}
	}

	// Sort by ValidFrom descending (most recent first)
	for i := 0; i < len(flags)-1; i++ {
		for j := i + 1; j < len(flags); j++ {
			if flags[i].ValidFrom.Before(flags[j].ValidFrom) {
				flags[i], flags[j] = flags[j], flags[i]
			}
		}
	}

	return flags, nil
}

// ListFlags lists all flag ranges for a project/stage ordered by key and recency.
func (r *InMemoryRepository) ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	flags := make([]FeatureFlag, 0)
	for _, flag := range r.flags {
		if flag.Project == project && flag.Stage == stage {
			flags = append(flags, cloneFlag(flag))
		}
	}

	sort.SliceStable(flags, func(i, j int) bool {
		if flags[i].Key == flags[j].Key {
			return flags[i].ValidFrom.After(flags[j].ValidFrom)
		}
		return flags[i].Key < flags[j].Key
	})

	return flags, nil
}

// UpsertFlag creates or updates a flag range (validates non-overlapping ranges)
func (r *InMemoryRepository) UpsertFlag(ctx context.Context, flag FeatureFlag) error {
	if err := validateFlag(flag); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFlag, err)
	}

	// Validate temporal range
	if err := ValidateTemporalRange(flag.ValidFrom, flag.ValidTo); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTimeRange, err)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	// Check for overlapping ranges
	hasOverlap, err := r.checkOverlapLocked(flag.Project, flag.Stage, flag.Key, flag.ValidFrom, flag.ValidTo, flag.ID)
	if err != nil {
		return fmt.Errorf("failed to check overlap: %w", err)
	}
	if hasOverlap {
		return ErrRangeOverlap
	}

	now := r.now()

	// Update existing flag range
	if flag.ID > 0 {
		existing, exists := r.flags[flag.ID]
		if !exists {
			return ErrFlagNotFound
		}

		// Optimistic locking check
		if !flag.UpdatedAt.IsZero() && !flag.UpdatedAt.Equal(existing.UpdatedAt) {
			return ErrFlagConflict
		}

		flag.CreatedAt = existing.CreatedAt
		flag.UpdatedAt = now
		r.flags[flag.ID] = cloneFlag(flag)
		return nil
	}

	// Insert new flag range
	flag.ID = r.nextID
	r.nextID++
	flag.CreatedAt = now
	flag.UpdatedAt = now

	// Default to active if not specified
	if !flag.Active {
		flag.Active = true
	}

	r.flags[flag.ID] = cloneFlag(flag)
	return nil
}

// DeleteFlag deletes a specific flag range by ID
func (r *InMemoryRepository) DeleteFlag(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.flags[id]; !exists {
		return ErrFlagNotFound
	}

	delete(r.flags, id)
	return nil
}

// ActivateFlag activates a flag range (checks for overlapping active ranges)
func (r *InMemoryRepository) ActivateFlag(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	flag, exists := r.flags[id]
	if !exists {
		return ErrFlagNotFound
	}

	// Check if already active
	if flag.Active {
		return nil // Already active, nothing to do
	}

	// Check for overlapping active ranges
	hasOverlap, err := r.checkActiveOverlapLocked(flag.Project, flag.Stage, flag.Key, flag.ValidFrom, flag.ValidTo, id)
	if err != nil {
		return fmt.Errorf("failed to check active overlap: %w", err)
	}
	if hasOverlap {
		return ErrActiveRangeOverlap
	}

	// Activate the range
	flag.Active = true
	r.flags[id] = flag

	return nil
}

// DeactivateFlag deactivates a flag range
func (r *InMemoryRepository) DeactivateFlag(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	flag, exists := r.flags[id]
	if !exists {
		return ErrFlagNotFound
	}

	flag.Active = false
	r.flags[id] = flag

	return nil
}

// CheckOverlap checks if a time range overlaps with any existing ranges (active or inactive)
func (r *InMemoryRepository) CheckOverlap(ctx context.Context, project, stage, key string, validFrom time.Time, validTo *time.Time, excludeID int64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.checkOverlapLocked(project, stage, key, validFrom, validTo, excludeID)
}

// checkOverlapLocked checks overlap without acquiring lock (assumes caller holds lock)
func (r *InMemoryRepository) checkOverlapLocked(project, stage, key string, validFrom time.Time, validTo *time.Time, excludeID int64) (bool, error) {
	newRange := TimeRange{From: validFrom, To: validTo}

	for _, flag := range r.flags {
		if flag.ID == excludeID {
			continue
		}
		if flag.Project == project && flag.Stage == stage && flag.Key == key {
			existingRange := TimeRange{From: flag.ValidFrom, To: flag.ValidTo}
			if RangesOverlap(newRange, existingRange) {
				return true, nil
			}
		}
	}

	return false, nil
}

// checkActiveOverlapLocked checks if a time range overlaps with any active ranges
func (r *InMemoryRepository) checkActiveOverlapLocked(project, stage, key string, validFrom time.Time, validTo *time.Time, excludeID int64) (bool, error) {
	newRange := TimeRange{From: validFrom, To: validTo}

	for _, flag := range r.flags {
		if flag.ID == excludeID {
			continue
		}
		if flag.Project == project && flag.Stage == stage && flag.Key == key && flag.Active {
			existingRange := TimeRange{From: flag.ValidFrom, To: flag.ValidTo}
			if RangesOverlap(newRange, existingRange) {
				return true, nil
			}
		}
	}

	return false, nil
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
