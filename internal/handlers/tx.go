package handlers

import (
	"net/http"

	"github.com/labstack/echo/v4"
)

// Tx handles GET /tx/:txid.
func (h *Handlers) Tx(c echo.Context) error {
	txid := c.Param("txid")
	if len(txid) != 64 || !isHex(txid) {
		return echo.NewHTTPError(http.StatusBadRequest, "txid must be 64 hex characters")
	}

	vm, err := h.ex.BuildTx(c.Request().Context(), txid)
	if err != nil {
		h.log.Error("BuildTx", "txid", txid, "err", err)
		return httpError(err)
	}
	return h.page(c, "tx", "Tx "+txid[:8]+"…", vm)
}
