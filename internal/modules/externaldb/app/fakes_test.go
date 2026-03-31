package app

import (
	"context"
	"errors"

	"github.com/onizukazaza/anc-portal-be-fake/internal/database"
)

// ─── Fake DBProvider ───

type fakeDBProvider struct {
	conns map[string]database.ExternalConn
	err   error
}

func (f *fakeDBProvider) External(name string) (database.ExternalConn, error) {
	if f.err != nil {
		return nil, f.err
	}
	conn, ok := f.conns[name]
	if !ok {
		return nil, errors.New("not found: " + name)
	}
	return conn, nil
}

func (f *fakeDBProvider) HealthCheck(_ context.Context) error {
	return f.err
}
