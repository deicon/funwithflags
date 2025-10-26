package flag_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	flagpkg "github.com/deicon/funwithflags/internal/flag"
	"github.com/deicon/funwithflags/internal/httpserver"
	"github.com/open-feature/go-sdk/openfeature"
)

type httpProvider struct {
	baseURL  string
	client   *http.Client
	metadata openfeature.Metadata
}

type apiEvaluateRequest struct {
	Context map[string]any `json:"context,omitempty"`
}

type apiEvaluateResponse struct {
	FlagKey       string `json:"flagKey"`
	VariationKey  string `json:"variationKey"`
	VariationType string `json:"variationType"`
	Value         any    `json:"value"`
	Reason        string `json:"reason"`
}

type apiErrorResponse struct {
	Error string `json:"error"`
}

func newHTTPProvider(baseURL string) *httpProvider {
	return &httpProvider{
		baseURL:  baseURL,
		metadata: openfeature.Metadata{Name: "funwithflags-http"},
	}
}

func (p *httpProvider) Metadata() openfeature.Metadata {
	return p.metadata
}

func (p *httpProvider) Hooks() []openfeature.Hook {
	return nil
}

func (p *httpProvider) Init(openfeature.EvaluationContext) error {
	if p.client == nil {
		p.client = &http.Client{Timeout: 2 * time.Second}
	}
	return nil
}

func (p *httpProvider) Shutdown() {
	if p.client != nil {
		p.client.CloseIdleConnections()
	}
}

func (p *httpProvider) BooleanEvaluation(ctx context.Context, flagKey string, defaultValue bool, flatCtx openfeature.FlattenedContext) openfeature.BoolResolutionDetail {
	resp, detail, err := p.evaluate(ctx, flagKey, flatCtx)
	if err != nil {
		return openfeature.BoolResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	value, ok := resp.Value.(bool)
	if !ok {
		detail.ResolutionError = openfeature.NewTypeMismatchResolutionError("variation value is not boolean")
		detail.Reason = openfeature.ErrorReason
		return openfeature.BoolResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	return openfeature.BoolResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}

func (p *httpProvider) StringEvaluation(ctx context.Context, flagKey string, defaultValue string, flatCtx openfeature.FlattenedContext) openfeature.StringResolutionDetail {
	resp, detail, err := p.evaluate(ctx, flagKey, flatCtx)
	if err != nil {
		return openfeature.StringResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	value, ok := resp.Value.(string)
	if !ok {
		detail.ResolutionError = openfeature.NewTypeMismatchResolutionError("variation value is not string")
		detail.Reason = openfeature.ErrorReason
		return openfeature.StringResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	return openfeature.StringResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}

func (p *httpProvider) FloatEvaluation(ctx context.Context, flagKey string, defaultValue float64, flatCtx openfeature.FlattenedContext) openfeature.FloatResolutionDetail {
	resp, detail, err := p.evaluate(ctx, flagKey, flatCtx)
	if err != nil {
		return openfeature.FloatResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	value, ok := coerceFloat64(resp.Value)
	if !ok {
		detail.ResolutionError = openfeature.NewTypeMismatchResolutionError("variation value is not float")
		detail.Reason = openfeature.ErrorReason
		return openfeature.FloatResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	return openfeature.FloatResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}

func (p *httpProvider) IntEvaluation(ctx context.Context, flagKey string, defaultValue int64, flatCtx openfeature.FlattenedContext) openfeature.IntResolutionDetail {
	resp, detail, err := p.evaluate(ctx, flagKey, flatCtx)
	if err != nil {
		return openfeature.IntResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	value, ok := coerceInt64(resp.Value)
	if !ok {
		detail.ResolutionError = openfeature.NewTypeMismatchResolutionError("variation value is not int")
		detail.Reason = openfeature.ErrorReason
		return openfeature.IntResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	return openfeature.IntResolutionDetail{Value: value, ProviderResolutionDetail: detail}
}

func (p *httpProvider) ObjectEvaluation(ctx context.Context, flagKey string, defaultValue any, flatCtx openfeature.FlattenedContext) openfeature.InterfaceResolutionDetail {
	resp, detail, err := p.evaluate(ctx, flagKey, flatCtx)
	if err != nil {
		return openfeature.InterfaceResolutionDetail{Value: defaultValue, ProviderResolutionDetail: detail}
	}
	return openfeature.InterfaceResolutionDetail{Value: resp.Value, ProviderResolutionDetail: detail}
}

func (p *httpProvider) evaluate(ctx context.Context, flagKey string, flatCtx openfeature.FlattenedContext) (apiEvaluateResponse, openfeature.ProviderResolutionDetail, error) {
	if p.client == nil {
		err := errors.New("http client not initialized")
		return apiEvaluateResponse{}, openfeature.ProviderResolutionDetail{
			Reason:          openfeature.ErrorReason,
			ResolutionError: openfeature.NewGeneralResolutionError(err.Error()),
		}, err
	}

	attrs := make(map[string]any, len(flatCtx))
	for k, v := range flatCtx {
		attrs[k] = v
	}

	payload, err := json.Marshal(apiEvaluateRequest{Context: attrs})
	if err != nil {
		detail := openfeature.ProviderResolutionDetail{
			Reason:          openfeature.ErrorReason,
			ResolutionError: openfeature.NewGeneralResolutionError(fmt.Sprintf("encode request: %v", err)),
		}
		return apiEvaluateResponse{}, detail, err
	}

	base := strings.TrimSuffix(p.baseURL, "/")
	endpoint := fmt.Sprintf("%s/api/v1/flags/%s/evaluate", base, url.PathEscape(flagKey))
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		detail := openfeature.ProviderResolutionDetail{
			Reason:          openfeature.ErrorReason,
			ResolutionError: openfeature.NewGeneralResolutionError(fmt.Sprintf("create request: %v", err)),
		}
		return apiEvaluateResponse{}, detail, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		detail := openfeature.ProviderResolutionDetail{
			Reason:          openfeature.ErrorReason,
			ResolutionError: openfeature.NewGeneralResolutionError(fmt.Sprintf("http request: %v", err)),
		}
		return apiEvaluateResponse{}, detail, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		var apiErr apiErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err != nil {
			apiErr.Error = "flag not found"
		}
		msg := apiErr.Error
		if msg == "" {
			msg = "flag not found"
		}
		detail := openfeature.ProviderResolutionDetail{
			Reason:          openfeature.ErrorReason,
			ResolutionError: openfeature.NewFlagNotFoundResolutionError(msg),
		}
		return apiEvaluateResponse{}, detail, errors.New(msg)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			msg = fmt.Sprintf("unexpected status %d", resp.StatusCode)
		} else {
			msg = fmt.Sprintf("status %d: %s", resp.StatusCode, msg)
		}
		detail := openfeature.ProviderResolutionDetail{
			Reason:          openfeature.ErrorReason,
			ResolutionError: openfeature.NewGeneralResolutionError(msg),
		}
		return apiEvaluateResponse{}, detail, errors.New(msg)
	}

	var apiResp apiEvaluateResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		detail := openfeature.ProviderResolutionDetail{
			Reason:          openfeature.ErrorReason,
			ResolutionError: openfeature.NewGeneralResolutionError(fmt.Sprintf("decode response: %v", err)),
		}
		return apiEvaluateResponse{}, detail, err
	}

	detail := openfeature.ProviderResolutionDetail{
		Reason:  mapReason(apiResp.Reason),
		Variant: apiResp.VariationKey,
	}

	return apiResp, detail, nil
}

func mapReason(reason string) openfeature.Reason {
	switch reason {
	case flagpkg.ReasonTargetMatch:
		return openfeature.TargetingMatchReason
	case flagpkg.ReasonPercentageRollout:
		return openfeature.SplitReason
	case flagpkg.ReasonDisabled:
		return openfeature.DisabledReason
	case flagpkg.ReasonDefault:
		return openfeature.DefaultReason
	default:
		return openfeature.UnknownReason
	}
}

func coerceFloat64(value any) (float64, bool) {
	switch v := value.(type) {
	case float32:
		return float64(v), true
	case float64:
		return v, true
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
	default:
		return 0, false
	}
}

func coerceInt64(value any) (int64, bool) {
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
	case float32:
		return int64(v), true
	case float64:
		return int64(v), true
	default:
		return 0, false
	}
}

func TestOpenFeatureClientHTTPIntegration(t *testing.T) {
	openfeature.Shutdown()

	repo := flagpkg.NewInMemoryRepository()
	engine := flagpkg.NewEngine()
	service, err := flagpkg.NewService(repo, engine)
	if err != nil {
		t.Fatalf("NewService: %v", err)
	}

	flagDef := flagpkg.FeatureFlag{
		Key:        "checkout-flow",
		Enabled:    true,
		DefaultKey: "control",
		Variations: []flagpkg.Variation{
			{Key: "control", Type: flagpkg.BooleanVariation, Value: false},
			{Key: "enabled", Type: flagpkg.BooleanVariation, Value: true},
		},
		Rules: []flagpkg.Rule{
			{
				ID:           "country-de",
				VariationKey: "enabled",
				Conditions: []flagpkg.Condition{
					{Attribute: "country", Operator: flagpkg.MatcherEquals, Value: "DE"},
				},
			},
		},
	}

	if err := repo.UpsertFlag(context.Background(), flagDef); err != nil {
		t.Fatalf("UpsertFlag: %v", err)
	}

	router, err := httpserver.NewRouter(httpserver.Config{FlagService: service})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}

	server := httptest.NewServer(router)
	defer server.Close()

	provider := newHTTPProvider(server.URL)
	if err := openfeature.SetProviderAndWait(provider); err != nil {
		t.Fatalf("SetProviderAndWait: %v", err)
	}
	t.Cleanup(openfeature.Shutdown)

	client := openfeature.NewClient("checkout-http")
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
