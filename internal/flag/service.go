package flag

import (
	"context"
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

func (s *Service) EvaluateFlag(ctx context.Context, project, stage, key string, attrs EvaluationContext) (EvaluationResult, error) {
	if key == "" {
		return EvaluationResult{}, fmt.Errorf("flag key is required")
	}
	flag, err := s.repo.GetFlag(ctx, project, stage, key)
	if err != nil {
		return EvaluationResult{}, err
	}
	if attrs == nil {
		attrs = EvaluationContext{}
	}
	return s.evaluator.Evaluate(ctx, flag, attrs)
}

// Admin operations

func (s *Service) GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error) {
	if key == "" {
		return FeatureFlag{}, fmt.Errorf("flag key is required")
	}
	return s.repo.GetFlag(ctx, project, stage, key)
}

func (s *Service) ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error) {
	flags, err := s.repo.ListFlags(ctx, project, stage)
	if err != nil {
		return nil, err
	}
	if flags == nil {
		return []FeatureFlag{}, nil
	}
	return flags, nil
}

func (s *Service) CreateFlag(ctx context.Context, flag FeatureFlag, performedBy string) error {
	if flag.Project == "" {
		return fmt.Errorf("flag project is required")
	}
	if flag.Stage == "" {
		return fmt.Errorf("flag stage is required")
	}
	if flag.Key == "" {
		return fmt.Errorf("flag key is required")
	}

	// Create the flag (temporal ranges allow multiple entries per key)
	if err := s.repo.UpsertFlag(ctx, flag); err != nil {
		return err
	}

	// Audit log
	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, flag.Project, flag.Stage, flag.Key, ActionCreate, performedBy, nil, &flag)
	}

	return nil
}

func (s *Service) UpdateFlag(ctx context.Context, flag FeatureFlag, performedBy string) error {
	if flag.ID == 0 {
		return fmt.Errorf("flag ID is required for updates")
	}
	if flag.Project == "" {
		return fmt.Errorf("flag project is required")
	}
	if flag.Stage == "" {
		return fmt.Errorf("flag stage is required")
	}
	if flag.Key == "" {
		return fmt.Errorf("flag key is required")
	}

	// Get existing flag for audit
	oldFlag, err := s.repo.GetFlagByID(ctx, flag.ID)
	if err != nil {
		return err
	}

	// Update the flag
	if err := s.repo.UpsertFlag(ctx, flag); err != nil {
		return err
	}

	// Audit log
	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, flag.Project, flag.Stage, flag.Key, ActionUpdate, performedBy, &oldFlag, &flag)
	}

	return nil
}

func (s *Service) DeleteFlag(ctx context.Context, id int64, performedBy string) error {
	// Get existing flag for audit
	oldFlag, err := s.repo.GetFlagByID(ctx, id)
	if err != nil {
		return err
	}

	// Delete the flag range
	if err := s.repo.DeleteFlag(ctx, id); err != nil {
		return err
	}

	// Audit log
	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, oldFlag.Project, oldFlag.Stage, oldFlag.Key, ActionDelete, performedBy, &oldFlag, nil)
	}

	return nil
}

// Temporal range operations

func (s *Service) GetFlagByID(ctx context.Context, id int64) (FeatureFlag, error) {
	return s.repo.GetFlagByID(ctx, id)
}

func (s *Service) GetFlagRanges(ctx context.Context, project, stage, key string) ([]FeatureFlag, error) {
	if project == "" {
		return nil, fmt.Errorf("flag project is required")
	}
	if stage == "" {
		return nil, fmt.Errorf("flag stage is required")
	}
	if key == "" {
		return nil, fmt.Errorf("flag key is required")
	}
	flags, err := s.repo.GetFlagRanges(ctx, project, stage, key)
	if err != nil {
		return nil, err
	}
	if flags == nil {
		return []FeatureFlag{}, nil
	}
	return flags, nil
}

func (s *Service) ActivateFlag(ctx context.Context, id int64, performedBy string) error {
	// Get flag for audit
	oldFlag, err := s.repo.GetFlagByID(ctx, id)
	if err != nil {
		return err
	}

	// Activate the flag range
	if err := s.repo.ActivateFlag(ctx, id); err != nil {
		return err
	}

	// Get updated flag for audit
	newFlag, err := s.repo.GetFlagByID(ctx, id)
	if err != nil {
		return err
	}

	// Audit log
	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, newFlag.Project, newFlag.Stage, newFlag.Key, "activate", performedBy, &oldFlag, &newFlag)
	}

	return nil
}

func (s *Service) DeactivateFlag(ctx context.Context, id int64, performedBy string) error {
	// Get flag for audit
	oldFlag, err := s.repo.GetFlagByID(ctx, id)
	if err != nil {
		return err
	}

	// Deactivate the flag range
	if err := s.repo.DeactivateFlag(ctx, id); err != nil {
		return err
	}

	// Get updated flag for audit
	newFlag, err := s.repo.GetFlagByID(ctx, id)
	if err != nil {
		return err
	}

	// Audit log
	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, newFlag.Project, newFlag.Stage, newFlag.Key, "deactivate", performedBy, &oldFlag, &newFlag)
	}

	return nil
}

func (s *Service) GetAuditLogs(ctx context.Context, project, stage, flagKey string, limit int) ([]AuditLog, error) {
	if s.auditService == nil {
		return nil, fmt.Errorf("audit service not configured")
	}
	logs, err := s.auditService.GetAuditLogs(ctx, project, stage, flagKey, limit)
	if err != nil {
		return nil, err
	}
	if logs == nil {
		return []AuditLog{}, nil
	}
	return logs, nil
}
