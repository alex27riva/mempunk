package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// healthResponse is the JSON body for GET /health.
type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// Health handles GET /health.
func (h *Handlers) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, healthResponse{
		Status:  "ok",
		Version: h.version,
	})
}
