package auth

import (
	"context"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// Service provides high-level user management operations.
type Service struct {
	repo Repository
}

func NewService(repo Repository) (*Service, error) {
	if repo == nil {
		return nil, fmt.Errorf("auth repository is required")
	}
	return &Service{repo: repo}, nil
}

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	records, err := s.repo.List(ctx)
	if err != nil {
		return nil, err
	}
	users := make([]User, 0, len(records))
	for _, record := range records {
		users = append(users, User{Username: record.Username, Role: record.Role})
	}
	return users, nil
}

func (s *Service) CreateUser(ctx context.Context, username, password string, role Role) (User, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" {
		return User{}, fmt.Errorf("username is required")
	}
	if password == "" {
		return User{}, fmt.Errorf("password is required")
	}
	if role == "" {
		role = RoleUser
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	record := StoredUser{
		Username:     username,
		PasswordHash: hash,
		Role:         role,
	}

	if _, err := s.repo.Create(ctx, record); err != nil {
		return User{}, err
	}

	return User{Username: username, Role: role}, nil
}

func (s *Service) EnsureUser(ctx context.Context, username, password string, role Role) error {
	if username == "" || password == "" {
		return fmt.Errorf("username and password are required")
	}

	_, err := s.repo.GetByUsername(ctx, username)
	if err == nil {
		return nil
	}
	if err != nil && err != ErrUserNotFound {
		return err
	}

	_, err = s.CreateUser(ctx, username, password, role)
	return err
}

func (s *Service) DeleteUser(ctx context.Context, username string) error {
	if strings.TrimSpace(username) == "" {
		return fmt.Errorf("username is required")
	}
	return s.repo.Delete(ctx, username)
}

func (s *Service) UpdatePassword(ctx context.Context, username, password string) error {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if password == "" {
		return fmt.Errorf("password is required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	return s.repo.UpdatePassword(ctx, username, hash)
}

func (s *Service) UpdateRole(ctx context.Context, username string, role Role) error {
	username = strings.TrimSpace(username)
	if username == "" {
		return fmt.Errorf("username is required")
	}
	if role == "" {
		role = RoleUser
	}
	return s.repo.UpdateRole(ctx, username, role)
}
