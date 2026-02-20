package memory

import (
	"context"
	"fmt"
	"sync"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/repository"

	"github.com/google/uuid"
)

// UserRepository is an in-memory implementation of repository.UserRepository for tests.
type UserRepository struct {
	mu      sync.RWMutex
	users   map[string]models.User // key: userId
	byEmail map[string]string      // email → userId
}

// NewUserRepository creates an empty in-memory UserRepository.
func NewUserRepository() *UserRepository {
	return &UserRepository{
		users:   make(map[string]models.User),
		byEmail: make(map[string]string),
	}
}

// SaveUser stores the user in memory. Generates a UserId if empty.
func (r *UserRepository) SaveUser(ctx context.Context, user models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if user.UserId == "" {
		user.UserId = uuid.New().String()
	}
	r.users[user.UserId] = user
	r.byEmail[user.Email] = user.UserId
	return nil
}

// GetUserByID fetches a user by ID.
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	u, ok := r.users[id]
	if !ok {
		return models.User{}, fmt.Errorf("%w: user %s", repository.ErrNotFound, id)
	}
	return u, nil
}

// GetUserByEmail fetches a user by email.
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	id, ok := r.byEmail[email]
	if !ok {
		return models.User{}, fmt.Errorf("%w: email %s", repository.ErrNotFound, email)
	}
	u, ok := r.users[id]
	if !ok {
		return models.User{}, fmt.Errorf("%w: user %s", repository.ErrNotFound, id)
	}
	return u, nil
}
