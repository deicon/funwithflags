package flag

import (
	"context"
	"testing"
)

func TestEngine_DefaultWhenNoRules(t *testing.T) {
	e := &Engine{}
	f := FeatureFlag{
		Key: "test", Enabled: true, DefaultKey: "off",
		Variations: []Variation{
			{Key: "off", Type: BooleanVariation, Value: false},
			{Key: "on", Type: BooleanVariation, Value: true},
		},
	}
	v := RangeVersion{Rules: []Rule{}}

	result, err := e.Evaluate(context.Background(), f, v, EvaluationContext{})
	if err != nil {
		t.Fatal(err)
	}
	if result.Reason != ReasonDefault {
		t.Errorf("got reason %q, want %q", result.Reason, ReasonDefault)
	}
	if result.Variation.Key != "off" {
		t.Errorf("got variation %q, want off", result.Variation.Key)
	}
}

func TestEngine_StaticRuleMatch(t *testing.T) {
	e := &Engine{}
	f := FeatureFlag{
		Key: "test", Enabled: true, DefaultKey: "off",
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
	if result.Reason != ReasonTargetMatch {
		t.Errorf("got reason %q, want %q", result.Reason, ReasonTargetMatch)
	}
}

func TestEngine_NoMatch_ReturnsDefault(t *testing.T) {
	e := &Engine{}
	f := FeatureFlag{
		Key: "test", Enabled: true, DefaultKey: "off",
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

	result, err := e.Evaluate(context.Background(), f, v, EvaluationContext{"country": "US"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Variation.Key != "off" {
		t.Errorf("got %q, want off", result.Variation.Key)
	}
	if result.Reason != ReasonDefault {
		t.Errorf("got reason %q, want %q", result.Reason, ReasonDefault)
	}
}
