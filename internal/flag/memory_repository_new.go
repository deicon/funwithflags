package flag

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// InMemoryRepository implements the Repository interface with in-memory storage.
// It uses three separate maps for flags, ranges, and versions, with composite-key
// indexing for efficient flag lookups by project/stage/key.
type InMemoryRepository struct {
	mu       sync.RWMutex
	flagSeq  int64
	rangeSeq int64
	verSeq   int64
	flags    map[int64]FeatureFlag
	flagKeys map[string]int64 // "project/stage/key" → flag ID
	ranges   map[int64]FlagRange
	versions map[int64]RangeVersion
}

// NewInMemoryRepository creates a new empty InMemoryRepository.
func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		flags:    make(map[int64]FeatureFlag),
		flagKeys: make(map[string]int64),
		ranges:   make(map[int64]FlagRange),
		versions: make(map[int64]RangeVersion),
	}
}

func flagKeyComposite(project, stage, key string) string {
	return project + "/" + stage + "/" + key
}

// --- Clone helpers ---

func cloneFlag(f FeatureFlag) FeatureFlag {
	cpy := f
	if len(f.Variations) > 0 {
		cpy.Variations = make([]Variation, len(f.Variations))
		copy(cpy.Variations, f.Variations)
	}
	return cpy
}

func cloneRules(rules []Rule) []Rule {
	if rules == nil {
		return nil
	}
	out := make([]Rule, len(rules))
	for i, rule := range rules {
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
		out[i] = cpy
	}
	return out
}

func cloneVersion(v RangeVersion) RangeVersion {
	cpy := v
	cpy.Rules = cloneRules(v.Rules)
	return cpy
}

// --- Flag identity CRUD ---

func (r *InMemoryRepository) CreateFlag(ctx context.Context, flag FeatureFlag) (FeatureFlag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	composite := flagKeyComposite(flag.Project, flag.Stage, flag.Key)
	if _, exists := r.flagKeys[composite]; exists {
		return FeatureFlag{}, fmt.Errorf("%w: duplicate key %s", ErrInvalidFlag, composite)
	}

	now := time.Now().UTC()
	r.flagSeq++
	flag.ID = r.flagSeq
	flag.CreatedAt = now
	flag.UpdatedAt = now

	r.flags[flag.ID] = cloneFlag(flag)
	r.flagKeys[composite] = flag.ID

	return cloneFlag(flag), nil
}

func (r *InMemoryRepository) GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	composite := flagKeyComposite(project, stage, key)
	id, ok := r.flagKeys[composite]
	if !ok {
		return FeatureFlag{}, ErrFlagNotFound
	}

	f, ok := r.flags[id]
	if !ok {
		return FeatureFlag{}, ErrFlagNotFound
	}

	return cloneFlag(f), nil
}

func (r *InMemoryRepository) GetFlagByID(ctx context.Context, id int64) (FeatureFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.flags[id]
	if !ok {
		return FeatureFlag{}, ErrFlagNotFound
	}

	return cloneFlag(f), nil
}

func (r *InMemoryRepository) UpdateFlag(ctx context.Context, flag FeatureFlag) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.flags[flag.ID]
	if !ok {
		return ErrFlagNotFound
	}

	// Keep project/stage/key immutable
	flag.Project = existing.Project
	flag.Stage = existing.Stage
	flag.Key = existing.Key

	// Preserve CreatedAt, update UpdatedAt
	flag.CreatedAt = existing.CreatedAt
	flag.UpdatedAt = time.Now().UTC()

	r.flags[flag.ID] = cloneFlag(flag)
	return nil
}

func (r *InMemoryRepository) DeleteFlag(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, ok := r.flags[id]
	if !ok {
		return ErrFlagNotFound
	}

	// Cascade delete: find all ranges belonging to this flag and their versions
	for rID, rng := range r.ranges {
		if rng.FlagID == id {
			// Delete versions belonging to this range
			for vID, v := range r.versions {
				if v.RangeID == rID {
					delete(r.versions, vID)
				}
			}
			delete(r.ranges, rID)
		}
	}

	// Remove composite key mapping and flag
	composite := flagKeyComposite(f.Project, f.Stage, f.Key)
	delete(r.flagKeys, composite)
	delete(r.flags, id)

	return nil
}

func (r *InMemoryRepository) ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]FeatureFlag, 0)
	for _, f := range r.flags {
		if f.Project == project && f.Stage == stage {
			result = append(result, cloneFlag(f))
		}
	}

	return result, nil
}

// --- Range CRUD (stubs) ---

func (r *InMemoryRepository) CreateRange(ctx context.Context, rng FlagRange) (FlagRange, error) {
	return FlagRange{}, fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) GetRange(ctx context.Context, id int64) (FlagRange, error) {
	return FlagRange{}, fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) UpdateRange(ctx context.Context, rng FlagRange) error {
	return fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) DeleteRange(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) ListRanges(ctx context.Context, flagID int64) ([]FlagRange, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) ActivateRange(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) DeactivateRange(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) GetActiveRange(ctx context.Context, flagID int64, at time.Time) (FlagRange, error) {
	return FlagRange{}, fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) CheckRangeOverlap(ctx context.Context, flagID int64, validFrom time.Time, validTo *time.Time, excludeID int64) (bool, error) {
	return false, fmt.Errorf("not implemented")
}

// --- Version CRUD (stubs) ---

func (r *InMemoryRepository) CreateVersion(ctx context.Context, v RangeVersion) (RangeVersion, error) {
	return RangeVersion{}, fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) GetVersion(ctx context.Context, id int64) (RangeVersion, error) {
	return RangeVersion{}, fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) UpdateVersion(ctx context.Context, v RangeVersion) error {
	return fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) DeleteDraftVersion(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) ListVersions(ctx context.Context, rangeID int64) ([]RangeVersion, error) {
	return nil, fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) GetPublishedVersion(ctx context.Context, rangeID int64) (RangeVersion, error) {
	return RangeVersion{}, fmt.Errorf("not implemented")
}

func (r *InMemoryRepository) PublishVersion(ctx context.Context, id int64) error {
	return fmt.Errorf("not implemented")
}
