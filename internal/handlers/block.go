package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

// Block handles GET /block/:id where id is a block hash (64 hex chars) or height.
func (h *Handlers) Block(c echo.Context) error {
	id := c.Param("id")
	if err := validateBlockID(id); err != nil {
		return err
	}

	limit := h.cfg.Explorer.TxPageSize()
	if raw := c.QueryParam("txlimit"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n < 1 {
			return echo.NewHTTPError(http.StatusBadRequest, "txlimit must be a positive integer")
		}
		limit = n
	}

	vm, err := h.ex.BuildBlock(c.Request().Context(), id, limit)
	if err != nil {
		h.log.Error("BuildBlock", "id", id, "err", err)
		return httpError(err)
	}
	title := "Block " + id
	if vm != nil {
		title = "Block " + formatBlockTitle(vm.Height, vm.Hash)
	}
	return h.page(c, "block", title, vm)
}

func formatBlockTitle(height int64, hash string) string {
	short := hash
	if len(short) > 8 {
		short = short[:8] + "…"
	}
	return strings.Join([]string{itoa64(height), short}, " · ")
}

func itoa64(n int64) string {
	if n == 0 {
		return "0"
	}
	buf := [20]byte{}
	pos := len(buf)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		pos--
		buf[pos] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		pos--
		buf[pos] = '-'
	}
	return string(buf[pos:])
}

// validateBlockID rejects obviously bad inputs before they reach RPC.
// Valid: decimal height (all digits) or 64-char lowercase hex hash.
func validateBlockID(id string) error {
	if id == "" {
		return echo.NewHTTPError(http.StatusBadRequest, "block id required")
	}
	if isAllDigits(id) {
		return nil // height
	}
	if len(id) == 64 && isHex(id) {
		return nil // hash
	}
	return echo.NewHTTPError(http.StatusBadRequest, "block id must be a height or 64-char hex hash")
}

func isAllDigits(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return len(s) > 0
}

func isHex(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
