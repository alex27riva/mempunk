package explorer

import (
	"context"
	"fmt"
)

// BuildAddress validates an address and builds an AddressVM. The two opt-in
// slow scans (UTXO set, block history) are handled separately in handlers.
func (e *Explorer) BuildAddress(ctx context.Context, addr string) (*AddressVM, error) {
	raw, err := e.rpc.Call(ctx, "validateaddress", addr)
	if err != nil {
		return nil, fmt.Errorf("validateaddress %s: %w", addr, err)
	}
	var va rpcValidateAddress
	if err := unmarshal(raw, &va, "validateaddress"); err != nil {
		return nil, err
	}
	return &AddressVM{
		Address:   addr,
		IsValid:   va.IsValid,
		Network:   string(e.cfg.Network),
		IsScript:  va.IsScript,
		IsWitness: va.IsWitness,
	}, nil
}
