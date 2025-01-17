package handler

import (
	"github.com/vertex-center/vertex/core/port"
	"github.com/vertex-center/vertex/pkg/router"
)

type HardwareHandler struct {
	hardwareService port.HardwareService
}

func NewHardwareHandler(hardwareService port.HardwareService) port.HardwareHandler {
	return &HardwareHandler{
		hardwareService: hardwareService,
	}
}

func (h *HardwareHandler) Get(c *router.Context) {
	c.JSON(h.hardwareService.Get())
}
