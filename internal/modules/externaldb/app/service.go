package app

import (
	"context"

	appOtel "github.com/onizukazaza/anc-portal-be-fake/pkg/otel"

	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/externaldb/domain"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/externaldb/ports"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/enum"
)

// Service ทดสอบ connectivity + diagnostic ของ external databases ทั้งหมด.
type Service struct {
	db    ports.DBProvider
	names []string
}

func NewService(db ports.DBProvider, names []string) *Service {
	return &Service{db: db, names: names}
}

// CheckAll ตรวจสอบทุก external database ที่ลงทะเบียนไว้.
func (s *Service) CheckAll(ctx context.Context) []domain.DBStatus {
	ctx, span := appOtel.Tracer(appOtel.TracerExtDBService).Start(ctx, "CheckAll")
	defer span.End()

	results := make([]domain.DBStatus, 0, len(s.names))
	for _, name := range s.names {
		results = append(results, s.check(ctx, name))
	}
	return results
}

// CheckByName ตรวจสอบ external database ตามชื่อ.
func (s *Service) CheckByName(ctx context.Context, name string) domain.DBStatus {
	ctx, span := appOtel.Tracer(appOtel.TracerExtDBService).Start(ctx, "CheckByName")
	defer span.End()

	return s.check(ctx, name)
}

func (s *Service) check(ctx context.Context, name string) domain.DBStatus {
	ctx, span := appOtel.Tracer(appOtel.TracerExtDBService).Start(ctx, "check")
	defer span.End()

	pool, err := s.db.External(name)
	if err != nil {
		return domain.DBStatus{Name: name, Status: enum.DBError, Error: err.Error()}
	}

	var currentDB, version string
	if err := pool.QueryRow(ctx, "SELECT current_database(), version()").Scan(&currentDB, &version); err != nil {
		return domain.DBStatus{Name: name, Status: enum.DBUnhealthy, Error: err.Error()}
	}

	return domain.DBStatus{
		Name:            name,
		Status:          enum.DBHealthy,
		CurrentDatabase: currentDB,
		Version:         version,
	}
}
