package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Address handles GET /address/:addr.
func (h *Handlers) Address(c echo.Context) error {
	addr := c.Param("addr")
	if err := validateAddr(addr); err != nil {
		return err
	}

	vm, err := h.ex.BuildAddress(c.Request().Context(), addr)
	if err != nil {
		h.log.Error("BuildAddress", "addr", addr, "err", err)
		return httpError(err)
	}
	return h.page(c, "address", "Address "+addr[:8]+"…", vm)
}

// validateAddr rejects obviously malformed inputs before they reach RPC.
func validateAddr(addr string) error {
	if len(addr) < 10 || len(addr) > 100 {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid address length")
	}
	for _, r := range addr {
		if !isAddrChar(r) {
			return echo.NewHTTPError(http.StatusBadRequest, "address contains invalid characters")
		}
	}
	return nil
}

func isAddrChar(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
