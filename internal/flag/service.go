package flag

import (
	"context"
	"fmt"
)

type Service struct {
	repo      Repository
	evaluator Evaluator
}

func NewService(repo Repository, evaluator Evaluator) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	if evaluator == nil {
		return nil, fmt.Errorf("evaluator is required")
	}
	return &Service{repo: repo, evaluator: evaluator}, nil
}

func (s *Service) EvaluateFlag(ctx context.Context, key string, attrs EvaluationContext) (EvaluationResult, error) {
	if key == "" {
		return EvaluationResult{}, fmt.Errorf("flag key is required")
	}
	flag, err := s.repo.GetFlag(ctx, key)
	if err != nil {
		return EvaluationResult{}, err
	}
	if attrs == nil {
		attrs = EvaluationContext{}
	}
	return s.evaluator.Evaluate(ctx, flag, attrs)
}
