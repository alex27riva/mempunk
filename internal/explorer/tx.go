package explorer

import (
	"context"
	"encoding/json"
	"fmt"

	"golang.org/x/sync/errgroup"
)

// BuildTx fetches a transaction by txid and builds a TxVM. Works for both
// confirmed and mempool transactions.
func (e *Explorer) BuildTx(ctx context.Context, txid string) (*TxVM, error) {
	raw, err := e.cachedTx(ctx, txid)
	if err != nil {
		return nil, err
	}

	var tx rpcTx
	if err := unmarshal(raw, &tx, "tx"); err != nil {
		return nil, err
	}

	pretty, _ := json.MarshalIndent(raw, "", "  ")

	vm := &TxVM{
		Txid:      tx.Txid,
		Hash:      tx.Hash,
		Version:   tx.Version,
		Size:      tx.Size,
		VSize:     tx.VSize,
		Weight:    tx.Weight,
		LockTime:  tx.LockTime,
		Confirmed: tx.Confirmations > 0,
		RawJSON:   string(pretty),
	}
	if vm.Confirmed {
		vm.BlockHash = tx.BlockHash
		vm.BlockTime = tx.BlockTime
		vm.Age = FormatAge(tx.BlockTime)
		vm.Confirmations = tx.Confirmations
	}

	vm.Coinbase = len(tx.Vin) > 0 && tx.Vin[0].Coinbase != ""

	// Build inputs.
	vm.Inputs = make([]InputVM, len(tx.Vin))
	var totalIn int64
	for i, vin := range tx.Vin {
		inp := InputVM{
			Index:      i,
			Txid:       vin.Txid,
			Vout:       vin.Vout,
			Sequence:   vin.Sequence,
			IsCoinbase: vin.Coinbase != "",
		}
		if vin.Prevout != nil {
			inp.ValueSats = BTCToSats(vin.Prevout.Value)
			inp.ValueBTC = FormatBTC(inp.ValueSats)
			inp.Address = scriptAddress(vin.Prevout.ScriptPubKey)
			inp.ScriptType = vin.Prevout.ScriptPubKey.Type
			totalIn += inp.ValueSats
		}
		vm.Inputs[i] = inp
	}

	// Build outputs.
	vm.Outputs = make([]OutputVM, len(tx.Vout))
	var totalOut int64
	for i, vout := range tx.Vout {
		sats := BTCToSats(vout.Value)
		totalOut += sats
		vm.Outputs[i] = OutputVM{
			Index:      vout.N,
			ValueSats:  sats,
			ValueBTC:   FormatBTC(sats),
			Address:    scriptAddress(vout.ScriptPubKey),
			ScriptType: vout.ScriptPubKey.Type,
		}
	}

	vm.TotalOut = totalOut
	if vm.Coinbase {
		vm.TotalIn = -1
		vm.Fee = -1
	} else {
		vm.TotalIn = totalIn
		vm.Fee = totalIn - totalOut
		if vm.VSize > 0 {
			vm.FeeRate = float64(vm.Fee) / float64(vm.VSize)
		}
	}

	// Check unspent status for each output of a confirmed tx (best-effort).
	if vm.Confirmed {
		g, gctx := errgroup.WithContext(ctx)
		for i := range vm.Outputs {
			i := i
			g.Go(func() error {
				raw, err := e.rpc.Call(gctx, "gettxout", tx.Txid, vm.Outputs[i].Index)
				if err != nil {
					return nil // error = treat as unknown, not spent
				}
				vm.Outputs[i].Spent = string(raw) == "null"
				return nil
			})
		}
		_ = g.Wait()
	}

	// For mempool txs, enrich with mempoolentry data.
	if !vm.Confirmed {
		mraw, err := e.rpc.Call(ctx, "getmempoolentry", txid)
		if err == nil {
			var me rpcMempoolEntry
			if json.Unmarshal(mraw, &me) == nil {
				vm.MempoolTime = me.Time
				vm.MempoolFee = BTCToSats(me.Fees.Base)
				if vm.Fee == 0 && !vm.Coinbase {
					vm.Fee = vm.MempoolFee
				}
			}
		}
	}

	return vm, nil
}

// cachedTx fetches getrawtransaction verbosity 3, caching confirmed txs only.
func (e *Explorer) cachedTx(ctx context.Context, txid string) (json.RawMessage, error) {
	if raw, ok := e.cache.Get("tx:" + txid); ok {
		e.log.Debug("cache hit", "key", "tx:"+txid)
		return raw, nil
	}
	raw, err := e.rpc.Call(ctx, "getrawtransaction", txid, 3)
	if err != nil {
		return nil, fmt.Errorf("getrawtransaction %s: %w", txid, err)
	}
	// Only cache confirmed (immutable) txs.
	var check struct {
		BlockHash string `json:"blockhash"`
	}
	if json.Unmarshal(raw, &check) == nil && check.BlockHash != "" {
		e.cache.Put("tx:"+txid, raw)
	}
	return raw, nil
}
