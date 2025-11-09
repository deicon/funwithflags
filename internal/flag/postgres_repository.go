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
	ID          int64
	Project     string
	Stage       string
	Key         string
	Name        string
	Description string
	Enabled     bool
	Active      bool
	ValidFrom   time.Time
	ValidTo     *time.Time
	DefaultKey  string
	Config      json.RawMessage
	Version     int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type flagConfig struct {
	Variations []Variation `json:"variations"`
	Rules      []Rule      `json:"rules"`
}

// GetFlag gets the currently active flag valid at the current time
func (r *PostgresRepository) GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error) {
	return r.GetFlagAt(ctx, project, stage, key, time.Now())
}

// GetFlagByID gets a specific flag range by ID
func (r *PostgresRepository) GetFlagByID(ctx context.Context, id int64) (FeatureFlag, error) {
	query := `
		SELECT id, project, stage, key, name, description, enabled, active, valid_from, valid_to,
		       default_key, config, version, created_at, updated_at
		FROM feature_flags
		WHERE id = $1
	`

	var dbF dbFlag
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&dbF.ID,
		&dbF.Project,
		&dbF.Stage,
		&dbF.Key,
		&dbF.Name,
		&dbF.Description,
		&dbF.Enabled,
		&dbF.Active,
		&dbF.ValidFrom,
		&dbF.ValidTo,
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
		return FeatureFlag{}, fmt.Errorf("failed to get flag by ID: %w", err)
	}

	return dbFlagToFeatureFlag(dbF)
}

// GetFlagAt gets the active flag valid at a specific time
func (r *PostgresRepository) GetFlagAt(ctx context.Context, project, stage, key string, at time.Time) (FeatureFlag, error) {
	query := `
		SELECT id, project, stage, key, name, description, enabled, active, valid_from, valid_to,
		       default_key, config, version, created_at, updated_at
		FROM feature_flags
		WHERE project = $1 AND stage = $2 AND key = $3
		  AND active = true
		  AND valid_from <= $4
		  AND (valid_to IS NULL OR valid_to > $4)
		ORDER BY valid_from DESC
		LIMIT 1
	`

	var dbF dbFlag
	err := r.pool.QueryRow(ctx, query, project, stage, key, at).Scan(
		&dbF.ID,
		&dbF.Project,
		&dbF.Stage,
		&dbF.Key,
		&dbF.Name,
		&dbF.Description,
		&dbF.Enabled,
		&dbF.Active,
		&dbF.ValidFrom,
		&dbF.ValidTo,
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
		return FeatureFlag{}, fmt.Errorf("failed to get flag at time: %w", err)
	}

	return dbFlagToFeatureFlag(dbF)
}

// GetFlagRanges gets all temporal ranges (active and inactive) for a flag
func (r *PostgresRepository) GetFlagRanges(ctx context.Context, project, stage, key string) ([]FeatureFlag, error) {
	query := `
		SELECT id, project, stage, key, name, description, enabled, active, valid_from, valid_to,
		       default_key, config, version, created_at, updated_at
		FROM feature_flags
		WHERE project = $1 AND stage = $2 AND key = $3
		ORDER BY valid_from DESC
	`

	rows, err := r.pool.Query(ctx, query, project, stage, key)
	if err != nil {
		return nil, fmt.Errorf("failed to get flag ranges: %w", err)
	}
	defer rows.Close()

	flags := make([]FeatureFlag, 0)
	for rows.Next() {
		var dbF dbFlag
		err := rows.Scan(
			&dbF.ID,
			&dbF.Project,
			&dbF.Stage,
			&dbF.Key,
			&dbF.Name,
			&dbF.Description,
			&dbF.Enabled,
			&dbF.Active,
			&dbF.ValidFrom,
			&dbF.ValidTo,
			&dbF.DefaultKey,
			&dbF.Config,
			&dbF.Version,
			&dbF.CreatedAt,
			&dbF.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan flag range: %w", err)
		}

		flag, err := dbFlagToFeatureFlag(dbF)
		if err != nil {
			return nil, err
		}
		flags = append(flags, flag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating flag ranges: %w", err)
	}

	return flags, nil
}

// ListFlags lists all flag ranges for a project/stage ordered by newest range first per key.
func (r *PostgresRepository) ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error) {
	query := `
		SELECT id, project, stage, key, name, description, enabled, active,
		       valid_from, valid_to, default_key, config, version, created_at, updated_at
		FROM feature_flags
		WHERE project = $1 AND stage = $2
		ORDER BY key, valid_from DESC
	`

	rows, err := r.pool.Query(ctx, query, project, stage)
	if err != nil {
		return nil, fmt.Errorf("failed to list flags: %w", err)
	}
	defer rows.Close()

	flags := make([]FeatureFlag, 0)
	for rows.Next() {
		var dbF dbFlag
		err := rows.Scan(
			&dbF.ID,
			&dbF.Project,
			&dbF.Stage,
			&dbF.Key,
			&dbF.Name,
			&dbF.Description,
			&dbF.Enabled,
			&dbF.Active,
			&dbF.ValidFrom,
			&dbF.ValidTo,
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

// UpsertFlag creates or updates a flag range (validates non-overlapping ranges)
func (r *PostgresRepository) UpsertFlag(ctx context.Context, flag FeatureFlag) error {
	if err := validateFlag(flag); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidFlag, err)
	}

	// Validate temporal range
	if err := ValidateTemporalRange(flag.ValidFrom, flag.ValidTo); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidTimeRange, err)
	}

	config := flagConfig{
		Variations: flag.Variations,
		Rules:      flag.Rules,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal flag config: %w", err)
	}

	// Check for overlapping ranges
	hasOverlap, err := r.CheckOverlap(ctx, flag.Project, flag.Stage, flag.Key, flag.ValidFrom, flag.ValidTo, flag.ID)
	if err != nil {
		return fmt.Errorf("failed to check overlap: %w", err)
	}
	if hasOverlap {
		return ErrRangeOverlap
	}

	// Update existing flag range
	if flag.ID > 0 {
		// Get existing for optimistic locking
		existing, err := r.GetFlagByID(ctx, flag.ID)
		if err != nil {
			return err
		}

		// Optimistic locking check
		if !flag.UpdatedAt.IsZero() && !flag.UpdatedAt.Equal(existing.UpdatedAt) {
			return ErrFlagConflict
		}

		updateQuery := `
			UPDATE feature_flags
			SET name = $2, description = $3, enabled = $4, active = $5,
			    valid_from = $6, valid_to = $7, default_key = $8, config = $9
			WHERE id = $1
			RETURNING version, updated_at
		`

		var newVersion int
		var newUpdatedAt time.Time
		err = r.pool.QueryRow(ctx, updateQuery,
			flag.ID,
			flag.Name,
			flag.Description,
			flag.Enabled,
			flag.Active,
			flag.ValidFrom,
			flag.ValidTo,
			flag.DefaultKey,
			configJSON,
		).Scan(&newVersion, &newUpdatedAt)

		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrFlagNotFound
			}
			return fmt.Errorf("failed to update flag: %w", err)
		}

		return nil
	}

	// Insert new flag range
	// Default to active if not specified
	active := flag.Active
	if flag.ID == 0 && !flag.Active {
		active = true // Default for new flags
	}

	insertQuery := `
		INSERT INTO feature_flags (project, stage, key, name, description, enabled, active,
		                           valid_from, valid_to, default_key, config, version)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, 1)
		RETURNING id
	`

	var newID int64
	err = r.pool.QueryRow(ctx, insertQuery,
		flag.Project,
		flag.Stage,
		flag.Key,
		flag.Name,
		flag.Description,
		flag.Enabled,
		active,
		flag.ValidFrom,
		flag.ValidTo,
		flag.DefaultKey,
		configJSON,
	).Scan(&newID)

	if err != nil {
		return fmt.Errorf("failed to insert flag: %w", err)
	}

	return nil
}

// DeleteFlag deletes a specific flag range by ID
func (r *PostgresRepository) DeleteFlag(ctx context.Context, id int64) error {
	query := `DELETE FROM feature_flags WHERE id = $1`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete flag: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrFlagNotFound
	}

	return nil
}

// ActivateFlag activates a flag range (checks for overlapping active ranges)
func (r *PostgresRepository) ActivateFlag(ctx context.Context, id int64) error {
	// Get the flag range
	flag, err := r.GetFlagByID(ctx, id)
	if err != nil {
		return err
	}

	// Check if already active
	if flag.Active {
		return nil // Already active, nothing to do
	}

	// Check for overlapping active ranges
	hasOverlap, err := r.checkActiveOverlap(ctx, flag.Project, flag.Stage, flag.Key, flag.ValidFrom, flag.ValidTo, id)
	if err != nil {
		return fmt.Errorf("failed to check active overlap: %w", err)
	}
	if hasOverlap {
		return ErrActiveRangeOverlap
	}

	// Activate the range
	query := `UPDATE feature_flags SET active = true WHERE id = $1`
	_, err = r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to activate flag: %w", err)
	}

	return nil
}

// DeactivateFlag deactivates a flag range
func (r *PostgresRepository) DeactivateFlag(ctx context.Context, id int64) error {
	// Check if flag exists
	_, err := r.GetFlagByID(ctx, id)
	if err != nil {
		return err
	}

	query := `UPDATE feature_flags SET active = false WHERE id = $1`
	_, err = r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to deactivate flag: %w", err)
	}

	return nil
}

// CheckOverlap checks if a time range overlaps with any existing ranges (active or inactive)
func (r *PostgresRepository) CheckOverlap(ctx context.Context, project, stage, key string, validFrom time.Time, validTo *time.Time, excludeID int64) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM feature_flags
		WHERE project = $1 AND stage = $2 AND key = $3
		  AND id != $4
		  AND (
		    (valid_from < $6 OR $6 IS NULL OR valid_from < $5)
		    AND (valid_to IS NULL OR valid_to > $5)
		  )
	`

	var count int
	err := r.pool.QueryRow(ctx, query, project, stage, key, excludeID, validFrom, validTo).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check overlap: %w", err)
	}

	return count > 0, nil
}

// checkActiveOverlap checks if a time range overlaps with any active ranges
func (r *PostgresRepository) checkActiveOverlap(ctx context.Context, project, stage, key string, validFrom time.Time, validTo *time.Time, excludeID int64) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM feature_flags
		WHERE project = $1 AND stage = $2 AND key = $3
		  AND id != $4
		  AND active = true
		  AND (
		    (valid_from < $6 OR $6 IS NULL OR valid_from < $5)
		    AND (valid_to IS NULL OR valid_to > $5)
		  )
	`

	var count int
	err := r.pool.QueryRow(ctx, query, project, stage, key, excludeID, validFrom, validTo).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check active overlap: %w", err)
	}

	return count > 0, nil
}

func dbFlagToFeatureFlag(dbF dbFlag) (FeatureFlag, error) {
	var config flagConfig
	if err := json.Unmarshal(dbF.Config, &config); err != nil {
		return FeatureFlag{}, fmt.Errorf("failed to unmarshal flag config: %w", err)
	}

	return FeatureFlag{
		ID:          dbF.ID,
		Project:     dbF.Project,
		Stage:       dbF.Stage,
		Key:         dbF.Key,
		Name:        dbF.Name,
		Description: dbF.Description,
		Enabled:     dbF.Enabled,
		Active:      dbF.Active,
		ValidFrom:   dbF.ValidFrom,
		ValidTo:     dbF.ValidTo,
		DefaultKey:  dbF.DefaultKey,
		Variations:  config.Variations,
		Rules:       config.Rules,
		CreatedAt:   dbF.CreatedAt,
		UpdatedAt:   dbF.UpdatedAt,
	}, nil
}
