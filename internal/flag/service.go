package flag

import (
	"context"
	"fmt"
	"time"
)

type Service struct {
	repo         Repository
	evaluator    Evaluator
	auditService AuditService
}

func NewService(repo Repository) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("repository is required")
	}
	return &Service{repo: repo}, nil
}

func (s *Service) SetEvaluator(evaluator Evaluator) {
	s.evaluator = evaluator
}

func (s *Service) SetAuditService(auditService AuditService) {
	s.auditService = auditService
}

// --- Flag identity operations ---

func (s *Service) CreateFlag(ctx context.Context, f FeatureFlag, performedBy string) (FeatureFlag, error) {
	if f.Project == "" {
		return FeatureFlag{}, fmt.Errorf("flag project is required")
	}
	if f.Stage == "" {
		return FeatureFlag{}, fmt.Errorf("flag stage is required")
	}
	if f.Key == "" {
		return FeatureFlag{}, fmt.Errorf("flag key is required")
	}

	created, err := s.repo.CreateFlag(ctx, f)
	if err != nil {
		return FeatureFlag{}, err
	}

	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, ActionCreate, performedBy, nil, &created)
	}
	return created, nil
}

func (s *Service) GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error) {
	return s.repo.GetFlag(ctx, project, stage, key)
}

func (s *Service) GetFlagByID(ctx context.Context, id int64) (FeatureFlag, error) {
	return s.repo.GetFlagByID(ctx, id)
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

func (s *Service) UpdateFlag(ctx context.Context, f FeatureFlag, performedBy string) error {
	if f.ID == 0 {
		return fmt.Errorf("flag ID is required")
	}

	oldFlag, err := s.repo.GetFlagByID(ctx, f.ID)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateFlag(ctx, f); err != nil {
		return err
	}

	if s.auditService != nil {
		newFlag, _ := s.repo.GetFlagByID(ctx, f.ID)
		_ = s.auditService.LogAction(ctx, oldFlag.Project, oldFlag.Stage, oldFlag.Key, ActionUpdate, performedBy, &oldFlag, &newFlag)
	}
	return nil
}

func (s *Service) DeleteFlag(ctx context.Context, id int64, performedBy string) error {
	oldFlag, err := s.repo.GetFlagByID(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteFlag(ctx, id); err != nil {
		return err
	}

	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, oldFlag.Project, oldFlag.Stage, oldFlag.Key, ActionDelete, performedBy, &oldFlag, nil)
	}
	return nil
}

// --- Range operations ---

func (s *Service) CreateRange(ctx context.Context, r FlagRange, rules []Rule, performedBy string) (FlagRange, RangeVersion, error) {
	// Validate flag exists
	f, err := s.repo.GetFlagByID(ctx, r.FlagID)
	if err != nil {
		return FlagRange{}, RangeVersion{}, err
	}

	// Create range
	created, err := s.repo.CreateRange(ctx, r)
	if err != nil {
		return FlagRange{}, RangeVersion{}, err
	}

	// Create v1 draft
	v, err := s.repo.CreateVersion(ctx, RangeVersion{
		RangeID: created.ID,
		Rules:   rules,
	})
	if err != nil {
		// Rollback range creation
		_ = s.repo.DeleteRange(ctx, created.ID)
		return FlagRange{}, RangeVersion{}, err
	}

	if s.auditService != nil {
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "create_range", performedBy, nil, &created)
	}
	return created, v, nil
}

func (s *Service) GetRange(ctx context.Context, id int64) (FlagRange, error) {
	return s.repo.GetRange(ctx, id)
}

func (s *Service) ListRanges(ctx context.Context, flagID int64) ([]FlagRange, error) {
	ranges, err := s.repo.ListRanges(ctx, flagID)
	if err != nil {
		return nil, err
	}
	if ranges == nil {
		return []FlagRange{}, nil
	}
	return ranges, nil
}

func (s *Service) UpdateRange(ctx context.Context, r FlagRange, performedBy string) error {
	old, err := s.repo.GetRange(ctx, r.ID)
	if err != nil {
		return err
	}

	if err := s.repo.UpdateRange(ctx, r); err != nil {
		return err
	}

	if s.auditService != nil {
		f, _ := s.repo.GetFlagByID(ctx, old.FlagID)
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "update_range", performedBy, &old, &r)
	}
	return nil
}

func (s *Service) DeleteRange(ctx context.Context, id int64, performedBy string) error {
	old, err := s.repo.GetRange(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.DeleteRange(ctx, id); err != nil {
		return err
	}

	if s.auditService != nil {
		f, _ := s.repo.GetFlagByID(ctx, old.FlagID)
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "delete_range", performedBy, &old, nil)
	}
	return nil
}

func (s *Service) ActivateRange(ctx context.Context, id int64, performedBy string) error {
	rng, err := s.repo.GetRange(ctx, id)
	if err != nil {
		return err
	}

	// Must have published version
	_, err = s.repo.GetPublishedVersion(ctx, id)
	if err != nil {
		return fmt.Errorf("cannot activate range: %w", ErrNoPublishedVersion)
	}

	if err := s.repo.ActivateRange(ctx, id); err != nil {
		return err
	}

	if s.auditService != nil {
		f, _ := s.repo.GetFlagByID(ctx, rng.FlagID)
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "activate", performedBy, nil, nil)
	}
	return nil
}

func (s *Service) DeactivateRange(ctx context.Context, id int64, performedBy string) error {
	rng, err := s.repo.GetRange(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.DeactivateRange(ctx, id); err != nil {
		return err
	}

	if s.auditService != nil {
		f, _ := s.repo.GetFlagByID(ctx, rng.FlagID)
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "deactivate", performedBy, nil, nil)
	}
	return nil
}

// --- Version operations ---

func (s *Service) CreateVersion(ctx context.Context, rangeID int64, rules []Rule, performedBy string) (RangeVersion, error) {
	rng, err := s.repo.GetRange(ctx, rangeID)
	if err != nil {
		return RangeVersion{}, err
	}

	v, err := s.repo.CreateVersion(ctx, RangeVersion{
		RangeID: rangeID,
		Rules:   rules,
	})
	if err != nil {
		return RangeVersion{}, err
	}

	if s.auditService != nil {
		f, _ := s.repo.GetFlagByID(ctx, rng.FlagID)
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "create_version", performedBy, nil, &v)
	}
	return v, nil
}

func (s *Service) GetVersion(ctx context.Context, id int64) (RangeVersion, error) {
	return s.repo.GetVersion(ctx, id)
}

func (s *Service) ListVersions(ctx context.Context, rangeID int64) ([]RangeVersion, error) {
	versions, err := s.repo.ListVersions(ctx, rangeID)
	if err != nil {
		return nil, err
	}
	if versions == nil {
		return []RangeVersion{}, nil
	}
	return versions, nil
}

func (s *Service) UpdateVersion(ctx context.Context, v RangeVersion, performedBy string) error {
	return s.repo.UpdateVersion(ctx, v)
}

func (s *Service) DeleteDraftVersion(ctx context.Context, id int64, performedBy string) error {
	return s.repo.DeleteDraftVersion(ctx, id)
}

func (s *Service) PublishVersion(ctx context.Context, id int64, performedBy string) error {
	v, err := s.repo.GetVersion(ctx, id)
	if err != nil {
		return err
	}

	if err := s.repo.PublishVersion(ctx, id); err != nil {
		return err
	}

	if s.auditService != nil {
		rng, _ := s.repo.GetRange(ctx, v.RangeID)
		f, _ := s.repo.GetFlagByID(ctx, rng.FlagID)
		_ = s.auditService.LogAction(ctx, f.Project, f.Stage, f.Key, "publish", performedBy, nil, &v)
	}
	return nil
}

func (s *Service) RollbackVersion(ctx context.Context, sourceVersionID int64, performedBy string) (RangeVersion, error) {
	source, err := s.repo.GetVersion(ctx, sourceVersionID)
	if err != nil {
		return RangeVersion{}, err
	}

	// Check if draft exists — if so, overwrite its rules
	versions, err := s.repo.ListVersions(ctx, source.RangeID)
	if err != nil {
		return RangeVersion{}, err
	}

	for _, v := range versions {
		if v.Status == VersionStatusDraft {
			v.Rules = source.Rules
			if err := s.repo.UpdateVersion(ctx, v); err != nil {
				return RangeVersion{}, err
			}
			return v, nil
		}
	}

	// No draft exists — create new one
	return s.repo.CreateVersion(ctx, RangeVersion{
		RangeID: source.RangeID,
		Rules:   source.Rules,
	})
}

// --- Evaluation ---

func (s *Service) EvaluateFlag(ctx context.Context, project, stage, key string, attrs EvaluationContext) (EvaluationResult, error) {
	if key == "" {
		return EvaluationResult{}, fmt.Errorf("flag key is required")
	}

	f, err := s.repo.GetFlag(ctx, project, stage, key)
	if err != nil {
		return EvaluationResult{}, err
	}

	if !f.Enabled {
		return defaultResult(f, ReasonDisabled), nil
	}

	rng, err := s.repo.GetActiveRange(ctx, f.ID, time.Now().UTC())
	if err != nil {
		return defaultResult(f, ReasonNoActiveRange), nil
	}

	v, err := s.repo.GetPublishedVersion(ctx, rng.ID)
	if err != nil {
		return defaultResult(f, ReasonNoPublishedVersion), nil
	}

	if attrs == nil {
		attrs = EvaluationContext{}
	}
	return s.evaluator.Evaluate(ctx, f, v, attrs)
}

func defaultResult(f FeatureFlag, reason string) EvaluationResult {
	for _, v := range f.Variations {
		if v.Key == f.DefaultKey {
			return EvaluationResult{
				Variation: v,
				Reason:    reason,
			}
		}
	}
	return EvaluationResult{
		Variation: Variation{Key: f.DefaultKey},
		Reason:    reason,
	}
}

// --- Audit log query ---

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
