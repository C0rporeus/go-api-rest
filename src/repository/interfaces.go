package repository

import (
	"context"
	"errors"

	models "backend-yonathan/src/models"
)

// ErrNotFound is returned when a requested resource does not exist.
var ErrNotFound = errors.New("not found")

// UserRepository defines the data access contract for user persistence.
type UserRepository interface {
	SaveUser(ctx context.Context, user models.User) error
	GetUserByID(ctx context.Context, id string) (models.User, error)
	GetUserByEmail(ctx context.Context, email string) (models.User, error)
}

// ExperienceRepository defines the data access contract for experience/skill persistence.
type ExperienceRepository interface {
	List(ctx context.Context) ([]models.Experience, error)
	GetByID(ctx context.Context, id string) (models.Experience, error)
	Create(ctx context.Context, exp models.Experience) error
	Update(ctx context.Context, exp models.Experience) error
	Delete(ctx context.Context, id string) error
}
