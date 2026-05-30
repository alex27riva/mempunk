package handlers

import "github.com/labstack/echo/v4"

// Node handles GET /node.
func (h *Handlers) Node(c echo.Context) error {
	vm, err := h.ex.BuildNode(c.Request().Context())
	if err != nil {
		h.log.Error("BuildNode", "err", err)
		return httpError(err)
	}
	return h.page(c, "node", "Node Info", vm)
}
