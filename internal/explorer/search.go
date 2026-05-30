package explorer

import "context"

// HexType is the kind of object a 64-char hex string resolves to.
type HexType string

const (
	HexBlock HexType = "block"
	HexTx    HexType = "tx"
)

// ProbeHex determines whether a 64-char hex string is a block hash or a txid
// by making cheap existence-check RPC calls. Returns an error if neither.
func (e *Explorer) ProbeHex(ctx context.Context, hex string) (HexType, error) {
	// getblockheader verbosity=false returns raw hex — fast block existence check.
	if _, err := e.rpc.Call(ctx, "getblockheader", hex, false); err == nil {
		return HexBlock, nil
	}
	// getrawtransaction verbosity=false returns raw hex — fast tx existence check.
	if _, err := e.rpc.Call(ctx, "getrawtransaction", hex, false); err == nil {
		return HexTx, nil
	}
	return "", &notFoundError{hex}
}

type notFoundError struct{ id string }

func (e *notFoundError) Error() string { return "not found: " + e.id }
