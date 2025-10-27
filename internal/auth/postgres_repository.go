package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) *PostgresRepository {
	return &PostgresRepository{pool: pool}
}

func (r *PostgresRepository) GetByUsername(ctx context.Context, username string) (StoredUser, error) {
	const query = `
		SELECT id, username, password_hash, role, created_at, updated_at
		FROM auth_users
		WHERE username = $1
	`

	var user StoredUser
	var passwordHash string
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID,
		&user.Username,
		&passwordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredUser{}, ErrUserNotFound
		}
		return StoredUser{}, fmt.Errorf("get auth user: %w", err)
	}
	user.PasswordHash = []byte(passwordHash)
	return user, nil
}

func (r *PostgresRepository) Create(ctx context.Context, user StoredUser) (StoredUser, error) {
	const query = `
		INSERT INTO auth_users (username, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, created_at, updated_at
	`

	err := r.pool.QueryRow(ctx, query, user.Username, string(user.PasswordHash), user.Role).Scan(
		&user.ID,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return StoredUser{}, ErrUserExists
		}
		return StoredUser{}, fmt.Errorf("create auth user: %w", err)
	}
	return user, nil
}

func (r *PostgresRepository) UpdatePassword(ctx context.Context, username string, passwordHash []byte) error {
	const query = `
		UPDATE auth_users
		SET password_hash = $2, updated_at = NOW()
		WHERE username = $1
	`

	tag, err := r.pool.Exec(ctx, query, username, string(passwordHash))
	if err != nil {
		return fmt.Errorf("update auth user password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PostgresRepository) UpdateRole(ctx context.Context, username string, role Role) error {
	const query = `
		UPDATE auth_users
		SET role = $2, updated_at = NOW()
		WHERE username = $1
	`

	tag, err := r.pool.Exec(ctx, query, username, role)
	if err != nil {
		return fmt.Errorf("update auth user role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PostgresRepository) Delete(ctx context.Context, username string) error {
	const query = `DELETE FROM auth_users WHERE username = $1`

	tag, err := r.pool.Exec(ctx, query, username)
	if err != nil {
		return fmt.Errorf("delete auth user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]StoredUser, error) {
	const query = `
		SELECT id, username, password_hash, role, created_at, updated_at
		FROM auth_users
		ORDER BY username
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list auth users: %w", err)
	}
	defer rows.Close()

	users := make([]StoredUser, 0)
	for rows.Next() {
		var user StoredUser
		var passwordHash string
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&passwordHash,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan auth user: %w", err)
		}
		user.PasswordHash = []byte(passwordHash)
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate auth users: %w", err)
	}

	return users, nil
}

func isUniqueViolation(err error) bool {
	return err != nil && strings.Contains(err.Error(), "duplicate key value")
}
