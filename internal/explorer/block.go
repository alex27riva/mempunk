package explorer

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"golang.org/x/sync/errgroup"
)

// BuildBlock fetches a block by hash or decimal height and builds a BlockVM.
// limit caps how many transactions are included in vm.Txs (0 = no cap).
func (e *Explorer) BuildBlock(ctx context.Context, id string, limit int) (*BlockVM, error) {
	hash, err := e.resolveBlockID(ctx, id)
	if err != nil {
		return nil, err
	}

	var blockRaw, statsRaw json.RawMessage
	g, gctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		raw, err := e.cachedCall(gctx, "blk:"+hash, "getblock", hash, 2)
		if err != nil {
			return fmt.Errorf("getblock: %w", err)
		}
		blockRaw = raw
		return nil
	})
	g.Go(func() error {
		raw, err := e.cachedCall(gctx, "blkstats:"+hash, "getblockstats", hash)
		if err != nil {
			return fmt.Errorf("getblockstats: %w", err)
		}
		statsRaw = raw
		return nil
	})
	if err := g.Wait(); err != nil {
		return nil, err
	}

	var b rpcBlockV2
	if err := unmarshal(blockRaw, &b, "block"); err != nil {
		return nil, err
	}
	var stats rpcBlockStats
	if err := unmarshal(statsRaw, &stats, "blockstats"); err != nil {
		return nil, err
	}

	vm := &BlockVM{
		Hash:          b.Hash,
		Height:        b.Height,
		Time:          b.Time,
		Age:           FormatAge(b.Time),
		Confirmations: b.Confirmations,
		TxCount:       b.NTx,
		Size:          b.Size,
		Weight:        b.Weight,
		StrippedSize:  b.StrippedSize,
		Difficulty:    b.Difficulty,
		MerkleRoot:    b.MerkleRoot,
		PrevHash:      b.PrevHash,
		NextHash:      b.NextHash,
		Nonce:         b.Nonce,
		Bits:          b.Bits,
		Version:       b.Version,
		TotalFeeSats:  stats.TotalFee,
		TotalFeeBTC:   FormatBTC(stats.TotalFee),
		AvgFeeRate:    stats.AvgFeeRate,
	}

	if e.cfg.Explorer.ShowTxDetails() {
		total := len(b.Tx)
		shown := total
		if limit > 0 && limit < total {
			shown = limit
		}
		vm.Txs = make([]TxSummaryVM, shown)
		for i := 0; i < shown; i++ {
			tx := b.Tx[i]
			outs := make([]OutputVM, len(tx.Vout))
			for j, vout := range tx.Vout {
				sats := BTCToSats(vout.Value)
				outs[j] = OutputVM{
					Index:      vout.N,
					ValueSats:  sats,
					ValueBTC:   FormatBTC(sats),
					Address:    scriptAddress(vout.ScriptPubKey),
					ScriptType: vout.ScriptPubKey.Type,
				}
			}
			vm.Txs[i] = TxSummaryVM{
				Txid:    tx.Txid,
				VSize:   tx.VSize,
				Weight:  tx.Weight,
				Outputs: outs,
			}
		}
		vm.TxShown = shown
		vm.HasMoreTxs = shown < total
	}

	return vm, nil
}

// resolveBlockID returns the block hash for a numeric height or passthrough if
// id is already a hash.
func (e *Explorer) resolveBlockID(ctx context.Context, id string) (string, error) {
	if !isNumeric(id) {
		return id, nil
	}
	height, err := strconv.ParseInt(id, 10, 64)
	if err != nil {
		return "", fmt.Errorf("invalid block id %q: %w", id, err)
	}
	raw, err := e.rpc.Call(ctx, "getblockhash", height)
	if err != nil {
		return "", fmt.Errorf("getblockhash %d: %w", height, err)
	}
	var hash string
	if err := json.Unmarshal(raw, &hash); err != nil {
		return "", fmt.Errorf("decode blockhash: %w", err)
	}
	return hash, nil
}
