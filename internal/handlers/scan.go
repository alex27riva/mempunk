package handlers

import "github.com/labstack/echo/v4"

// UTXOs handles GET /address/:addr/utxos.
func (h *Handlers) UTXOs(c echo.Context) error {
	addr := c.Param("addr")
	if err := validateAddr(addr); err != nil {
		return err
	}
	vm, err := h.ex.PollUTXOScan(c.Request().Context(), addr)
	if err != nil {
		h.log.Error("PollUTXOScan", "addr", addr, "err", err)
		return httpError(err)
	}
	title := "UTXO scan · " + addr[:8] + "…"
	if vm.Done {
		title = "UTXOs · " + addr[:8] + "…"
	}
	return h.page(c, "utxos", title, vm)
}

// History handles GET /address/:addr/history.
func (h *Handlers) History(c echo.Context) error {
	addr := c.Param("addr")
	if err := validateAddr(addr); err != nil {
		return err
	}
	vm, err := h.ex.PollHistoryScan(c.Request().Context(), addr)
	if err != nil {
		h.log.Error("PollHistoryScan", "addr", addr, "err", err)
		return httpError(err)
	}
	title := "History scan · " + addr[:8] + "…"
	if vm.Done {
		title = "History · " + addr[:8] + "…"
	}
	return h.page(c, "history", title, vm)
}
