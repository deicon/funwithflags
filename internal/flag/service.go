package flag

import (
	"context"
	"errors"
	"fmt"
)

type Service struct {
	repo         Repository
	evaluator    Evaluator
	auditService AuditService
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

func (s *Service) SetAuditService(auditService AuditService) {
	s.auditService = auditService
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

// Admin operations

func (s *Service) GetFlag(ctx context.Context, key string) (FeatureFlag, error) {
	if key == "" {
		return FeatureFlag{}, fmt.Errorf("flag key is required")
	}
	return s.repo.GetFlag(ctx, key)
}

func (s *Service) ListFlags(ctx context.Context) ([]FeatureFlag, error) {
	return s.repo.ListFlags(ctx)
}

func (s *Service) CreateFlag(ctx context.Context, flag FeatureFlag, performedBy string) error {
	if flag.Key == "" {
		return fmt.Errorf("flag key is required")
	}

	// Check if flag already exists
	_, err := s.repo.GetFlag(ctx, flag.Key)
	if err == nil {
		return fmt.Errorf("flag %q already exists", flag.Key)
	}
	if !errors.Is(err, ErrFlagNotFound) {
		return err
	}

	// Create the flag
	if err := s.repo.UpsertFlag(ctx, flag); err != nil {
		return err
	}

	// Audit log
	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, flag.Key, ActionCreate, performedBy, nil, &flag)
	}

	return nil
}

func (s *Service) UpdateFlag(ctx context.Context, flag FeatureFlag, performedBy string) error {
	if flag.Key == "" {
		return fmt.Errorf("flag key is required")
	}

	// Get existing flag for audit
	oldFlag, err := s.repo.GetFlag(ctx, flag.Key)
	if err != nil {
		return err
	}

	// Update the flag
	if err := s.repo.UpsertFlag(ctx, flag); err != nil {
		return err
	}

	// Audit log
	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, flag.Key, ActionUpdate, performedBy, &oldFlag, &flag)
	}

	return nil
}

func (s *Service) DeleteFlag(ctx context.Context, key string, performedBy string) error {
	if key == "" {
		return fmt.Errorf("flag key is required")
	}

	// Get existing flag for audit
	oldFlag, err := s.repo.GetFlag(ctx, key)
	if err != nil {
		return err
	}

	// Delete the flag
	if err := s.repo.DeleteFlag(ctx, key); err != nil {
		return err
	}

	// Audit log
	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, key, ActionDelete, performedBy, &oldFlag, nil)
	}

	return nil
}

func (s *Service) GetAuditLogs(ctx context.Context, flagKey string, limit int) ([]AuditLog, error) {
	if s.auditService == nil {
		return nil, fmt.Errorf("audit service not configured")
	}
	return s.auditService.GetAuditLogs(ctx, flagKey, limit)
}
