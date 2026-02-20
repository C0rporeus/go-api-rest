package firestorerepo

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	models "backend-yonathan/src/models"
	"backend-yonathan/src/repository"

	"github.com/google/uuid"
	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const usersCollection = "users"

// UserRepository is the Firestore implementation of repository.UserRepository.
type UserRepository struct {
	client *firestore.Client
}

// NewUserRepository creates a new Firestore-backed UserRepository.
func NewUserRepository(client *firestore.Client) *UserRepository {
	return &UserRepository{client: client}
}

func (r *UserRepository) col() *firestore.CollectionRef {
	return r.client.Collection(usersCollection)
}

// SaveUser persists a user to Firestore. Generates a UserId if empty.
func (r *UserRepository) SaveUser(ctx context.Context, user models.User) error {
	if user.UserId == "" {
		user.UserId = uuid.NewString()
	}

	_, err := r.col().Doc(user.UserId).Set(ctx, map[string]interface{}{
		"userId":   user.UserId,
		"email":    user.Email,
		"password": user.Password,
		"username": user.UserName,
	})
	return err
}

// GetUserByID fetches a user by document ID.
func (r *UserRepository) GetUserByID(ctx context.Context, id string) (models.User, error) {
	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return models.User{}, fmt.Errorf("%w: user %s", repository.ErrNotFound, id)
		}
		return models.User{}, err
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		return models.User{}, err
	}
	return user, nil
}

// GetUserByEmail queries users by email field.
func (r *UserRepository) GetUserByEmail(ctx context.Context, email string) (models.User, error) {
	iter := r.col().Where("email", "==", email).Limit(1).Documents(ctx)
	defer iter.Stop()

	doc, err := iter.Next()
	if err == iterator.Done {
		return models.User{}, fmt.Errorf("%w: user with email %s", repository.ErrNotFound, email)
	}
	if err != nil {
		return models.User{}, err
	}

	var user models.User
	if err := doc.DataTo(&user); err != nil {
		return models.User{}, err
	}
	return user, nil
}
