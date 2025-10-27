package auth

import (
	"context"
	"time"
)

// StoredUser represents the persisted user entity including the password hash.
type StoredUser struct {
	ID           int64
	Username     string
	PasswordHash []byte
	Role         Role
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// Repository defines persistence operations for authentication users.
type Repository interface {
	GetByUsername(ctx context.Context, username string) (StoredUser, error)
	Create(ctx context.Context, user StoredUser) (StoredUser, error)
	UpdatePassword(ctx context.Context, username string, passwordHash []byte) error
	UpdateRole(ctx context.Context, username string, role Role) error
	Delete(ctx context.Context, username string) error
	List(ctx context.Context) ([]StoredUser, error)
}
