package flag

import (
	"encoding/json"
	"testing"
	"time"
)

func TestFeatureFlagJSON(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name string
		flag FeatureFlag
	}{
		{
			name: "minimal flag",
			flag: FeatureFlag{
				ID:      1,
				Project: "proj",
				Stage:   "prod",
				Key:     "my-flag",
				Name:    "My Flag",
				Enabled: true,
				DefaultKey: "on",
				Variations: []Variation{
					{Key: "on", Type: BooleanVariation, Value: true},
					{Key: "off", Type: BooleanVariation, Value: false},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "flag with description",
			flag: FeatureFlag{
				ID:          2,
				Project:     "proj",
				Stage:       "staging",
				Key:         "banner-color",
				Name:        "Banner Color",
				Description: "Controls the banner color",
				Enabled:     false,
				DefaultKey:  "blue",
				Variations: []Variation{
					{Key: "blue", Type: StringVariation, Value: "#0000FF", Description: "Blue banner"},
					{Key: "red", Type: StringVariation, Value: "#FF0000"},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "flag with number variations",
			flag: FeatureFlag{
				ID:         3,
				Project:    "proj",
				Stage:      "dev",
				Key:        "rate-limit",
				Name:       "Rate Limit",
				Enabled:    true,
				DefaultKey: "default",
				Variations: []Variation{
					{Key: "default", Type: NumberVariation, Value: float64(100)},
					{Key: "high", Type: NumberVariation, Value: float64(1000)},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "flag with object variation",
			flag: FeatureFlag{
				ID:         4,
				Project:    "proj",
				Stage:      "prod",
				Key:        "config",
				Name:       "Config",
				Enabled:    true,
				DefaultKey: "v1",
				Variations: []Variation{
					{Key: "v1", Type: ObjectVariation, Value: map[string]any{"timeout": float64(30)}},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.flag)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			var got FeatureFlag
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			if got.ID != tt.flag.ID {
				t.Errorf("ID: got %d, want %d", got.ID, tt.flag.ID)
			}
			if got.Project != tt.flag.Project {
				t.Errorf("Project: got %q, want %q", got.Project, tt.flag.Project)
			}
			if got.Stage != tt.flag.Stage {
				t.Errorf("Stage: got %q, want %q", got.Stage, tt.flag.Stage)
			}
			if got.Key != tt.flag.Key {
				t.Errorf("Key: got %q, want %q", got.Key, tt.flag.Key)
			}
			if got.Name != tt.flag.Name {
				t.Errorf("Name: got %q, want %q", got.Name, tt.flag.Name)
			}
			if got.Description != tt.flag.Description {
				t.Errorf("Description: got %q, want %q", got.Description, tt.flag.Description)
			}
			if got.Enabled != tt.flag.Enabled {
				t.Errorf("Enabled: got %v, want %v", got.Enabled, tt.flag.Enabled)
			}
			if got.DefaultKey != tt.flag.DefaultKey {
				t.Errorf("DefaultKey: got %q, want %q", got.DefaultKey, tt.flag.DefaultKey)
			}
			if len(got.Variations) != len(tt.flag.Variations) {
				t.Errorf("Variations length: got %d, want %d", len(got.Variations), len(tt.flag.Variations))
			}

			// Verify old fields are absent from JSON
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal raw error: %v", err)
			}
			for _, removed := range []string{"active", "validFrom", "validTo", "rules"} {
				if _, ok := raw[removed]; ok {
					t.Errorf("JSON should not contain %q field", removed)
				}
			}
		})
	}
}

func TestFlagRangeJSON(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	later := now.Add(24 * time.Hour)

	tests := []struct {
		name  string
		r     FlagRange
		hasTo bool
	}{
		{
			name: "open-ended range",
			r: FlagRange{
				ID:        1,
				FlagID:    10,
				Active:    true,
				ValidFrom: now,
				ValidTo:   nil,
				CreatedAt: now,
				UpdatedAt: now,
			},
			hasTo: false,
		},
		{
			name: "bounded range",
			r: FlagRange{
				ID:        2,
				FlagID:    10,
				Active:    false,
				ValidFrom: now,
				ValidTo:   &later,
				CreatedAt: now,
				UpdatedAt: now,
			},
			hasTo: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.r)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			var got FlagRange
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			if got.ID != tt.r.ID {
				t.Errorf("ID: got %d, want %d", got.ID, tt.r.ID)
			}
			if got.FlagID != tt.r.FlagID {
				t.Errorf("FlagID: got %d, want %d", got.FlagID, tt.r.FlagID)
			}
			if got.Active != tt.r.Active {
				t.Errorf("Active: got %v, want %v", got.Active, tt.r.Active)
			}
			if !got.ValidFrom.Equal(tt.r.ValidFrom) {
				t.Errorf("ValidFrom: got %v, want %v", got.ValidFrom, tt.r.ValidFrom)
			}

			// Check ValidTo presence in JSON
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal raw error: %v", err)
			}
			_, hasValidTo := raw["validTo"]
			if hasValidTo != tt.hasTo {
				t.Errorf("validTo presence: got %v, want %v", hasValidTo, tt.hasTo)
			}

			if tt.hasTo {
				if got.ValidTo == nil {
					t.Fatal("ValidTo should not be nil")
				}
				if !got.ValidTo.Equal(*tt.r.ValidTo) {
					t.Errorf("ValidTo: got %v, want %v", *got.ValidTo, *tt.r.ValidTo)
				}
			} else {
				if got.ValidTo != nil {
					t.Errorf("ValidTo should be nil, got %v", *got.ValidTo)
				}
			}
		})
	}
}

func TestRangeVersionJSON(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name    string
		version RangeVersion
	}{
		{
			name: "draft with no rules",
			version: RangeVersion{
				ID:        1,
				RangeID:   100,
				Version:   1,
				Status:    VersionStatusDraft,
				Rules:     []Rule{},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "published with rules",
			version: RangeVersion{
				ID:      2,
				RangeID: 100,
				Version: 2,
				Status:  VersionStatusPublished,
				Rules: []Rule{
					{
						ID:          "rule-1",
						Description: "beta users",
						Conditions: []Condition{
							{
								Attribute: "userGroup",
								Operator:  MatcherEquals,
								Value:     "beta",
							},
						},
						VariationKey: "on",
					},
					{
						ID: "rule-2",
						Rollout: &PercentageRollout{
							Attribute: "userId",
							Seed:      "experiment-1",
							Buckets: []RolloutBucket{
								{VariationKey: "on", Weight: 0.5},
								{VariationKey: "off", Weight: 0.5},
							},
						},
					},
				},
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.version)
			if err != nil {
				t.Fatalf("marshal error: %v", err)
			}

			var got RangeVersion
			if err := json.Unmarshal(data, &got); err != nil {
				t.Fatalf("unmarshal error: %v", err)
			}

			if got.ID != tt.version.ID {
				t.Errorf("ID: got %d, want %d", got.ID, tt.version.ID)
			}
			if got.RangeID != tt.version.RangeID {
				t.Errorf("RangeID: got %d, want %d", got.RangeID, tt.version.RangeID)
			}
			if got.Version != tt.version.Version {
				t.Errorf("Version: got %d, want %d", got.Version, tt.version.Version)
			}
			if got.Status != tt.version.Status {
				t.Errorf("Status: got %q, want %q", got.Status, tt.version.Status)
			}
			if len(got.Rules) != len(tt.version.Rules) {
				t.Fatalf("Rules length: got %d, want %d", len(got.Rules), len(tt.version.Rules))
			}

			// Verify status serialization
			var raw map[string]json.RawMessage
			if err := json.Unmarshal(data, &raw); err != nil {
				t.Fatalf("unmarshal raw error: %v", err)
			}
			var status string
			if err := json.Unmarshal(raw["status"], &status); err != nil {
				t.Fatalf("unmarshal status error: %v", err)
			}
			if status != string(tt.version.Status) {
				t.Errorf("status JSON value: got %q, want %q", status, tt.version.Status)
			}

			// For the version with rules, verify rule details survived round-trip
			if len(got.Rules) > 0 {
				if got.Rules[0].ID != tt.version.Rules[0].ID {
					t.Errorf("Rule[0].ID: got %q, want %q", got.Rules[0].ID, tt.version.Rules[0].ID)
				}
				if got.Rules[0].VariationKey != tt.version.Rules[0].VariationKey {
					t.Errorf("Rule[0].VariationKey: got %q, want %q", got.Rules[0].VariationKey, tt.version.Rules[0].VariationKey)
				}
			}

			if len(got.Rules) > 1 && got.Rules[1].Rollout != nil {
				if got.Rules[1].Rollout.Seed != tt.version.Rules[1].Rollout.Seed {
					t.Errorf("Rule[1].Rollout.Seed: got %q, want %q", got.Rules[1].Rollout.Seed, tt.version.Rules[1].Rollout.Seed)
				}
				if len(got.Rules[1].Rollout.Buckets) != len(tt.version.Rules[1].Rollout.Buckets) {
					t.Errorf("Rule[1].Rollout.Buckets length: got %d, want %d",
						len(got.Rules[1].Rollout.Buckets), len(tt.version.Rules[1].Rollout.Buckets))
				}
			}
		})
	}
}
