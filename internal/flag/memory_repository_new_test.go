package flag

import (
	"context"
	"errors"
	"testing"
	"time"
)

// createTestFlag is a helper that creates a flag in the repo and returns it.
func createTestFlag(t *testing.T, repo *InMemoryRepository) FeatureFlag {
	t.Helper()
	f, err := repo.CreateFlag(context.Background(), FeatureFlag{
		Project: "proj", Stage: "dev", Key: "test-flag",
		Name: "Test Flag", Enabled: true, DefaultKey: "on",
		Variations: []Variation{{Key: "on", Type: BooleanVariation, Value: true}},
	})
	if err != nil {
		t.Fatalf("createTestFlag: %v", err)
	}
	return f
}

// createTestFlagAndRange creates a flag and a range for that flag.
func createTestFlagAndRange(t *testing.T, repo *InMemoryRepository) (FeatureFlag, FlagRange) {
	t.Helper()
	f := createTestFlag(t, repo)
	now := time.Now().UTC()
	r, err := repo.CreateRange(context.Background(), FlagRange{
		FlagID:    f.ID,
		ValidFrom: now,
	})
	if err != nil {
		t.Fatalf("createTestFlagAndRange: %v", err)
	}
	return f, r
}

func TestMemRepo_CreateAndGetFlag(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	f := FeatureFlag{
		Project:    "proj",
		Stage:      "dev",
		Key:        "my-flag",
		Name:       "My Flag",
		Enabled:    true,
		DefaultKey: "on",
		Variations: []Variation{
			{Key: "on", Type: BooleanVariation, Value: true},
			{Key: "off", Type: BooleanVariation, Value: false},
		},
	}

	created, err := repo.CreateFlag(ctx, f)
	if err != nil {
		t.Fatalf("CreateFlag: %v", err)
	}

	if created.ID == 0 {
		t.Fatal("expected non-zero ID")
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}
	if created.UpdatedAt.IsZero() {
		t.Fatal("expected UpdatedAt to be set")
	}

	// GetFlag by composite key
	got, err := repo.GetFlag(ctx, "proj", "dev", "my-flag")
	if err != nil {
		t.Fatalf("GetFlag: %v", err)
	}
	if got.ID != created.ID {
		t.Fatalf("GetFlag ID: got %d, want %d", got.ID, created.ID)
	}
	if got.Name != "My Flag" {
		t.Fatalf("GetFlag Name: got %q, want %q", got.Name, "My Flag")
	}

	// GetFlagByID
	got2, err := repo.GetFlagByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetFlagByID: %v", err)
	}
	if got2.Key != "my-flag" {
		t.Fatalf("GetFlagByID Key: got %q, want %q", got2.Key, "my-flag")
	}

	// Verify clone isolation: mutating returned value doesn't affect stored
	got.Variations[0].Value = false
	original, _ := repo.GetFlag(ctx, "proj", "dev", "my-flag")
	if original.Variations[0].Value.(bool) != true {
		t.Fatal("stored variation should be unaffected by caller mutation")
	}
}

func TestMemRepo_CreateFlag_DuplicateKey(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	f := FeatureFlag{
		Project:    "proj",
		Stage:      "dev",
		Key:        "dup-flag",
		Name:       "Dup",
		Enabled:    true,
		DefaultKey: "on",
		Variations: []Variation{
			{Key: "on", Type: BooleanVariation, Value: true},
		},
	}

	if _, err := repo.CreateFlag(ctx, f); err != nil {
		t.Fatalf("first CreateFlag: %v", err)
	}

	_, err := repo.CreateFlag(ctx, f)
	if !errors.Is(err, ErrInvalidFlag) {
		t.Fatalf("expected ErrInvalidFlag on duplicate, got %v", err)
	}
}

func TestMemRepo_GetFlag_NotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	_, err := repo.GetFlag(ctx, "proj", "dev", "nonexistent")
	if !errors.Is(err, ErrFlagNotFound) {
		t.Fatalf("expected ErrFlagNotFound, got %v", err)
	}

	_, err = repo.GetFlagByID(ctx, 999)
	if !errors.Is(err, ErrFlagNotFound) {
		t.Fatalf("expected ErrFlagNotFound from GetFlagByID, got %v", err)
	}
}

func TestMemRepo_UpdateFlag(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	f := FeatureFlag{
		Project:    "proj",
		Stage:      "dev",
		Key:        "upd-flag",
		Name:       "Original",
		Enabled:    true,
		DefaultKey: "on",
		Variations: []Variation{
			{Key: "on", Type: BooleanVariation, Value: true},
			{Key: "off", Type: BooleanVariation, Value: false},
		},
	}

	created, err := repo.CreateFlag(ctx, f)
	if err != nil {
		t.Fatalf("CreateFlag: %v", err)
	}

	// Update the name
	created.Name = "Updated"
	if err := repo.UpdateFlag(ctx, created); err != nil {
		t.Fatalf("UpdateFlag: %v", err)
	}

	got, err := repo.GetFlagByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetFlagByID: %v", err)
	}

	if got.Name != "Updated" {
		t.Fatalf("Name: got %q, want %q", got.Name, "Updated")
	}

	// CreatedAt must be preserved
	if !got.CreatedAt.Equal(created.CreatedAt) {
		t.Fatalf("CreatedAt changed: got %v, want %v", got.CreatedAt, created.CreatedAt)
	}

	// UpdatedAt must be at or after CreatedAt (may be same instant in fast tests)
	if got.UpdatedAt.Before(created.CreatedAt) {
		t.Fatalf("UpdatedAt should not be before CreatedAt")
	}

	// project/stage/key must remain immutable (lookup by original key still works)
	got2, err := repo.GetFlag(ctx, "proj", "dev", "upd-flag")
	if err != nil {
		t.Fatalf("GetFlag by original key: %v", err)
	}
	if got2.Name != "Updated" {
		t.Fatalf("GetFlag should still find updated flag via original key")
	}
}

func TestMemRepo_UpdateFlag_NotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	err := repo.UpdateFlag(ctx, FeatureFlag{ID: 999})
	if !errors.Is(err, ErrFlagNotFound) {
		t.Fatalf("expected ErrFlagNotFound, got %v", err)
	}
}

func TestMemRepo_DeleteFlag(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	f := FeatureFlag{
		Project:    "proj",
		Stage:      "dev",
		Key:        "del-flag",
		Name:       "To Delete",
		Enabled:    true,
		DefaultKey: "on",
		Variations: []Variation{
			{Key: "on", Type: BooleanVariation, Value: true},
		},
	}

	created, err := repo.CreateFlag(ctx, f)
	if err != nil {
		t.Fatalf("CreateFlag: %v", err)
	}

	if err := repo.DeleteFlag(ctx, created.ID); err != nil {
		t.Fatalf("DeleteFlag: %v", err)
	}

	_, err = repo.GetFlagByID(ctx, created.ID)
	if !errors.Is(err, ErrFlagNotFound) {
		t.Fatalf("expected ErrFlagNotFound after delete, got %v", err)
	}

	_, err = repo.GetFlag(ctx, "proj", "dev", "del-flag")
	if !errors.Is(err, ErrFlagNotFound) {
		t.Fatalf("expected ErrFlagNotFound via composite key after delete, got %v", err)
	}
}

func TestMemRepo_DeleteFlag_NotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	err := repo.DeleteFlag(ctx, 999)
	if !errors.Is(err, ErrFlagNotFound) {
		t.Fatalf("expected ErrFlagNotFound, got %v", err)
	}
}

func TestMemRepo_ListFlags(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	// Empty list returns empty slice, not nil
	list, err := repo.ListFlags(ctx, "proj", "dev")
	if err != nil {
		t.Fatalf("ListFlags: %v", err)
	}
	if list == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 flags, got %d", len(list))
	}

	// Create flags in different project/stages
	for _, f := range []FeatureFlag{
		{Project: "proj", Stage: "dev", Key: "a", Name: "A", Enabled: true, DefaultKey: "on",
			Variations: []Variation{{Key: "on", Type: BooleanVariation, Value: true}}},
		{Project: "proj", Stage: "dev", Key: "b", Name: "B", Enabled: true, DefaultKey: "on",
			Variations: []Variation{{Key: "on", Type: BooleanVariation, Value: true}}},
		{Project: "proj", Stage: "prod", Key: "c", Name: "C", Enabled: true, DefaultKey: "on",
			Variations: []Variation{{Key: "on", Type: BooleanVariation, Value: true}}},
		{Project: "other", Stage: "dev", Key: "d", Name: "D", Enabled: true, DefaultKey: "on",
			Variations: []Variation{{Key: "on", Type: BooleanVariation, Value: true}}},
	} {
		if _, err := repo.CreateFlag(ctx, f); err != nil {
			t.Fatalf("CreateFlag %s: %v", f.Key, err)
		}
	}

	list, err = repo.ListFlags(ctx, "proj", "dev")
	if err != nil {
		t.Fatalf("ListFlags: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 flags for proj/dev, got %d", len(list))
	}

	// Should only see proj/prod
	list, err = repo.ListFlags(ctx, "proj", "prod")
	if err != nil {
		t.Fatalf("ListFlags: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 flag for proj/prod, got %d", len(list))
	}
}

// --- Range CRUD tests ---

func TestMemRepo_CreateAndGetRange(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	f := createTestFlag(t, repo)

	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	later := now.Add(24 * time.Hour)

	r, err := repo.CreateRange(ctx, FlagRange{
		FlagID:    f.ID,
		ValidFrom: now,
		ValidTo:   &later,
	})
	if err != nil {
		t.Fatalf("CreateRange: %v", err)
	}

	if r.ID == 0 {
		t.Fatal("expected non-zero range ID")
	}
	if r.Active {
		t.Fatal("new range should start inactive")
	}
	if r.CreatedAt.IsZero() {
		t.Fatal("expected CreatedAt to be set")
	}

	// GetRange
	got, err := repo.GetRange(ctx, r.ID)
	if err != nil {
		t.Fatalf("GetRange: %v", err)
	}
	if got.FlagID != f.ID {
		t.Fatalf("FlagID: got %d, want %d", got.FlagID, f.ID)
	}
	if !got.ValidFrom.Equal(now) {
		t.Fatalf("ValidFrom: got %v, want %v", got.ValidFrom, now)
	}
	if got.ValidTo == nil || !got.ValidTo.Equal(later) {
		t.Fatal("ValidTo mismatch")
	}
}

func TestMemRepo_CreateRange_FlagNotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	_, err := repo.CreateRange(ctx, FlagRange{
		FlagID:    999,
		ValidFrom: time.Now().UTC(),
	})
	if !errors.Is(err, ErrFlagNotFound) {
		t.Fatalf("expected ErrFlagNotFound, got %v", err)
	}
}

func TestMemRepo_CreateRange_InvalidTimeRange(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	f := createTestFlag(t, repo)

	now := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	before := now.Add(-time.Hour)

	_, err := repo.CreateRange(ctx, FlagRange{
		FlagID:    f.ID,
		ValidFrom: now,
		ValidTo:   &before,
	})
	if !errors.Is(err, ErrInvalidTimeRange) {
		t.Fatalf("expected ErrInvalidTimeRange, got %v", err)
	}
}

func TestMemRepo_RangeOverlap(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	f := createTestFlag(t, repo)

	t1 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)
	t3 := t2.Add(24 * time.Hour)

	// Create range [t1, t2)
	_, err := repo.CreateRange(ctx, FlagRange{
		FlagID:    f.ID,
		ValidFrom: t1,
		ValidTo:   &t2,
	})
	if err != nil {
		t.Fatalf("CreateRange 1: %v", err)
	}

	// Overlapping range [t1+12h, t3) should fail
	mid := t1.Add(12 * time.Hour)
	_, err = repo.CreateRange(ctx, FlagRange{
		FlagID:    f.ID,
		ValidFrom: mid,
		ValidTo:   &t3,
	})
	if !errors.Is(err, ErrRangeOverlap) {
		t.Fatalf("expected ErrRangeOverlap, got %v", err)
	}

	// Non-overlapping range [t2, t3) should succeed
	_, err = repo.CreateRange(ctx, FlagRange{
		FlagID:    f.ID,
		ValidFrom: t2,
		ValidTo:   &t3,
	})
	if err != nil {
		t.Fatalf("CreateRange non-overlapping: %v", err)
	}

	// CheckRangeOverlap public method
	overlap, err := repo.CheckRangeOverlap(ctx, f.ID, t1, &t2, 0)
	if err != nil {
		t.Fatalf("CheckRangeOverlap: %v", err)
	}
	if !overlap {
		t.Fatal("expected overlap to be true")
	}
}

func TestMemRepo_GetActiveRange(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	f := createTestFlag(t, repo)

	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	validFrom := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	validTo := time.Date(2025, 6, 2, 0, 0, 0, 0, time.UTC)

	r, err := repo.CreateRange(ctx, FlagRange{
		FlagID:    f.ID,
		ValidFrom: validFrom,
		ValidTo:   &validTo,
	})
	if err != nil {
		t.Fatalf("CreateRange: %v", err)
	}

	// Range is inactive by default — GetActiveRange should fail
	_, err = repo.GetActiveRange(ctx, f.ID, now)
	if !errors.Is(err, ErrRangeNotFound) {
		t.Fatalf("expected ErrRangeNotFound for inactive range, got %v", err)
	}

	// Activate it
	if err := repo.ActivateRange(ctx, r.ID); err != nil {
		t.Fatalf("ActivateRange: %v", err)
	}

	// Now GetActiveRange should find it
	active, err := repo.GetActiveRange(ctx, f.ID, now)
	if err != nil {
		t.Fatalf("GetActiveRange: %v", err)
	}
	if active.ID != r.ID {
		t.Fatalf("active range ID: got %d, want %d", active.ID, r.ID)
	}

	// Outside the range — should not find
	outside := time.Date(2025, 6, 3, 0, 0, 0, 0, time.UTC)
	_, err = repo.GetActiveRange(ctx, f.ID, outside)
	if !errors.Is(err, ErrRangeNotFound) {
		t.Fatalf("expected ErrRangeNotFound for outside time, got %v", err)
	}

	// Deactivate and verify
	if err := repo.DeactivateRange(ctx, r.ID); err != nil {
		t.Fatalf("DeactivateRange: %v", err)
	}
	_, err = repo.GetActiveRange(ctx, f.ID, now)
	if !errors.Is(err, ErrRangeNotFound) {
		t.Fatalf("expected ErrRangeNotFound after deactivate, got %v", err)
	}
}

func TestMemRepo_UpdateRange(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	f := createTestFlag(t, repo)

	t1 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	t2 := t1.Add(24 * time.Hour)
	t3 := t2.Add(24 * time.Hour)

	r, err := repo.CreateRange(ctx, FlagRange{
		FlagID:    f.ID,
		ValidFrom: t1,
		ValidTo:   &t2,
	})
	if err != nil {
		t.Fatalf("CreateRange: %v", err)
	}

	// Update to extend the range
	r.ValidTo = &t3
	if err := repo.UpdateRange(ctx, r); err != nil {
		t.Fatalf("UpdateRange: %v", err)
	}

	got, err := repo.GetRange(ctx, r.ID)
	if err != nil {
		t.Fatalf("GetRange: %v", err)
	}
	if !got.ValidTo.Equal(t3) {
		t.Fatalf("ValidTo: got %v, want %v", *got.ValidTo, t3)
	}
	// CreatedAt preserved
	if !got.CreatedAt.Equal(r.CreatedAt) {
		t.Fatalf("CreatedAt changed")
	}
}

func TestMemRepo_UpdateRange_NotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	err := repo.UpdateRange(context.Background(), FlagRange{ID: 999})
	if !errors.Is(err, ErrRangeNotFound) {
		t.Fatalf("expected ErrRangeNotFound, got %v", err)
	}
}

func TestMemRepo_DeleteRange(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	_, r := createTestFlagAndRange(t, repo)

	if err := repo.DeleteRange(ctx, r.ID); err != nil {
		t.Fatalf("DeleteRange: %v", err)
	}

	_, err := repo.GetRange(ctx, r.ID)
	if !errors.Is(err, ErrRangeNotFound) {
		t.Fatalf("expected ErrRangeNotFound after delete, got %v", err)
	}
}

func TestMemRepo_DeleteRange_NotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	err := repo.DeleteRange(context.Background(), 999)
	if !errors.Is(err, ErrRangeNotFound) {
		t.Fatalf("expected ErrRangeNotFound, got %v", err)
	}
}

func TestMemRepo_ListRanges(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()
	f := createTestFlag(t, repo)

	// Empty list
	list, err := repo.ListRanges(ctx, f.ID)
	if err != nil {
		t.Fatalf("ListRanges: %v", err)
	}
	if list == nil {
		t.Fatal("expected non-nil empty slice")
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 ranges, got %d", len(list))
	}

	// Add two non-overlapping ranges
	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

	if _, err := repo.CreateRange(ctx, FlagRange{FlagID: f.ID, ValidFrom: t1, ValidTo: &t2}); err != nil {
		t.Fatalf("CreateRange 1: %v", err)
	}
	if _, err := repo.CreateRange(ctx, FlagRange{FlagID: f.ID, ValidFrom: t2, ValidTo: &t3}); err != nil {
		t.Fatalf("CreateRange 2: %v", err)
	}

	list, err = repo.ListRanges(ctx, f.ID)
	if err != nil {
		t.Fatalf("ListRanges: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 ranges, got %d", len(list))
	}

	// Ranges for a different flag should be empty
	list, err = repo.ListRanges(ctx, 999)
	if err != nil {
		t.Fatalf("ListRanges: %v", err)
	}
	if len(list) != 0 {
		t.Fatalf("expected 0 ranges for non-existent flag, got %d", len(list))
	}
}

func TestMemRepo_ActivateDeactivateRange_NotFound(t *testing.T) {
	repo := NewInMemoryRepository()
	ctx := context.Background()

	if err := repo.ActivateRange(ctx, 999); !errors.Is(err, ErrRangeNotFound) {
		t.Fatalf("expected ErrRangeNotFound for ActivateRange, got %v", err)
	}
	if err := repo.DeactivateRange(ctx, 999); !errors.Is(err, ErrRangeNotFound) {
		t.Fatalf("expected ErrRangeNotFound for DeactivateRange, got %v", err)
	}
}
