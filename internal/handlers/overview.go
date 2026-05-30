package handlers

import "github.com/labstack/echo/v4"

// Overview handles GET /.
func (h *Handlers) Overview(c echo.Context) error {
	vm, err := h.ex.BuildOverview(c.Request().Context())
	if err != nil {
		h.log.Error("BuildOverview", "err", err)
		return httpError(err)
	}
	return h.page(c, "overview", "Overview", vm)
}
