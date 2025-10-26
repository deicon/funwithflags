package flag

import (
	"context"
	"errors"
	"math"
	"testing"

	"github.com/open-feature/go-sdk/openfeature"
)

type serviceProvider struct {
	service  *Service
	project  string
	stage    string
	metadata openfeature.Metadata
}

func newServiceProvider(svc *Service, project, stage string) *serviceProvider {
	return &serviceProvider{
		service:  svc,
		project:  project,
		stage:    stage,
		metadata: openfeature.Metadata{Name: "funwithflags-service"},
	}
}

func (p *serviceProvider) Metadata() openfeature.Metadata {
	return p.metadata
}

func (p *serviceProvider) Hooks() []openfeature.Hook {
	return nil
}

func (p *serviceProvider) BooleanEvaluation(ctx context.Context, flagKey string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	result, detail, err := p.evaluate(ctx, flagKey, flatCtx)
	if err != nil {
		return openfeature.BoolResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	value, ok := result.Variation.Value.(bool)
	if !ok {
		detail.ResolutionError = openfeature.NewTypeMismatchResolutionError("variation value is not boolean")
		detail.Reason = openfeature.ErrorReason
		return openfeature.BoolResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	return openfeature.BoolResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}

func (p *serviceProvider) StringEvaluation(ctx context.Context, flagKey string, defaultValue string, flatCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	result, detail, err := p.evaluate(ctx, flagKey, flatCtx)
	if err != nil {
		return openfeature.StringResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	value, ok := result.Variation.Value.(string)
	if !ok {
		detail.ResolutionError = openfeature.NewTypeMismatchResolutionError("variation value is not string")
		detail.Reason = openfeature.ErrorReason
		return openfeature.StringResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	return openfeature.StringResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}

func (p *serviceProvider) FloatEvaluation(ctx context.Context, flagKey string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	result, detail, err := p.evaluate(ctx, flagKey, flatCtx)
	if err != nil {
		return openfeature.FloatResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	value, ok := toFloat64(result.Variation.Value)
	if !ok {
		detail.ResolutionError = openfeature.NewTypeMismatchResolutionError("variation value is not float")
		detail.Reason = openfeature.ErrorReason
		return openfeature.FloatResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	return openfeature.FloatResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}

func (p *serviceProvider) IntEvaluation(ctx context.Context, flagKey string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	result, detail, err := p.evaluate(ctx, flagKey, flatCtx)
	if err != nil {
		return openfeature.IntResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	value, ok := toInt64(result.Variation.Value)
	if !ok {
		detail.ResolutionError = openfeature.NewTypeMismatchResolutionError("variation value is not int")
		detail.Reason = openfeature.ErrorReason
		return openfeature.IntResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	return openfeature.IntResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}

func (p *serviceProvider) ObjectEvaluation(ctx context.Context, flagKey string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	result, detail, err := p.evaluate(ctx, flagKey, flatCtx)
	if err != nil {
		return openfeature.InterfaceResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	return openfeature.InterfaceResolutionDetail{Value: result.Variation.Value, ProviderResolutionDetail: detail}
}

func (p *serviceProvider) evaluate(ctx context.Context, flagKey string, flatCtx openfeature.FlattenedContext) (EvaluationResult, openfeature.ProviderResolutionDetail, error) {
	if p.service == nil {
		return EvaluationResult{}, openfeature.ProviderResolutionDetail{
			ResolutionError: openfeature.NewGeneralResolutionError("service not configured"),
			Reason:          openfeature.ErrorReason,
		}, errors.New("service not configured")
	}

	attrs := make(EvaluationContext, len(flatCtx))
	for k, v := range flatCtx {
		attrs[k] = v
	}

	res, err := p.service.EvaluateFlag(ctx, p.project, p.stage, flagKey, attrs)
	if err != nil {
		detail := openfeature.ProviderResolutionDetail{
			Reason: openfeature.ErrorReason,
		}
		switch {
		case errors.Is(err, ErrFlagNotFound):
			detail.ResolutionError = openfeature.NewFlagNotFoundResolutionError(err.Error())
		case errors.Is(err, ErrInvalidFlag):
			detail.ResolutionError = openfeature.NewParseErrorResolutionError(err.Error())
		default:
			detail.ResolutionError = openfeature.NewGeneralResolutionError(err.Error())
		}
		return EvaluationResult{}, detail, err
	}

	return res, openfeature.ProviderResolutionDetail{
		Reason:  mapReason(res.Reason),
		Variant: res.Variation.Key,
	}, nil
}

func mapReason(reason string) openfeature.Reason {
	switch reason {
	case ReasonTargetMatch:
		return openfeature.TargetingMatchReason
	case ReasonPercentageRollout:
		return openfeature.SplitReason
	case ReasonDisabled:
		return openfeature.DisabledReason
	case ReasonDefault:
		return openfeature.DefaultReason
	default:
		return openfeature.UnknownReason
	}
}

func toInt64(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int8:
		return int64(v), true
	case int16:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case uint:
		return int64(v), true
	case uint8:
		return int64(v), true
	case uint16:
		return int64(v), true
	case uint32:
		return int64(v), true
	case uint64:
		if v > math.MaxInt64 {
			return 0, false
		}
		return int64(v), true
	default:
		return 0, false
	}
}

func TestOpenFeatureClientIntegration(t *testing.T) {
	repo := NewInMemoryRepository()
	engine := NewEngine()
	service, err := NewService(repo, engine)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	flagDef := FeatureFlag{
		Project:    "test-project",
		Stage:      "dev",
		Key:        "checkout-flow",
		Enabled:    true,
		DefaultKey: "control",
		Variations: []Variation{
			{Key: "control", Type: BooleanVariation, Value: false},
			{Key: "enabled", Type: BooleanVariation, Value: true},
		},
		Rules: []Rule{
			{
				ID:           "country-de",
				VariationKey: "enabled",
				Conditions: []Condition{
					{Attribute: "country", Operator: MatcherEquals, Value: "DE"},
				},
			},
		},
	}

	if err := repo.UpsertFlag(context.Background(), flagDef); err != nil {
		t.Fatalf("UpsertFlag: %v", err)
	}

	provider := newServiceProvider(service, "test-project", "dev")
	if err := openfeature.SetProviderAndWait(provider); err != nil {
		t.Fatalf("SetProviderAndWait: %v", err)
	}
	t.Cleanup(openfeature.Shutdown)

	client := openfeature.NewClient("checkout")
	targetedCtx := openfeature.NewTargetlessEvaluationContext(map[string]any{"country": "DE"})

	details, err := client.BooleanValueDetails(context.Background(), "checkout-flow", false, targetedCtx)
	if err != nil {
		t.Fatalf("BooleanValueDetails: %v", err)
	}
	if details.Value != true {
		t.Fatalf("expected true variation for targeted context, got %v", details.Value)
	}
	if details.Reason != openfeature.TargetingMatchReason {
		t.Fatalf("expected reason %s, got %s", openfeature.TargetingMatchReason, details.Reason)
	}
	if details.Variant != "enabled" {
		t.Fatalf("expected variant 'enabled', got %s", details.Variant)
	}

	defaultCtx := openfeature.NewTargetlessEvaluationContext(map[string]any{"country": "US"})
	defaultDetails, err := client.BooleanValueDetails(context.Background(), "checkout-flow", false, defaultCtx)
	if err != nil {
		t.Fatalf("BooleanValueDetails default: %v", err)
	}
	if defaultDetails.Value != false {
		t.Fatalf("expected default false variation, got %v", defaultDetails.Value)
	}
	if defaultDetails.Reason != openfeature.DefaultReason {
		t.Fatalf("expected reason %s, got %s", openfeature.DefaultReason, defaultDetails.Reason)
	}
	if defaultDetails.Variant != "control" {
		t.Fatalf("expected variant 'control', got %s", defaultDetails.Variant)
	}
}
