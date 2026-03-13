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

func TestService_CreateRange_WithDraft(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	f, _ := svc.CreateFlag(ctx, FeatureFlag{
		Project: "p", Stage: "s", Key: "k", Name: "n", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}},
	}, "admin")

	rng, v, err := svc.CreateRange(ctx, FlagRange{
		FlagID:    f.ID,
		ValidFrom: f.CreatedAt,
	}, []Rule{{ID: "r1", VariationKey: "d"}}, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if rng.ID == 0 {
		t.Error("expected non-zero range ID")
	}
	if v.Version != 1 {
		t.Errorf("expected version 1, got %d", v.Version)
	}
	if v.Status != VersionStatusDraft {
		t.Errorf("expected draft, got %q", v.Status)
	}
}

func TestService_ActivateRange_RequiresPublishedVersion(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	f, _ := svc.CreateFlag(ctx, FeatureFlag{
		Project: "p", Stage: "s", Key: "k", Name: "n", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}},
	}, "admin")

	rng, _, _ := svc.CreateRange(ctx, FlagRange{
		FlagID:    f.ID,
		ValidFrom: f.CreatedAt,
	}, []Rule{}, "admin")

	// Should fail — only has draft
	err := svc.ActivateRange(ctx, rng.ID, "admin")
	if err == nil {
		t.Error("expected error activating range without published version")
	}
}

func TestService_RollbackVersion(t *testing.T) {
	svc := newTestService(t)
	ctx := context.Background()

	f, _ := svc.CreateFlag(ctx, FeatureFlag{
		Project: "p", Stage: "s", Key: "k", Name: "n", DefaultKey: "d",
		Variations: []Variation{{Key: "d", Type: BooleanVariation, Value: false}},
	}, "admin")

	rng, v1, _ := svc.CreateRange(ctx, FlagRange{
		FlagID:    f.ID,
		ValidFrom: f.CreatedAt,
	}, []Rule{{ID: "original", VariationKey: "d"}}, "admin")

	// Publish v1
	svc.PublishVersion(ctx, v1.ID, "admin")

	// Create v2 draft with different rules
	v2, _ := svc.CreateVersion(ctx, rng.ID, []Rule{{ID: "new", VariationKey: "d"}}, "admin")

	// Rollback to v1 — should overwrite v2 draft's rules
	rolled, err := svc.RollbackVersion(ctx, v1.ID, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if rolled.ID != v2.ID {
		t.Errorf("expected rollback to update existing draft %d, got %d", v2.ID, rolled.ID)
	}
	if rolled.Rules[0].ID != "original" {
		t.Errorf("expected original rules, got %q", rolled.Rules[0].ID)
	}
}
