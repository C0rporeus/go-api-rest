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
)

// Injectable file I/O vars (swap in tests).
var (
	readFileFunc = func(path string) ([]byte, error) {
		return os.ReadFile(path)
	}
	writeFileFunc = func(path string, data []byte, perm os.FileMode) error {
		return os.WriteFile(path, data, perm)
	}
	mkdirAllFunc = func(path string, perm os.FileMode) error {
		return os.MkdirAll(path, perm)
	}
)

// ExperienceRepository is the JSON-file implementation of repository.ExperienceRepository.
type ExperienceRepository struct {
	mu sync.RWMutex
}

// NewExperienceRepository creates a new JSON-file-backed ExperienceRepository.
func NewExperienceRepository() *ExperienceRepository {
	return &ExperienceRepository{}
}

func (r *ExperienceRepository) filePath() string {
	dataDir := os.Getenv(constants.DataDirEnvVar)
	if dataDir == "" {
		dataDir = constants.DefaultDataDir
	}
	return filepath.Join(dataDir, constants.ExperiencesFilename)
}

func (r *ExperienceRepository) load() ([]models.Experience, error) {
	data, err := readFileFunc(r.filePath())
	if err != nil {
		if os.IsNotExist(err) {
			return []models.Experience{}, nil
		}
		return nil, err
	}
	var experiences []models.Experience
	if len(data) == 0 {
		return []models.Experience{}, nil
	}
	if err := json.Unmarshal(data, &experiences); err != nil {
		return nil, err
	}
	return experiences, nil
}

func (r *ExperienceRepository) save(experiences []models.Experience) error {
	fp := r.filePath()
	if err := mkdirAllFunc(filepath.Dir(fp), constants.DirPermission); err != nil {
		return err
	}
	data, err := json.MarshalIndent(experiences, "", "  ")
	if err != nil {
		return err
	}
	return writeFileFunc(fp, data, constants.FilePermission)
}

// List returns all experiences.
func (r *ExperienceRepository) List(ctx context.Context) ([]models.Experience, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.load()
}

// GetByID returns an experience by ID.
func (r *ExperienceRepository) GetByID(ctx context.Context, id string) (models.Experience, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	experiences, err := r.load()
	if err != nil {
		return models.Experience{}, err
	}
	for _, exp := range experiences {
		if exp.ID == id {
			return exp, nil
		}
	}
	return models.Experience{}, fmt.Errorf("%w: experience %s", repository.ErrNotFound, id)
}

// Create appends a new experience and persists.
func (r *ExperienceRepository) Create(ctx context.Context, exp models.Experience) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	experiences, err := r.load()
	if err != nil {
		return err
	}
	experiences = append(experiences, exp)
	return r.save(experiences)
}

// Update replaces an existing experience by ID and persists.
func (r *ExperienceRepository) Update(ctx context.Context, exp models.Experience) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	experiences, err := r.load()
	if err != nil {
		return err
	}
	for i, item := range experiences {
		if item.ID == exp.ID {
			experiences[i] = exp
			return r.save(experiences)
		}
	}
	return fmt.Errorf("%w: experience %s", repository.ErrNotFound, exp.ID)
}

// Delete removes an experience by ID and persists.
func (r *ExperienceRepository) Delete(ctx context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	experiences, err := r.load()
	if err != nil {
		return err
	}
	filtered := make([]models.Experience, 0, len(experiences))
	found := false
	for _, item := range experiences {
		if item.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, item)
	}
	if !found {
		return fmt.Errorf("%w: experience %s", repository.ErrNotFound, id)
	}
	return r.save(filtered)
}
