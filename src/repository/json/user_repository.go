package jsonrepo

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/pkg/constants"
	"backend-yonathan/src/repository"

	"github.com/google/uuid"
)

// UserRepository is the JSON-file implementation of repository.UserRepository.
type UserRepository struct {
	mu sync.RWMutex
}

// NewUserRepository creates a new JSON-file-backed UserRepository.
func NewUserRepository() *UserRepository {
	return &UserRepository{}
}

func (r *UserRepository) filePath() string {
	dataDir := os.Getenv(constants.DataDirEnvVar)
	if dataDir == "" {
		dataDir = constants.DefaultDataDir
	}
	return filepath.Join(dataDir, constants.UsersFilename)
}

func (r *UserRepository) load() ([]models.User, error) {
	data, err := readFileFunc(r.filePath())
	if err != nil {
		if os.IsNotExist(err) {
			return []models.User{}, nil
		}
		return nil, err
	}
	var users []models.User
	if len(data) == 0 {
		return []models.User{}, nil
	}
	if err := json.Unmarshal(data, &users); err != nil {
		return nil, err
	}
	return users, nil
}

func (r *UserRepository) save(users []models.User) error {
	fp := r.filePath()
	if err := mkdirAllFunc(filepath.Dir(fp), constants.DirPermission); err != nil {
		return err
	}
	data, err := json.MarshalIndent(users, "", "  ")
	if err != nil {
		return err
	}
	return writeFileFunc(fp, data, constants.FilePermission)
}

// SaveUser persists a user. If UserId is empty a new UUID is generated.
// If a user with the same ID already exists it is replaced.
func (r *UserRepository) SaveUser(ctx context.Context, user models.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if user.UserId == "" {
		user.UserId = uuid.NewString()
	}
	users, err := r.load()
	if err != nil {
		return err
	}
	for i, u := range users {
		if u.UserId == user.UserId {
			users[i] = user
			return r.save(users)
		}
	}
	users = append(users, user)
	return r.save(users)
}

// GetUserByID returns the user with the given ID.
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	users, err := r.load()
	if err != nil {
		return models.User{}, err
	}
	for _, u := range users {
		if u.UserId == id {
			return u, nil
		}
	}
	return models.User{}, fmt.Errorf("%w: user %s", repository.ErrNotFound, id)
}

// GetUserByEmail returns the user with the given email.
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	users, err := r.load()
	if err != nil {
		return models.User{}, err
	}
	for _, u := range users {
		if u.Email == email {
			return u, nil
		}
	}
	return models.User{}, fmt.Errorf("%w: user with email %s", repository.ErrNotFound, email)
}
