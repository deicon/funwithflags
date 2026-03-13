# Temporal Validity Redesign Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Split the monolithic FeatureFlag model into three entities — FeatureFlag (identity), FlagRange (time windows), RangeVersion (rules with draft/publish) — across backend, database, and frontend.

**Architecture:** Two-table split plus versioning. FeatureFlag holds identity (name, variations, enabled). FlagRange holds temporal windows (validFrom/validTo, active). RangeVersion holds rules with a draft/publish lifecycle. Evaluation joins all three: flag → active range at now → latest published version.

**Tech Stack:** Go 1.24, PostgreSQL 16 (pgx), React + TypeScript + Vite

**Spec:** `docs/superpowers/specs/2026-03-13-temporal-validity-redesign-design.md`

---

## Chunk 1: Domain Models, Errors, Temporal Logic

### Task 1: Define new model structs

**Files:**
- Modify: `internal/flag/model.go`

This task replaces the current monolithic `FeatureFlag` struct with three separate structs and updates the `Repository` and `Evaluator` interfaces.

- [ ] **Step 1: Write tests for new model JSON serialization**

Create `internal/flag/model_test.go` (or add to existing test file) with tests that verify the new structs serialize to the expected JSON shape:

```go
// internal/flag/model_test.go
package flag

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFeatureFlagJSON(t *testing.T) {
	f := FeatureFlag{
		ID:         1,
		Project:    "proj",
		Stage:      "prod",
		Key:        "my-flag",
		Name:       "My Flag",
		Enabled:    true,
		DefaultKey: "control",
		Variations: []Variation{{Key: "control", Type: BooleanVariation, Value: false}},
	}
	data, err := json.Marshal(f)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]interface{}
	json.Unmarshal(data, &out)

	// Must NOT have validFrom, validTo, active, rules fields
	for _, field := range []string{"validFrom", "validTo", "active", "rules"} {
		if _, ok := out[field]; ok {
			t.Errorf("FeatureFlag JSON should not contain %q", field)
		}
	}
	// Must have identity fields
	for _, field := range []string{"id", "project", "stage", "key", "name", "enabled", "defaultKey", "variations"} {
		if _, ok := out[field]; !ok {
			t.Errorf("FeatureFlag JSON missing %q", field)
		}
	}
}

func TestFlagRangeJSON(t *testing.T) {
	now := time.Now().UTC()
	r := FlagRange{
		ID:        1,
		FlagID:    10,
		Active:    true,
		ValidFrom: now,
	}
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]interface{}
	json.Unmarshal(data, &out)

	for _, field := range []string{"id", "flagId", "active", "validFrom"} {
		if _, ok := out[field]; !ok {
			t.Errorf("FlagRange JSON missing %q", field)
		}
	}
	// validTo omitted when nil
	if _, ok := out["validTo"]; ok {
		t.Error("FlagRange JSON should omit validTo when nil")
	}
}

func TestRangeVersionJSON(t *testing.T) {
	v := RangeVersion{
		ID:      1,
		RangeID: 5,
		Version: 1,
		Status:  VersionStatusDraft,
		Rules:   []Rule{},
	}
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]interface{}
	json.Unmarshal(data, &out)

	for _, field := range []string{"id", "rangeId", "version", "status", "rules"} {
		if _, ok := out[field]; !ok {
			t.Errorf("RangeVersion JSON missing %q", field)
		}
	}
	// publishedAt omitted when nil
	if _, ok := out["publishedAt"]; ok {
		t.Error("RangeVersion JSON should omit publishedAt when nil")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/flag/ -run TestFeatureFlagJSON -v`
Expected: FAIL (FlagRange and RangeVersion types don't exist yet)

- [ ] **Step 3: Rewrite model.go with new structs**

Replace the `FeatureFlag` struct and `Repository`/`Evaluator` interfaces in `internal/flag/model.go`. **IMPORTANT: Keep ALL existing types unchanged** — `Rule`, `Condition`, `MatcherOperator`, `PercentageRollout`, `RolloutBucket`, `Variation`, `VariationType` constants, and `EvaluationResult` must remain exactly as they are. Only change: remove `Active`, `ValidFrom`, `ValidTo`, `Rules` from `FeatureFlag`; add `FlagRange` and `RangeVersion` structs; update `Repository` and `Evaluator` interfaces.

```go
package flag

import (
	"context"
	"time"
)

// --- Value types (UNCHANGED — keep exact existing names and fields) ---

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

// --- Core entities ---

// FeatureFlag is the flag identity — stored once per (project, stage, key).
// Fields removed: Active, ValidFrom, ValidTo, Rules (moved to FlagRange/RangeVersion).
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

// FlagRange is a temporal window for a flag.
type FlagRange struct {
	ID        int64      `json:"id"`
	FlagID    int64      `json:"flagId"`
	Active    bool       `json:"active"`
	ValidFrom time.Time  `json:"validFrom"`
	ValidTo   *time.Time `json:"validTo,omitempty"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

const (
	VersionStatusDraft     = "draft"
	VersionStatusPublished = "published"
)

// RangeVersion holds rules for a range with draft/publish lifecycle.
type RangeVersion struct {
	ID          int64      `json:"id"`
	RangeID     int64      `json:"rangeId"`
	Version     int        `json:"version"`
	Status      string     `json:"status"`
	Rules       []Rule     `json:"rules"`
	PublishedAt *time.Time `json:"publishedAt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
}

// --- Evaluation (UNCHANGED) ---

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

// --- Matcher/Rule types (UNCHANGED — keep exact existing names and fields) ---

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

// --- Interfaces ---

type Repository interface {
	// Flag identity CRUD
	CreateFlag(ctx context.Context, flag FeatureFlag) (FeatureFlag, error)
	GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error)
	GetFlagByID(ctx context.Context, id int64) (FeatureFlag, error)
	UpdateFlag(ctx context.Context, flag FeatureFlag) error
	DeleteFlag(ctx context.Context, id int64) error
	ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error)

	// Range CRUD
	CreateRange(ctx context.Context, r FlagRange) (FlagRange, error)
	GetRange(ctx context.Context, id int64) (FlagRange, error)
	UpdateRange(ctx context.Context, r FlagRange) error
	DeleteRange(ctx context.Context, id int64) error
	ListRanges(ctx context.Context, flagID int64) ([]FlagRange, error)
	ActivateRange(ctx context.Context, id int64) error
	DeactivateRange(ctx context.Context, id int64) error
	GetActiveRange(ctx context.Context, flagID int64, at time.Time) (FlagRange, error)
	CheckRangeOverlap(ctx context.Context, flagID int64, validFrom time.Time, validTo *time.Time, excludeID int64) (bool, error)

	// Version CRUD
	CreateVersion(ctx context.Context, v RangeVersion) (RangeVersion, error)
	GetVersion(ctx context.Context, id int64) (RangeVersion, error)
	UpdateVersion(ctx context.Context, v RangeVersion) error
	DeleteDraftVersion(ctx context.Context, id int64) error
	ListVersions(ctx context.Context, rangeID int64) ([]RangeVersion, error)
	GetPublishedVersion(ctx context.Context, rangeID int64) (RangeVersion, error)
	PublishVersion(ctx context.Context, id int64) error
}

type Evaluator interface {
	Evaluate(ctx context.Context, flag FeatureFlag, v RangeVersion, attrs EvaluationContext) (EvaluationResult, error)
}

// --- Audit ---

const (
	ActionCreate = "create"
	ActionUpdate = "update"
	ActionDelete = "delete"
)

type AuditService interface {
	LogAction(ctx context.Context, project, stage, flagKey, action, performedBy string, oldValue, newValue any) error
	GetAuditLogs(ctx context.Context, project, stage, flagKey string, limit int) ([]AuditLog, error)
}

// AuditLog struct stays in audit.go — do not redefine here. Only update audit.go
// to add the RangeID field and change LogAction to accept any values.
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/flag/ -run "TestFeatureFlagJSON|TestFlagRangeJSON|TestRangeVersionJSON" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/flag/model.go internal/flag/model_test.go
git commit -m "refactor(flag): split FeatureFlag into identity, range, and version structs"
```

---

### Task 2: Update error types

**Files:**
- Modify: `internal/flag/errors.go`

- [ ] **Step 1: Update errors.go with new error types**

```go
package flag

import "errors"

var (
	// Flag identity errors
	ErrFlagNotFound  = errors.New("flag not found")
	ErrFlagConflict  = errors.New("flag conflict (optimistic lock)")
	ErrInvalidFlag   = errors.New("invalid flag")

	// Range errors
	ErrRangeNotFound      = errors.New("range not found")
	ErrRangeOverlap       = errors.New("range overlaps with existing range")
	ErrInvalidTimeRange   = errors.New("invalid time range")
	ErrNoPublishedVersion = errors.New("range has no published version")

	// Version errors
	ErrVersionNotFound   = errors.New("version not found")
	ErrVersionNotDraft   = errors.New("version is not a draft")
	ErrDraftExists       = errors.New("a draft version already exists")
	ErrCannotDeletePublished = errors.New("cannot delete a published version")
)
```

- [ ] **Step 2: Verify build**

Run: `go build ./internal/flag/...`
Expected: Build succeeds (may have errors from other files referencing old types — that's expected, we'll fix them in subsequent tasks)

- [ ] **Step 3: Commit**

```bash
git add internal/flag/errors.go
git commit -m "refactor(flag): update error types for three-entity model"
```

---

### Task 3: Update temporal validation

**Files:**
- Modify: `internal/flag/temporal.go`
- Modify: `internal/flag/temporal_test.go` (create if not exists)

- [ ] **Step 1: Write tests for temporal validation**

```go
// internal/flag/temporal_test.go
package flag

import (
	"testing"
	"time"
)

func TestValidateTemporalRange(t *testing.T) {
	now := time.Now().UTC()
	future := now.Add(24 * time.Hour)
	past := now.Add(-24 * time.Hour)

	tests := []struct {
		name      string
		validFrom time.Time
		validTo   *time.Time
		wantErr   bool
	}{
		{"valid with no end", now, nil, false},
		{"valid with end", now, &future, false},
		{"zero from", time.Time{}, nil, true},
		{"end before start", now, &past, true},
		{"end equals start", now, &now, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTemporalRange(tt.validFrom, tt.validTo)
			if (err != nil) != tt.wantErr {
				t.Errorf("got err=%v, wantErr=%v", err, tt.wantErr)
			}
		})
	}
}

func TestRangesOverlap(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	t4 := time.Date(2026, 4, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		a, b    TimeRange
		overlap bool
	}{
		{"no overlap", TimeRange{t1, &t2}, TimeRange{t3, &t4}, false},
		{"adjacent no overlap", TimeRange{t1, &t2}, TimeRange{t2, &t3}, false},
		{"overlap", TimeRange{t1, &t3}, TimeRange{t2, &t4}, true},
		{"contained", TimeRange{t1, &t4}, TimeRange{t2, &t3}, true},
		{"unbounded a", TimeRange{t1, nil}, TimeRange{t2, &t3}, true},
		{"unbounded both", TimeRange{t1, nil}, TimeRange{t2, nil}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RangesOverlap(tt.a, tt.b)
			if got != tt.overlap {
				t.Errorf("got %v, want %v", got, tt.overlap)
			}
		})
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/flag/ -run "TestValidateTemporalRange|TestRangesOverlap" -v`
Expected: PASS (temporal.go is unchanged, these tests validate existing behavior)

- [ ] **Step 3: Commit**

```bash
git add internal/flag/temporal_test.go
git commit -m "test(flag): add temporal validation tests"
```

---

## Chunk 2: Memory Repository

### Task 4: Memory repository — flag identity CRUD

**Files:**
- Rewrite: `internal/flag/memory_repository.go`
- Rewrite: `internal/flag/memory_repository_test.go`

The memory repository is the foundation for testing. It needs to implement the full new `Repository` interface. We'll build it incrementally — flag identity first, then ranges, then versions.

- [ ] **Step 1: Write flag identity tests**

```go
// internal/flag/memory_repository_test.go
package flag

import (
	"context"
	"testing"
)

func newTestRepo() *InMemoryRepository {
	return NewInMemoryRepository()
}

func TestMemRepo_CreateAndGetFlag(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	f := FeatureFlag{
		Project:    "proj",
		Stage:      "prod",
		Key:        "flag-1",
		Name:       "Flag 1",
		DefaultKey: "control",
		Variations: []Variation{{Key: "control", Type: BooleanVariation, Value: false}},
	}
	created, err := repo.CreateFlag(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Error("expected non-zero ID")
	}

	got, err := repo.GetFlag(ctx, "proj", "prod", "flag-1")
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != created.ID {
		t.Errorf("got ID %d, want %d", got.ID, created.ID)
	}
	if got.Name != "Flag 1" {
		t.Errorf("got Name %q, want %q", got.Name, "Flag 1")
	}
}

func TestMemRepo_CreateFlag_DuplicateKey(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	f := FeatureFlag{Project: "p", Stage: "s", Key: "k", Name: "n", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}}}
	_, err := repo.CreateFlag(ctx, f)
	if err != nil {
		t.Fatal(err)
	}
	_, err = repo.CreateFlag(ctx, f)
	if err == nil {
		t.Error("expected error on duplicate key")
	}
}

func TestMemRepo_GetFlagByID(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	f := FeatureFlag{Project: "p", Stage: "s", Key: "k", Name: "n", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}}}
	created, _ := repo.CreateFlag(ctx, f)

	got, err := repo.GetFlagByID(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Key != "k" {
		t.Errorf("got Key %q, want %q", got.Key, "k")
	}
}

func TestMemRepo_UpdateFlag(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	f := FeatureFlag{Project: "p", Stage: "s", Key: "k", Name: "old", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}}}
	created, _ := repo.CreateFlag(ctx, f)

	created.Name = "new"
	err := repo.UpdateFlag(ctx, created)
	if err != nil {
		t.Fatal(err)
	}

	got, _ := repo.GetFlagByID(ctx, created.ID)
	if got.Name != "new" {
		t.Errorf("got Name %q, want %q", got.Name, "new")
	}
}

func TestMemRepo_DeleteFlag(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	f := FeatureFlag{Project: "p", Stage: "s", Key: "k", Name: "n", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}}}
	created, _ := repo.CreateFlag(ctx, f)

	err := repo.DeleteFlag(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repo.GetFlagByID(ctx, created.ID)
	if err == nil {
		t.Error("expected ErrFlagNotFound")
	}
}

func TestMemRepo_ListFlags(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	for _, key := range []string{"a", "b", "c"} {
		repo.CreateFlag(ctx, FeatureFlag{Project: "p", Stage: "s", Key: key, Name: key, DefaultKey: "d",
			Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}}})
	}
	// Different project
	repo.CreateFlag(ctx, FeatureFlag{Project: "other", Stage: "s", Key: "x", Name: "x", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}}})

	flags, err := repo.ListFlags(ctx, "p", "s")
	if err != nil {
		t.Fatal(err)
	}
	if len(flags) != 3 {
		t.Errorf("got %d flags, want 3", len(flags))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/flag/ -run "TestMemRepo_Create|TestMemRepo_Get|TestMemRepo_Update|TestMemRepo_Delete|TestMemRepo_List" -v`
Expected: FAIL (InMemoryRepository doesn't implement new interface)

- [ ] **Step 3: Implement memory repository — flag identity methods**

Rewrite `internal/flag/memory_repository.go` with the new struct and flag identity methods. Start with just the struct definition and flag CRUD — range and version methods will be stubs returning errors for now:

```go
package flag

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu       sync.RWMutex
	flagSeq  int64
	rangeSeq int64
	verSeq   int64

	flags    map[int64]FeatureFlag           // by flag ID
	flagKeys map[string]int64                // "project/stage/key" → flag ID
	ranges   map[int64]FlagRange             // by range ID
	versions map[int64]RangeVersion           // by version ID
}

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

func (r *InMemoryRepository) CreateFlag(ctx context.Context, f FeatureFlag) (FeatureFlag, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	ck := flagKeyComposite(f.Project, f.Stage, f.Key)
	if _, exists := r.flagKeys[ck]; exists {
		return FeatureFlag{}, fmt.Errorf("flag %s already exists: %w", ck, ErrInvalidFlag)
	}

	r.flagSeq++
	f.ID = r.flagSeq
	now := time.Now().UTC()
	f.CreatedAt = now
	f.UpdatedAt = now

	r.flags[f.ID] = f
	r.flagKeys[ck] = f.ID
	return f, nil
}

func (r *InMemoryRepository) GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	ck := flagKeyComposite(project, stage, key)
	id, ok := r.flagKeys[ck]
	if !ok {
		return FeatureFlag{}, ErrFlagNotFound
	}
	return r.flags[id], nil
}

func (r *InMemoryRepository) GetFlagByID(ctx context.Context, id int64) (FeatureFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	f, ok := r.flags[id]
	if !ok {
		return FeatureFlag{}, ErrFlagNotFound
	}
	return f, nil
}

func (r *InMemoryRepository) UpdateFlag(ctx context.Context, f FeatureFlag) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.flags[f.ID]
	if !ok {
		return ErrFlagNotFound
	}

	f.CreatedAt = existing.CreatedAt
	f.UpdatedAt = time.Now().UTC()
	// Keep project/stage/key immutable
	f.Project = existing.Project
	f.Stage = existing.Stage
	f.Key = existing.Key

	r.flags[f.ID] = f
	return nil
}

func (r *InMemoryRepository) DeleteFlag(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	f, ok := r.flags[id]
	if !ok {
		return ErrFlagNotFound
	}

	ck := flagKeyComposite(f.Project, f.Stage, f.Key)
	delete(r.flags, id)
	delete(r.flagKeys, ck)

	// Cascade: delete ranges and their versions
	for rid, rng := range r.ranges {
		if rng.FlagID == id {
			for vid, ver := range r.versions {
				if ver.RangeID == rid {
					delete(r.versions, vid)
				}
			}
			delete(r.ranges, rid)
		}
	}
	return nil
}

func (r *InMemoryRepository) ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []FeatureFlag
	for _, f := range r.flags {
		if f.Project == project && f.Stage == stage {
			result = append(result, f)
		}
	}
	if result == nil {
		return []FeatureFlag{}, nil
	}
	return result, nil
}
```

Add stub methods for range and version operations that return `fmt.Errorf("not implemented")` so the file compiles against the Repository interface:

```go
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
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/flag/ -run "TestMemRepo_" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/flag/memory_repository.go internal/flag/memory_repository_test.go
git commit -m "feat(flag): implement memory repository flag identity CRUD"
```

---

### Task 5: Memory repository — range CRUD

**Files:**
- Modify: `internal/flag/memory_repository.go` (replace stubs)
- Modify: `internal/flag/memory_repository_test.go` (add range tests)

- [ ] **Step 1: Write range tests**

Add to `memory_repository_test.go`:

```go
func createTestFlagAndRange(t *testing.T, repo *InMemoryRepository) (FeatureFlag, FlagRange) {
	t.Helper()
	ctx := context.Background()
	f, err := repo.CreateFlag(ctx, FeatureFlag{
		Project: "p", Stage: "s", Key: "k", Name: "n", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}},
	})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	rng, err := repo.CreateRange(ctx, FlagRange{
		FlagID: f.ID, ValidFrom: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	return f, rng
}

func TestMemRepo_CreateAndGetRange(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	f, _ := repo.CreateFlag(ctx, FeatureFlag{
		Project: "p", Stage: "s", Key: "k", Name: "n", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}},
	})

	now := time.Now().UTC()
	rng, err := repo.CreateRange(ctx, FlagRange{FlagID: f.ID, ValidFrom: now})
	if err != nil {
		t.Fatal(err)
	}
	if rng.ID == 0 {
		t.Error("expected non-zero range ID")
	}

	got, err := repo.GetRange(ctx, rng.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.FlagID != f.ID {
		t.Errorf("got FlagID %d, want %d", got.FlagID, f.ID)
	}
}

func TestMemRepo_RangeOverlap(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	f, _ := repo.CreateFlag(ctx, FeatureFlag{
		Project: "p", Stage: "s", Key: "k", Name: "n", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}},
	})

	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	_, err := repo.CreateRange(ctx, FlagRange{FlagID: f.ID, ValidFrom: t1, ValidTo: &t2})
	if err != nil {
		t.Fatal(err)
	}

	// Non-overlapping: should succeed
	_, err = repo.CreateRange(ctx, FlagRange{FlagID: f.ID, ValidFrom: t2, ValidTo: &t3})
	if err != nil {
		t.Errorf("expected no error for adjacent range, got %v", err)
	}

	// Overlapping: should fail
	mid := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	_, err = repo.CreateRange(ctx, FlagRange{FlagID: f.ID, ValidFrom: mid, ValidTo: &t3})
	if err == nil {
		t.Error("expected overlap error")
	}
}

func TestMemRepo_GetActiveRange(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	f, _ := repo.CreateFlag(ctx, FeatureFlag{
		Project: "p", Stage: "s", Key: "k", Name: "n", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}},
	})

	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	rng, _ := repo.CreateRange(ctx, FlagRange{FlagID: f.ID, ValidFrom: t1, ValidTo: &t2})

	// Range starts inactive — should not be found
	at := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	_, err := repo.GetActiveRange(ctx, f.ID, at)
	if err == nil {
		t.Error("expected ErrRangeNotFound for inactive range")
	}

	// Activate explicitly
	repo.ActivateRange(ctx, rng.ID)

	got, err := repo.GetActiveRange(ctx, f.ID, at)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID == 0 {
		t.Error("expected to find active range")
	}

	// Outside range
	outside := time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)
	_, err = repo.GetActiveRange(ctx, f.ID, outside)
	if err == nil {
		t.Error("expected ErrRangeNotFound for time outside range")
	}
}

func TestMemRepo_ListRanges(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	f, _ := repo.CreateFlag(ctx, FeatureFlag{
		Project: "p", Stage: "s", Key: "k", Name: "n", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}},
	})

	t1 := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	repo.CreateRange(ctx, FlagRange{FlagID: f.ID, ValidFrom: t1, ValidTo: &t2})
	repo.CreateRange(ctx, FlagRange{FlagID: f.ID, ValidFrom: t2, ValidTo: &t3})

	ranges, err := repo.ListRanges(ctx, f.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(ranges) != 2 {
		t.Errorf("got %d ranges, want 2", len(ranges))
	}
}

// NOTE: TestMemRepo_DeleteRange_Cascades is in Task 6 (requires version methods)
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/flag/ -run "TestMemRepo_.*Range" -v`
Expected: FAIL (range methods are stubs)

- [ ] **Step 3: Implement range CRUD methods**

Replace the range stub methods in `memory_repository.go`:

```go
func (r *InMemoryRepository) CreateRange(ctx context.Context, rng FlagRange) (FlagRange, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.flags[rng.FlagID]; !ok {
		return FlagRange{}, ErrFlagNotFound
	}

	if err := ValidateTemporalRange(rng.ValidFrom, rng.ValidTo); err != nil {
		return FlagRange{}, fmt.Errorf("%w: %v", ErrInvalidTimeRange, err)
	}

	// Check overlap
	if r.checkOverlapLocked(rng.FlagID, rng.ValidFrom, rng.ValidTo, 0) {
		return FlagRange{}, ErrRangeOverlap
	}

	r.rangeSeq++
	rng.ID = r.rangeSeq
	now := time.Now().UTC()
	rng.CreatedAt = now
	rng.UpdatedAt = now
	// Ranges start inactive — must publish a version then activate explicitly

	r.ranges[rng.ID] = rng
	return rng, nil
}

// Note: CreateRange does NOT default Active=true. Ranges start inactive.
// They must have a published version before ActivateRange can be called.

func (r *InMemoryRepository) GetRange(ctx context.Context, id int64) (FlagRange, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	rng, ok := r.ranges[id]
	if !ok {
		return FlagRange{}, ErrRangeNotFound
	}
	return rng, nil
}

func (r *InMemoryRepository) UpdateRange(ctx context.Context, rng FlagRange) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.ranges[rng.ID]
	if !ok {
		return ErrRangeNotFound
	}

	if err := ValidateTemporalRange(rng.ValidFrom, rng.ValidTo); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTimeRange, err)
	}

	if r.checkOverlapLocked(existing.FlagID, rng.ValidFrom, rng.ValidTo, rng.ID) {
		return ErrRangeOverlap
	}

	rng.FlagID = existing.FlagID
	rng.CreatedAt = existing.CreatedAt
	rng.Active = existing.Active
	rng.UpdatedAt = time.Now().UTC()

	r.ranges[rng.ID] = rng
	return nil
}

func (r *InMemoryRepository) DeleteRange(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.ranges[id]; !ok {
		return ErrRangeNotFound
	}

	// Cascade delete versions
	for vid, ver := range r.versions {
		if ver.RangeID == id {
			delete(r.versions, vid)
		}
	}
	delete(r.ranges, id)
	return nil
}

func (r *InMemoryRepository) ListRanges(ctx context.Context, flagID int64) ([]FlagRange, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []FlagRange
	for _, rng := range r.ranges {
		if rng.FlagID == flagID {
			result = append(result, rng)
		}
	}
	if result == nil {
		return []FlagRange{}, nil
	}
	return result, nil
}

func (r *InMemoryRepository) ActivateRange(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rng, ok := r.ranges[id]
	if !ok {
		return ErrRangeNotFound
	}
	rng.Active = true
	rng.UpdatedAt = time.Now().UTC()
	r.ranges[id] = rng
	return nil
}

func (r *InMemoryRepository) DeactivateRange(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	rng, ok := r.ranges[id]
	if !ok {
		return ErrRangeNotFound
	}
	rng.Active = false
	rng.UpdatedAt = time.Now().UTC()
	r.ranges[id] = rng
	return nil
}

func (r *InMemoryRepository) GetActiveRange(ctx context.Context, flagID int64, at time.Time) (FlagRange, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, rng := range r.ranges {
		if rng.FlagID == flagID && rng.Active && IsValidInRange(at, rng.ValidFrom, rng.ValidTo) {
			return rng, nil
		}
	}
	return FlagRange{}, ErrRangeNotFound
}

func (r *InMemoryRepository) CheckRangeOverlap(ctx context.Context, flagID int64, validFrom time.Time, validTo *time.Time, excludeID int64) (bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.checkOverlapLocked(flagID, validFrom, validTo, excludeID), nil
}

func (r *InMemoryRepository) checkOverlapLocked(flagID int64, validFrom time.Time, validTo *time.Time, excludeID int64) bool {
	newRange := TimeRange{From: validFrom, To: validTo}
	for _, rng := range r.ranges {
		if rng.FlagID == flagID && rng.ID != excludeID {
			existing := TimeRange{From: rng.ValidFrom, To: rng.ValidTo}
			if RangesOverlap(newRange, existing) {
				return true
			}
		}
	}
	return false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/flag/ -run "TestMemRepo_" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/flag/memory_repository.go internal/flag/memory_repository_test.go
git commit -m "feat(flag): implement memory repository range CRUD"
```

---

### Task 6: Memory repository — version CRUD

**Files:**
- Modify: `internal/flag/memory_repository.go` (replace version stubs)
- Modify: `internal/flag/memory_repository_test.go` (add version tests)

- [ ] **Step 1: Write version tests**

Add to `memory_repository_test.go`:

```go
func TestMemRepo_DeleteRange_Cascades(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()

	f, rng := createTestFlagAndRange(t, repo)
	_ = f

	// Create a version on the range
	_, err := repo.CreateVersion(ctx, RangeVersion{RangeID: rng.ID, Rules: []Rule{}})
	if err != nil {
		t.Fatal(err)
	}

	err = repo.DeleteRange(ctx, rng.ID)
	if err != nil {
		t.Fatal(err)
	}

	// Range gone
	_, err = repo.GetRange(ctx, rng.ID)
	if err == nil {
		t.Error("expected range to be deleted")
	}

	// Versions gone
	versions, err := repo.ListVersions(ctx, rng.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 0 {
		t.Errorf("expected 0 versions after cascade delete, got %d", len(versions))
	}
}

func TestMemRepo_CreateAndGetVersion(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()
	_, rng := createTestFlagAndRange(t, repo)

	v, err := repo.CreateVersion(ctx, RangeVersion{
		RangeID: rng.ID,
		Rules:   []Rule{{ID: "r1"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if v.Version != 1 {
		t.Errorf("got version %d, want 1", v.Version)
	}
	if v.Status != VersionStatusDraft {
		t.Errorf("got status %q, want %q", v.Status, VersionStatusDraft)
	}

	got, err := repo.GetVersion(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Rules) != 1 {
		t.Errorf("got %d rules, want 1", len(got.Rules))
	}
}

func TestMemRepo_CreateVersion_OnlyOneDraft(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()
	_, rng := createTestFlagAndRange(t, repo)

	_, err := repo.CreateVersion(ctx, RangeVersion{RangeID: rng.ID, Rules: []Rule{}})
	if err != nil {
		t.Fatal(err)
	}

	// Second draft should fail
	_, err = repo.CreateVersion(ctx, RangeVersion{RangeID: rng.ID, Rules: []Rule{}})
	if err == nil {
		t.Error("expected ErrDraftExists")
	}
}

func TestMemRepo_PublishVersion(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()
	_, rng := createTestFlagAndRange(t, repo)

	v, _ := repo.CreateVersion(ctx, RangeVersion{RangeID: rng.ID, Rules: []Rule{}})

	err := repo.PublishVersion(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}

	got, _ := repo.GetVersion(ctx, v.ID)
	if got.Status != VersionStatusPublished {
		t.Errorf("got status %q, want %q", got.Status, VersionStatusPublished)
	}
	if got.PublishedAt == nil {
		t.Error("expected PublishedAt to be set")
	}
}

func TestMemRepo_GetPublishedVersion(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()
	_, rng := createTestFlagAndRange(t, repo)

	// No published version yet
	_, err := repo.GetPublishedVersion(ctx, rng.ID)
	if err == nil {
		t.Error("expected ErrNoPublishedVersion")
	}

	// Create and publish v1
	v1, _ := repo.CreateVersion(ctx, RangeVersion{RangeID: rng.ID, Rules: []Rule{{ID: "r1"}}})
	repo.PublishVersion(ctx, v1.ID)

	// Create and publish v2
	v2, _ := repo.CreateVersion(ctx, RangeVersion{RangeID: rng.ID, Rules: []Rule{{ID: "r2"}}})
	repo.PublishVersion(ctx, v2.ID)

	// Should return v2 (highest version number)
	got, err := repo.GetPublishedVersion(ctx, rng.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Version != 2 {
		t.Errorf("got version %d, want 2", got.Version)
	}
}

func TestMemRepo_DeleteDraftVersion(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()
	_, rng := createTestFlagAndRange(t, repo)

	v, _ := repo.CreateVersion(ctx, RangeVersion{RangeID: rng.ID, Rules: []Rule{}})

	err := repo.DeleteDraftVersion(ctx, v.ID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = repo.GetVersion(ctx, v.ID)
	if err == nil {
		t.Error("expected version to be deleted")
	}
}

func TestMemRepo_DeleteDraftVersion_RejectsPublished(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()
	_, rng := createTestFlagAndRange(t, repo)

	v, _ := repo.CreateVersion(ctx, RangeVersion{RangeID: rng.ID, Rules: []Rule{}})
	repo.PublishVersion(ctx, v.ID)

	err := repo.DeleteDraftVersion(ctx, v.ID)
	if err == nil {
		t.Error("expected ErrCannotDeletePublished")
	}
}

func TestMemRepo_UpdateVersion_DraftOnly(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()
	_, rng := createTestFlagAndRange(t, repo)

	v, _ := repo.CreateVersion(ctx, RangeVersion{RangeID: rng.ID, Rules: []Rule{}})

	v.Rules = []Rule{{ID: "updated"}}
	err := repo.UpdateVersion(ctx, v)
	if err != nil {
		t.Fatal(err)
	}

	got, _ := repo.GetVersion(ctx, v.ID)
	if len(got.Rules) != 1 || got.Rules[0].ID != "updated" {
		t.Error("rules not updated")
	}

	// Publish, then try to update
	repo.PublishVersion(ctx, v.ID)
	v.Rules = []Rule{{ID: "nope"}}
	err = repo.UpdateVersion(ctx, v)
	if err == nil {
		t.Error("expected error updating published version")
	}
}

func TestMemRepo_ListVersions(t *testing.T) {
	repo := newTestRepo()
	ctx := context.Background()
	_, rng := createTestFlagAndRange(t, repo)

	v1, _ := repo.CreateVersion(ctx, RangeVersion{RangeID: rng.ID, Rules: []Rule{}})
	repo.PublishVersion(ctx, v1.ID)
	repo.CreateVersion(ctx, RangeVersion{RangeID: rng.ID, Rules: []Rule{}})

	versions, err := repo.ListVersions(ctx, rng.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 2 {
		t.Errorf("got %d versions, want 2", len(versions))
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/flag/ -run "TestMemRepo_.*Version" -v`
Expected: FAIL (version methods are stubs)

- [ ] **Step 3: Implement version CRUD methods**

Replace the version stub methods in `memory_repository.go`:

```go
func (r *InMemoryRepository) CreateVersion(ctx context.Context, v RangeVersion) (RangeVersion, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.ranges[v.RangeID]; !ok {
		return RangeVersion{}, ErrRangeNotFound
	}

	// Check for existing draft
	maxVersion := 0
	for _, existing := range r.versions {
		if existing.RangeID == v.RangeID {
			if existing.Status == VersionStatusDraft {
				return RangeVersion{}, ErrDraftExists
			}
			if existing.Version > maxVersion {
				maxVersion = existing.Version
			}
		}
	}

	r.verSeq++
	v.ID = r.verSeq
	v.Version = maxVersion + 1
	v.Status = VersionStatusDraft
	v.PublishedAt = nil
	v.CreatedAt = time.Now().UTC()

	r.versions[v.ID] = v
	return v, nil
}

func (r *InMemoryRepository) GetVersion(ctx context.Context, id int64) (RangeVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	v, ok := r.versions[id]
	if !ok {
		return RangeVersion{}, ErrVersionNotFound
	}
	return v, nil
}

func (r *InMemoryRepository) UpdateVersion(ctx context.Context, v RangeVersion) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, ok := r.versions[v.ID]
	if !ok {
		return ErrVersionNotFound
	}
	if existing.Status != VersionStatusDraft {
		return ErrVersionNotDraft
	}

	existing.Rules = v.Rules
	r.versions[v.ID] = existing
	return nil
}

func (r *InMemoryRepository) DeleteDraftVersion(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	v, ok := r.versions[id]
	if !ok {
		return ErrVersionNotFound
	}
	if v.Status != VersionStatusDraft {
		return ErrCannotDeletePublished
	}
	delete(r.versions, id)
	return nil
}

func (r *InMemoryRepository) ListVersions(ctx context.Context, rangeID int64) ([]RangeVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []RangeVersion
	for _, v := range r.versions {
		if v.RangeID == rangeID {
			result = append(result, v)
		}
	}
	if result == nil {
		return []RangeVersion{}, nil
	}
	return result, nil
}

func (r *InMemoryRepository) GetPublishedVersion(ctx context.Context, rangeID int64) (RangeVersion, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var best *RangeVersion
	for _, v := range r.versions {
		if v.RangeID == rangeID && v.Status == VersionStatusPublished {
			if best == nil || v.Version > best.Version {
				vCopy := v
				best = &vCopy
			}
		}
	}
	if best == nil {
		return RangeVersion{}, ErrNoPublishedVersion
	}
	return *best, nil
}

func (r *InMemoryRepository) PublishVersion(ctx context.Context, id int64) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	v, ok := r.versions[id]
	if !ok {
		return ErrVersionNotFound
	}
	if v.Status != VersionStatusDraft {
		return ErrVersionNotDraft
	}

	now := time.Now().UTC()
	v.Status = VersionStatusPublished
	v.PublishedAt = &now
	r.versions[id] = v
	return nil
}
```

- [ ] **Step 4: Run all memory repository tests**

Run: `go test ./internal/flag/ -run "TestMemRepo_" -v`
Expected: ALL PASS

- [ ] **Step 5: Commit**

```bash
git add internal/flag/memory_repository.go internal/flag/memory_repository_test.go
git commit -m "feat(flag): implement memory repository version CRUD with draft/publish"
```

---

## Chunk 3: Service Layer & Evaluator

### Task 7: Service layer — flag operations

**Files:**
- Rewrite: `internal/flag/service.go`

- [ ] **Step 1: Write service flag operation tests**

Create `internal/flag/service_test.go`:

```go
package flag

import (
	"context"
	"testing"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	repo := NewInMemoryRepository()
	svc, err := NewService(repo)
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

func TestService_CreateFlag(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	f, err := svc.CreateFlag(ctx, FeatureFlag{
		Project: "p", Stage: "s", Key: "k", Name: "n", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}},
	}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if f.ID == 0 {
		t.Error("expected non-zero ID")
	}
}

func TestService_CreateFlag_Validation(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	_, err := svc.CreateFlag(ctx, FeatureFlag{}, "admin")
	if err == nil {
		t.Error("expected validation error")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/flag/ -run "TestService_" -v`
Expected: FAIL (Service has old signature)

- [ ] **Step 3: Rewrite service.go**

The new service wraps the repository and adds business logic. `CreateFlag` now returns the created flag. Range and version operations have service methods with audit logging.

```go
package flag

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	repo         Repository
	auditService AuditService
}

func NewService(repo Repository) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	return &Service{repo: repo}, nil
}

func (s *Service) SetAuditService(auditService AuditService) {
	s.auditService = auditService
}

// --- Flag identity operations ---

func (s *Service) CreateFlag(ctx context.Context, f FeatureFlag, performedBy string) (FeatureFlag, error) {
	if f.Project == "" {
		return FeatureFlag{}, fmt.Errorf("flag project is required")
	}
	if f.Stage == "" {
		return FeatureFlag{}, fmt.Errorf("flag stage is required")
	}
	if f.Key == "" {
		return FeatureFlag{}, fmt.Errorf("flag key is required")
	}

	created, err := s.repo.CreateFlag(ctx, f)
	if err != nil {
		return FeatureFlag{}, err
	}

	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, ActionCreate, performedBy, nil, &created)
	}
	return created, nil
}

func (s *Service) GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error) {
	return s.repo.GetFlag(ctx, project, stage, key)
}

func (s *Service) GetFlagByID(ctx context.Context, id int64) (FeatureFlag, error) {
	return s.repo.GetFlagByID(ctx, id)
}

func (s *Service) ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error) {
	flags, err := s.repo.ListFlags(ctx, project, stage)
	if err != nil {
		return nil, err
	}
	if flags == nil {
		return []FeatureFlag{}, nil
	}
	return flags, nil
}

func (s *Service) UpdateFlag(ctx context.Context, f FeatureFlag, performedBy string) error {
	if f.ID == 0 {
		return fmt.Errorf("flag ID is required")
	}

	oldFlag, err := s.repo.GetFlagByID(ctx, f.ID)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateFlag(ctx, f); err != nil {
		return err
	}

	if s.auditService != nil {
		newFlag, _ := s.repo.GetFlagByID(ctx, f.ID)
		_ = s.auditService.LogAction(ctx, oldFlag.Project, oldFlag.Stage, oldFlag.Key, ActionUpdate, performedBy, &oldFlag, &newFlag)
	}
	return nil
}

func (s *Service) DeleteFlag(ctx context.Context, id int64, performedBy string) error {
	oldFlag, err := s.repo.GetFlagByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteFlag(ctx, id); err != nil {
		return err
	}

	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, oldFlag.Project, oldFlag.Stage, oldFlag.Key, ActionDelete, performedBy, &oldFlag, nil)
	}
	return nil
}

// --- Range operations ---

func (s *Service) CreateRange(ctx context.Context, r FlagRange, rules []Rule, performedBy string) (FlagRange, RangeVersion, error) {
	// Validate flag exists
	f, err := s.repo.GetFlagByID(ctx, r.FlagID)
	if err != nil {
		return FlagRange{}, RangeVersion{}, err
	}

	// Create range
	created, err := s.repo.CreateRange(ctx, r)
	if err != nil {
		return FlagRange{}, RangeVersion{}, err
	}

	// Create v1 draft
	v, err := s.repo.CreateVersion(ctx, RangeVersion{
		RangeID: created.ID,
		Rules:   rules,
	})
	if err != nil {
		// Rollback range creation
		_ = s.repo.DeleteRange(ctx, created.ID)
		return FlagRange{}, RangeVersion{}, err
	}

	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "create_range", performedBy, nil, &created)
	}
	return created, v, nil
}

func (s *Service) GetRange(ctx context.Context, id int64) (FlagRange, error) {
	return s.repo.GetRange(ctx, id)
}

func (s *Service) ListRanges(ctx context.Context, flagID int64) ([]FlagRange, error) {
	ranges, err := s.repo.ListRanges(ctx, flagID)
	if err != nil {
		return nil, err
	}
	if ranges == nil {
		return []FlagRange{}, nil
	}
	return ranges, nil
}

func (s *Service) UpdateRange(ctx context.Context, r FlagRange, performedBy string) error {
	old, err := s.repo.GetRange(ctx, r.ID)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateRange(ctx, r); err != nil {
		return err
	}

	if s.auditService != nil {
		f, _ := s.repo.GetFlagByID(ctx, old.FlagID)
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "update_range", performedBy, &old, &r)
	}
	return nil
}

func (s *Service) DeleteRange(ctx context.Context, id int64, performedBy string) error {
	old, err := s.repo.GetRange(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteRange(ctx, id); err != nil {
		return err
	}

	if s.auditService != nil {
		f, _ := s.repo.GetFlagByID(ctx, old.FlagID)
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "delete_range", performedBy, &old, nil)
	}
	return nil
}

func (s *Service) ActivateRange(ctx context.Context, id int64, performedBy string) error {
	rng, err := s.repo.GetRange(ctx, id)
	if err != nil {
		return err
	}

	// Must have published version
	_, err = s.repo.GetPublishedVersion(ctx, id)
	if err != nil {
		return fmt.Errorf("cannot activate range: %w", ErrNoPublishedVersion)
	}

	if err := s.repo.ActivateRange(ctx, id); err != nil {
		return err
	}

	if s.auditService != nil {
		f, _ := s.repo.GetFlagByID(ctx, rng.FlagID)
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "activate", performedBy, nil, nil)
	}
	return nil
}

func (s *Service) DeactivateRange(ctx context.Context, id int64, performedBy string) error {
	rng, err := s.repo.GetRange(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.DeactivateRange(ctx, id); err != nil {
		return err
	}

	if s.auditService != nil {
		f, _ := s.repo.GetFlagByID(ctx, rng.FlagID)
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "deactivate", performedBy, nil, nil)
	}
	return nil
}

// --- Version operations ---

func (s *Service) CreateVersion(ctx context.Context, rangeID int64, rules []Rule, performedBy string) (RangeVersion, error) {
	rng, err := s.repo.GetRange(ctx, rangeID)
	if err != nil {
		return RangeVersion{}, err
	}

	v, err := s.repo.CreateVersion(ctx, RangeVersion{
		RangeID: rangeID,
		Rules:   rules,
	})
	if err != nil {
		return RangeVersion{}, err
	}

	if s.auditService != nil {
		f, _ := s.repo.GetFlagByID(ctx, rng.FlagID)
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "create_version", performedBy, nil, &v)
	}
	return v, nil
}

func (s *Service) GetVersion(ctx context.Context, id int64) (RangeVersion, error) {
	return s.repo.GetVersion(ctx, id)
}

func (s *Service) ListVersions(ctx context.Context, rangeID int64) ([]RangeVersion, error) {
	versions, err := s.repo.ListVersions(ctx, rangeID)
	if err != nil {
		return nil, err
	}
	if versions == nil {
		return []RangeVersion{}, nil
	}
	return versions, nil
}

func (s *Service) UpdateVersion(ctx context.Context, v RangeVersion, performedBy string) error {
	return s.repo.UpdateVersion(ctx, v)
}

func (s *Service) DeleteDraftVersion(ctx context.Context, id int64, performedBy string) error {
	return s.repo.DeleteDraftVersion(ctx, id)
}

func (s *Service) PublishVersion(ctx context.Context, id int64, performedBy string) error {
	v, err := s.repo.GetVersion(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.PublishVersion(ctx, id); err != nil {
		return err
	}

	if s.auditService != nil {
		rng, _ := s.repo.GetRange(ctx, v.RangeID)
		f, _ := s.repo.GetFlagByID(ctx, rng.FlagID)
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "publish", performedBy, nil, &v)
	}
	return nil
}

func (s *Service) RollbackVersion(ctx context.Context, sourceVersionID int64, performedBy string) (RangeVersion, error) {
	source, err := s.repo.GetVersion(ctx, sourceVersionID)
	if err != nil {
		return RangeVersion{}, err
	}

	// Check if draft exists — if so, overwrite its rules
	versions, err := s.repo.ListVersions(ctx, source.RangeID)
	if err != nil {
		return RangeVersion{}, err
	}

	for _, v := range versions {
		if v.Status == VersionStatusDraft {
			v.Rules = source.Rules
			if err := s.repo.UpdateVersion(ctx, v); err != nil {
				return RangeVersion{}, err
			}
			return v, nil
		}
	}

	// No draft exists — create new one
	return s.repo.CreateVersion(ctx, RangeVersion{
		RangeID: source.RangeID,
		Rules:   source.Rules,
	})
}

// --- Evaluation ---

func (s *Service) EvaluateFlag(ctx context.Context, project, stage, key string, attrs EvaluationContext) (EvaluationResult, error) {
	if key == "" {
		return EvaluationResult{}, fmt.Errorf("flag key is required")
	}

	f, err := s.repo.GetFlag(ctx, project, stage, key)
	if err != nil {
		return EvaluationResult{}, err
	}

	if !f.Enabled {
		return defaultResult(f, "DISABLED"), nil
	}

	rng, err := s.repo.GetActiveRange(ctx, f.ID, time.Now().UTC())
	if err != nil {
		return defaultResult(f, "NO_ACTIVE_RANGE"), nil
	}

	v, err := s.repo.GetPublishedVersion(ctx, rng.ID)
	if err != nil {
		return defaultResult(f, "NO_PUBLISHED_VERSION"), nil
	}

	if attrs == nil {
		attrs = EvaluationContext{}
	}
	return s.evaluator.Evaluate(ctx, f, v, attrs)
}

func defaultResult(f FeatureFlag, reason string) EvaluationResult {
	for _, v := range f.Variations {
		if v.Key == f.DefaultKey {
			return EvaluationResult{
				Variation: v,
				Reason:    reason,
			}
		}
	}
	return EvaluationResult{
		Variation: Variation{Key: f.DefaultKey},
		Reason:    reason,
	}
}

// --- Audit log query ---

func (s *Service) GetAuditLogs(ctx context.Context, project, stage, flagKey string, limit int) ([]AuditLog, error) {
	if s.auditService == nil {
		return nil, fmt.Errorf("audit service not configured")
	}
	logs, err := s.auditService.GetAuditLogs(ctx, project, stage, flagKey, limit)
	if err != nil {
		return nil, err
	}
	if logs == nil {
		return []AuditLog{}, nil
	}
	return logs, nil
}
```

Note: The service still needs an `evaluator` field. Add it:

```go
type Service struct {
	repo         Repository
	evaluator    Evaluator
	auditService AuditService
}

func NewService(repo Repository) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	return &Service{repo: repo}, nil
}

func (s *Service) SetEvaluator(evaluator Evaluator) {
	s.evaluator = evaluator
}
```

- [ ] **Step 4: Run tests**

Run: `go test ./internal/flag/ -run "TestService_" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/flag/service.go internal/flag/service_test.go
git commit -m "feat(flag): rewrite service layer for three-entity model"
```

---

### Task 8: Update evaluator and fix integration tests

**Files:**
- Modify: `internal/flag/evaluator.go`
- Modify: `internal/flag/evaluator_test.go`
- Modify: `internal/flag/openfeature_integration_test.go`
- Modify: `internal/flag/openfeature_http_integration_test.go`

The evaluator signature changes from `Evaluate(ctx, flag, attrs)` to `Evaluate(ctx, flag, version, attrs)`. Rules now come from the version, not the flag.

**IMPORTANT:** Also update integration tests in this task to prevent a broken build window. The integration tests reference the old `NewService(repo, engine)` constructor (now `NewService(repo)` + `SetEvaluator`) and the old `FeatureFlag` shape. Update them to use the three-entity flow: create flag → create range → create version → publish → activate → evaluate.

- [ ] **Step 1: Write evaluator test with new signature**

Update `internal/flag/evaluator_test.go` to use the new signature:

```go
func TestEngine_ReturnsDefaultWhenDisabled(t *testing.T) {
	e := &Engine{}
	f := FeatureFlag{
		Key:        "test",
		Enabled:    false,
		DefaultKey: "off",
		Variations: []Variation{{Key: "off", Type: BooleanVariation, Value: false}},
	}
	v := RangeVersion{Rules: []Rule{}}

	result, err := e.Evaluate(context.Background(), f, v, EvaluationContext{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != ReasonDisabled {
		t.Errorf("got reason %q, want %q", result.Reason, ReasonDisabled)
	}
}

func TestEngine_StaticRuleMatch(t *testing.T) {
	e := &Engine{}
	f := FeatureFlag{
		Key:     "test",
		Enabled: true,
		DefaultKey: "off",
		Variations: []Variation{
			{Key: "off", Type: BooleanVariation, Value: false},
			{Key: "on", Type: BooleanVariation, Value: true},
		},
	}
	v := RangeVersion{
		Rules: []Rule{{
			ID:           "rule-1",
			Conditions:   []Condition{{Attribute: "country", Operator: MatcherEquals, Value: "DE"}},
			VariationKey: "on",
		}},
	}

	result, err := e.Evaluate(context.Background(), f, v, EvaluationContext{"country": "DE"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Variation.Key != "on" {
		t.Errorf("got %q, want on", result.Variation.Key)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/flag/ -run "TestEngine_" -v`
Expected: FAIL (wrong signature)

- [ ] **Step 3: Update evaluator.go**

Change the `Evaluate` method signature. Replace `flag.Rules` references with `v.Rules`:

In `evaluator.go`, change:
```go
func (e *Engine) Evaluate(ctx context.Context, flag FeatureFlag, attrs EvaluationContext) (EvaluationResult, error) {
```
to:
```go
func (e *Engine) Evaluate(ctx context.Context, flag FeatureFlag, v RangeVersion, attrs EvaluationContext) (EvaluationResult, error) {
```

And change all references from `flag.Rules` to `v.Rules` within the method body.

- [ ] **Step 4: Run tests**

Run: `go test ./internal/flag/ -run "TestEngine_" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/flag/evaluator.go internal/flag/evaluator_test.go
git commit -m "refactor(flag): update evaluator to accept RangeVersion for rules"
```

---

## Chunk 4: HTTP Handlers & Routing

### Task 9: Flag identity handlers

**Files:**
- Rewrite: `internal/httpserver/admin.go`
- Modify: `internal/httpserver/router.go`

- [ ] **Step 1: Rewrite admin.go flag handlers**

The handlers split into three groups: flag identity, range, and version. Start with flag identity handlers. Each handler follows the pattern: parse request → call service → write JSON response.

```go
// Flag identity handlers

func newListFlagsHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := r.PathValue("project")
		stage := r.PathValue("stage")
		flags, err := svc.ListFlags(r.Context(), project, stage)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"flags": flags})
	}
}

func newCreateFlagHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := r.PathValue("project")
		stage := r.PathValue("stage")

		var f flag.FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		f.Project = project
		f.Stage = stage

		created, err := svc.CreateFlag(r.Context(), f, requestUser(r))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, created)
	}
}

func newGetFlagHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		project := r.PathValue("project")
		stage := r.PathValue("stage")
		key := r.PathValue("key")
		f, err := svc.GetFlag(r.Context(), project, stage, key)
		if err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, f)
	}
}

func newGetFlagByIDHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}
		f, err := svc.GetFlagByID(r.Context(), id)
		if err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, f)
	}
}

func newUpdateFlagHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}

		var f flag.FeatureFlag
		if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		f.ID = id

		if err := svc.UpdateFlag(r.Context(), f, requestUser(r)); err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

func newDeleteFlagHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}
		if err := svc.DeleteFlag(r.Context(), id, requestUser(r)); err != nil {
			if errors.Is(err, flag.ErrFlagNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}
```

- [ ] **Step 2: Add range handlers**

```go
// Range handlers

func newListRangesHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		flagID, err := strconv.ParseInt(r.PathValue("flagId"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}
		ranges, err := svc.ListRanges(r.Context(), flagID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"ranges": ranges})
	}
}

func newCreateRangeHandler(svc *flag.Service) http.HandlerFunc {
	type createRangeRequest struct {
		ValidFrom time.Time  `json:"validFrom"`
		ValidTo   *time.Time `json:"validTo,omitempty"`
		Rules     []flag.Rule `json:"rules"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		flagID, err := strconv.ParseInt(r.PathValue("flagId"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid flag ID")
			return
		}

		var req createRangeRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}

		rng, ver, err := svc.CreateRange(r.Context(), flag.FlagRange{
			FlagID:    flagID,
			ValidFrom: req.ValidFrom,
			ValidTo:   req.ValidTo,
		}, req.Rules, requestUser(r))
		if err != nil {
			status := http.StatusBadRequest
			if errors.Is(err, flag.ErrFlagNotFound) {
				status = http.StatusNotFound
			}
			writeError(w, status, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, map[string]interface{}{"range": rng, "version": ver})
	}
}

func newGetRangeHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}
		rng, err := svc.GetRange(r.Context(), id)
		if err != nil {
			if errors.Is(err, flag.ErrRangeNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rng)
	}
}

func newUpdateRangeHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}
		var rng flag.FlagRange
		if err := json.NewDecoder(r.Body).Decode(&rng); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		rng.ID = id
		if err := svc.UpdateRange(r.Context(), rng, requestUser(r)); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

func newDeleteRangeHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}
		if err := svc.DeleteRange(r.Context(), id, requestUser(r)); err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func newActivateRangeHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}
		if err := svc.ActivateRange(r.Context(), id, requestUser(r)); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "activated"})
	}
}

func newDeactivateRangeHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}
		if err := svc.DeactivateRange(r.Context(), id, requestUser(r)); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deactivated"})
	}
}
```

- [ ] **Step 3: Add version handlers**

```go
// Version handlers

func newListVersionsHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rangeID, err := strconv.ParseInt(r.PathValue("rangeId"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}
		versions, err := svc.ListVersions(r.Context(), rangeID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"versions": versions})
	}
}

func newCreateVersionHandler(svc *flag.Service) http.HandlerFunc {
	type createVersionRequest struct {
		Rules []flag.Rule `json:"rules"`
	}
	return func(w http.ResponseWriter, r *http.Request) {
		rangeID, err := strconv.ParseInt(r.PathValue("rangeId"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid range ID")
			return
		}
		var req createVersionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		v, err := svc.CreateVersion(r.Context(), rangeID, req.Rules, requestUser(r))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, v)
	}
}

func newGetVersionHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid version ID")
			return
		}
		v, err := svc.GetVersion(r.Context(), id)
		if err != nil {
			if errors.Is(err, flag.ErrVersionNotFound) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, v)
	}
}

func newUpdateVersionHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid version ID")
			return
		}
		var v flag.RangeVersion
		if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
			writeError(w, http.StatusBadRequest, "invalid request body")
			return
		}
		v.ID = id
		if err := svc.UpdateVersion(r.Context(), v, requestUser(r)); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "updated"})
	}
}

func newDeleteVersionHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid version ID")
			return
		}
		if err := svc.DeleteDraftVersion(r.Context(), id, requestUser(r)); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
	}
}

func newPublishVersionHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid version ID")
			return
		}
		if err := svc.PublishVersion(r.Context(), id, requestUser(r)); err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "published"})
	}
}

func newRollbackVersionHandler(svc *flag.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid version ID")
			return
		}
		v, err := svc.RollbackVersion(r.Context(), id, requestUser(r))
		if err != nil {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		writeJSON(w, http.StatusCreated, v)
	}
}
```

- [ ] **Step 4: Update evaluate.go**

Update the evaluation handler — the endpoint path and service call remain the same, but verify it compiles with the new service signature. The handler itself needs no logic changes since `svc.EvaluateFlag` handles the three-entity flow internally.

- [ ] **Step 5: Update router.go with new routes**

Replace the flag route registration block with:

Use the existing pattern `wrapAuth(cfg.AuthManager, true, handler)`. Replace the flag route block in `NewRouter()`:

```go
// Flag identity
mux.HandleFunc("GET /api/v1/admin/{project}/{stage}/flags",
    wrapAuth(cfg.AuthManager, true, newListFlagsHandler(cfg.FlagService)))
mux.HandleFunc("POST /api/v1/admin/{project}/{stage}/flags",
    wrapAuth(cfg.AuthManager, true, newCreateFlagHandler(cfg.FlagService)))
mux.HandleFunc("GET /api/v1/admin/{project}/{stage}/flags/{key}",
    wrapAuth(cfg.AuthManager, true, newGetFlagHandler(cfg.FlagService)))
mux.HandleFunc("GET /api/v1/admin/flags/{id}",
    wrapAuth(cfg.AuthManager, true, newGetFlagByIDHandler(cfg.FlagService)))
mux.HandleFunc("PUT /api/v1/admin/flags/{id}",
    wrapAuth(cfg.AuthManager, true, newUpdateFlagHandler(cfg.FlagService)))
mux.HandleFunc("DELETE /api/v1/admin/flags/{id}",
    wrapAuth(cfg.AuthManager, true, newDeleteFlagHandler(cfg.FlagService)))

// Ranges
mux.HandleFunc("GET /api/v1/admin/flags/{flagId}/ranges",
    wrapAuth(cfg.AuthManager, true, newListRangesHandler(cfg.FlagService)))
mux.HandleFunc("POST /api/v1/admin/flags/{flagId}/ranges",
    wrapAuth(cfg.AuthManager, true, newCreateRangeHandler(cfg.FlagService)))
mux.HandleFunc("GET /api/v1/admin/ranges/{id}",
    wrapAuth(cfg.AuthManager, true, newGetRangeHandler(cfg.FlagService)))
mux.HandleFunc("PUT /api/v1/admin/ranges/{id}",
    wrapAuth(cfg.AuthManager, true, newUpdateRangeHandler(cfg.FlagService)))
mux.HandleFunc("DELETE /api/v1/admin/ranges/{id}",
    wrapAuth(cfg.AuthManager, true, newDeleteRangeHandler(cfg.FlagService)))
mux.HandleFunc("POST /api/v1/admin/ranges/{id}/activate",
    wrapAuth(cfg.AuthManager, true, newActivateRangeHandler(cfg.FlagService)))
mux.HandleFunc("POST /api/v1/admin/ranges/{id}/deactivate",
    wrapAuth(cfg.AuthManager, true, newDeactivateRangeHandler(cfg.FlagService)))

// Versions
mux.HandleFunc("GET /api/v1/admin/ranges/{rangeId}/versions",
    wrapAuth(cfg.AuthManager, true, newListVersionsHandler(cfg.FlagService)))
mux.HandleFunc("POST /api/v1/admin/ranges/{rangeId}/versions",
    wrapAuth(cfg.AuthManager, true, newCreateVersionHandler(cfg.FlagService)))
mux.HandleFunc("GET /api/v1/admin/versions/{id}",
    wrapAuth(cfg.AuthManager, true, newGetVersionHandler(cfg.FlagService)))
mux.HandleFunc("PUT /api/v1/admin/versions/{id}",
    wrapAuth(cfg.AuthManager, true, newUpdateVersionHandler(cfg.FlagService)))
mux.HandleFunc("DELETE /api/v1/admin/versions/{id}",
    wrapAuth(cfg.AuthManager, true, newDeleteVersionHandler(cfg.FlagService)))
mux.HandleFunc("POST /api/v1/admin/versions/{id}/publish",
    wrapAuth(cfg.AuthManager, true, newPublishVersionHandler(cfg.FlagService)))
mux.HandleFunc("POST /api/v1/admin/versions/{id}/rollback",
    wrapAuth(cfg.AuthManager, true, newRollbackVersionHandler(cfg.FlagService)))

// Audit
mux.HandleFunc("GET /api/v1/admin/{project}/{stage}/flags/{key}/audit",
    wrapAuth(cfg.AuthManager, true, newGetAuditLogsHandler(cfg.FlagService)))
```

- [ ] **Step 6: Update app.go wiring**

In `internal/app/app.go`, update the `NewService` call — it no longer takes an evaluator in the constructor. Set it separately:

```go
flagSvc, err := flag.NewService(flagRepo)
if err != nil { ... }
flagSvc.SetEvaluator(&flag.Engine{})
```

- [ ] **Step 7: Verify build compiles**

Run: `go build ./...`
Expected: Build succeeds

- [ ] **Step 8: Commit**

```bash
git add internal/httpserver/admin.go internal/httpserver/router.go internal/httpserver/evaluate.go internal/app/app.go
git commit -m "feat(httpserver): add range and version handlers, update routing"
```

---

## Chunk 5: Database Migration & Postgres Repository

### Task 10: Write database migration

**Files:**
- Create: `migrations/000006_temporal_redesign.up.sql`

- [ ] **Step 1: Write migration SQL**

```sql
-- 000006_temporal_redesign.up.sql
-- Splits feature_flags into identity + ranges + versions

BEGIN;

-- 1. Create new identity table
CREATE TABLE feature_flags_new (
    id          BIGSERIAL PRIMARY KEY,
    project     VARCHAR(255) NOT NULL,
    stage       VARCHAR(255) NOT NULL,
    key         VARCHAR(255) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    enabled     BOOLEAN NOT NULL DEFAULT false,
    default_key VARCHAR(255) NOT NULL,
    variations  JSONB NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(project, stage, key)
);

-- 2. Create ranges table
CREATE TABLE flag_ranges (
    id         BIGSERIAL PRIMARY KEY,
    flag_id    BIGINT NOT NULL,
    active     BOOLEAN NOT NULL DEFAULT true,
    valid_from TIMESTAMPTZ NOT NULL,
    valid_to   TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- 3. Create versions table
CREATE TABLE range_versions (
    id           BIGSERIAL PRIMARY KEY,
    range_id     BIGINT NOT NULL,
    version      INTEGER NOT NULL,
    status       VARCHAR(20) NOT NULL DEFAULT 'draft',
    rules        JSONB NOT NULL DEFAULT '[]',
    published_at TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(range_id, version)
);

-- 4. Migrate data: insert one identity per distinct (project, stage, key)
INSERT INTO feature_flags_new (project, stage, key, name, description, enabled, default_key, variations, created_at, updated_at)
SELECT DISTINCT ON (project, stage, key)
    project, stage, key, name, description, enabled, default_key,
    COALESCE(config->'variations', '[]'::jsonb),
    created_at, updated_at
FROM feature_flags
ORDER BY project, stage, key, updated_at DESC;

-- 5. Migrate ranges
INSERT INTO flag_ranges (flag_id, active, valid_from, valid_to, created_at, updated_at)
SELECT
    fn.id,
    ff.active,
    ff.valid_from,
    ff.valid_to,
    ff.created_at,
    ff.updated_at
FROM feature_flags ff
JOIN feature_flags_new fn ON fn.project = ff.project AND fn.stage = ff.stage AND fn.key = ff.key;

-- 6. Migrate versions (v1 published for each range)
INSERT INTO range_versions (range_id, version, status, rules, published_at, created_at)
SELECT
    fr.id,
    1,
    'published',
    COALESCE(ff.config->'rules', '[]'::jsonb),
    ff.updated_at,
    ff.created_at
FROM feature_flags ff
JOIN feature_flags_new fn ON fn.project = ff.project AND fn.stage = ff.stage AND fn.key = ff.key
JOIN flag_ranges fr ON fr.flag_id = fn.id AND fr.valid_from = ff.valid_from AND (fr.valid_to = ff.valid_to OR (fr.valid_to IS NULL AND ff.valid_to IS NULL));

-- 7. Add foreign keys after data migration
ALTER TABLE flag_ranges ADD CONSTRAINT fk_flag_ranges_flag_id FOREIGN KEY (flag_id) REFERENCES feature_flags_new(id) ON DELETE CASCADE;
ALTER TABLE range_versions ADD CONSTRAINT fk_range_versions_range_id FOREIGN KEY (range_id) REFERENCES flag_ranges(id) ON DELETE CASCADE;

-- 8. Indices
CREATE INDEX idx_flag_ranges_flag_id ON flag_ranges(flag_id);
CREATE INDEX idx_flag_ranges_active_time ON flag_ranges(flag_id, valid_from, valid_to) WHERE active = true;
CREATE INDEX idx_range_versions_range_id ON range_versions(range_id);
CREATE INDEX idx_range_versions_published ON range_versions(range_id, version) WHERE status = 'published';

-- 9. Update audit_logs
ALTER TABLE audit_logs ADD COLUMN IF NOT EXISTS range_id BIGINT;
UPDATE audit_logs SET flag_id = NULL; -- old flag_ids are invalid

-- 10. Drop old table, rename new
DROP TABLE feature_flags;
ALTER TABLE feature_flags_new RENAME TO feature_flags;

-- 11. Recreate the updated_at trigger for new tables
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_feature_flags_updated_at BEFORE UPDATE ON feature_flags FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_flag_ranges_updated_at BEFORE UPDATE ON flag_ranges FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMIT;
```

- [ ] **Step 2: Commit**

```bash
git add migrations/000006_temporal_redesign.up.sql
git commit -m "feat(db): add migration to split feature_flags into identity, ranges, versions"
```

---

### Task 11: Rewrite Postgres repository

**Files:**
- Rewrite: `internal/flag/postgres_repository.go`

- [ ] **Step 1: Rewrite postgres_repository.go**

The Postgres repository implements the same `Repository` interface as the memory implementation. Follow the same method signatures. Each method maps to SQL queries against the new three-table schema.

Key patterns:
- Flag identity: queries `feature_flags` table
- Ranges: queries `flag_ranges` table, JOIN to `feature_flags` when needed
- Versions: queries `range_versions` table
- `GetPublishedVersion`: `SELECT ... WHERE range_id = $1 AND status = 'published' ORDER BY version DESC LIMIT 1`
- `GetActiveRange`: `SELECT ... WHERE flag_id = $1 AND active = true AND valid_from <= $2 AND (valid_to IS NULL OR valid_to > $2) ORDER BY valid_from DESC LIMIT 1`
- `CheckRangeOverlap`: use the `RangesOverlap` logic in SQL: `WHERE flag_id = $1 AND id != $4 AND valid_from < COALESCE($3, 'infinity'::timestamptz) AND (valid_to IS NULL OR valid_to > $2)`
- Variations stored as JSONB in `feature_flags.variations`
- Rules stored as JSONB in `range_versions.rules`
- Use `pgx` directly (no ORM, per CLAUDE.md)

The implementation should follow the exact same patterns as the existing `postgres_repository.go` — use `pgx.Pool`, scan rows with `pgx.CollectRows`, marshal/unmarshal JSONB with `json.Marshal`/`json.Unmarshal`.

- [ ] **Step 2: Verify build**

Run: `go build ./internal/flag/...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add internal/flag/postgres_repository.go
git commit -m "feat(flag): rewrite postgres repository for three-entity model"
```

---

## Chunk 6: Audit Service Update

### Task 12: Update audit service

**Files:**
- Modify: `internal/flag/audit.go`

- [ ] **Step 1: Update AuditService to accept interface{} values**

The current audit service marshals `*FeatureFlag` to JSON. With the three-entity model, it needs to accept any value type (`interface{}`). Update `LogAction` to accept `interface{}` for old/new values instead of `*FeatureFlag`.

The `AuditLog` struct gains an optional `RangeID` field (already defined in model.go). The SQL INSERT adds the `range_id` column.

- [ ] **Step 2: Verify build**

Run: `go build ./internal/flag/...`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add internal/flag/audit.go
git commit -m "refactor(flag): update audit service for three-entity model"
```

---

## Chunk 7: Frontend

### Task 13: Update TypeScript types

**Files:**
- Modify: `frontend/src/types.ts`

- [ ] **Step 1: Update types.ts**

Replace the monolithic `FeatureFlag` interface with three interfaces:

```typescript
export interface FeatureFlag {
  id: number;
  project: string;
  stage: string;
  key: string;
  name: string;
  description?: string;
  enabled: boolean;
  defaultKey: string;
  variations: Variation[];
  createdAt: string;
  updatedAt: string;
}

export interface FlagRange {
  id: number;
  flagId: number;
  active: boolean;
  validFrom: string;
  validTo?: string;
  createdAt: string;
  updatedAt: string;
}

export interface RangeVersion {
  id: number;
  rangeId: number;
  version: number;
  status: 'draft' | 'published';
  rules: Rule[];
  publishedAt?: string;
  createdAt: string;
}
```

Keep `Variation`, `Rule`, `Condition`, `PercentageRollout`, `RolloutBucket` unchanged.

- [ ] **Step 2: Commit**

```bash
git add frontend/src/types.ts
git commit -m "refactor(frontend): split FeatureFlag type into identity, range, version"
```

---

### Task 14: Update FlagsPage

**Files:**
- Modify: `frontend/src/pages/FlagsPage.tsx`

- [ ] **Step 1: Update FlagsPage**

Key changes:
- List response is now `{ flags: FeatureFlag[] }` — no temporal fields per item
- Create modal sends only identity fields (key, name, description, defaultKey, variations) — no validFrom/validTo
- Toggle `enabled` calls `PUT /api/v1/admin/flags/{id}` with identity-only payload
- Remove any references to `active`, `validFrom`, `validTo`, `rules` from list items

- [ ] **Step 2: Verify frontend builds**

Run: `cd frontend && npm run build`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/FlagsPage.tsx
git commit -m "feat(frontend): update FlagsPage for identity-only flag model"
```

---

### Task 15: Redesign FlagDetailPage

**Files:**
- Modify: `frontend/src/pages/FlagDetailPage.tsx`

This is the biggest frontend change. The page becomes a two-level view: flag identity at top, ranges list below with expandable version management.

- [ ] **Step 1: Redesign FlagDetailPage**

The page structure:

1. **Header**: flag key, enabled toggle, delete
2. **Details card**: name, description (inline edit), default variation, variations list
3. **Ranges section**: fetched from `GET /api/v1/admin/flags/{flagId}/ranges`
   - Each range shows: validFrom, validTo, active badge, draft indicator
   - Actions: activate/deactivate, delete, expand to manage versions
4. **Add Range button**: opens modal with validFrom/validTo date pickers
5. **Expanded range view**: shows published rules, version history
   - "Edit Rules" → creates draft or edits existing draft
   - "Publish" → publishes draft
   - "Discard Draft" → deletes draft version
   - Version history list with rollback buttons

Key API calls:
- `GET /api/v1/admin/{project}/{stage}/flags/{key}` — flag identity
- `GET /api/v1/admin/flags/{flagId}/ranges` — ranges
- `GET /api/v1/admin/ranges/{rangeId}/versions` — versions for expanded range
- `POST /api/v1/admin/flags/{flagId}/ranges` — create range
- `POST /api/v1/admin/ranges/{rangeId}/versions` — create draft
- `PUT /api/v1/admin/versions/{id}` — update draft rules
- `POST /api/v1/admin/versions/{id}/publish` — publish
- `DELETE /api/v1/admin/versions/{id}` — discard draft
- `POST /api/v1/admin/versions/{id}/rollback` — rollback

- [ ] **Step 2: Verify frontend builds**

Run: `cd frontend && npm run build`
Expected: Build succeeds

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/FlagDetailPage.tsx
git commit -m "feat(frontend): redesign FlagDetailPage with ranges and version management"
```

---

## Chunk 8: Integration & Cleanup

### Task 16: Update integration tests

**Files:**
- Modify: `internal/flag/openfeature_integration_test.go`
- Modify: `internal/flag/openfeature_http_integration_test.go`

- [ ] **Step 1: Update integration tests**

The integration tests use the old monolithic `FeatureFlag` struct. Update them to:
1. Create flag identity
2. Create range with v1 draft
3. Publish v1
4. Activate range
5. Then evaluate

This ensures the full three-entity flow works end-to-end.

- [ ] **Step 2: Run all tests**

Run: `go test ./... -count=1`
Expected: ALL PASS

- [ ] **Step 3: Commit**

```bash
git add internal/flag/openfeature_integration_test.go internal/flag/openfeature_http_integration_test.go
git commit -m "test(flag): update integration tests for three-entity model"
```

---

### Task 17: Final cleanup and verification

- [ ] **Step 1: Remove old code**

Check for any remaining references to the old `FeatureFlag.ValidFrom`, `FeatureFlag.ValidTo`, `FeatureFlag.Active`, `FeatureFlag.Rules` fields. Search with `grep -r "ValidFrom\|ValidTo\|\.Active\|\.Rules" internal/flag/` and clean up any stragglers.

Also remove the old `Evaluator` field from `NewService` if it was previously a constructor parameter, and remove `ErrActiveRangeOverlap` from errors.go if no longer used.

- [ ] **Step 2: Run full test suite**

Run: `go test ./... -count=1 -v`
Expected: ALL PASS

- [ ] **Step 3: Run frontend build**

Run: `cd frontend && npm run build`
Expected: Build succeeds

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "chore: cleanup old model references after temporal redesign"
```
