package app

import (
	"context"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/example/domain"
)

// fakeRepository — in-memory fake for testing (no external dependencies).
type fakeRepository struct {
	items map[string]*domain.Example
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{items: make(map[string]*domain.Example)}
}

func (f *fakeRepository) FindByID(_ context.Context, id string) (*domain.Example, error) {
	item, ok := f.items[id]
	if !ok {
		return nil, nil
	}
	return item, nil
}

func (f *fakeRepository) FindAll(_ context.Context) ([]domain.Example, error) {
	result := make([]domain.Example, 0, len(f.items))
	for _, item := range f.items {
		result = append(result, *item)
	}
	return result, nil
}
