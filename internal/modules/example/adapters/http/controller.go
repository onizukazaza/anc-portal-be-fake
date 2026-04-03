package http

import "github.com/onizukazaza/anc-portal-be-fake/internal/modules/example/app"

// Controller — HTTP handler group for Example module.
type Controller struct {
	service *app.Service
}

func NewController(service *app.Service) *Controller {
	return &Controller{service: service}
}
