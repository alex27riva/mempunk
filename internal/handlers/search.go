package handlers

import (
	"net/http"
	"strings"

	"github.com/alex27riva/mempunk/internal/explorer"
	"github.com/labstack/echo/v4"
)

// Search handles GET /search?q=... and redirects to the appropriate view.
// Classification order:
//  1. All digits → /block/:q (height)
//  2. 64-char hex → probe block then tx; redirect to whichever exists, else 404
//  3. LooksLikeAddress → /address/:q
//  4. else → 404
func (h *Handlers) Search(c echo.Context) error {
	q := strings.TrimSpace(c.QueryParam("q"))
	if q == "" {
		return c.Redirect(http.StatusFound, "/")
	}

	// Height
	if isAllDigits(q) {
		return c.Redirect(http.StatusFound, "/block/"+q)
	}

	// 64-char hex: block hash or txid
	if len(q) == 64 && isHex(q) {
		kind, err := h.ex.ProbeHex(c.Request().Context(), q)
		if err != nil {
			return echo.NewHTTPError(http.StatusNotFound, "no block or transaction found for: "+q)
		}
		switch kind {
		case explorer.HexBlock:
			return c.Redirect(http.StatusFound, "/block/"+q)
		case explorer.HexTx:
			return c.Redirect(http.StatusFound, "/tx/"+q)
		}
	}

	// Address
	if h.cfg.Params().LooksLikeAddress(q) {
		return c.Redirect(http.StatusFound, "/address/"+q)
	}

	return echo.NewHTTPError(http.StatusNotFound, "no block, transaction, or address matched: "+q)
}
