package flag

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	ActionCreate = "CREATE"
	ActionUpdate = "UPDATE"
	ActionDelete = "DELETE"
)

type AuditLog struct {
	ID          int64
	FlagKey     string
	Action      string
	PerformedBy string
	OldValue    json.RawMessage
	NewValue    json.RawMessage
	CreatedAt   time.Time
}

type AuditService interface {
	LogAction(ctx context.Context, flagKey, action, performedBy string, oldValue, newValue *FeatureFlag) error
	GetAuditLogs(ctx context.Context, flagKey string, limit int) ([]AuditLog, error)
}

type PostgresAuditService struct {
	pool *pgxpool.Pool
}

func NewPostgresAuditService(pool *pgxpool.Pool) *PostgresAuditService {
	return &PostgresAuditService{
		pool: pool,
	}
}

func (s *PostgresAuditService) LogAction(ctx context.Context, flagKey, action, performedBy string, oldValue, newValue *FeatureFlag) error {
	var oldJSON, newJSON json.RawMessage
	var err error

	if oldValue != nil {
		oldJSON, err = json.Marshal(oldValue)
		if err != nil {
			return fmt.Errorf("failed to marshal old value: %w", err)
		}
	}

	if newValue != nil {
		newJSON, err = json.Marshal(newValue)
		if err != nil {
			return fmt.Errorf("failed to marshal new value: %w", err)
		}
	}

	query := `
		INSERT INTO audit_logs (flag_key, action, performed_by, old_value, new_value)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err = s.pool.Exec(ctx, query, flagKey, action, performedBy, oldJSON, newJSON)
	if err != nil {
		return fmt.Errorf("failed to insert audit log: %w", err)
	}

	return nil
}

func (s *PostgresAuditService) GetAuditLogs(ctx context.Context, flagKey string, limit int) ([]AuditLog, error) {
	if limit <= 0 {
		limit = 100
	}

	query := `
		SELECT id, flag_key, action, performed_by, old_value, new_value, created_at
		FROM audit_logs
		WHERE flag_key = $1
		ORDER BY created_at DESC
		LIMIT $2
	`

	rows, err := s.pool.Query(ctx, query, flagKey, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query audit logs: %w", err)
	}
	defer rows.Close()

	var logs []AuditLog
	for rows.Next() {
		var log AuditLog
		err := rows.Scan(
			&log.ID,
			&log.FlagKey,
			&log.Action,
			&log.PerformedBy,
			&log.OldValue,
			&log.NewValue,
			&log.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan audit log: %w", err)
		}
		logs = append(logs, log)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating audit logs: %w", err)
	}

	return logs, nil
}
