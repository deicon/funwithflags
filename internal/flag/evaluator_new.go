package flag

import (
	"context"
	"fmt"
	"hash/fnv"
	"reflect"
	"strings"
)

type Engine struct{}

func (e *Engine) Evaluate(ctx context.Context, flag FeatureFlag, v RangeVersion, attrs EvaluationContext) (EvaluationResult, error) {
	variationMap := buildVariationMap(flag.Variations)
	if !flag.Enabled {
		return buildResultFromKey(flag.DefaultKey, variationMap, ReasonDisabled)
	}

	for _, rule := range v.Rules {
		result, matched, err := evaluateRule(rule, attrs, variationMap)
		if err != nil {
			return EvaluationResult{}, err
		}
		if matched {
			return result, nil
		}
	}

	return buildResultFromKey(flag.DefaultKey, variationMap, ReasonDefault)
}

func evaluateRule(rule Rule, attrs EvaluationContext, variations map[string]Variation) (EvaluationResult, bool, error) {
	matched, err := conditionsMatch(rule.Conditions, attrs)
	if err != nil || !matched {
		return EvaluationResult{}, matched, err
	}

	if rule.Rollout != nil {
		return evaluateRollout(rule.Rollout, attrs, variations)
	}

	result, err := buildResultFromKey(rule.VariationKey, variations, ReasonTargetMatch)
	if err != nil {
		return EvaluationResult{}, false, err
	}
	return result, true, nil
}

func buildVariationMap(variations []Variation) map[string]Variation {
	m := make(map[string]Variation, len(variations))
	for _, variation := range variations {
		m[variation.Key] = variation
	}
	return m
}

func buildResultFromKey(key string, variations map[string]Variation, reason string) (EvaluationResult, error) {
	variation, ok := variations[key]
	if !ok {
		return EvaluationResult{}, fmt.Errorf("variation %q not defined", key)
	}
	return EvaluationResult{Variation: variation, Reason: reason}, nil
}

func conditionsMatch(conditions []Condition, attrs EvaluationContext) (bool, error) {
	for _, condition := range conditions {
		value, exists := attrs[condition.Attribute]
		switch condition.Operator {
		case MatcherExists:
			if !exists {
				return false, nil
			}
		case MatcherEquals:
			if !exists || !equals(value, condition.Value) {
				return false, nil
			}
		case MatcherNotEquals:
			if !exists || equals(value, condition.Value) {
				return false, nil
			}
		case MatcherContains:
			if !exists || !contains(value, condition.Value) {
				return false, nil
			}
		case MatcherStartsWith:
			if !exists || !startsWith(value, condition.Value) {
				return false, nil
			}
		case MatcherEndsWith:
			if !exists || !endsWith(value, condition.Value) {
				return false, nil
			}
		case MatcherGreater:
			if !exists || !compareNumeric(value, condition.Value, func(a, b float64) bool { return a > b }) {
				return false, nil
			}
		case MatcherLess:
			if !exists || !compareNumeric(value, condition.Value, func(a, b float64) bool { return a < b }) {
				return false, nil
			}
		case MatcherIn:
			if !exists || !inCollection(value, condition.Value) {
				return false, nil
			}
		default:
			return false, fmt.Errorf("unsupported operator %q", condition.Operator)
		}
	}
	return true, nil
}

func equals(lhs, rhs any) bool {
	return reflect.DeepEqual(normalize(lhs), normalize(rhs))
}

func normalize(value any) any {
	switch v := value.(type) {
	case int:
		return float64(v)
	case int8:
		return float64(v)
	case int16:
		return float64(v)
	case int32:
		return float64(v)
	case int64:
		return float64(v)
	case uint:
		return float64(v)
	case uint8:
		return float64(v)
	case uint16:
		return float64(v)
	case uint32:
		return float64(v)
	case uint64:
		return float64(v)
	case float32:
		return float64(v)
	default:
		return v
	}
}

func contains(attr, value any) bool {
	attrStr, ok := attr.(string)
	if !ok {
		return false
	}
	valueStr, ok := value.(string)
	if !ok {
		return false
	}

	return strings.Contains(attrStr, valueStr)
}

func startsWith(attr, value any) bool {
	attrStr, ok := attr.(string)
	if !ok {
		return false
	}
	valueStr, ok := value.(string)
	if !ok {
		return false
	}
	return strings.HasPrefix(attrStr, valueStr)
}

func endsWith(attr, value any) bool {
	attrStr, ok := attr.(string)
	if !ok {
		return false
	}
	valueStr, ok := value.(string)
	if !ok {
		return false
	}
	return strings.HasSuffix(attrStr, valueStr)
}

func compareNumeric(lhs, rhs any, comparator func(a, b float64) bool) bool {
	left, ok := toFloat64(lhs)
	if !ok {
		return false
	}
	right, ok := toFloat64(rhs)
	if !ok {
		return false
	}
	return comparator(left, right)
}

func toFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case int:
		return float64(v), true
	case int8:
		return float64(v), true
	case int16:
		return float64(v), true
	case int32:
		return float64(v), true
	case int64:
		return float64(v), true
	case uint:
		return float64(v), true
	case uint8:
		return float64(v), true
	case uint16:
		return float64(v), true
	case uint32:
		return float64(v), true
	case uint64:
		return float64(v), true
	case float32:
		return float64(v), true
	case float64:
		return v, true
	default:
		return 0, false
	}
}

func inCollection(value any, collection any) bool {
	col := reflect.ValueOf(collection)
	if !col.IsValid() {
		return false
	}
	if col.Kind() != reflect.Slice && col.Kind() != reflect.Array {
		return false
	}
	for i := 0; i < col.Len(); i++ {
		if equals(value, col.Index(i).Interface()) {
			return true
		}
	}
	return false
}

func evaluateRollout(rollout *PercentageRollout, attrs EvaluationContext, variations map[string]Variation) (EvaluationResult, bool, error) {
	value, ok := attrs[rollout.Attribute]
	if !ok {
		return EvaluationResult{}, false, nil
	}

	hashInput := fmt.Sprintf("%s:%v", rollout.Seed, value)
	hasher := fnv.New32a()
	if _, err := hasher.Write([]byte(hashInput)); err != nil {
		return EvaluationResult{}, false, fmt.Errorf("hash rollout seed: %w", err)
	}

	point := float64(hasher.Sum32()%10000) / 100.0
	var cumulative float64
	for _, bucket := range rollout.Buckets {
		cumulative += bucket.Weight
		if point < cumulative {
			result, err := buildResultFromKey(bucket.VariationKey, variations, ReasonPercentageRollout)
			if err != nil {
				return EvaluationResult{}, false, err
			}
			return result, true, nil
		}
	}

	last := rollout.Buckets[len(rollout.Buckets)-1]
	result, err := buildResultFromKey(last.VariationKey, variations, ReasonPercentageRollout)
	if err != nil {
		return EvaluationResult{}, false, err
	}
	return result, true, nil
}
