package memory

import (
	"context"
	"fmt"
	"sync"

	models "backend-yonathan/src/models"
	"backend-yonathan/src/repository"
)

// ExperienceRepository is an in-memory implementation of repository.ExperienceRepository for tests.
type ExperienceRepository struct {
	mu          sync.RWMutex
	experiences map[string]models.Experience
	order       []string // preserves insertion order for List
}

// NewExperienceRepository creates an empty in-memory ExperienceRepository.
func NewExperienceRepository() *ExperienceRepository {
	return &ExperienceRepository{
		experiences: make(map[string]models.Experience),
	}
}

// List returns all experiences in insertion order.
func (r *ExperienceRepository) List(ctx context.Context) ([]models.Experience, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]models.Experience, 0, len(r.order))
	for _, id := range r.order {
		if exp, ok := r.experiences[id]; ok {
			result = append(result, exp)
		}
	}
	return result, nil
}

// GetByID returns an experience by ID.
func (r *ExperienceRepository) GetByID(ctx context.Context, id string) (models.Experience, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	exp, ok := r.experiences[id]
	if !ok {
		return models.Experience{}, fmt.Errorf("%w: experience %s", repository.ErrNotFound, id)
	}
	return exp, nil
}

// Create stores a new experience in memory.
func (r *ExperienceRepository) Create(ctx context.Context, exp models.Experience) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.experiences[exp.ID] = exp
	r.order = append(r.order, exp.ID)
	return nil
}

// Update replaces an existing experience in memory.
func (r *ExperienceRepository) Update(ctx context.Context, exp models.Experience) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.experiences[exp.ID]; !ok {
		return fmt.Errorf("%w: experience %s", repository.ErrNotFound, exp.ID)
	}
	r.experiences[exp.ID] = exp
	return nil
}

// Delete removes an experience from memory.
func (r *ExperienceRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.experiences[id]; !ok {
		return fmt.Errorf("%w: experience %s", repository.ErrNotFound, id)
	}
	delete(r.experiences, id)
	newOrder := make([]string, 0, len(r.order)-1)
	for _, oid := range r.order {
		if oid != id {
			newOrder = append(newOrder, oid)
		}
	}
	r.order = newOrder
	return nil
}
