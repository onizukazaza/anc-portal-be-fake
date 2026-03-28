package app

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ─── Fake DBProvider ───

type fakeDBProvider struct {
	pools map[string]*pgxpool.Pool
	err   error
}

func (f *fakeDBProvider) External(name string) (*pgxpool.Pool, error) {
	if f.err != nil {
		return nil, f.err
	}
	pool, ok := f.pools[name]
	if !ok {
		return nil, errors.New("not found: " + name)
	}
	return pool, nil
}

func (f *fakeDBProvider) HealthCheck(_ context.Context) error {
	return f.err
}
