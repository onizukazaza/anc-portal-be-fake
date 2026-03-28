package importer

import (
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type RunRequest struct {
	ServiceType string
	FilePath    string
	DB          *pgxpool.Pool
}

func Run(req RunRequest) error {
	switch req.ServiceType {
	case "insurer":
		return ImportInsurer(req.DB, req.FilePath)
	case "insurer_installment":
		return ImportInsurerInstallment(req.DB, req.FilePath)
	case "province":
		return ImportProvince(req.DB, req.FilePath)
	case "user":
		return ImportUser(req.DB, req.FilePath)
	default:
		return fmt.Errorf("unsupported service_type: %s", req.ServiceType)
	}
}
