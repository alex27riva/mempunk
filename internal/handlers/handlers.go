// Package handlers contains thin Echo HTTP handlers for each mempunk view.
package handlers

import (
	"log/slog"
	"net/http"

	"github.com/alex27riva/mempunk/internal/config"
	"github.com/alex27riva/mempunk/internal/explorer"
	"github.com/alex27riva/mempunk/internal/render"
	"github.com/labstack/echo/v4"
)

// Handlers holds shared dependencies for all HTTP handlers.
type Handlers struct {
	ex      *explorer.Explorer
	cfg     *config.Config
	log     *slog.Logger
	version string
}

// New creates a Handlers instance. All parameters are required.
func New(ex *explorer.Explorer, cfg *config.Config, log *slog.Logger, version string) *Handlers {
	return &Handlers{ex: ex, cfg: cfg, log: log, version: version}
}

// page renders name with data wrapped in a render.Page built from cfg.
func (h *Handlers) page(c echo.Context, name string, title string, data any) error {
	p := h.cfg.Params()
	return c.Render(http.StatusOK, name, render.Page{
		Title:       title,
		Network:     string(h.cfg.Network),
		AccentClass: p.AccentClass,
		Version:     h.version,
		Data:        data,
	})
}

// httpError maps an error to an appropriate HTTP status. RPC errors that look
// like "not found" become 404; everything else becomes 500.
func httpError(err error) *echo.HTTPError {
	if err == nil {
		return nil
	}
	msg := err.Error()
	for _, needle := range []string{"not found", "no such", "invalid", "does not exist"} {
		for i := 0; i+len(needle) <= len(msg); i++ {
			if msg[i:i+len(needle)] == needle {
				return echo.NewHTTPError(http.StatusNotFound, msg)
			}
		}
	}
	return echo.NewHTTPError(http.StatusInternalServerError, msg)
}
