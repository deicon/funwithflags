package flag

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{
		pool: pool,
	}
}

type dbFlag struct {
	Key        string
	Name       string
	Description string
	Enabled    bool
	DefaultKey string
	Config     json.RawMessage
	Version    int
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type flagConfig struct {
	Variations []Variation `json:"variations"`
	Rules      []Rule      `json:"rules"`
}

func (r *PostgresRepository) GetFlag(ctx context.Context, key string) (FeatureFlag, error) {
	query := `
		SELECT key, name, description, enabled, default_key, config, version, created_at, updated_at
		FROM feature_flags
		WHERE key = $1
	`

	var dbF dbFlag
	err := r.pool.QueryRow(ctx, query, key).Scan(
		&dbF.Key,
		&dbF.Name,
		&dbF.Description,
		&dbF.Enabled,
		&dbF.DefaultKey,
		&dbF.Config,
		&dbF.Version,
		&dbF.CreatedAt,
		&dbF.UpdatedAt,
	)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FeatureFlag{}, ErrFlagNotFound
		}
		return FeatureFlag{}, fmt.Errorf("failed to get flag: %w", err)
	}

	return dbFlagToFeatureFlag(dbF)
}

func (r *PostgresRepository) ListFlags(ctx context.Context) ([]FeatureFlag, error) {
	query := `
		SELECT key, name, description, enabled, default_key, config, version, created_at, updated_at
		FROM feature_flags
		ORDER BY key
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list flags: %w", err)
	}
	defer rows.Close()

	var flags []FeatureFlag
	for rows.Next() {
		var dbF dbFlag
		err := rows.Scan(
			&dbF.Key,
			&dbF.Name,
			&dbF.Description,
			&dbF.Enabled,
			&dbF.DefaultKey,
			&dbF.Config,
			&dbF.Version,
			&dbF.CreatedAt,
			&dbF.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan flag: %w", err)
		}

		flag, err := dbFlagToFeatureFlag(dbF)
		if err != nil {
			return nil, err
		}
		flags = append(flags, flag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating flags: %w", err)
	}

	return flags, nil
}

func (r *PostgresRepository) UpsertFlag(ctx context.Context, flag FeatureFlag) error {
	if err := validateFlag(flag); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFlag, err)
	}

	config := flagConfig{
		Variations: flag.Variations,
		Rules:      flag.Rules,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal flag config: %w", err)
	}

	// Check if flag exists
	var existingVersion int
	var existingUpdatedAt time.Time
	checkQuery := `SELECT version, updated_at FROM feature_flags WHERE key = $1`
	err = r.pool.QueryRow(ctx, checkQuery, flag.Key).Scan(&existingVersion, &existingUpdatedAt)

	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("failed to check existing flag: %w", err)
	}

	exists := err == nil

	if exists {
		// Optimistic locking check
		if !flag.UpdatedAt.IsZero() && !flag.UpdatedAt.Equal(existingUpdatedAt) {
			return ErrFlagConflict
		}

		// Update existing flag
		updateQuery := `
			UPDATE feature_flags
			SET name = $2, description = $3, enabled = $4, default_key = $5, config = $6
			WHERE key = $1 AND version = $7
			RETURNING version, updated_at
		`

		var newVersion int
		var newUpdatedAt time.Time
		err = r.pool.QueryRow(ctx, updateQuery,
			flag.Key,
			flag.Name,
			flag.Description,
			flag.Enabled,
			flag.DefaultKey,
			configJSON,
			existingVersion,
		).Scan(&newVersion, &newUpdatedAt)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrFlagConflict
			}
			return fmt.Errorf("failed to update flag: %w", err)
		}

		return nil
	}

	// Insert new flag
	insertQuery := `
		INSERT INTO feature_flags (key, name, description, enabled, default_key, config, version)
		VALUES ($1, $2, $3, $4, $5, $6, 1)
	`

	_, err = r.pool.Exec(ctx, insertQuery,
		flag.Key,
		flag.Name,
		flag.Description,
		flag.Enabled,
		flag.DefaultKey,
		configJSON,
	)

	if err != nil {
		return fmt.Errorf("failed to insert flag: %w", err)
	}

	return nil
}

func (r *PostgresRepository) DeleteFlag(ctx context.Context, key string) error {
	query := `DELETE FROM feature_flags WHERE key = $1`

	result, err := r.pool.Exec(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete flag: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrFlagNotFound
	}

	return nil
}

func dbFlagToFeatureFlag(dbF dbFlag) (FeatureFlag, error) {
	var config flagConfig
	if err := json.Unmarshal(dbF.Config, &config); err != nil {
		return FeatureFlag{}, fmt.Errorf("failed to unmarshal flag config: %w", err)
	}

	return FeatureFlag{
		Key:         dbF.Key,
		Name:        dbF.Name,
		Description: dbF.Description,
		Enabled:     dbF.Enabled,
		DefaultKey:  dbF.DefaultKey,
		Variations:  config.Variations,
		Rules:       config.Rules,
		CreatedAt:   dbF.CreatedAt,
		UpdatedAt:   dbF.UpdatedAt,
	}, nil
}
