package quotation

import (
	"github.com/gofiber/fiber/v2"

	qthttp "github.com/onizukazaza/anc-portal-be-fake/internal/modules/quotation/adapters/http"
	qtpg "github.com/onizukazaza/anc-portal-be-fake/internal/modules/quotation/adapters/postgres"
	"github.com/onizukazaza/anc-portal-be-fake/internal/modules/quotation/app"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/module"
)

// Register wires quotation module dependencies and mounts routes.
func Register(router fiber.Router, deps module.Deps) {
	// ดึง pool จาก external database ชื่อ "meprakun"
	pool, err := deps.DB.External("meprakun")
	if err != nil {
		// ถ้าไม่มี external DB "meprakun" ลงทะเบียนไว้ ก็ข้ามไป (ไม่ register routes)
		return
	}

	repo := qtpg.NewQuotationRepository(pool)
	service := app.NewService(repo)
	controller := qthttp.NewQuotationController(service)

	group := router.Group("/quotations")
	group.Get("/", controller.ListByCustomer)
	group.Get("/:id", controller.GetByID)
}
