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

// PostgresRepository implements the Repository interface using PostgreSQL.
type PostgresRepository struct {
	pool *pgxpool.Pool
}

// NewPostgresRepository creates a new PostgresRepository backed by pgxpool.
func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

// --- Flag identity CRUD ---

func (r *PostgresRepository) CreateFlag(ctx context.Context, f FeatureFlag) (FeatureFlag, error) {
	variationsJSON, err := json.Marshal(f.Variations)
	if err != nil {
		return FeatureFlag{}, fmt.Errorf("marshal variations: %w", err)
	}

	query := `
		INSERT INTO feature_flags (project, stage, key, name, description, enabled, default_key, variations)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, created_at, updated_at
	`

	err = r.pool.QueryRow(ctx, query,
		f.Project, f.Stage, f.Key, f.Name, f.Description, f.Enabled, f.DefaultKey, variationsJSON,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
	if err != nil {
		return FeatureFlag{}, fmt.Errorf("insert flag: %w", err)
	}

	return f, nil
}

func (r *PostgresRepository) GetFlag(ctx context.Context, project, stage, key string) (FeatureFlag, error) {
	query := `
		SELECT id, project, stage, key, name, description, enabled, default_key, variations,
		       created_at, updated_at
		FROM feature_flags
		WHERE project = $1 AND stage = $2 AND key = $3
	`

	return r.scanFlag(r.pool.QueryRow(ctx, query, project, stage, key))
}

func (r *PostgresRepository) GetFlagByID(ctx context.Context, id int64) (FeatureFlag, error) {
	query := `
		SELECT id, project, stage, key, name, description, enabled, default_key, variations,
		       created_at, updated_at
		FROM feature_flags
		WHERE id = $1
	`

	return r.scanFlag(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresRepository) UpdateFlag(ctx context.Context, f FeatureFlag) error {
	variationsJSON, err := json.Marshal(f.Variations)
	if err != nil {
		return fmt.Errorf("marshal variations: %w", err)
	}

	query := `
		UPDATE feature_flags
		SET name = $2, description = $3, enabled = $4, default_key = $5, variations = $6
		WHERE id = $1
		RETURNING updated_at
	`

	err = r.pool.QueryRow(ctx, query,
		f.ID, f.Name, f.Description, f.Enabled, f.DefaultKey, variationsJSON,
	).Scan(&f.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrFlagNotFound
		}
		return fmt.Errorf("update flag: %w", err)
	}

	return nil
}

func (r *PostgresRepository) DeleteFlag(ctx context.Context, id int64) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM feature_flags WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete flag: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrFlagNotFound
	}
	return nil
}

func (r *PostgresRepository) ListFlags(ctx context.Context, project, stage string) ([]FeatureFlag, error) {
	query := `
		SELECT id, project, stage, key, name, description, enabled, default_key, variations,
		       created_at, updated_at
		FROM feature_flags
		WHERE project = $1 AND stage = $2
		ORDER BY key
	`

	rows, err := r.pool.Query(ctx, query, project, stage)
	if err != nil {
		return nil, fmt.Errorf("list flags: %w", err)
	}
	defer rows.Close()

	flags := make([]FeatureFlag, 0)
	for rows.Next() {
		f, err := r.scanFlagRow(rows)
		if err != nil {
			return nil, err
		}
		flags = append(flags, f)
	}

	return flags, rows.Err()
}

func (r *PostgresRepository) scanFlag(row pgx.Row) (FeatureFlag, error) {
	var f FeatureFlag
	var variationsJSON []byte

	err := row.Scan(
		&f.ID, &f.Project, &f.Stage, &f.Key, &f.Name, &f.Description,
		&f.Enabled, &f.DefaultKey, &variationsJSON,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FeatureFlag{}, ErrFlagNotFound
		}
		return FeatureFlag{}, fmt.Errorf("scan flag: %w", err)
	}

	if err := json.Unmarshal(variationsJSON, &f.Variations); err != nil {
		return FeatureFlag{}, fmt.Errorf("unmarshal variations: %w", err)
	}

	return f, nil
}

func (r *PostgresRepository) scanFlagRow(rows pgx.Rows) (FeatureFlag, error) {
	var f FeatureFlag
	var variationsJSON []byte

	err := rows.Scan(
		&f.ID, &f.Project, &f.Stage, &f.Key, &f.Name, &f.Description,
		&f.Enabled, &f.DefaultKey, &variationsJSON,
		&f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		return FeatureFlag{}, fmt.Errorf("scan flag row: %w", err)
	}

	if err := json.Unmarshal(variationsJSON, &f.Variations); err != nil {
		return FeatureFlag{}, fmt.Errorf("unmarshal variations: %w", err)
	}

	return f, nil
}

// --- Range CRUD ---

func (r *PostgresRepository) CreateRange(ctx context.Context, rng FlagRange) (FlagRange, error) {
	query := `
		INSERT INTO flag_ranges (flag_id, active, valid_from, valid_to)
		VALUES ($1, $2, $3, $4)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		rng.FlagID, rng.Active, rng.ValidFrom, rng.ValidTo,
	).Scan(&rng.ID, &rng.CreatedAt, &rng.UpdatedAt)
	if err != nil {
		return FlagRange{}, fmt.Errorf("insert range: %w", err)
	}

	return rng, nil
}

func (r *PostgresRepository) GetRange(ctx context.Context, id int64) (FlagRange, error) {
	query := `
		SELECT id, flag_id, active, valid_from, valid_to, created_at, updated_at
		FROM flag_ranges
		WHERE id = $1
	`

	var rng FlagRange
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&rng.ID, &rng.FlagID, &rng.Active, &rng.ValidFrom, &rng.ValidTo,
		&rng.CreatedAt, &rng.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FlagRange{}, ErrRangeNotFound
		}
		return FlagRange{}, fmt.Errorf("get range: %w", err)
	}

	return rng, nil
}

func (r *PostgresRepository) UpdateRange(ctx context.Context, rng FlagRange) error {
	query := `
		UPDATE flag_ranges
		SET valid_from = $2, valid_to = $3
		WHERE id = $1
		RETURNING updated_at
	`

	err := r.pool.QueryRow(ctx, query,
		rng.ID, rng.ValidFrom, rng.ValidTo,
	).Scan(&rng.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRangeNotFound
		}
		return fmt.Errorf("update range: %w", err)
	}

	return nil
}

func (r *PostgresRepository) DeleteRange(ctx context.Context, id int64) error {
	result, err := r.pool.Exec(ctx, `DELETE FROM flag_ranges WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("delete range: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrRangeNotFound
	}
	return nil
}

func (r *PostgresRepository) ListRanges(ctx context.Context, flagID int64) ([]FlagRange, error) {
	query := `
		SELECT id, flag_id, active, valid_from, valid_to, created_at, updated_at
		FROM flag_ranges
		WHERE flag_id = $1
		ORDER BY valid_from DESC
	`

	rows, err := r.pool.Query(ctx, query, flagID)
	if err != nil {
		return nil, fmt.Errorf("list ranges: %w", err)
	}
	defer rows.Close()

	ranges := make([]FlagRange, 0)
	for rows.Next() {
		var rng FlagRange
		err := rows.Scan(
			&rng.ID, &rng.FlagID, &rng.Active, &rng.ValidFrom, &rng.ValidTo,
			&rng.CreatedAt, &rng.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan range: %w", err)
		}
		ranges = append(ranges, rng)
	}

	return ranges, rows.Err()
}

func (r *PostgresRepository) ActivateRange(ctx context.Context, id int64) error {
	result, err := r.pool.Exec(ctx, `UPDATE flag_ranges SET active = true WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("activate range: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrRangeNotFound
	}
	return nil
}

func (r *PostgresRepository) DeactivateRange(ctx context.Context, id int64) error {
	result, err := r.pool.Exec(ctx, `UPDATE flag_ranges SET active = false WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("deactivate range: %w", err)
	}
	if result.RowsAffected() == 0 {
		return ErrRangeNotFound
	}
	return nil
}

func (r *PostgresRepository) GetActiveRange(ctx context.Context, flagID int64, at time.Time) (FlagRange, error) {
	query := `
		SELECT id, flag_id, active, valid_from, valid_to, created_at, updated_at
		FROM flag_ranges
		WHERE flag_id = $1
		  AND active = true
		  AND valid_from <= $2
		  AND (valid_to IS NULL OR valid_to > $2)
		ORDER BY valid_from DESC
		LIMIT 1
	`

	var rng FlagRange
	err := r.pool.QueryRow(ctx, query, flagID, at).Scan(
		&rng.ID, &rng.FlagID, &rng.Active, &rng.ValidFrom, &rng.ValidTo,
		&rng.CreatedAt, &rng.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return FlagRange{}, ErrRangeNotFound
		}
		return FlagRange{}, fmt.Errorf("get active range: %w", err)
	}

	return rng, nil
}

func (r *PostgresRepository) CheckRangeOverlap(ctx context.Context, flagID int64, validFrom time.Time, validTo *time.Time, excludeID int64) (bool, error) {
	query := `
		SELECT COUNT(*)
		FROM flag_ranges
		WHERE flag_id = $1
		  AND id != $2
		  AND valid_from < COALESCE($4, 'infinity'::timestamptz)
		  AND (valid_to IS NULL OR valid_to > $3)
	`

	var count int
	err := r.pool.QueryRow(ctx, query, flagID, excludeID, validFrom, validTo).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check range overlap: %w", err)
	}

	return count > 0, nil
}

// --- Version CRUD ---

func (r *PostgresRepository) CreateVersion(ctx context.Context, v RangeVersion) (RangeVersion, error) {
	rulesJSON, err := json.Marshal(v.Rules)
	if err != nil {
		return RangeVersion{}, fmt.Errorf("marshal rules: %w", err)
	}

	// Auto-increment version within range, start as draft
	query := `
		INSERT INTO range_versions (range_id, version, status, rules)
		VALUES ($1, COALESCE((SELECT MAX(version) FROM range_versions WHERE range_id = $1), 0) + 1, 'draft', $2)
		RETURNING id, version, status, published_at, created_at
	`

	err = r.pool.QueryRow(ctx, query, v.RangeID, rulesJSON).Scan(
		&v.ID, &v.Version, &v.Status, &v.PublishedAt, &v.CreatedAt,
	)
	if err != nil {
		return RangeVersion{}, fmt.Errorf("insert version: %w", err)
	}

	return v, nil
}

func (r *PostgresRepository) GetVersion(ctx context.Context, id int64) (RangeVersion, error) {
	query := `
		SELECT id, range_id, version, status, rules, published_at, created_at
		FROM range_versions
		WHERE id = $1
	`

	return r.scanVersion(r.pool.QueryRow(ctx, query, id))
}

func (r *PostgresRepository) UpdateVersion(ctx context.Context, v RangeVersion) error {
	rulesJSON, err := json.Marshal(v.Rules)
	if err != nil {
		return fmt.Errorf("marshal rules: %w", err)
	}

	// Only allow updating draft versions
	query := `
		UPDATE range_versions
		SET rules = $2
		WHERE id = $1 AND status = 'draft'
	`

	result, err := r.pool.Exec(ctx, query, v.ID, rulesJSON)
	if err != nil {
		return fmt.Errorf("update version: %w", err)
	}
	if result.RowsAffected() == 0 {
		// Check if version exists at all
		_, err := r.GetVersion(ctx, v.ID)
		if err != nil {
			return ErrVersionNotFound
		}
		return ErrVersionNotDraft
	}

	return nil
}

func (r *PostgresRepository) DeleteDraftVersion(ctx context.Context, id int64) error {
	// Only allow deleting draft versions
	result, err := r.pool.Exec(ctx, `DELETE FROM range_versions WHERE id = $1 AND status = 'draft'`, id)
	if err != nil {
		return fmt.Errorf("delete draft version: %w", err)
	}
	if result.RowsAffected() == 0 {
		// Check if version exists at all
		_, err := r.GetVersion(ctx, id)
		if err != nil {
			return ErrVersionNotFound
		}
		return ErrCannotDeletePublished
	}
	return nil
}

func (r *PostgresRepository) ListVersions(ctx context.Context, rangeID int64) ([]RangeVersion, error) {
	query := `
		SELECT id, range_id, version, status, rules, published_at, created_at
		FROM range_versions
		WHERE range_id = $1
		ORDER BY version DESC
	`

	rows, err := r.pool.Query(ctx, query, rangeID)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()

	versions := make([]RangeVersion, 0)
	for rows.Next() {
		v, err := r.scanVersionRow(rows)
		if err != nil {
			return nil, err
		}
		versions = append(versions, v)
	}

	return versions, rows.Err()
}

func (r *PostgresRepository) GetPublishedVersion(ctx context.Context, rangeID int64) (RangeVersion, error) {
	query := `
		SELECT id, range_id, version, status, rules, published_at, created_at
		FROM range_versions
		WHERE range_id = $1 AND status = 'published'
		ORDER BY version DESC
		LIMIT 1
	`

	v, err := r.scanVersion(r.pool.QueryRow(ctx, query, rangeID))
	if err != nil {
		if errors.Is(err, ErrVersionNotFound) {
			return RangeVersion{}, ErrNoPublishedVersion
		}
		return RangeVersion{}, err
	}

	return v, nil
}

func (r *PostgresRepository) PublishVersion(ctx context.Context, id int64) error {
	query := `
		UPDATE range_versions
		SET status = 'published', published_at = NOW()
		WHERE id = $1 AND status = 'draft'
	`

	result, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return fmt.Errorf("publish version: %w", err)
	}
	if result.RowsAffected() == 0 {
		_, err := r.GetVersion(ctx, id)
		if err != nil {
			return ErrVersionNotFound
		}
		return ErrVersionNotDraft
	}

	return nil
}

func (r *PostgresRepository) scanVersion(row pgx.Row) (RangeVersion, error) {
	var v RangeVersion
	var rulesJSON []byte

	err := row.Scan(
		&v.ID, &v.RangeID, &v.Version, &v.Status, &rulesJSON, &v.PublishedAt, &v.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return RangeVersion{}, ErrVersionNotFound
		}
		return RangeVersion{}, fmt.Errorf("scan version: %w", err)
	}

	if err := json.Unmarshal(rulesJSON, &v.Rules); err != nil {
		return RangeVersion{}, fmt.Errorf("unmarshal rules: %w", err)
	}

	return v, nil
}

func (r *PostgresRepository) scanVersionRow(rows pgx.Rows) (RangeVersion, error) {
	var v RangeVersion
	var rulesJSON []byte

	err := rows.Scan(
		&v.ID, &v.RangeID, &v.Version, &v.Status, &rulesJSON, &v.PublishedAt, &v.CreatedAt,
	)
	if err != nil {
		return RangeVersion{}, fmt.Errorf("scan version row: %w", err)
	}

	if err := json.Unmarshal(rulesJSON, &v.Rules); err != nil {
		return RangeVersion{}, fmt.Errorf("unmarshal rules: %w", err)
	}

	return v, nil
}
