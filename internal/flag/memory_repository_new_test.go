package flag

import (
	"context"
	"errors"
	"testing"
)

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
