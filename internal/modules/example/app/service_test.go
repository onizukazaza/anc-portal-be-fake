package app

import (
	"context"
	"testing"
	"time"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/example/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/testkit"
)

func TestGetByID_Found(t *testing.T) {
	repo := newFakeRepository()
	repo.items["ex-1"] = &domain.Example{
		ID:        "ex-1",
		Name:      "Test Item",
		CreatedAt: time.Now(),
	}

	svc := NewService(repo)
	got, err := svc.GetByID(context.Background(), "ex-1")

	testkit.NoError(t, err)
	testkit.NotNil(t, got)
	testkit.Equal(t, "ex-1", got.ID)
}

func TestGetByID_NotFound(t *testing.T) {
	repo := newFakeRepository()
	svc := NewService(repo)

	_, err := svc.GetByID(context.Background(), "nonexistent")

	testkit.Error(t, err)
	testkit.ErrorIs(t, err, ErrNotFound)
}
