package auth

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"
)

type InMemoryRepository struct {
	mu    sync.RWMutex
	users map[string]StoredUser
	next  int64
}

func NewInMemoryRepository() *InMemoryRepository {
	return &InMemoryRepository{
		users: make(map[string]StoredUser),
		next:  1,
	}
}

func (r *InMemoryRepository) GetByUsername(_ context.Context, username string) (StoredUser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, ok := r.users[username]
	if !ok {
		return StoredUser{}, ErrUserNotFound
	}
	return user, nil
}

func (r *InMemoryRepository) Create(_ context.Context, user StoredUser) (StoredUser, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.users[user.Username]; exists {
		return StoredUser{}, ErrUserExists
	}

	now := time.Now()
	user.ID = r.next
	r.next++
	user.CreatedAt = now
	user.UpdatedAt = now

	r.users[user.Username] = user
	return user, nil
}

func (r *InMemoryRepository) UpdatePassword(_ context.Context, username string, passwordHash []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[username]
	if !ok {
		return ErrUserNotFound
	}

	user.PasswordHash = passwordHash
	user.UpdatedAt = time.Now()
	r.users[username] = user
	return nil
}

func (r *InMemoryRepository) UpdateRole(_ context.Context, username string, role Role) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	user, ok := r.users[username]
	if !ok {
		return ErrUserNotFound
	}

	user.Role = role
	user.UpdatedAt = time.Now()
	r.users[username] = user
	return nil
}

func (r *InMemoryRepository) Delete(_ context.Context, username string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.users[username]; !ok {
		return ErrUserNotFound
	}
	delete(r.users, username)
	return nil
}

func (r *InMemoryRepository) List(_ context.Context) ([]StoredUser, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.users) == 0 {
		return []StoredUser{}, nil
	}

	keys := make([]string, 0, len(r.users))
	for key := range r.users {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]StoredUser, 0, len(keys))
	for _, key := range keys {
		result = append(result, r.users[key])
	}
	return result, nil
}

// Seed inserts the provided users without hashing (for tests).
func (r *InMemoryRepository) Seed(users []StoredUser) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, user := range users {
		if user.Username == "" {
			continue
		}
		if _, exists := r.users[user.Username]; exists {
			continue
		}
		if user.ID == 0 {
			user.ID = r.next
			r.next++
		}
		if user.CreatedAt.IsZero() {
			user.CreatedAt = time.Now()
		}
		if user.UpdatedAt.IsZero() {
			user.UpdatedAt = user.CreatedAt
		}
		r.users[user.Username] = user
	}
}

// MustGet is a helper intended for tests.
func (r *InMemoryRepository) MustGet(username string) StoredUser {
	user, err := r.GetByUsername(context.Background(), username)
	if err != nil {
		panic(fmt.Sprintf("user %s not found: %v", username, err))
	}
	return user
}
