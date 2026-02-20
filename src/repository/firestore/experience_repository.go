package firestorerepo

import (
	"context"
	"fmt"

	"cloud.google.com/go/firestore"
	models "backend-yonathan/src/models"
	"backend-yonathan/src/repository"

	"google.golang.org/api/iterator"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const experiencesCollection = "experiences"

// ExperienceRepository is the Firestore implementation of repository.ExperienceRepository.
type ExperienceRepository struct {
	client *firestore.Client
}

// NewExperienceRepository creates a new Firestore-backed ExperienceRepository.
func NewExperienceRepository(client *firestore.Client) *ExperienceRepository {
	return &ExperienceRepository{client: client}
}

func (r *ExperienceRepository) col() *firestore.CollectionRef {
	return r.client.Collection(experiencesCollection)
}

// List returns all experiences.
func (r *ExperienceRepository) List(ctx context.Context) ([]models.Experience, error) {
	iter := r.col().Documents(ctx)
	defer iter.Stop()

	experiences := make([]models.Experience, 0)
	for {
		doc, err := iter.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return nil, err
		}

		var exp models.Experience
		if err := doc.DataTo(&exp); err != nil {
			return nil, err
		}
		experiences = append(experiences, exp)
	}
	return experiences, nil
}

// GetByID returns an experience by document ID.
func (r *ExperienceRepository) GetByID(ctx context.Context, id string) (models.Experience, error) {
	doc, err := r.col().Doc(id).Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return models.Experience{}, fmt.Errorf("%w: experience %s", repository.ErrNotFound, id)
		}
		return models.Experience{}, err
	}

	var exp models.Experience
	if err := doc.DataTo(&exp); err != nil {
		return models.Experience{}, err
	}
	return exp, nil
}

// Create persists a new experience using its ID as the document key.
func (r *ExperienceRepository) Create(ctx context.Context, exp models.Experience) error {
	_, err := r.col().Doc(exp.ID).Set(ctx, exp)
	return err
}

// Update replaces an existing experience.
func (r *ExperienceRepository) Update(ctx context.Context, exp models.Experience) error {
	docRef := r.col().Doc(exp.ID)
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("%w: experience %s", repository.ErrNotFound, exp.ID)
		}
		return err
	}
	_ = doc

	_, err = docRef.Set(ctx, exp)
	return err
}

// Delete removes an experience by ID.
func (r *ExperienceRepository) Delete(ctx context.Context, id string) error {
	docRef := r.col().Doc(id)
	doc, err := docRef.Get(ctx)
	if err != nil {
		if status.Code(err) == codes.NotFound {
			return fmt.Errorf("%w: experience %s", repository.ErrNotFound, id)
		}
		return err
	}
	_ = doc

	_, err = docRef.Delete(ctx)
	return err
}
