package handlers

import (
	"fmt"
	"net/http"

	"github.com/alex27riva/mempunk/internal/render"
	"github.com/labstack/echo/v4"
)

type errorData struct {
	Code    int
	Message string
}

// ErrorHandler is Echo's custom HTTPErrorHandler. Renders error.html for HTML
// requests; falls back to plain text if template rendering fails.
func (h *Handlers) ErrorHandler(err error, c echo.Context) {
	if c.Response().Committed {
		return
	}

	code := http.StatusInternalServerError
	msg := "internal server error"

	if he, ok := err.(*echo.HTTPError); ok {
		code = he.Code
		switch m := he.Message.(type) {
		case string:
			msg = m
		default:
			msg = fmt.Sprintf("%v", m)
		}
	}

	p := h.cfg.Params()
	page := render.Page{
		Title:       http.StatusText(code),
		Network:     string(h.cfg.Network),
		AccentClass: p.AccentClass,
		Data:        errorData{Code: code, Message: msg},
	}

	if err2 := c.Render(code, "error", page); err2 != nil {
		_ = c.String(code, fmt.Sprintf("%d %s", code, msg))
	}
}
