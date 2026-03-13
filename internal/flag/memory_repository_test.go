//go:build ignore

package flag

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestInMemoryRepository_UpsertAndGet(t *testing.T) {
	repo := NewInMemoryRepository()
	now := time.Date(2024, 10, 1, 12, 0, 0, 0, time.UTC)
	repo.now = func() time.Time { return now }

	flag := FeatureFlag{
		Project:    "test-project",
		Stage:      "dev",
		Key:        "checkout-experience",
		Name:       "Checkout Experience",
		Enabled:    true,
		Active:     true,
		ValidFrom:  now,
		DefaultKey: "control",
		Variations: []Variation{
			{Key: "control", Type: BooleanVariation, Value: false},
			{Key: "variant", Type: BooleanVariation, Value: true},
		},
	}

	if err := repo.UpsertFlag(context.Background(), flag); err != nil {
		t.Fatalf("UpsertFlag: %v", err)
	}

	stored, err := repo.GetFlag(context.Background(), flag.Project, flag.Stage, flag.Key)
	if err != nil {
		t.Fatalf("GetFlag: %v", err)
	}

	if stored.CreatedAt != now || stored.UpdatedAt != now {
		t.Fatalf("expected timestamps to be set to %v, got created=%v updated=%v", now, stored.CreatedAt, stored.UpdatedAt)
	}

	if !stored.Enabled {
		t.Fatalf("expected stored flag to remain enabled")
	}

	stored.Variations[0].Value = true

	original, err := repo.GetFlag(context.Background(), flag.Project, flag.Stage, flag.Key)
	if err != nil {
		t.Fatalf("GetFlag: %v", err)
	}

	if original.Variations[0].Value.(bool) != false {
		t.Fatalf("expected stored variation to be unaffected by caller mutation")
	}
}

func TestInMemoryRepository_UpdateWithOptimisticLock(t *testing.T) {
	repo := NewInMemoryRepository()
	first := time.Date(2024, 10, 1, 12, 0, 0, 0, time.UTC)
	second := first.Add(time.Minute)

	repo.now = func() time.Time { return first }
	flag := FeatureFlag{
		Project:    "test-project",
		Stage:      "dev",
		Key:        "search-layout",
		Active:     true,
		ValidFrom:  first,
		DefaultKey: "old",
		Variations: []Variation{
			{Key: "old", Type: StringVariation, Value: "classic"},
			{Key: "new", Type: StringVariation, Value: "modern"},
		},
	}
	if err := repo.UpsertFlag(context.Background(), flag); err != nil {
		t.Fatalf("UpsertFlag initial: %v", err)
	}

	current, err := repo.GetFlag(context.Background(), flag.Project, flag.Stage, flag.Key)
	if err != nil {
		t.Fatalf("GetFlag: %v", err)
	}

	repo.now = func() time.Time { return second }
	current.DefaultKey = "new"
	if err := repo.UpsertFlag(context.Background(), current); err != nil {
		t.Fatalf("UpsertFlag update: %v", err)
	}

	updated, err := repo.GetFlag(context.Background(), flag.Project, flag.Stage, flag.Key)
	if err != nil {
		t.Fatalf("GetFlag: %v", err)
	}

	if updated.DefaultKey != "new" {
		t.Fatalf("expected default key to update, got %q", updated.DefaultKey)
	}
	if updated.CreatedAt != first {
		t.Fatalf("expected created at to remain unchanged, got %v", updated.CreatedAt)
	}
	if updated.UpdatedAt != second {
		t.Fatalf("expected updated at to change to %v, got %v", second, updated.UpdatedAt)
	}
}

func TestInMemoryRepository_UpdateConflict(t *testing.T) {
	repo := NewInMemoryRepository()
	now := time.Date(2024, 10, 1, 12, 0, 0, 0, time.UTC)
	repo.now = func() time.Time { return now }

	flag := FeatureFlag{
		Project:    "test-project",
		Stage:      "dev",
		Key:        "recommendations",
		Active:     true,
		ValidFrom:  now,
		DefaultKey: "off",
		Variations: []Variation{
			{Key: "off", Type: BooleanVariation, Value: false},
			{Key: "on", Type: BooleanVariation, Value: true},
		},
	}

	if err := repo.UpsertFlag(context.Background(), flag); err != nil {
		t.Fatalf("UpsertFlag initial: %v", err)
	}

	// Get the current flag to see its UpdatedAt
	current, err := repo.GetFlag(context.Background(), flag.Project, flag.Stage, flag.Key)
	if err != nil {
		t.Fatalf("GetFlag: %v", err)
	}

	// Create a stale version with a different UpdatedAt
	stale := current
	stale.UpdatedAt = current.UpdatedAt.Add(-time.Hour)

	if err := repo.UpsertFlag(context.Background(), stale); !errors.Is(err, ErrFlagConflict) {
		t.Fatalf("expected conflict error, got %v", err)
	}
}

func TestInMemoryRepository_Delete(t *testing.T) {
	repo := NewInMemoryRepository()
	now := time.Date(2024, 10, 1, 12, 0, 0, 0, time.UTC)
	repo.now = func() time.Time { return now }

	flag := FeatureFlag{
		Project:    "test-project",
		Stage:      "dev",
		Key:        "recommendations",
		Active:     true,
		ValidFrom:  now,
		DefaultKey: "off",
		Variations: []Variation{
			{Key: "off", Type: BooleanVariation, Value: false},
			{Key: "on", Type: BooleanVariation, Value: true},
		},
	}
	if err := repo.UpsertFlag(context.Background(), flag); err != nil {
		t.Fatalf("UpsertFlag initial: %v", err)
	}

	// Get the flag to obtain its ID
	stored, err := repo.GetFlag(context.Background(), flag.Project, flag.Stage, flag.Key)
	if err != nil {
		t.Fatalf("GetFlag: %v", err)
	}

	if err := repo.DeleteFlag(context.Background(), stored.ID); err != nil {
		t.Fatalf("DeleteFlag: %v", err)
	}

	if _, err := repo.GetFlag(context.Background(), flag.Project, flag.Stage, flag.Key); !errors.Is(err, ErrFlagNotFound) {
		t.Fatalf("expected not found after delete, got %v", err)
	}
}

func TestInMemoryRepository_InvalidFlag(t *testing.T) {
	repo := NewInMemoryRepository()
	now := time.Date(2024, 10, 1, 12, 0, 0, 0, time.UTC)

	flag := FeatureFlag{
		Project:    "test-project",
		Stage:      "dev",
		Key:        "badflag",
		Active:     true,
		ValidFrom:  now,
		DefaultKey: "missing",
		Variations: []Variation{
			{Key: "control", Type: BooleanVariation, Value: true},
		},
	}

	if err := repo.UpsertFlag(context.Background(), flag); !errors.Is(err, ErrInvalidFlag) {
		t.Fatalf("expected invalid flag error, got %v", err)
	}
}
