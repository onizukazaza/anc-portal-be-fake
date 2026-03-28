package cmi

import (
	"github.com/gofiber/fiber/v2"

	cmihttp "github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/adapters/http"
	cmipg "github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/adapters/postgres"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/cmi/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/module"
)

// Register wires CMI module dependencies and mounts routes.
func Register(router fiber.Router, deps module.Deps) {
	// ดึง pool จาก external database ชื่อ "meprakun"
	pool, err := deps.DB.External("meprakun")
	if err != nil {
		// ถ้าไม่มี external DB "meprakun" ก็ข้ามไป
		return
	}

	repo := cmipg.NewCMIPolicyRepository(pool)
	service := app.NewService(repo)
	controller := cmihttp.NewCMIController(service)

	group := router.Group("/cmi")
	group.Get("/:job_id/request-policy-single-cmi", controller.GetPolicyByJobID)
}
