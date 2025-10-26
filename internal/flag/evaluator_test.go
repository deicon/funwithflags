package flag

import (
	"context"
	"fmt"
	"hash/fnv"
	"testing"
)

func TestEngine_ReturnsDefaultWhenDisabled(t *testing.T) {
	engine := NewEngine()
	flag := FeatureFlag{
		Key:        "checkout",
		Enabled:    false,
		DefaultKey: "control",
		Variations: []Variation{
			{Key: "control", Type: BooleanVariation, Value: false},
			{Key: "variant", Type: BooleanVariation, Value: true},
		},
	}

	result, err := engine.Evaluate(context.Background(), flag, EvaluationContext{"country": "DE"})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if result.Variation.Key != "control" {
		t.Fatalf("expected default variation, got %s", result.Variation.Key)
	}
	if result.Reason != ReasonDisabled {
		t.Fatalf("expected reason %s, got %s", ReasonDisabled, result.Reason)
	}
}

func TestEngine_StaticRuleMatch(t *testing.T) {
	engine := NewEngine()
	flag := FeatureFlag{
		Key:        "search-layout",
		Enabled:    true,
		DefaultKey: "control",
		Variations: []Variation{
			{Key: "control", Type: StringVariation, Value: "legacy"},
			{Key: "experiment", Type: StringVariation, Value: "modern"},
		},
		Rules: []Rule{
			{
				ID:           "vip-eu",
				VariationKey: "experiment",
				Conditions: []Condition{
					{Attribute: "country", Operator: MatcherEquals, Value: "DE"},
					{Attribute: "plan", Operator: MatcherExists},
					{Attribute: "tenure", Operator: MatcherGreater, Value: 12},
					{Attribute: "segment", Operator: MatcherIn, Value: []string{"beta", "vip"}},
				},
			},
		},
	}

	attrs := EvaluationContext{
		"country": "DE",
		"plan":    "pro",
		"tenure":  18,
		"segment": "vip",
	}

	result, err := engine.Evaluate(context.Background(), flag, attrs)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if result.Variation.Key != "experiment" {
		t.Fatalf("expected experiment variation, got %s", result.Variation.Key)
	}
	if result.Reason != ReasonTargetMatch {
		t.Fatalf("expected reason %s, got %s", ReasonTargetMatch, result.Reason)
	}
}

func TestEngine_ReturnsDefaultWhenNoRuleMatches(t *testing.T) {
	engine := NewEngine()
	flag := FeatureFlag{
		Key:        "search-layout",
		Enabled:    true,
		DefaultKey: "control",
		Variations: []Variation{
			{Key: "control", Type: StringVariation, Value: "legacy"},
			{Key: "experiment", Type: StringVariation, Value: "modern"},
		},
		Rules: []Rule{
			{
				ID:           "vip-eu",
				VariationKey: "experiment",
				Conditions: []Condition{
					{Attribute: "country", Operator: MatcherEquals, Value: "DE"},
				},
			},
		},
	}

	attrs := EvaluationContext{
		"country": "US",
	}

	result, err := engine.Evaluate(context.Background(), flag, attrs)
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if result.Variation.Key != "control" {
		t.Fatalf("expected default variation, got %s", result.Variation.Key)
	}
	if result.Reason != ReasonDefault {
		t.Fatalf("expected reason %s, got %s", ReasonDefault, result.Reason)
	}
}

func TestEngine_PercentageRolloutDeterministic(t *testing.T) {
	engine := NewEngine()
	flag := FeatureFlag{
		Key:        "feed-algorithm",
		Enabled:    true,
		DefaultKey: "control",
		Variations: []Variation{
			{Key: "control", Type: StringVariation, Value: "classic"},
			{Key: "ml", Type: StringVariation, Value: "personalized"},
		},
		Rules: []Rule{
			{
				ID: "emea-rollout",
				Conditions: []Condition{
					{Attribute: "region", Operator: MatcherEquals, Value: "EMEA"},
				},
				Rollout: &PercentageRollout{
					Attribute: "user_id",
					Seed:      "feed-algorithm",
					Buckets: []RolloutBucket{
						{VariationKey: "control", Weight: 40},
						{VariationKey: "ml", Weight: 60},
					},
				},
			},
		},
	}

	attrs := EvaluationContext{
		"region":  "EMEA",
		"user_id": "user-123",
	}

	expected := expectedRolloutVariation(flag.Rules[0].Rollout, attrs["user_id"])

	first, err := engine.Evaluate(context.Background(), flag, attrs)
	if err != nil {
		t.Fatalf("Evaluate first: %v", err)
	}
	second, err := engine.Evaluate(context.Background(), flag, attrs)
	if err != nil {
		t.Fatalf("Evaluate second: %v", err)
	}

	if first.Variation.Key != expected {
		t.Fatalf("expected variation %s from rollout, got %s", expected, first.Variation.Key)
	}
	if first.Reason != ReasonPercentageRollout {
		t.Fatalf("expected reason %s, got %s", ReasonPercentageRollout, first.Reason)
	}
	if second.Variation.Key != first.Variation.Key {
		t.Fatalf("expected deterministic rollout, got %s and %s", first.Variation.Key, second.Variation.Key)
	}
}

func expectedRolloutVariation(rollout *PercentageRollout, attrValue any) string {
	hashInput := fmt.Sprintf("%s:%v", rollout.Seed, attrValue)
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(hashInput))

	point := float64(hasher.Sum32()%10000) / 100.0
	var cumulative float64
	for _, bucket := range rollout.Buckets {
		cumulative += bucket.Weight
		if point < cumulative {
			return bucket.VariationKey
		}
	}
	return rollout.Buckets[len(rollout.Buckets)-1].VariationKey
}
